package db

import "database/sql"

type categorySibling struct {
	ID int64
	seedNode
}

// planCategoryAdditions keeps existing identities and custom ordering, inserting
// missing defaults before the catch-all category. Repeated calls are a no-op.
func planCategoryAdditions(existing []categorySibling, nodes []seedNode) []categorySibling {
	result := append([]categorySibling(nil), existing...)
	for _, node := range nodes {
		found := false
		order := node.Order
		for _, sibling := range result {
			if sibling.Name == node.Name {
				found = true
			}
			if sibling.Order >= order {
				order = sibling.Order + 1
			}
		}
		if found {
			continue
		}
		for _, sibling := range result {
			if sibling.Name == "其他" || sibling.Name == "其他收入" {
				order = sibling.Order
				break
			}
		}
		for i := range result {
			if result[i].Order >= order {
				result[i].Order++
			}
		}
		node.Order = order
		result = append(result, categorySibling{seedNode: node})
	}
	return result
}

func ensureCategoryAdditions(tx *sql.Tx, userID int64, kind string, parentID sql.NullInt64, nodes []seedNode) error {
	rows, err := tx.Query(`SELECT id, name, sort_order FROM categories
		WHERE user_id = $1 AND type = $2 AND scope = 'normal'
		AND parent_id IS NOT DISTINCT FROM $3 ORDER BY sort_order, id`, userID, kind, parentID)
	if err != nil {
		return err
	}
	var existing []categorySibling
	for rows.Next() {
		var sibling categorySibling
		if err := rows.Scan(&sibling.ID, &sibling.Name, &sibling.Order); err != nil {
			rows.Close()
			return err
		}
		existing = append(existing, sibling)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	oldOrders := map[int64]int{}
	for _, sibling := range existing {
		oldOrders[sibling.ID] = sibling.Order
	}
	planned := planCategoryAdditions(existing, nodes)
	for i := range planned {
		sibling := &planned[i]
		if sibling.ID == 0 {
			err = tx.QueryRow(`INSERT INTO categories
				(user_id, name, type, icon, color_hex, sort_order, is_system, parent_id, scope)
				VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7, 'normal') RETURNING id`,
				userID, sibling.Name, kind, sibling.Icon, sibling.Color, sibling.Order, parentID).Scan(&sibling.ID)
		} else if oldOrders[sibling.ID] != sibling.Order {
			_, err = tx.Exec(`UPDATE categories SET sort_order = $1 WHERE id = $2 AND user_id = $3`, sibling.Order, sibling.ID, userID)
		}
		if err != nil {
			return err
		}
	}
	for _, node := range nodes {
		if len(node.Children) == 0 {
			continue
		}
		for _, sibling := range planned {
			if sibling.Name == node.Name {
				if err := ensureCategoryAdditions(tx, userID, kind, sql.NullInt64{Int64: sibling.ID, Valid: true}, node.Children); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

// Only normal categories are expanded; historical transactions and trip categories
// retain their references. Per-user transactions prevent partially applied trees.
func migrateCategoryExpansion(db *sql.DB) error {
	rows, err := db.Query(`SELECT DISTINCT user_id FROM categories WHERE scope = 'normal' AND parent_id IS NULL`)
	if err != nil {
		return err
	}
	var userIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		userIDs = append(userIDs, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var incomes []seedNode
	for _, node := range incomeTree {
		if node.Name == "报销" || node.Name == "二手卖出" || node.Name == "礼金红包" || node.Name == "奖励返现" {
			incomes = append(incomes, node)
		}
	}
	for _, userID := range userIDs {
		err := func() error {
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()
			if _, err := tx.Exec(`SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
				return err
			}
			var transportID int64
			err = tx.QueryRow(`SELECT id FROM categories WHERE user_id = $1 AND type = 'expense'
				AND scope = 'normal' AND parent_id IS NULL AND name = '交通' ORDER BY is_system DESC, id LIMIT 1`, userID).Scan(&transportID)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == nil {
				if err := ensureCategoryAdditions(tx, userID, "expense", sql.NullInt64{Int64: transportID, Valid: true}, []seedNode{
					{Name: "电瓶车充电", Icon: "🔋", Color: "#10B981", Order: 104},
				}); err != nil {
					return err
				}
			}
			if err := ensureCategoryAdditions(tx, userID, "income", sql.NullInt64{}, incomes); err != nil {
				return err
			}
			return tx.Commit()
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
