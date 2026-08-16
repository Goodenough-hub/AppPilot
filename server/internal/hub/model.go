package hub

import (
	"errors"
	"time"
)

// Item 类型常量（与 hub_items.type CHECK 约束对齐）
const (
	TypeBookmark = "bookmark"
	TypePrompt   = "prompt"
	TypeSkill    = "skill"
)

// Item 对应 hub_items 表一行；tags 用 []string 与 PG TEXT[] 映射。
type Item struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"` // 服务端注入，不出对外 JSON
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	URL       *string   `json:"url"`     // 可空
	Content   *string   `json:"content"` // 可空
	Tags      []string  `json:"tags"`
	Favorite  bool      `json:"favorite"`
	Folder    string    `json:"folder"`   // 文件夹名，空串 = 未分类；命名空间随 item.Type
	Icon      string    `json:"icon"`     // 自定义图标 URL，空串 = 按站点 favicon 自动探测
	Position  int       `json:"position"` // 文件夹内手动排序位（0 = 未排序，排过的为 1..n）
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Validate 校验必填字段与长度约束。返回稳定字符串错误便于测试。
func (i *Item) Validate() error {
	if i.Title == "" {
		return errors.New("title required")
	}
	if len(i.Title) > 500 {
		return errors.New("title too long")
	}
	if len(i.Folder) > 200 {
		return errors.New("folder too long")
	}
	if len(i.Icon) > 1000 {
		return errors.New("icon too long")
	}
	switch i.Type {
	case TypeBookmark, TypePrompt, TypeSkill:
		return nil
	default:
		return errors.New("invalid type")
	}
}

// Folder 对应 hub_folders 表一行。命名空间按 (user_id, type) 隔离：
// 同名文件夹在 bookmark / prompt / skill 下互不相干。
type Folder struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"` // 服务端注入，不出对外 JSON
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	ItemCount int       `json:"itemCount"` // 列表接口聚合得出，不落库
	CreatedAt time.Time `json:"createdAt"`
}

// Validate 校验文件夹字段。返回稳定字符串错误便于测试。
func (f *Folder) Validate() error {
	if f.Name == "" {
		return errors.New("name required")
	}
	if len(f.Name) > 200 {
		return errors.New("name too long")
	}
	switch f.Type {
	case TypeBookmark, TypePrompt, TypeSkill:
		return nil
	default:
		return errors.New("invalid type")
	}
}
