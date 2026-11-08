package oj

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) RegisterPublic(rg *gin.RouterGroup) {
	rg.GET("/problems", h.listPublic)
	rg.GET("/problems/:key", h.getPublic)
}

func (h *Handler) RegisterUser(rg *gin.RouterGroup) {
	rg.POST("/runs", h.createRun)
	rg.POST("/submissions", h.createSubmission)
	rg.GET("/submissions", h.listSubmissions)
	rg.GET("/submissions/:id", h.getSubmission)
}

func (h *Handler) RegisterAdmin(rg *gin.RouterGroup) {
	rg.GET("/problems", h.listAdmin)
	rg.GET("/problems/:key", h.getAdmin)
	rg.POST("/problems", h.createProblem)
	rg.PUT("/problems/:id", h.updateProblem)
	rg.POST("/problems/:id/publish", h.publishProblem)
}

func (h *Handler) listPublic(c *gin.Context) { h.listProblems(c, false) }
func (h *Handler) listAdmin(c *gin.Context)  { h.listProblems(c, true) }
func (h *Handler) listProblems(c *gin.Context, admin bool) {
	items, err := h.repo.ListProblems(admin, NormalizeQuery(c.Query("q")), c.Query("difficulty"))
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}
func (h *Handler) getPublic(c *gin.Context) { h.getProblem(c, false) }
func (h *Handler) getAdmin(c *gin.Context)  { h.getProblem(c, true) }
func (h *Handler) getProblem(c *gin.Context, admin bool) {
	p, err := h.repo.GetProblem(c.Param("key"), admin)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func bindProblem(c *gin.Context) (ProblemInput, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
	var in ProblemInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "题目 JSON 无效或超过 10MB"})
		return in, false
	}
	if err := in.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return in, false
	}
	return in, true
}
func (h *Handler) createProblem(c *gin.Context) {
	in, ok := bindProblem(c)
	if !ok {
		return
	}
	id, err := h.repo.CreateProblem(in)
	if IsUnique(err) {
		c.JSON(http.StatusConflict, gin.H{"error": "题目标识已存在"})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": strconv.FormatInt(id, 10)})
}
func (h *Handler) updateProblem(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	in, ok := bindProblem(c)
	if !ok {
		return
	}
	err := h.repo.UpdateProblem(id, in)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}
	if IsUnique(err) {
		c.JSON(http.StatusConflict, gin.H{"error": "题目标识已存在"})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *Handler) publishProblem(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in struct {
		Published *bool `json:"published"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Published == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要 published 布尔值"})
		return
	}
	err := h.repo.PublishProblem(id, *in.Published)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在"})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type sourceRequest struct {
	ProblemID string `json:"problemId"`
	Code      string `json:"code"`
	Input     string `json:"input"`
}

func (h *Handler) createRun(c *gin.Context)        { h.createSource(c, "run") }
func (h *Handler) createSubmission(c *gin.Context) { h.createSource(c, "submit") }
func (h *Handler) createSource(c *gin.Context, kind string) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 160<<10)
	var in sourceRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求 JSON 无效或过大"})
		return
	}
	id, err := strconv.ParseInt(in.ProblemID, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "题目 ID 无效"})
		return
	}
	if kind == "submit" && in.Input != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "正式提交不接受自定义输入"})
		return
	}
	if err := ValidateSource(in.Code, in.Input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, _ := c.Get("userID")
	uid, _ := userID.(int64)
	subID, err := h.repo.CreateSubmission(uid, id, kind, in.Code, in.Input)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "题目不存在或尚未发布"})
		return
	}
	if errors.Is(err, ErrBusy) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"id": strconv.FormatInt(subID, 10), "status": "queued"})
}
func (h *Handler) listSubmissions(c *gin.Context) {
	userID, _ := c.Get("userID")
	items, err := h.repo.ListSubmissions(userID.(int64))
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}
func (h *Handler) getSubmission(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	userID, _ := c.Get("userID")
	s, err := h.repo.GetSubmission(id, userID.(int64))
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "提交不存在"})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, s)
}
func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 无效"})
		return 0, false
	}
	return id, true
}
func serverError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "服务暂时不可用"})
}
