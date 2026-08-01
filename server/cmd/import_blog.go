package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"apppilot-server/internal/blog"
	"apppilot-server/internal/db"
	"apppilot-server/pkg/config"
)

// importBlogCmd 把 FluxBlog 存量 Markdown（src/content/posts/*.md）幂等导入 blog_drafts。
// 内容以 DB 为权威源后，公开站改由 /api/v1/blog/posts 读取，不再依赖 Git 仓库内容。
func importBlogCmd(cfg *config.Config) *cobra.Command {
	var dir, username string
	c := &cobra.Command{
		Use:   "import-blog",
		Short: "Import FluxBlog posts from Markdown files into the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			pg, err := db.NewPostgres(cfg.DSN)
			if err != nil {
				return err
			}
			defer pg.Close()
			if err := db.Migrate(pg); err != nil {
				return err
			}
			repo := blog.NewRepository(pg)
			u, err := repo.FindByUsernameActive(username)
			if err != nil {
				return fmt.Errorf("find blog user %q: %w", username, err)
			}
			files, err := filepath.Glob(filepath.Join(dir, "*.md"))
			if err != nil {
				return err
			}
			var imported, skipped int
			for _, f := range files {
				base := filepath.Base(f)
				if strings.HasPrefix(base, "_") {
					continue // 下划线前缀的文件被 content 集合忽略，跳过。
				}
				d, publishedAt, skip, err := parsePostFile(f)
				if err != nil {
					klog.Errorf("parse %s: %v", base, err)
					continue
				}
				if skip {
					skipped++
					continue
				}
				if _, err := repo.ImportDraft(u.ID, d, publishedAt, true); err != nil {
					return fmt.Errorf("import %s: %w", base, err)
				}
				imported++
				klog.Infof("imported: %s (slug=%s)", base, d.Slug)
			}
			fmt.Printf("import-blog done: imported=%d skipped=%d user=%s\n", imported, skipped, username)
			return nil
		},
	}
	c.Flags().StringVar(&dir, "dir", "src/content/posts", "directory of FluxBlog post .md files")
	c.Flags().StringVar(&username, "username", "", "blog username to own imported posts (required)")
	_ = c.MarkFlagRequired("username")
	return c
}

type postFrontmatter struct {
	Title       string   `yaml:"title"`
	Slug        string   `yaml:"slug"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Cover       *string  `yaml:"cover"`
	PublishedAt string   `yaml:"publishedAt"`
	UpdatedAt   string   `yaml:"updatedAt"`
	Draft       bool     `yaml:"draft"`
}

// parsePostFile 解析 Markdown 文件：YAML frontmatter + 正文。
// 返回 Draft、publishedAt（可能为零值 nil）、skip（draft:true 跳过）、error。
func parsePostFile(path string) (blog.Draft, *time.Time, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return blog.Draft{}, nil, false, err
	}
	body := string(data)
	if !strings.HasPrefix(body, "---\n") {
		return blog.Draft{}, nil, false, fmt.Errorf("missing frontmatter")
	}
	// 找闭合 ---（第二个独占一行的 ---）。
	rest := body[4:]
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		// 末尾可能是 \n--- 无换行。
		idx = strings.Index(rest, "\n---")
		if idx < 0 {
			return blog.Draft{}, nil, false, fmt.Errorf("unterminated frontmatter")
		}
	}
	fmText := rest[:idx]
	var fm postFrontmatter
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return blog.Draft{}, nil, false, fmt.Errorf("parse yaml: %w", err)
	}
	if fm.Draft {
		return blog.Draft{}, nil, true, nil
	}
	markdown := strings.TrimSpace(rest[idx+4:])

	slug := fm.Slug
	if slug == "" {
		slug = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	tags := fm.Tags
	if tags == nil {
		tags = []string{}
	}
	// publishedAt：支持 2006-01-02 与 RFC3339；缺省交给 ImportDraft 用 NOW()。
	var pubAt *time.Time
	if fm.PublishedAt != "" {
		if t, err := parseDate(fm.PublishedAt); err == nil {
			pubAt = &t
		}
	}

	d := blog.Draft{
		Slug:        slug,
		Title:       fm.Title,
		Description: fm.Description,
		Tags:        tags,
		Cover:       fm.Cover,
		Markdown:    markdown,
		Version:     1,
		Visibility:  blog.VisibilityPublic,
	}
	return d, pubAt, false, nil
}

// parseDate 尝试用多种格式解析日期/时间字符串。
func parseDate(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02 15:04:05",
	}
	var lastErr error
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}
