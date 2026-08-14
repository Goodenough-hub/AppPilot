# FinFlow 收入分类扩充：退款 / 报销 / 他人转入

- 日期：2026-08-14
- 涉及仓库：`AppPilot`（server / db seed）
- 关联业务：FinFlow 记账，收入分类

## 背景与问题

当前 FinFlow 的收入顶级分类只有 4 个（`seed.go:126-136`）：工资、投资、兼职、其他收入。数量偏少，且缺常见项。用户反馈"至少要加：退款、转账"。

## 关键前置澄清（不做什么）

**不新增名为"转账"的收入分类。** 前端 `TransactionType` 已把 `transfer` 定义为与 `income/expense` 并列的第三种交易类型（`FinFlow/web/src/db/models.ts:1,85`），记账界面顶部有独立的"转账" tab，转账时不选分类而是选"从账户 → 到账户"。若在收入分类里再叫"转账"，语义会与之冲突（用户可能把内部资金移动误记成收入，虚增总收入）。

用户实际想解决的场景是"别人转钱给我"（红包、AA 回款、代付回款等，从外部账户流入），语义上是**收入**、不是**资金内部转移**，因此以"**他人转入**"命名。

## 目标

在收入分类中新增 3 个顶级分类：**退款、报销、他人转入**，插在"其他收入"之前，保持"其他收入"位于最末。

## 分类清单（最终）

| sort_order | 名称 | icon | color | 说明 |
|---|---|---|---|---|
| 0 | 工资 | 💰 | `#10B981` | 原有 |
| 1 | 投资 | 📈 | `#3B82F6` | 原有 |
| 2 | 兼职 | 💼 | `#8B5CF6` | 原有 |
| 3 | **退款** | ↩️ | `#10B981` | 新增 |
| 4 | **报销** | 🧾 | `#3B82F6` | 新增 |
| 5 | **他人转入** | 🤝 | `#8B5CF6` | 新增 |
| 6 | 其他收入 | ⋯ | `#6B7280` | sort_order 由 3 挪到 6 |

命名 / 图标 / 配色对齐 `seed.go:906-909` 已有的旅游收入组（同伴回款/退款/报销/其他），保持系统内一致。

## 实现方案

### 改动范围

**仅后端一个包**：`AppPilot/server/internal/db/`

1. **`seed.go`**
   - `incomeTree`（第 126-136 行）：追加 3 条顶级 income seedNode（Order 3/4/5），把"其他收入"Order 从 3 改为 6。
   - 新增函数 `migrateIncomeAddRefundReimburseTransferIn(db *sql.DB) error`：对每个已有 income 顶级分类的用户，补齐这 3 项 + 挪"其他收入"位。

2. **`migrations.go`**
   - 在 `MigrateTripCategoriesV2(db)` 之后（约第 212 行、`migrateMoveWeixinReadSubscription` 之前）插入 `if err := migrateIncomeAddRefundReimburseTransferIn(db); err != nil { return err }`。

**前端不改动**。分类是从 `GET /api/v1/finflow/categories` 拉的，新增顶级分类自动出现在"收入"tab；`api/finflow.ts` 无契约变化。

### 迁移函数逻辑

参考已有 `migrateDigitalServiceTree`（`seed.go:400-495`）的写法：

```
对每个 (SELECT DISTINCT user_id FROM categories) 的用户：
  幂等 gate：查询该用户是否已有顶级 income "退款"，是则跳过。
  BEGIN TX
    UPDATE categories SET sort_order=6 WHERE user_id=$1 AND name='其他收入' AND type='income' AND parent_id IS NULL
    INSERT 退款   (sort_order=3, icon='↩️', color='#10B981', is_system=TRUE, parent_id=NULL)
    INSERT 报销   (sort_order=4, icon='🧾', color='#3B82F6', is_system=TRUE, parent_id=NULL)
    INSERT 他人转入 (sort_order=5, icon='🤝', color='#8B5CF6', is_system=TRUE, parent_id=NULL)
  COMMIT
```

### 幂等策略

**以"退款是否存在"作为整体开关**：若用户已跑过一次迁移，就已经有"退款"了，整个跳过。用户当前确认没有手动建过这三个同名分类，因此这个策略不会漏补也不会重复插。

若未来某个用户已手建同名分类，此迁移会跳过其他两项——这是显式取舍，避免破坏用户已有数据。未来若需支持"逐项补齐"，另写一个精细化迁移即可，不必现在做（YAGNI）。

## 验证

- `cd AppPilot/server && go test ./internal/db/...`：跑 `seed_test.go` 保证 seed 不 regress。若测试对分类总数有断言（当前 78 → 应改为 81），一并更新。
- `go vet ./...` + `make fmt`。
- 手动验证（可选）：在本地 PG 上 `go run . serve` 起服务，用现有 finflow 账号 `GET /api/v1/finflow/categories?type=income`，确认返回按 sort_order 排序为 工资/投资/兼职/退款/报销/他人转入/其他收入。

## 风险与回滚

- **风险低**：只加 3 行数据 + 挪 1 行 sort_order。迁移在事务内，失败自动回滚。
- **回滚**：若上线后需撤，写反向迁移（DELETE 3 项 + 把"其他收入"改回 sort_order=3）。已有交易若引用了这 3 个分类，`transactions.category_id ON DELETE SET NULL` 会保留交易本身。

## 不做

- 不改 `TransactionType` / `TripGroups`。
- 不动前端。
- 不做"红包/奖金/经营收入"等本轮未选中的项。
