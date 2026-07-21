package db

import (
	"database/sql"
	"fmt"
)

type seedNode struct {
	Name     string
	Icon     string
	Color    string
	Order    int
	Children []seedNode
}

var expenseTree = []seedNode{
	{Name: "餐饮", Icon: "🍴", Color: "#FF6B35", Order: 0, Children: []seedNode{
		{Name: "早餐", Icon: "🥐", Color: "#FF6B35", Order: 100},
		{Name: "午餐", Icon: "🍱", Color: "#F59E0B", Order: 101},
		{Name: "晚餐", Icon: "🍽️", Color: "#EF4444", Order: 102},
		{Name: "夜宵", Icon: "🌙", Color: "#6366F1", Order: 103},
		{Name: "小吃", Icon: "🍡", Color: "#8B5CF6", Order: 104},
		{Name: "饮料", Icon: "🥤", Color: "#06B6D4", Order: 105},
		{Name: "聚餐AA", Icon: "👥", Color: "#8B5CF6", Order: 106},
		{Name: "聚餐请客", Icon: "❤️", Color: "#EC4899", Order: 107},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 108},
	}},
	{Name: "交通", Icon: "🚗", Color: "#3B82F6", Order: 1, Children: []seedNode{
		{Name: "地铁", Icon: "🚇", Color: "#3B82F6", Order: 100},
		{Name: "公交", Icon: "🚌", Color: "#10B981", Order: 101},
		{Name: "打车", Icon: "🚕", Color: "#F59E0B", Order: 102},
		{Name: "高铁", Icon: "🚄", Color: "#6366F1", Order: 103},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 104},
	}},
	{Name: "购物", Icon: "🛍️", Color: "#8B5CF6", Order: 2, Children: []seedNode{
		{Name: "京东", Icon: "📦", Color: "#EF4444", Order: 100},
		{Name: "淘宝", Icon: "🛍️", Color: "#F59E0B", Order: 101},
		{Name: "拼多多", Icon: "🛒", Color: "#EF4444", Order: 102},
		{Name: "抖音", Icon: "🎵", Color: "#6B7280", Order: 103},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 104},
	}},
	{Name: "住房", Icon: "🏠", Color: "#10B981", Order: 3, Children: []seedNode{
		{Name: "租金", Icon: "🔑", Color: "#10B981", Order: 100},
		{Name: "水电", Icon: "⚡", Color: "#F59E0B", Order: 101},
		{Name: "物业", Icon: "🏢", Color: "#3B82F6", Order: 102},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 103},
	}},
	{Name: "娱乐", Icon: "🎮", Color: "#F59E0B", Order: 4, Children: []seedNode{
		{Name: "游戏", Icon: "🎮", Color: "#F59E0B", Order: 100, Children: []seedNode{
			{Name: "王者荣耀", Icon: "👑", Color: "#F59E0B", Order: 201},
			{Name: "和平精英", Icon: "🎯", Color: "#10B981", Order: 202},
			{Name: "原神", Icon: "✨", Color: "#3B82F6", Order: 203},
			{Name: "Steam", Icon: "🔥", Color: "#EF4444", Order: 204},
			{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 205},
		}},
		{Name: "影视", Icon: "🎬", Color: "#8B5CF6", Order: 200, Children: []seedNode{
			{Name: "腾讯视频", Icon: "📺", Color: "#10B981", Order: 301},
			{Name: "B站", Icon: "▶️", Color: "#EF4444", Order: 302},
			{Name: "爱奇艺", Icon: "🎬", Color: "#10B981", Order: 303},
			{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 304},
		}},
		{Name: "音乐", Icon: "🎵", Color: "#06B6D4", Order: 300, Children: []seedNode{
			{Name: "Apple Music", Icon: "🎵", Color: "#EF4444", Order: 401},
			{Name: "网易云音乐", Icon: "🎙️", Color: "#EF4444", Order: 402},
			{Name: "QQ音乐", Icon: "🎶", Color: "#3B82F6", Order: 403},
			{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 404},
		}},
		{Name: "健身", Icon: "🏃", Color: "#10B981", Order: 400, Children: []seedNode{
			{Name: "健身房", Icon: "🏢", Color: "#10B981", Order: 601},
			{Name: "私教", Icon: "✅", Color: "#3B82F6", Order: 602},
			{Name: "团课", Icon: "👥", Color: "#8B5CF6", Order: 603},
			{Name: "跑步", Icon: "🏃", Color: "#F59E0B", Order: 604},
			{Name: "游泳", Icon: "🏊", Color: "#3B82F6", Order: 605},
			{Name: "瑜伽", Icon: "🧘", Color: "#EC4899", Order: 606},
			{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 607},
		}},
		{Name: "网盘", Icon: "☁️", Color: "#3B82F6", Order: 500, Children: []seedNode{
			{Name: "百度网盘", Icon: "☁️", Color: "#3B82F6", Order: 501},
			{Name: "阿里网盘", Icon: "☁️", Color: "#F59E0B", Order: 502},
			{Name: "天翼网盘", Icon: "☁️", Color: "#EF4444", Order: 503},
			{Name: "夸克网盘", Icon: "☁️", Color: "#8B5CF6", Order: 504},
			{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 505},
		}},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 600},
	}},
	{Name: "医疗", Icon: "⚕️", Color: "#EF4444", Order: 5, Children: []seedNode{
		{Name: "挂号", Icon: "⚕️", Color: "#EF4444", Order: 100},
		{Name: "药品", Icon: "💊", Color: "#F59E0B", Order: 101},
		{Name: "体检", Icon: "🩺", Color: "#10B981", Order: 102},
		{Name: "牙科", Icon: "🦷", Color: "#3B82F6", Order: 103},
		{Name: "眼科", Icon: "👁️", Color: "#8B5CF6", Order: 104},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 105},
	}},
	{Name: "教育", Icon: "📚", Color: "#6366F1", Order: 6, Children: []seedNode{
		{Name: "培训", Icon: "🎓", Color: "#6366F1", Order: 100},
		{Name: "书籍", Icon: "📚", Color: "#8B5CF6", Order: 101},
		{Name: "学费", Icon: "💳", Color: "#3B82F6", Order: 102},
		{Name: "课程", Icon: "📺", Color: "#F59E0B", Order: 103},
		{Name: "考试报名", Icon: "📄", Color: "#EF4444", Order: 104},
		{Name: "微信读书订阅", Icon: "📖", Color: "#10B981", Order: 105},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 106},
	}},
	{Name: "数字服务", Icon: "🌐", Color: "#06B6D4", Order: 7, Children: []seedNode{
		{Name: "服务器", Icon: "🖥️", Color: "#3B82F6", Order: 100},
		{Name: "域名", Icon: "🌍", Color: "#10B981", Order: 101},
		{Name: "软件订阅", Icon: "📦", Color: "#8B5CF6", Order: 102},
		{Name: "云服务", Icon: "☁️", Color: "#F59E0B", Order: 103},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 104},
	}},
	{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 8},
}

