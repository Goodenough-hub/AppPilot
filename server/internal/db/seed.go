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
		{Name: "聚餐AA", Icon: "👥", Color: "#8B5CF6", Order: 103},
		{Name: "聚餐请客", Icon: "❤️", Color: "#EC4899", Order: 104},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 105},
	}},
	{Name: "交通", Icon: "🚗", Color: "#3B82F6", Order: 1, Children: []seedNode{
		{Name: "地铁", Icon: "🚇", Color: "#3B82F6", Order: 100},
		{Name: "公交", Icon: "🚌", Color: "#10B981", Order: 101},
		{Name: "打车", Icon: "🚕", Color: "#F59E0B", Order: 102},
		{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 103},
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
	{Name: "其他", Icon: "⋯", Color: "#6B7280", Order: 7},
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
	Type  string
	Name  string
	Icon  string
	Color string
	Order int
}

var defaultAccounts = []defaultAccount{
	{Type: "alipay", Name: "支付宝", Icon: "支", Color: "#1677FF", Order: 0},
	{Type: "wechat", Name: "微信", Icon: "微", Color: "#07C160", Order: 1},
	{Type: "unionpay", Name: "云闪付", Icon: "银", Color: "#E60012", Order: 2},
	{Type: "fixed", Name: "定期", Icon: "定", Color: "#F59E0B", Order: 3},
}

// SeedForUser 在指定 user_id 下种子 78 个分类 + 4 个账户。
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
		_, err := tx.Exec(
			`INSERT INTO accounts (user_id, name, type, icon, color_hex, initial_balance, sort_order, is_system)
			 VALUES ($1, $2, $3, $4, $5, 0, $6, TRUE)`,
			userID, acc.Name, acc.Type, acc.Icon, acc.Color, acc.Order,
		)
		if err != nil {
			return fmt.Errorf("insert account %s: %w", acc.Name, err)
		}
	}

	return tx.Commit()
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
