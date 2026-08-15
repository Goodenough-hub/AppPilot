package hub

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// Register 挂路由。调用者负责在 rg 上先叠好 AuthRequired + AdminRequired。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/items", h.list)
	rg.POST("/items", h.create)
	rg.PATCH("/items/:id", h.update)
	rg.DELETE("/items/:id", h.remove)
	// Export/Import 在 Task 6 添加
}

func userIDOf(c *gin.Context) int64 {
	v, _ := c.Get("userID")
	id, _ := v.(int64)
	return id
}

func (h *Handler) list(c *gin.Context) {
	items, err := h.repo.List(userIDOf(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

type createRequest struct {
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	URL      *string  `json:"url"`
	Content  *string  `json:"content"`
	Tags     []string `json:"tags"`
	Favorite bool     `json:"favorite"`
}

func (h *Handler) create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	it := &Item{
		Type: req.Type, Title: req.Title,
		URL: req.URL, Content: req.Content,
		Tags: req.Tags, Favorite: req.Favorite,
	}
	if err := it.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.repo.Create(userIDOf(c), it)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

type updateRequest struct {
	Type     *string   `json:"type"`
	Title    *string   `json:"title"`
	URL      *string   `json:"url"`
	Content  *string   `json:"content"`
	Tags     *[]string `json:"tags"`
	Favorite *bool     `json:"favorite"`
}

func (h *Handler) update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 若同时提供 type/title 之一，做一次 Validate（用一个探针 Item 校验字段合法性）
	if req.Type != nil || req.Title != nil {
		probe := Item{
			Type:  "bookmark", // 默认合法占位
			Title: "x",        // 默认合法占位
		}
		if req.Type != nil {
			probe.Type = *req.Type
		}
		if req.Title != nil {
			probe.Title = *req.Title
		}
		if err := probe.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	updated, err := h.repo.Update(userIDOf(c), id, UpdatePatch{
		Type: req.Type, Title: req.Title, URL: req.URL,
		Content: req.Content, Tags: req.Tags, Favorite: req.Favorite,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) remove(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.Delete(userIDOf(c), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