var incomeTree = []seedNode{
	{Name: "工资", Icon: "💰", Color: "#10B981", Order: 0},
	{Name: "投资", Icon: "📈", Color: "#3B82F6", Order: 1, Children: []seedNode{
		{Name: "余额宝收益", Icon: "💰", Color: "#10B981", Order: 100},
		{Name: "零钱通收益", Icon: "💵", Color: "#10B981", Order: 101},
		{Name: "理财收益", Icon: "📈", Color: "#10B981", Order: 102},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 103},
	}},
	{Name: "兼职", Icon: "💼", Color: "#8B5CF6", Order: 2},
	{Name: "其他收入", Icon: "⋯", Color: "#6B7280", Order: 3},
}

type defaultAccount struct {
	Type     string
	Name     string
	Icon     string
	Color    string
	Order    int
	Children []defaultAccount
}

var defaultAccounts = []defaultAccount{
	{Type: "alipay", Name: "支付宝", Icon: "支", Color: "#1677FF", Order: 0, Children: []defaultAccount{
		{Type: "alipay", Name: "支付宝·余额", Icon: "支", Color: "#1677FF", Order: 0},
		{Type: "alipay", Name: "支付宝·余额宝", Icon: "💰", Color: "#10B981", Order: 1},
		{Type: "alipay", Name: "支付宝·理财", Icon: "📈", Color: "#F59E0B", Order: 2},
	}},
	{Type: "wechat", Name: "微信", Icon: "微", Color: "#07C160", Order: 1, Children: []defaultAccount{
		{Type: "wechat", Name: "微信·零钱", Icon: "微", Color: "#07C160", Order: 0},
		{Type: "wechat", Name: "微信·零钱通", Icon: "💰", Color: "#10B981", Order: 1},
		{Type: "wechat", Name: "微信·理财通", Icon: "📈", Color: "#F59E0B", Order: 2},
	}},
	{Type: "unionpay", Name: "云闪付", Icon: "银", Color: "#E60012", Order: 2},
	{Type: "fixed", Name: "定期", Icon: "定", Color: "#F59E0B", Order: 3},
}

