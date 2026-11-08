# AppPilot

工作区的共享 Go API、管理后台与 Hub 前端。FinFlow、TypResume、Hub 和 AlgoForge 复用 `users`、主 JWT 与 `app_scope` 授权；FluxBlog 使用独立博客账号与 JWT。各应用业务表隔离。

## 结构与入口

```text
AppPilot/
├── server/   Go + Gin + PostgreSQL API（/api/v1/）
├── admin/    React 管理后台（/admin/）
└── hub/      React 私人工作台（/hub/）
```

`server/internal/oj/` 提供 AlgoForge 的题库、提交和录题 API；`server/internal/db/migrations_oj.go` 建立 `oj_` 表族。C++ 编译与运行由独立 [AlgoForge Worker](../AlgoForge/README.md) 完成，不进入 API 进程。

## 本地开发

先准备 PostgreSQL，并设置：

```bash
export APPPLOT_DSN='postgres://<user>:<password>@127.0.0.1:5432/<database>?sslmode=disable'
export APPPLOT_JWT_SECRET='<至少 32 字符的随机密钥>'
cd server
go run . serve
```

服务器默认监听 `127.0.0.1:8080`，启动时自动执行幂等迁移。首次管理员账号可运行 `go run . create-admin --username <name> --password <password>`。前端分别在 `admin/`、`hub/` 下执行 `npm install && npm run dev`。

## AlgoForge 接口

| 接口 | 权限 | 用途 |
|---|---|---|
| `GET /api/v1/oj/problems`、`GET /api/v1/oj/problems/:slug` | 公开 | 已发布题库与题面，不含隐藏测试 |
| `POST /api/v1/oj/runs` | `oj` scope | 用自定义标准输入自测 |
| `POST /api/v1/oj/submissions` | `oj` scope | 提交到隐藏测试队列 |
| `GET /api/v1/oj/submissions[/:id]` | `oj` scope | 本人提交列表与详情 |
| `/api/v1/admin/oj/problems*` | admin | 录题、编辑版本、发布或撤回 |

管理员在用户管理页为普通账号勾选 `AlgoForge`，即可授予 `oj` scope。Worker 直接领取 PostgreSQL 中的 `oj_jobs`，并将判题结果回写至对应提交。

## 检查

```bash
cd server && go test ./... && make build
cd ../admin && npm run typecheck && npm test && npm run build
cd ../hub && npm run typecheck && npm test && npm run build
```

部署流程、凭据位置与 Git 规范见工作区 [AGENTS.md](../AGENTS.md)。
