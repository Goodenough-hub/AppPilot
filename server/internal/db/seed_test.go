package db

import "testing"

// diningChildren 返回「餐饮」一级分类的子分类列表（来自 expenseTree 种子定义）。
func diningChildren(t *testing.T) []seedNode {
	t.Helper()
	for _, root := range expenseTree {
		if root.Name == "餐饮" {
			return root.Children
		}
	}
	t.Fatalf("expenseTree 中找不到「餐饮」分类")
	return nil
}

func TestExpenseTreeDiningHasLateNightAndSnacks(t *testing.T) {
	subs := diningChildren(t)

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

func TestExpenseTreeDiningLateNightSnacksAfterDinner(t *testing.T) {
	subs := diningChildren(t)

	// 定位晚餐 / 夜宵 / 小吃 / 饮料 的下标，断言顺序：晚餐 < 夜宵 < 小吃 < 饮料
	idx := map[string]int{}
	for i, c := range subs {
		idx[c.Name] = i
	}
	want := []string{"晚餐", "夜宵", "小吃", "饮料"}
	for _, n := range want {
		if _, ok := idx[n]; !ok {
			t.Fatalf("缺少必要子分类「%s」", n)
		}
	}
	d := idx["晚餐"]
	ln := idx["夜宵"]
	sn := idx["小吃"]
	dr := idx["饮料"]
	if !(d < ln && ln < sn && sn < dr) {
		t.Errorf("顺序应为 晚餐 < 夜宵 < 小吃 < 饮料，实际下标: 晚餐=%d 夜宵=%d 小吃=%d 饮料=%d", d, ln, sn, dr)
	}

	// sort_order 与下标一致（递增），且夜宵/小吃/饮料 紧跟晚餐
	for i := 1; i < len(subs); i++ {
		if subs[i].Order <= subs[i-1].Order {
			t.Errorf("餐饮子分类 sort_order 非递增于位置 %d: %d <= %d", i, subs[i].Order, subs[i-1].Order)
		}
	}
	if subs[ln].Order != subs[d].Order+1 {
		t.Errorf("夜宵 sort_order 应为 晚餐+1: 夜宵=%d 晚餐=%d", subs[ln].Order, subs[d].Order)
	}
	if subs[sn].Order != subs[d].Order+2 {
		t.Errorf("小吃 sort_order 应为 晚餐+2: 小吃=%d 晚餐=%d", subs[sn].Order, subs[d].Order)
	}
	if subs[dr].Order != subs[d].Order+3 {
		t.Errorf("饮料 sort_order 应为 晚餐+3: 饮料=%d 晚餐=%d", subs[dr].Order, subs[d].Order)
	}
}