// SeedForUser 在指定 user_id 下种子 78 个分类 + 默认账户结构。
// 支付宝/微信为分组容器，下挂余额/理财等子账户；云闪付/定期为叶子账户。
// 幂等：若该用户已有分类则跳过。
func SeedForUser(db *sql.DB, userID int64) error {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM categories WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("check existing: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, root := range expenseTree {
		if err := insertTree(tx, userID, root, "expense", nil); err != nil {
			return err
		}
	}
	for _, root := range incomeTree {
		if err := insertTree(tx, userID, root, "income", nil); err != nil {
			return err
		}
	}

	for _, acc := range defaultAccounts {
		if err := insertAccountTree(tx, userID, acc, nil); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func insertAccountTree(tx *sql.Tx, userID int64, acc defaultAccount, parentID *int64) error {
	var id int64
	err := tx.QueryRow(
		`INSERT INTO accounts (user_id, name, type, icon, color_hex, initial_balance, sort_order, is_system, parent_id)
		 VALUES ($1, $2, $3, $4, $5, 0, $6, TRUE, $7) RETURNING id`,
		userID, acc.Name, acc.Type, acc.Icon, acc.Color, acc.Order, parentID,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("insert account %s: %w", acc.Name, err)
	}
	for _, child := range acc.Children {
		if err := insertAccountTree(tx, userID, child, &id); err != nil {
			return err
		}
	}
	return nil
}

func insertTree(tx *sql.Tx, userID int64, node seedNode, catType string, parentID *int64) error {
	var id int64
	err := tx.QueryRow(
		`INSERT INTO categories (user_id, name, type, icon, color_hex, sort_order, is_system, parent_id)
		 VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7) RETURNING id`,
		userID, node.Name, catType, node.Icon, node.Color, node.Order, parentID,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("insert category %s: %w", node.Name, err)
	}
	for _, child := range node.Children {
		if err := insertTree(tx, userID, child, catType, &id); err != nil {
			return err
		}
	}
	return nil
}

// MigrateAccountsHierarchy 老用户账户层级迁移：
// 把已有的「支付宝」「微信」单一账户升级为分组+子账户结构。
// 原账户保留为子账户（重命名为「支付宝·余额」/「微信·零钱」），新建一个分组容器作为父账户。
// 交易 accountId 不变，仍指向原账户（现在是子账户），交易数据完整保留。
// 幂等：已是分组或已迁移过的账户跳过。
func MigrateAccountsHierarchy(db *sql.DB) error {
	type target struct {
		Type      string
		OldName   string
		GroupIcon string
		GroupColor string
		ChildName string
		ChildIcon string
		ChildColor string
		ExtraChildren []defaultAccount
	}
	targets := []target{
		{Type: "alipay", OldName: "支付宝", GroupIcon: "支", GroupColor: "#1677FF",
			ChildName: "支付宝·余额", ChildIcon: "支", ChildColor: "#1677FF",
			ExtraChildren: []defaultAccount{
				{Type: "alipay", Name: "支付宝·余额宝", Icon: "💰", Color: "#10B981", Order: 1},
				{Type: "alipay", Name: "支付宝·理财", Icon: "📈", Color: "#F59E0B", Order: 2},
			}},
		{Type: "wechat", OldName: "微信", GroupIcon: "微", GroupColor: "#07C160",
			ChildName: "微信·零钱", ChildIcon: "微", ChildColor: "#07C160",
			ExtraChildren: []defaultAccount{
				{Type: "wechat", Name: "微信·零钱通", Icon: "💰", Color: "#10B981", Order: 1},
				{Type: "wechat", Name: "微信·理财通", Icon: "📈", Color: "#F59E0B", Order: 2},
			}},
	}

	for _, t := range targets {
		// 找到所有 type=t.Type 且 parent_id IS NULL 且无子账户的账户（待迁移的叶子）
		rows, err := db.Query(
			`SELECT id, user_id, name, icon, color_hex, initial_balance, sort_order
			 FROM accounts
			 WHERE type = $1 AND parent_id IS NULL
			   AND NOT EXISTS (SELECT 1 FROM accounts c WHERE c.parent_id = accounts.id)`,
			t.Type,
		)
		if err != nil {
			return fmt.Errorf("scan %s accounts: %w", t.Type, err)
		}
		type leaf struct {
			ID, UserID   int64
			Name, Icon, Color string
			InitialBalance float64
			SortOrder      int
		}
		var leaves []leaf
		for rows.Next() {
			var l leaf
			if err := rows.Scan(&l.ID, &l.UserID, &l.Name, &l.Icon, &l.Color, &l.InitialBalance, &l.SortOrder); err != nil {
				rows.Close()
				return err
			}
			leaves = append(leaves, l)
		}
		rows.Close()

		for _, l := range leaves {
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()

			// 1. 新建分组容器
			var groupID int64
			err = tx.QueryRow(
				`INSERT INTO accounts (user_id, name, type, icon, color_hex, initial_balance, sort_order, is_system, parent_id)
				 VALUES ($1, $2, $3, $4, $5, 0, $6, TRUE, NULL) RETURNING id`,
				l.UserID, t.OldName, t.Type, t.GroupIcon, t.GroupColor, l.SortOrder,
			).Scan(&groupID)
			if err != nil {
				return fmt.Errorf("create group %s: %w", t.OldName, err)
			}

			// 2. 原账户重命名为子账户，parent 指向新分组
			_, err = tx.Exec(
				`UPDATE accounts SET name = $3, icon = $4, color_hex = $5, sort_order = 0, parent_id = $2
				 WHERE id = $1`,
				l.ID, groupID, t.ChildName, t.ChildIcon, t.ChildColor,
			)
			if err != nil {
				return fmt.Errorf("update leaf %s: %w", t.OldName, err)
			}

			// 3. 补齐其他子账户（余额宝/理财等）
			for _, child := range t.ExtraChildren {
				_, err := tx.Exec(
					`INSERT INTO accounts (user_id, name, type, icon, color_hex, initial_balance, sort_order, is_system, parent_id)
					 VALUES ($1, $2, $3, $4, $5, 0, $6, TRUE, $7)`,
					l.UserID, child.Name, child.Type, child.Icon, child.Color, child.Order, groupID,
				)
				if err != nil {
					return fmt.Errorf("insert child %s: %w", child.Name, err)
				}
			}

			if err := tx.Commit(); err != nil {
				return err
			}
		}
	}
	return nil
}

// MigrateEntertainmentOther 老用户「娱乐」分类补「其他」子分类（seed 新增项）。
// 幂等：已存在则跳过。
func MigrateEntertainmentOther(db *sql.DB) error {
	rows, err := db.Query(
		`SELECT id, user_id FROM categories WHERE name = '娱乐' AND type = 'expense' AND parent_id IS NULL`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	type ent struct {
		ID     int64
		UserID int64
	}
	var ents []ent
	for rows.Next() {
		var e ent
		if err := rows.Scan(&e.ID, &e.UserID); err != nil {
			return err
		}
		ents = append(ents, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, e := range ents {
		var exists bool
		if err := db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM categories WHERE parent_id = $1 AND name = '其他')`,
			e.ID,
		).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := db.Exec(
			`INSERT INTO categories (user_id, name, type, icon, color_hex, sort_order, is_system, parent_id)
			 VALUES ($1, '其他', 'expense', '⋯', '#6B7280', 600, TRUE, $2)`,
			e.UserID, e.ID,
		); err != nil {
			return fmt.Errorf("insert 娱乐·其他: %w", err)
		}
	}
	return nil
}

// migrateDigitalServiceTree 老用户补「数字服务」顶级分类及其子分类（seed 新增项）。
// 同时把顶级「其他」的 sort_order 调到 8，让「数字服务」(7) 排在前面。
// 幂等：已有「数字服务」则跳过。
func migrateDigitalServiceTree(db *sql.DB) error {
	rows, err := db.Query(
		`SELECT id, user_id FROM categories WHERE name = '数字服务' AND type = 'expense' AND parent_id IS NULL`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	type u struct {
		ID     int64
		UserID int64
	}
	var existing []u
	for rows.Next() {
		var x u
		if err := rows.Scan(&x.ID, &x.UserID); err != nil {
			return err
		}
		existing = append(existing, x)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	alreadyUsers := make(map[int64]bool, len(existing))
	for _, x := range existing {
		alreadyUsers[x.UserID] = true
	}

	userRows, err := db.Query(`SELECT DISTINCT user_id FROM categories`)
	if err != nil {
		return err
	}
	defer userRows.Close()
	var userIDs []int64
	for userRows.Next() {
		var uid int64
		if err := userRows.Scan(&uid); err != nil {
			return err
		}
		userIDs = append(userIDs, uid)
	}
	if err := userRows.Err(); err != nil {
		return err
	}

	tree := []seedNode{
		{Name: "服务器", Icon: "🖥️", Color: "#3B82F6", Order: 100},
		{Name: "域名", Icon: "🌍", Color: "#10B981", Order: 101},
		{Name: "软件订阅", Icon: "📦", Color: "#8B5CF6", Order: 102},
		{Name: "云服务", Icon: "☁️", Color: "#F59E0B", Order: 103},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 104},
	}

	for _, uid := range userIDs {
		if alreadyUsers[uid] {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		var rootID int64
		err = tx.QueryRow(
			`INSERT INTO categories (user_id, name, type, icon, color_hex, sort_order, is_system, parent_id)
			 VALUES ($1, '数字服务', 'expense', '🌐', '#06B6D4', 7, TRUE, NULL) RETURNING id`,
			uid,
		).Scan(&rootID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("insert 数字服务: %w", err)
		}
		for _, child := range tree {
			if _, err := tx.Exec(
				`INSERT INTO categories (user_id, name, type, icon, color_hex, sort_order, is_system, parent_id)
				 VALUES ($1, $2, 'expense', $3, $4, $5, TRUE, $6)`,
				uid, child.Name, child.Icon, child.Color, child.Order, rootID,
			); err != nil {
				tx.Rollback()
				return fmt.Errorf("insert 数字服务·%s: %w", child.Name, err)
			}
		}
		// 顶级「其他」sort_order 调到 8（原来 7），让数字服务排前面
		if _, err := tx.Exec(
			`UPDATE categories SET sort_order = 8
			 WHERE user_id = $1 AND name = '其他' AND type = 'expense' AND parent_id IS NULL`,
			uid,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("update 其他 sort_order: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// migrateInsertAfterParent 在每个用户的指定顶级支出分类 rootName 下，
// 于锚点子分类 afterName 之后插入 nodes（sort_order 紧跟锚点递增）；
// 原排在锚点之后的子分类 sort_order 整体 +len(nodes) 腾位。
// 幂等：已存在 nodes[0].Name 则跳过；无锚点子分类则跳过（无法定位）。
//
// 用于「餐饮」补 夜宵/小吃/饮料（晚餐后）、「交通」补 高铁（打车后）等 seed 新增项。
func migrateInsertAfterParent(db *sql.DB, rootName, afterName string, nodes []seedNode) error {
	if len(nodes) == 0 {
		return nil
	}
	rows, err := db.Query(
		`SELECT id, user_id FROM categories WHERE name = $1 AND type = 'expense' AND parent_id IS NULL`,
		rootName,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	type root struct {
		ID     int64
		UserID int64
	}
	var roots []root
	for rows.Next() {
		var r root
		if err := rows.Scan(&r.ID, &r.UserID); err != nil {
			return err
		}
		roots = append(roots, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, r := range roots {
		// 幂等：已存在首个新子分类则跳过
		var exists bool
		if err := db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM categories WHERE parent_id = $1 AND name = $2)`,
			r.ID, nodes[0].Name,
		).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}

		// 定位锚点子分类的 sort_order
		var afterOrder int
		err := db.QueryRow(
			`SELECT sort_order FROM categories WHERE parent_id = $1 AND name = $2`,
			r.ID, afterName,
		).Scan(&afterOrder)
		if err != nil {
			// 无锚点子分类，无法定位，跳过
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// 锚点之后的子分类整体 +len(nodes) 腾位
		if _, err := tx.Exec(
			`UPDATE categories SET sort_order = sort_order + $3
			 WHERE parent_id = $1 AND sort_order > $2`,
			r.ID, afterOrder, len(nodes),
		); err != nil {
			return fmt.Errorf("shift %s children after %s: %w", rootName, afterName, err)
		}

		// 插入新子分类（sort_order 紧跟锚点递增）
		for i, n := range nodes {
			if _, err := tx.Exec(
				`INSERT INTO categories (user_id, name, type, icon, color_hex, sort_order, is_system, parent_id)
				 VALUES ($1, $2, 'expense', $3, $4, $5, TRUE, $6)`,
				r.UserID, n.Name, n.Icon, n.Color, afterOrder+i+1, r.ID,
			); err != nil {
				return fmt.Errorf("insert %s·%s: %w", rootName, n.Name, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}