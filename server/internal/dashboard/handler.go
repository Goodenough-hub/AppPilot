package dashboard

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler exposes dashboard CRUD and data source query endpoints under the
// admin API. All routes are mounted behind the admin auth middlewares.
type Handler struct {
	repo     *Repository
	registry *Registry
}

// NewHandler constructs a dashboard Handler backed by the given repository and
// data source registry.
func NewHandler(repo *Repository, registry *Registry) *Handler {
	return &Handler{repo: repo, registry: registry}
}

// RegisterAdmin wires every dashboard endpoint onto the given router group,
// applying the supplied middlewares (admin auth) to all routes.
func (h *Handler) RegisterAdmin(rg *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	g := rg.Use(middlewares...)
	{
		g.GET("/dashboards", h.listDashboards)
		g.GET("/dashboards/:id", h.getDashboard)
		g.PUT("/dashboards/:id", h.updateDashboard)
		g.POST("/dashboards/:id/widgets", h.createWidget)
		g.PUT("/dashboards/:id/widgets/:wid", h.updateWidget)
		g.DELETE("/dashboards/:id/widgets/:wid", h.deleteWidget)
		g.GET("/datasources", h.listDataSources)
		g.POST("/datasources/:key/query", h.queryDataSource)
	}
}

// ---- dashboards ----

func (h *Handler) listDashboards(c *gin.Context) {
	ds, err := h.repo.ListDashboards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ds)
}

func (h *Handler) getDashboard(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	d, err := h.repo.GetDashboard(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	widgets, err := h.repo.ListWidgets(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dashboard": d, "widgets": widgets})
}

// updateDashboardRequest carries the optional patch fields for a dashboard.
// Omitted JSON fields stay nil so the repository leaves them unchanged.
type updateDashboardRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (h *Handler) updateDashboard(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req updateDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, err := h.repo.UpdateDashboard(id, req.Title, req.Description)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dashboard not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, d)
}

// ---- widgets ----

func (h *Handler) createWidget(c *gin.Context) {
	dashboardID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var w Widget
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.repo.CreateWidget(dashboardID, w)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) updateWidget(c *gin.Context) {
	_, ok := parseIDParam(c, "id") // dashboard id validates the path scope
	if !ok {
		return
	}
	wid, ok := parseIDParam(c, "wid")
	if !ok {
		return
	}
	var w Widget
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.repo.UpdateWidget(wid, w)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteWidget(c *gin.Context) {
	_, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	wid, ok := parseIDParam(c, "wid")
	if !ok {
		return
	}
	if err := h.repo.DeleteWidget(wid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "widget not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- data sources ----

func (h *Handler) listDataSources(c *gin.Context) {
	c.JSON(http.StatusOK, h.registry.List())
}

// queryDataSource reads the data source key from the URL (e.g.
// "finflow:summary") and forwards any JSON body params to the source.
//
// Note on the colon: a data source key is a single path segment containing a
// ':' separator (finflow:summary). ':' is a valid character within a path
// segment, so Gin's ":key" captures the whole key with no encoding tricks.
func (h *Handler) queryDataSource(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing data source key"})
		return
	}
	params := map[string]any{}
	// An empty body is fine — just query with an empty param map.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	data, err := h.registry.Query(key, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "data": data})
}

// ---- helpers ----

// parseIDParam parses a base-10 int64 path param, writing a 400 response on
// failure. Returns ok=false (already responded) so callers can return early.
func parseIDParam(c *gin.Context, key string) (int64, bool) {
	s := c.Param(key)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + key})
		return 0, false
	}
	return id, true
}
