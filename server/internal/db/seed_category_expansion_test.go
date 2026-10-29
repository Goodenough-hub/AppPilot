package db

import (
	"reflect"
	"testing"
)

func TestCategoryExpansionDefaults(t *testing.T) {
	cases := map[string][]string{
		"报销":   {"公司报销", "学校报销", "其他报销"},
		"二手卖出": {"闲鱼", "爱回收", "转转", "线下回收", "熟人交易", "其他平台"},
		"礼金红包": {"节日红包", "生日红包", "礼金", "其他"},
		"奖励返现": {"活动奖励", "消费返现", "其他"},
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			node := findNode(incomeTree, name)
			if node == nil {
				t.Fatalf("缺少 %s", name)
			}
			var got []string
			for _, child := range node.Children {
				got = append(got, child.Name)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
	var transport []string
	for _, child := range childrenOf(t, "交通") {
		transport = append(transport, child.Name)
	}
	if !reflect.DeepEqual(transport, []string{"地铁", "公交", "打车", "高铁", "电瓶车充电", "其他"}) {
		t.Fatalf("交通新增范围不符: %v", transport)
	}
}

func TestPlanCategoryAdditions(t *testing.T) {
	nodes := findNode(incomeTree, "报销").Children
	t.Run("空分类补齐全部子分类", func(t *testing.T) {
		got := planCategoryAdditions(nil, nodes)
		if len(got) != 3 || got[0].Order != 100 || got[2].Order != 102 {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("保留已有分类身份且补齐部分缺失", func(t *testing.T) {
		existing := []categorySibling{{ID: 42, seedNode: seedNode{Name: "公司报销", Order: 7}}}
		got := planCategoryAdditions(existing, nodes)
		if len(got) != 3 || got[0].ID != 42 || got[0].Order != 7 {
			t.Fatalf("%+v", got)
		}
		if !reflect.DeepEqual(got, planCategoryAdditions(got, nodes)) {
			t.Fatal("重复执行不应再新增或重排")
		}
	})
	t.Run("插入其他之前且不改原始快照", func(t *testing.T) {
		existing := []categorySibling{{ID: 1, seedNode: seedNode{Name: "自定义", Order: 10}}, {ID: 2, seedNode: seedNode{Name: "其他", Order: 20}}}
		got := planCategoryAdditions(existing, nodes)
		if got[0].Order != 10 || got[1].Order != 23 || got[2].Order != 20 || got[4].Order != 22 {
			t.Fatalf("%+v", got)
		}
		if existing[1].Order != 20 {
			t.Fatal("不应修改输入")
		}
	})
	t.Run("空新增列表不修改已有分类", func(t *testing.T) {
		existing := []categorySibling{{ID: 3, seedNode: seedNode{Name: "手动分类", Order: 8}}}
		if !reflect.DeepEqual(existing, planCategoryAdditions(existing, nil)) {
			t.Fatal("空列表应保持原样")
		}
	})
}

func TestLifeTransferDefault(t *testing.T) {
	subs := childrenOf(t, "生活")
	assertChain(t, subs, "捐赠", "转账", "其他")
	var transfer seedNode
	count := 0
	for _, node := range subs {
		if node.Name == "转账" {
			transfer = node
			count++
		}
	}
	if count != 1 || transfer.Icon != "💸" || transfer.Color != "#14B8A6" {
		t.Fatalf("生活应包含唯一的转账分类并带有默认图标颜色: %+v", subs)
	}

	t.Run("老账号保留分类身份并在其他之前插入且重复执行不改变结果", func(t *testing.T) {
		existing := []categorySibling{
			{ID: 11, seedNode: seedNode{Name: "捐赠", Order: 103}},
			{ID: 12, seedNode: seedNode{Name: "其他", Order: 104}},
		}
		got := planCategoryAdditions(existing, []seedNode{transfer})
		if len(got) != 3 || got[0].ID != 11 || got[0].Order != 103 || got[1].ID != 12 || got[1].Order != 105 || got[2].Name != "转账" || got[2].Order != 104 {
			t.Fatalf("补分类结果不符: %+v", got)
		}
		if !reflect.DeepEqual(got, planCategoryAdditions(got, []seedNode{transfer})) {
			t.Fatal("重复迁移不应产生重复分类或再次改变顺序")
		}
	})

	t.Run("缺少默认兄弟分类时仍可以补充转账", func(t *testing.T) {
		got := planCategoryAdditions(nil, []seedNode{transfer})
		if len(got) != 1 || got[0].Name != "转账" || got[0].Order != 104 {
			t.Fatalf("空列表补分类结果不符: %+v", got)
		}
	})

	t.Run("保留用户已有同名分类及自定义顺序", func(t *testing.T) {
		existing := []categorySibling{{ID: 20, seedNode: seedNode{Name: "转账", Order: 8}}}
		if got := planCategoryAdditions(existing, []seedNode{transfer}); !reflect.DeepEqual(existing, got) {
			t.Fatalf("不应替换已有同名分类: %+v", got)
		}
	})
}
