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
