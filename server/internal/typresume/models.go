package typresume

import "time"

// Resume 是用户简历的持久化模型。
// ID 是服务器 PK；ClientID 是客户端生成的稳定 UUID，用于离线创建后同步。
type Resume struct {
	ID         int64             `json:"-"`
	ClientID   string            `json:"id"`         // 对外用 client_id 作为 id
	Name       string            `json:"name"`
	ActiveFile string            `json:"activeFile"`
	Files      map[string]string `json:"files"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}
