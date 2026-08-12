package typresume

import (
	"encoding/json"
	"time"
)

// Resume 是用户简历的持久化模型。
// ID 是服务器 PK；ClientID 是客户端生成的稳定 UUID，用于离线创建后同步。
// Mode: 'form' 或 'typst'。'form' 时 Content 是权威源，Files 是 render 产物。
// Content: 结构化简历数据（basics + sections），仅 form 模式使用。
type Resume struct {
	ID         int64             `json:"-"`
	ClientID   string            `json:"id"`
	Name       string            `json:"name"`
	Mode       string            `json:"mode"`
	Content    json.RawMessage   `json:"content,omitempty"`
	ActiveFile string            `json:"activeFile"`
	Files      map[string]string `json:"files"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}
