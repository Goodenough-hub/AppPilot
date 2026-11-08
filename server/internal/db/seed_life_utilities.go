package db

import (
	"database/sql"
	"fmt"
)

// lifeSeedChildren 返回 seed 中「生活」的标准子分类（含顺序）。
func lifeSeedChildren() []seedNode {
	for _, node := range expenseTree {
		if node.Name == "生活" {
			return node.Children
		}
	}
	return nil
}

// lifeChildPlan 描述把一个用户的「生活」子分类对齐到 seed 所需的动作。
type lifeChildPlan struct {
	Inserts  []seedNode    // 缺失的 seed 子分类（带目标 sort_order）
	Reorders map[int64]int // 已有子分类 id → 新 sort_order
}

func (p lifeChildPlan) empty() bool {
	return len(p.Inserts) == 0 && len(p.Reorders) == 0
}

// planLifeChildren 补齐缺失的 seed 子分类，并把 seed 内的子分类按 seed 顺序
// 重排；用户自建的其他子分类不动。幂等：已对齐时返回空计划。
func planLifeChildren(existing []categorySibling, desired []seedNode) lifeChildPlan {
	var plan lifeChildPlan
	byName := make(map[string]categorySibling, len(existing))
	for _, sibling := range existing {
		byName[sibling.Name] = sibling
	}
	for _, node := range desired {
		sibling, ok := byName[node.Name]
		if !ok {
			plan.Inserts = append(plan.Inserts, node)
			continue
		}
		if sibling.Order != node.Order {
			if plan.Reorders == nil {
				plan.Reorders = map[int64]int{}
			}
			plan.Reorders[sibling.ID] = node.Order
		}
	}
	return plan
}

// migrateLifeUtilityChildren 老用户「生活」补「饮用水」「话费」「文印」，
// 并按 seed 顺序重排。幂等。
func migrateLifeUtilityChildren(db *sql.DB) error {
	desired := lifeSeedChildren()
	if len(desired) == 0 {
		return nil
	}
	rows, err := db.Query(`SELECT id, user_id FROM categories
		WHERE name = '生活' AND type = 'expense' AND scope = 'normal' AND parent_id IS NULL`)
	if err != nil {
		return err
	}
	type lifeRoot struct {
		ID     int64
		UserID int64
	}
	var roots []lifeRoot
	for rows.Next() {
		var root lifeRoot
		if err := rows.Scan(&root.ID, &root.UserID); err != nil {
			rows.Close()
			return err
		}
		roots = append(roots, root)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}

	for _, root := range roots {
		if err := applyLifeChildPlan(db, root.ID, root.UserID, desired); err != nil {
			return fmt.Errorf("life children for user %d: %w", root.UserID, err)
		}
	}
	return nil
}

func applyLifeChildPlan(db *sql.DB, lifeID, userID int64, desired []seedNode) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	childRows, err := tx.Query(`SELECT id, name, sort_order FROM categories
		WHERE user_id = $1 AND type = 'expense' AND scope = 'normal' AND parent_id = $2
		ORDER BY sort_order, id`, userID, lifeID)
	if err != nil {
		return err
	}
	var existing []categorySibling
	for childRows.Next() {
		var sibling categorySibling
		if err := childRows.Scan(&sibling.ID, &sibling.Name, &sibling.Order); err != nil {
			childRows.Close()
			return err
		}
		existing = append(existing, sibling)
	}
	err = childRows.Err()
	childRows.Close()
	if err != nil {
		return err
	}

	plan := planLifeChildren(existing, desired)
	if plan.empty() {
		return nil
	}
	for _, node := range plan.Inserts {
		if _, err := tx.Exec(`INSERT INTO categories
			(user_id, name, type, icon, color_hex, sort_order, is_system, parent_id, scope)
			VALUES ($1, $2, 'expense', $3, $4, $5, TRUE, $6, 'normal')`,
			userID, node.Name, node.Icon, node.Color, node.Order, lifeID); err != nil {
			return fmt.Errorf("insert 生活·%s: %w", node.Name, err)
		}
	}
	for id, order := range plan.Reorders {
		if _, err := tx.Exec(`UPDATE categories SET sort_order = $1 WHERE id = $2 AND user_id = $3`,
			order, id, userID); err != nil {
			return fmt.Errorf("reorder 生活 child %d: %w", id, err)
		}
	}
	return tx.Commit()
}
