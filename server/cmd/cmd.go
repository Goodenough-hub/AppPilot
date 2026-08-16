package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"apppilot-server/internal/admin"
	"apppilot-server/internal/analytics"
	"apppilot-server/internal/auth"
	"apppilot-server/internal/blog"
	"apppilot-server/internal/dashboard"
	"apppilot-server/internal/db"
	"apppilot-server/internal/finflow"
	"apppilot-server/internal/hub"
	"apppilot-server/internal/middleware"
	"apppilot-server/internal/typresume"
	"apppilot-server/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func Run() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		klog.Fatalf("config invalid: %v", err)
	}

	root := &cobra.Command{
		Use:   "apppilot-server",
		Short: "AppPilot backend server",
	}
	root.AddCommand(serveCmd(&cfg))
	root.AddCommand(createAdminCmd(&cfg))
	root.AddCommand(seedCmd(&cfg))
	root.AddCommand(importBlogCmd(&cfg))
	if err := root.Execute(); err != nil {
		klog.Fatalf("execute: %v", err)
	}
}

func serveCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return serve(cfg)
		},
	}
}

func serve(cfg *config.Config) error {
	// decimal 字段（initialBalance/amount 等）以 number 而非 string 输出 JSON
	decimal.MarshalJSONWithoutQuotes = true

	pg, err := db.NewPostgres(cfg.DSN)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pg.Close()

	if err := db.Migrate(pg); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	v1 := r.Group("/api/v1")

	authRepo := auth.NewRepository(pg)
	authHandler := auth.NewHandler(authRepo, cfg.JWTSecret)
	authHandler.Register(v1.Group("/auth"))
	authHandler.RegisterProtected(
		v1.Group("/auth", middleware.AuthRequired(cfg.JWTSecret)),
	)

	finflowHandler := finflow.NewHandler(pg)
	finflowHandler.Register(
		v1.Group("/finflow"),
		middleware.AuthRequired(cfg.JWTSecret),
		middleware.AppScopeRequired("finflow"),
	)

	// TypResume：用户简历云端同步。受 typresume scope 保护，与 finflow 隔离。
	typresumeHandler := typresume.NewHandler(pg)
	typresumeHandler.Register(
		v1.Group("/typresume"),
		middleware.AuthRequired(cfg.JWTSecret),
		middleware.AppScopeRequired("typresume"),
	)

	// FluxBlog：独立博客写作 API。独立 JWT/账号/表族，与 finflow/admin 隔离。
	// 发布/撤回为 DB 内同步状态翻转，不再提交 Git。
	// adminVerifier 注入 auth.Repository 实现：供 blog login 交叉验证 admin 凭据、
	// admin-preview 端点校验身份。blog 包不直接依赖 internal/auth，仍保持隔离。
	blogRepo := blog.NewRepository(pg)
	blogHandler := blog.NewHandler(blogRepo, cfg.BlogJWTSecret, cfg.BlogAssetDir, authRepo)
	blogHandler.Register(v1.Group("/blog"))

	// 定时发布后台 worker：每 60s 扫描到点的 scheduled_publish_at 草稿并提升为 published。
	blogSched := blog.NewScheduler(blogRepo)
	blogCtx, blogCancel := context.WithCancel(context.Background())
	defer blogCancel()
	blogSched.Start(blogCtx)

	// 埋点：公开 track 端点 + admin 鉴权的分析查询
	analyticsRepo := analytics.NewRepository(pg)
	analyticsHandler := analytics.NewHandler(analyticsRepo)
	analyticsHandler.RegisterPublic(v1.Group("/analytics", middleware.AuthOptional(cfg.JWTSecret)))
	analyticsHandler.RegisterAdmin(
		v1.Group("/admin"),
		middleware.AuthRequired(cfg.JWTSecret),
		middleware.AdminRequired(),
	)

	adminHandler := admin.NewHandler(pg, authRepo, cfg.JWTSecret, blogRepo, cfg.BlogJWTSecret)
	adminHandler.Register(
		v1.Group("/admin"),
		middleware.AuthRequired(cfg.JWTSecret),
		middleware.AdminRequired(),
	)

	// 仪表盘：CRUD + 数据源查询，admin 鉴权。复用与 admin 相同的中间件链。
	dashboardRepo := dashboard.NewRepository(pg)
	dsRegistry := dashboard.NewRegistry(pg)
	dashboardHandler := dashboard.NewHandler(dashboardRepo, dsRegistry)
	dashboardHandler.RegisterAdmin(
		v1.Group("/admin"),
		middleware.AuthRequired(cfg.JWTSecret),
		middleware.AdminRequired(),
	)

	// Hub：私人书签/Prompts/Skills 工作台。复用 admin 账号 + 主 JWT。
	hubRepo := hub.NewRepository(pg)
	hubHandler := hub.NewHandler(hubRepo)
	hubHandler.Register(
		v1.Group("/hub",
			middleware.AuthRequired(cfg.JWTSecret),
			middleware.AdminRequired(),
		),
	)

	klog.Infof("listening on %s", cfg.Address)
	return r.Run(cfg.Address)
}

func createAdminCmd(cfg *config.Config) *cobra.Command {
	var username, password string
	c := &cobra.Command{
		Use:   "create-admin",
		Short: "Create an admin user",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg, err := db.NewPostgres(cfg.DSN)
			if err != nil {
				return err
			}
			defer pg.Close()
			if err := db.Migrate(pg); err != nil {
				return err
			}
			repo := auth.NewRepository(pg)
			u, err := repo.CreateAdmin(username, password)
			if err != nil {
				return err
			}
			if err := db.SeedForUser(pg, u.ID); err != nil {
				return fmt.Errorf("seed: %w", err)
			}
			fmt.Printf("admin created: id=%d username=%s\n", u.ID, u.Username)
			return nil
		},
	}
	c.Flags().StringVar(&username, "username", "", "admin username (required)")
	c.Flags().StringVar(&password, "password", "", "admin password (required)")
	_ = c.MarkFlagRequired("username")
	_ = c.MarkFlagRequired("password")
	return c
}

func seedCmd(cfg *config.Config) *cobra.Command {
	var username string
	c := &cobra.Command{
		Use:   "seed",
		Short: "Seed default categories & accounts for an existing user",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg, err := db.NewPostgres(cfg.DSN)
			if err != nil {
				return err
			}
			defer pg.Close()
			repo := auth.NewRepository(pg)
			u, err := repo.FindByUsername(username)
			if err != nil {
				return err
			}
			if err := db.SeedForUser(pg, u.ID); err != nil {
				return fmt.Errorf("seed: %w", err)
			}
			fmt.Printf("seeded: id=%d username=%s\n", u.ID, u.Username)
			return nil
		},
	}
	c.Flags().StringVar(&username, "username", "", "username to seed (required)")
	_ = c.MarkFlagRequired("username")
	return c
}
