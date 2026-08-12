package typresume

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{repo: NewRepository(db)}
}

func (h *Handler) Register(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	g := rg.Use(middlewares...)
	{
		g.GET("/resumes", h.list)
		g.POST("/resumes", h.upsert)
		g.PUT("/resumes/:client_id", h.update)
		g.DELETE("/resumes/:client_id", h.delete)
		g.POST("/resumes/bulk", h.bulk)
	}
}

func userID(c *gin.Context) int64 {
	v, _ := c.Get("userID")
	id, _ := v.(int64)
	return id
}

func (h *Handler) list(c *gin.Context) {
	uid := userID(c)
	items, err := h.repo.ListByUser(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) upsert(c *gin.Context) {
	var in Resume
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if in.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id (client_id) required"})
		return
	}
	out, err := h.repo.UpsertByClientID(userID(c), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// update 用 URL 路径参数指定 client_id，body 里也可以带同名字段（以 URL 为准）。
func (h *Handler) update(c *gin.Context) {
	var in Resume
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in.ClientID = c.Param("client_id")
	if in.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id required"})
		return
	}
	out, err := h.repo.UpsertByClientID(userID(c), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) delete(c *gin.Context) {
	clientID := c.Param("client_id")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id required"})
		return
	}
	if err := h.repo.DeleteByClientID(userID(c), clientID); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// bulk 首次登录迁移专用：批量 upsert；返回服务端权威数据。
func (h *Handler) bulk(c *gin.Context) {
	var in []Resume
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.BulkUpsert(userID(c), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}
