package hub

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// Register 挂路由。调用者负责在 rg 上先叠好 AuthRequired + AppScopeRequired("hub")。
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.GET("/items", h.list)
	rg.POST("/items", h.create)
	rg.PATCH("/items/:id", h.update)
	rg.DELETE("/items/:id", h.remove)
	rg.GET("/export", h.export)
	rg.POST("/import", h.importBatch)
	rg.GET("/folders", h.listFolders)
	rg.POST("/folders", h.createFolder)
	rg.PATCH("/folders/:id", h.renameFolder)
	rg.DELETE("/folders/:id", h.deleteFolder)
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
	Folder   string   `json:"folder"`
	Icon     string   `json:"icon"`
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
		Folder: req.Folder, Icon: req.Icon,
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
	Folder   *string   `json:"folder"`
	Icon     *string   `json:"icon"`
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
	// 若提供 type/title/folder/icon 之一，做一次 Validate（用一个探针 Item 校验字段合法性）
	if req.Type != nil || req.Title != nil || req.Folder != nil || req.Icon != nil {
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
		if req.Folder != nil {
			probe.Folder = *req.Folder
		}
		if req.Icon != nil {
			probe.Icon = *req.Icon
		}
		if err := probe.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	updated, err := h.repo.Update(userIDOf(c), id, UpdatePatch{
		Type: req.Type, Title: req.Title, URL: req.URL,
		Content: req.Content, Tags: req.Tags, Favorite: req.Favorite,
		Folder: req.Folder, Icon: req.Icon,
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

func (h *Handler) export(c *gin.Context) {
	items, err := h.repo.ExportAll(userIDOf(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 附件下载
	c.Header("Content-Disposition", `attachment; filename="hub-export.json"`)
	c.JSON(http.StatusOK, items)
}

func (h *Handler) importBatch(c *gin.Context) {
	mode := c.DefaultQuery("mode", "merge")
	if mode != "merge" && mode != "replace" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be merge or replace"})
		return
	}
	var items []Item
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// replace 模式必须至少 1 条，防止空 payload 静默清空数据（与 repository 双重保险）
	if mode == "replace" && len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "replace mode requires non-empty items"})
		return
	}
	// 逐条校验；merge 允许 id 已存在
	for i := range items {
		if err := items[i].Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "index": i})
			return
		}
	}
	n, err := h.repo.ImportBatch(userIDOf(c), items, mode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"affected": n, "mode": mode})
}

// ---- folders ----

func (h *Handler) listFolders(c *gin.Context) {
	typ := c.Query("type")
	// 复用 Folder.Validate 校验 type 合法性（Name 用占位符绕过非空校验）
	if err := (&Folder{Type: typ, Name: "x"}).Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
		return
	}
	folders, err := h.repo.ListFolders(userIDOf(c), typ)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, folders)
}

type folderRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (h *Handler) createFolder(c *gin.Context) {
	var req folderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	f := &Folder{Type: req.Type, Name: strings.TrimSpace(req.Name)}
	if err := f.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.repo.CreateFolder(userIDOf(c), f)
	if err != nil {
		if errors.Is(err, ErrFolderExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) renameFolder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	// 探针校验（Type 占位合法值，只验 name）
	if err := (&Folder{Type: TypeBookmark, Name: name}).Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.repo.RenameFolder(userIDOf(c), id, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if errors.Is(err, ErrFolderExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteFolder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.DeleteFolder(userIDOf(c), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
