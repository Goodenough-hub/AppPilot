package db

import "testing"

// findNode 在 expenseTree 中按名递归查找节点（含嵌套，如「影视」在「娱乐」下）。
func findNode(nodes []seedNode, name string) *seedNode {
	for i := range nodes {
		if nodes[i].Name == name {
			return &nodes[i]
		}
		if r := findNode(nodes[i].Children, name); r != nil {
			return r
		}
	}
	return nil
}

// childrenOf 返回 expenseTree 中指定分类（可为顶级或嵌套）的子分类列表。
func childrenOf(t *testing.T, root string) []seedNode {
	t.Helper()
	node := findNode(expenseTree, root)
	if node == nil {
		t.Fatalf("expenseTree 中找不到「%s」分类", root)
	}
	return node.Children
}

func TestExpenseTreeDiningHasLateNightAndSnacks(t *testing.T) {
	subs := childrenOf(t, "餐饮")

	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Name
	}

	// 夜宵、小吃、饮料必须存在
	want := []string{"夜宵", "小吃", "饮料"}
	for _, w := range want {
		found := false
		for _, n := range names {
			if n == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("餐饮子分类缺少「%s」，实际: %v", w, names)
		}
	}
}

// assertChain 校验锚点后紧跟的若干子分类顺序与 sort_order 连续。
func assertChain(t *testing.T, subs []seedNode, anchor string, after ...string) {
	t.Helper()
	idx := map[string]int{}
	for i, c := range subs {
		idx[c.Name] = i
	}
	if _, ok := idx[anchor]; !ok {
		t.Fatalf("缺少锚点子分类「%s」", anchor)
	}
	prev := anchor
	for _, n := range after {
		if _, ok := idx[n]; !ok {
			t.Fatalf("缺少必要子分类「%s」", n)
		}
		if !(idx[prev] < idx[n]) {
			t.Errorf("顺序应为 %s < %s", prev, n)
		}
		prev = n
	}
	// sort_order 整体递增
	for i := 1; i < len(subs); i++ {
		if subs[i].Order <= subs[i-1].Order {
			t.Errorf("子分类 sort_order 非递增于位置 %d: %d <= %d", i, subs[i].Order, subs[i-1].Order)
		}
	}
	// 锚点 + after 的 sort_order 连续递增
	byName := map[string]seedNode{}
	for _, c := range subs {
		byName[c.Name] = c
	}
	cur := byName[anchor].Order
	for _, n := range after {
		cur++
		if byName[n].Order != cur {
			t.Errorf("「%s」sort_order 应为 %d，实际 %d", n, cur, byName[n].Order)
		}
	}
}

func TestExpenseTreeDiningLateNightSnacksAfterDinner(t *testing.T) {
	subs := childrenOf(t, "餐饮")
	assertChain(t, subs, "晚餐", "夜宵", "小吃", "饮料")
}

func TestExpenseTreeTransportHasHighSpeedRail(t *testing.T) {
	subs := childrenOf(t, "交通")

	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Name
	}
	found := false
	for _, n := range names {
		if n == "高铁" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("交通子分类缺少「高铁」，实际: %v", names)
	}

	// 高铁位于打车之后，其他之前；sort_order 连续
	assertChain(t, subs, "打车", "高铁")
}

func TestExpenseTreeFilmHasCinema(t *testing.T) {
	// 影视是「娱乐」下的嵌套子分类（非顶级）
	subs := childrenOf(t, "影视")

	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Name
	}
	found := false
	for _, n := range names {
		if n == "影院" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("影视子分类缺少「影院」，实际: %v", names)
	}

	// 影院位于爱奇艺之后，其他之前；sort_order 连续
	assertChain(t, subs, "爱奇艺", "影院")
}

func TestExpenseTreeDiningHasTakeout(t *testing.T) {
	subs := childrenOf(t, "餐饮")

	found := false
	for _, c := range subs {
		if c.Name == "外卖" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("餐饮子分类缺少「外卖」")
	}
	// 外卖位于饮料之后；sort_order 连续
	assertChain(t, subs, "饮料", "外卖")
}

func TestExpenseTreeShoppingHasTakeout(t *testing.T) {
	subs := childrenOf(t, "购物")

	found := false
	for _, c := range subs {
		if c.Name == "外卖" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("购物子分类缺少「外卖」")
	}
	// 外卖位于抖音之后，其他之前；sort_order 连续
	assertChain(t, subs, "抖音", "外卖")
}

// TestIncomeTreeHasRefundReimburseTransferIn 校验收入顶级分类含
// 退款/报销/他人转入，且顺序为 工资→投资→兼职→退款→报销→他人转入→其他收入。
func TestIncomeTreeHasRefundReimburseTransferIn(t *testing.T) {
	names := make([]string, len(incomeTree))
	orderByName := map[string]int{}
	for i, c := range incomeTree {
		names[i] = c.Name
		orderByName[c.Name] = c.Order
	}
	want := []string{"工资", "投资", "兼职", "退款", "报销", "他人转入", "其他收入"}
	for i, w := range want {
		if i >= len(names) || names[i] != w {
			t.Errorf("incomeTree[%d] 期望 %q，实际 %v", i, w, names)
		}
		if orderByName[w] != i {
			t.Errorf("%q sort_order 期望 %d，实际 %d", w, i, orderByName[w])
		}
	}
}

