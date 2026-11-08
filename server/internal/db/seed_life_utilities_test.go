package db

import (
	"reflect"
	"testing"
)

func TestLifeSeedChildren(t *testing.T) {
	var got []string
	for _, child := range lifeSeedChildren() {
		got = append(got, child.Name)
	}
	want := []string{"理发", "饮用水", "话费", "快递", "文印", "礼金", "捐赠", "转账", "其他"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("生活子分类顺序不符: %v", got)
	}
	for i, child := range lifeSeedChildren() {
		if child.Order != 100+i {
			t.Fatalf("%s 的 sort_order 应为 %d，实际 %d", child.Name, 100+i, child.Order)
		}
	}
}

func TestPlanLifeChildren(t *testing.T) {
	desired := lifeSeedChildren()

	t.Run("空分类时补齐全部子分类", func(t *testing.T) {
		plan := planLifeChildren(nil, desired)
		if len(plan.Reorders) != 0 {
			t.Fatalf("不应重排: %+v", plan)
		}
		if len(plan.Inserts) != len(desired) {
			t.Fatalf("应补齐 %d 个，实际 %d", len(desired), len(plan.Inserts))
		}
	})

	t.Run("老账号补三个新分类并把既有分类重排到 seed 位置", func(t *testing.T) {
		existing := []categorySibling{
			{ID: 1, seedNode: seedNode{Name: "理发", Order: 100}},
			{ID: 2, seedNode: seedNode{Name: "快递", Order: 101}},
			{ID: 3, seedNode: seedNode{Name: "礼金", Order: 102}},
			{ID: 4, seedNode: seedNode{Name: "捐赠", Order: 103}},
			{ID: 5, seedNode: seedNode{Name: "转账", Order: 104}},
			{ID: 6, seedNode: seedNode{Name: "其他", Order: 105}},
		}
		plan := planLifeChildren(existing, desired)
		var inserted []string
		for _, node := range plan.Inserts {
			inserted = append(inserted, node.Name)
		}
		if !reflect.DeepEqual(inserted, []string{"饮用水", "话费", "文印"}) {
			t.Fatalf("新增分类不符: %v", inserted)
		}
		want := map[int64]int{2: 103, 3: 105, 4: 106, 5: 107, 6: 108}
		if !reflect.DeepEqual(plan.Reorders, want) {
			t.Fatalf("重排结果不符: %v", plan.Reorders)
		}
		if plan.empty() {
			t.Fatal("计划不应为空")
		}
	})

	t.Run("已对齐时重复执行不产生任何变更", func(t *testing.T) {
		var existing []categorySibling
		for i, node := range desired {
			existing = append(existing, categorySibling{ID: int64(i + 1), seedNode: node})
		}
		plan := planLifeChildren(existing, desired)
		if !plan.empty() {
			t.Fatalf("幂等性被破坏: %+v", plan)
		}
	})

	t.Run("已有同名分类不重复新增", func(t *testing.T) {
		existing := []categorySibling{{ID: 7, seedNode: seedNode{Name: "饮用水", Order: 101}}}
		plan := planLifeChildren(existing, desired)
		for _, node := range plan.Inserts {
			if node.Name == "饮用水" {
				t.Fatalf("已存在的「饮用水」不应再新增: %+v", plan.Inserts)
			}
		}
		if _, ok := plan.Reorders[7]; ok {
			t.Fatal("顺序已对齐时不应重排")
		}
	})

	t.Run("用户自建子分类不被重排或删除", func(t *testing.T) {
		existing := []categorySibling{{ID: 9, seedNode: seedNode{Name: "宠物", Order: 200}}}
		plan := planLifeChildren(existing, desired)
		if _, ok := plan.Reorders[9]; ok {
			t.Fatal("自建分类不应被重排")
		}
		if len(plan.Inserts) != len(desired) {
			t.Fatalf("应补齐 %d 个 seed 分类，实际 %d", len(desired), len(plan.Inserts))
		}
	})
}
