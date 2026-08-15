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
	switch i.Type {
	case TypeBookmark, TypePrompt, TypeSkill:
		return nil
	default:
		return errors.New("invalid type")
	}
}
