package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	Address   string
	DSN       string
	JWTSecret string

	// FluxBlog 独立配置。blog 与现有 finflow/admin 隔离：
	// 独立 JWT secret、独立 GitHub 仓库发布、独立草稿图片暂存目录。
	// 这些字段为空时 blog 路由仍注册，但请求会在运行期返回 503，
	// 不影响 finflow/admin（保持现有部署向后兼容）。
	BlogJWTSecret            string
	BlogGitHubToken          string
	BlogRepo                 string
	BlogBranch               string
	BlogAssetDir             string
	BlogDeployCallbackSecret string
}

func Load() Config {
	return Config{
		Address:   getenv("APPPLOT_ADDRESS", "127.0.0.1:8080"),
		DSN:       getenv("APPPLOT_DSN", ""),
		JWTSecret: getenv("APPPLOT_JWT_SECRET", ""),

		BlogJWTSecret:            getenv("APPPLOT_BLOG_JWT_SECRET", ""),
		BlogGitHubToken:          getenv("APPPLOT_BLOG_GITHUB_TOKEN", ""),
		BlogRepo:                 getenv("APPPLOT_BLOG_REPO", "Goodenough-hub/FluxBlog"),
		BlogBranch:               getenv("APPPLOT_BLOG_BRANCH", "main"),
		BlogAssetDir:             getenv("APPPLOT_BLOG_ASSET_DIR", "/var/lib/apppilot/fluxblog"),
		BlogDeployCallbackSecret: getenv("APPPLOT_BLOG_DEPLOY_CALLBACK_SECRET", ""),
	}
}

func (c Config) Validate() error {
	if c.DSN == "" {
		return errors.New("APPPLOT_DSN is required")
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("APPPLOT_JWT_SECRET must be at least 32 chars")
	}
	return nil
}

// BlogEnabled 报告 blog 是否具备可工作配置。
// 缺少 JWT secret 或 GitHub token 时视为未启用，blog 接口返回 503。
func (c Config) BlogEnabled() bool {
	return len(c.BlogJWTSecret) >= 32 && c.BlogGitHubToken != "" && c.BlogRepo != ""
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
