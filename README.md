# AppPilot

多 PWA 应用的共享后端 + 管理后台。第一个接入应用是 [FinFlow](https://github.com/Goodenough-hub/FinFlow)（个人记账）。

## 架构

```
AppPilot/
├── server/    # Go 后端（Gin + PostgreSQL + JWT）
└── admin/     # 管理后台 SPA（Vite + React + TS）
```

nginx 反向代理：
- `/` → FinFlow PWA 静态文件
- `/admin` → AppPilot admin 静态文件
- `/api/v1/*` → AppPilot server（127.0.0.1:8080）

## server

```bash
cd server
go run .          # 启动（需要 PostgreSQL 与环境变量）
go test ./...
make build        # 交叉编译 Linux 二进制到 bin/
```

环境变量：
- `APPPLOT_DSN` — PostgreSQL 连接串
- `APPPLOT_JWT_SECRET` — JWT 签名密钥（64 字符随机串）
- `APPPLOT_ADDRESS` — 监听地址（默认 127.0.0.1:8080）

## admin

```bash
cd admin
npm install
npm run dev       # 开发 :5076
npm run build     # 输出 dist/
```

## 接入新 PWA 应用

1. 在 server `internal/` 下加新业务包（如 `internal/blog/`）
2. 在 `cmd/cmd.go` 注册新路由组 `/api/v1/blog/*`
3. 在 admin 加新管理页面 `src/pages/blog/`
4. 用户表 `app_scope` 字段加新应用名