// TestPlatformCategoriesUseBrandIcon 校验 seed 里的平台子分类 icon 已改为 brand: 前缀，
// 且 categoryBrandRewrites 对应表覆盖 seed 中所有 brand: 分类（前后端映射一致的兜底）。
func TestPlatformCategoriesUseBrandIcon(t *testing.T) {
	wantBrand := map[string]string{
		"京东": "brand:jd", "淘宝": "brand:taobao", "拼多多": "brand:pinduoduo", "抖音": "brand:douyin",
		"王者荣耀": "brand:wangzhe", "和平精英": "brand:hepingjy", "原神": "brand:yuanshen", "Steam": "brand:steam",
		"腾讯视频": "brand:tencentvid", "B站": "brand:bilibili", "爱奇艺": "brand:iqiyi",
		"Apple Music": "brand:applemusic", "网易云音乐": "brand:neteasemus", "QQ音乐": "brand:qqmusic",
		"百度网盘": "brand:baiduyun", "阿里网盘": "brand:aliyunpan", "天翼网盘": "brand:tianyipan", "夸克网盘": "brand:quarkpan",
	}

	seen := map[string]string{}
	var walk func([]seedNode)
	walk = func(ns []seedNode) {
		for _, n := range ns {
			if wanted, ok := wantBrand[n.Name]; ok {
				seen[n.Name] = n.Icon
				if n.Icon != wanted {
					t.Errorf("seed 中 %q 的 icon 期望 %q，实际 %q", n.Name, wanted, n.Icon)
				}
			}
			walk(n.Children)
		}
	}
	walk(expenseTree)

	for name := range wantBrand {
		if _, ok := seen[name]; !ok {
			t.Errorf("expenseTree 中未找到平台子分类 %q", name)
		}
	}

	// 迁移映射表必须覆盖 seed 里所有 brand 分类（前后端一致的兜底）
	rewriteByName := map[string]string{}
	for _, r := range categoryBrandRewrites {
		rewriteByName[r.Name] = r.Icon
	}
	for name, icon := range wantBrand {
		if rewriteByName[name] != icon {
			t.Errorf("categoryBrandRewrites 缺 %q → %q（实际 %q）", name, icon, rewriteByName[name])
		}
	}
}

func TestTripGroupsSeed(t *testing.T) {
	if len(tripGroups) == 0 {
		t.Fatal("tripGroups 为空")
	}

	// 期望的组 → 必须包含的若干叶子
	wantGroups := map[string][]string{
		"交通": {"机票", "火车", "高铁", "打车", "地铁", "公交", "租车", "加油", "停车", "过路费"},
		"餐饮": {"早餐", "午餐", "晚餐", "小吃", "饮料", "咖啡", "酒水"},
		"住宿": {"酒店", "民宿"},
		"游玩": {"门票", "演出", "项目", "导游", "装备租赁"},
		"购物": {"特产", "纪念品", "伴手礼", "免税店"},
		"杂项": {"通讯流量", "签证", "保险", "医疗", "洗衣", "小费", "其他"},
		"收入": {"同伴回款", "退款", "报销", "其他"},
	}

	groupByName := map[string]tripGroup{}
	for _, g := range tripGroups {
		if g.Name == "" || g.Icon == "" || g.Color == "" {
			t.Errorf("旅游分类组字段不完整: %+v", g)
		}
		if g.Type != "expense" && g.Type != "income" {
			t.Errorf("旅游分类组 %s type 非法: %s", g.Name, g.Type)
		}
		if len(g.Children) == 0 {
			t.Errorf("旅游分类组 %s 没有叶子", g.Name)
		}
		// 组内叶子字段完整、无重名
		seen := map[string]bool{}
		for _, ch := range g.Children {
			if ch.Name == "" || ch.Icon == "" || ch.Color == "" {
				t.Errorf("组 %s 叶子字段不完整: %+v", g.Name, ch)
			}
			if seen[ch.Name] {
				t.Errorf("组 %s 内叶子重名: %s", g.Name, ch.Name)
			}
			seen[ch.Name] = true
		}
		groupByName[g.Name] = g
	}

	for gname, leaves := range wantGroups {
		g, ok := groupByName[gname]
		if !ok {
			t.Errorf("缺少旅游分类组「%s」", gname)
			continue
		}
		have := map[string]bool{}
		for _, ch := range g.Children {
			have[ch.Name] = true
		}
		for _, leaf := range leaves {
			if !have[leaf] {
				t.Errorf("组「%s」缺少叶子「%s」", gname, leaf)
			}
		}
	}

	// 收入组必须是 income 类型，其余为 expense
	if groupByName["收入"].Type != "income" {
		t.Errorf("「收入」组 type 应为 income")
	}
	for _, gname := range []string{"交通", "餐饮", "住宿", "游玩", "购物", "杂项"} {
		if groupByName[gname].Type != "expense" {
			t.Errorf("「%s」组 type 应为 expense", gname)
		}
	}
}
