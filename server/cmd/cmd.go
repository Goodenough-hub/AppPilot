package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"apppilot-server/internal/admin"
	"apppilot-server/internal/auth"
	"apppilot-server/internal/db"
	"apppilot-server/internal/finflow"
	"apppilot-server/internal/middleware"
	"apppilot-server/pkg/config"

	"github.com/gin-gonic/gin"
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

	finflowHandler := finflow.NewHandler(pg)
	finflowHandler.Register(
		v1.Group("/finflow"),
		middleware.AuthRequired(cfg.JWTSecret),
		middleware.AppScopeRequired("finflow"),
	)

	adminHandler := admin.NewHandler(pg, authRepo, cfg.JWTSecret)
	adminHandler.Register(
		v1.Group("/admin"),
		middleware.AuthRequired(cfg.JWTSecret),
		middleware.AdminRequired(),
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
