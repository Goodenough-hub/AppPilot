package finflow

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

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
		g.GET("/transactions", h.listTransactions)
		g.POST("/transactions", h.createTransaction)
		g.GET("/transactions/:id", h.getTransaction)
		g.PUT("/transactions/:id", h.updateTransaction)
		g.DELETE("/transactions/:id", h.deleteTransaction)

		g.GET("/categories", h.listCategories)
		g.POST("/categories", h.createCategory)
		g.PUT("/categories/:id", h.updateCategory)
		g.DELETE("/categories/:id", h.deleteCategory)

		g.GET("/accounts", h.listAccounts)
		g.POST("/accounts", h.createAccount)
		g.PUT("/accounts/:id", h.updateAccount)
		g.DELETE("/accounts/:id", h.deleteAccount)
		g.DELETE("/accounts", h.clearAccounts)

		g.GET("/budgets", h.listBudgets)
		g.POST("/budgets", h.upsertBudget)
		g.DELETE("/budgets/:id", h.deleteBudget)

		g.GET("/recurring", h.listRecurring)
		g.POST("/recurring", h.createRecurring)
		g.PUT("/recurring/:id", h.updateRecurring)
		g.DELETE("/recurring/:id", h.deleteRecurring)
		g.POST("/recurring/process", h.processRecurring)

		g.GET("/trips", h.listTrips)
		g.POST("/trips", h.createTrip)
		g.PUT("/trips/:id", h.updateTrip)
		g.DELETE("/trips/:id", h.deleteTrip)

		g.GET("/stats/summary", h.summary)
		g.GET("/stats/category-breakdown", h.categoryBreakdown)
		g.GET("/stats/daily-trend", h.dailyTrend)
	}
}

func userID(c *gin.Context) int64 {
	v, _ := c.Get("userID")
	id, _ := v.(int64)
	return id
}

func parseIDParam(c *gin.Context, key string) (int64, bool) {
	s := c.Param(key)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

// ---- Transactions ----

func (h *Handler) listTransactions(c *gin.Context) {
	f := ListTxFilter{}
	if s := c.Query("startDate"); s != "" {
		f.StartDate = &s
	}
	if s := c.Query("endDate"); s != "" {
		f.EndDate = &s
	}
	if s := c.Query("type"); s != "" {
		f.Type = &s
	}
	if s := c.Query("categoryId"); s != "" {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			f.CategoryID = &id
		}
	}
	if s := c.Query("accountId"); s != "" {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			f.AccountID = &id
		}
	}
	if s := c.Query("tripId"); s != "" {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			f.TripID = &id
		}
	}
	if s := c.Query("keyword"); s != "" {
		f.Keyword = &s
	}
	f.Limit, _ = strconv.Atoi(c.DefaultQuery("pageSize", "0"))
	f.Offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	txs, err := h.repo.ListTransactions(userID(c), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}

func (h *Handler) createTransaction(c *gin.Context) {
	var t Transaction
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.CreateTransaction(userID(c), t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) getTransaction(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	t, err := h.repo.GetTransaction(userID(c), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *Handler) updateTransaction(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var t Transaction
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.UpdateTransaction(userID(c), id, t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) deleteTransaction(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.repo.DeleteTransaction(userID(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- Categories ----

func (h *Handler) listCategories(c *gin.Context) {
	cats, err := h.repo.ListCategories(userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cats)
}

func (h *Handler) createCategory(c *gin.Context) {
	var cat Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.CreateCategory(userID(c), cat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) updateCategory(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var cat Category
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.UpdateCategory(userID(c), id, cat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found or system category"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) deleteCategory(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.repo.DeleteCategory(userID(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- Accounts ----

func (h *Handler) listAccounts(c *gin.Context) {
	accs, err := h.repo.ListAccounts(userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accs)
}

func (h *Handler) createAccount(c *gin.Context) {
	var a Account
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.CreateAccount(userID(c), a)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) updateAccount(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var a Account
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.UpdateAccount(userID(c), id, a)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) deleteAccount(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.repo.DeleteAccount(userID(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) clearAccounts(c *gin.Context) {
	if err := h.repo.ClearAccounts(userID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- Budgets ----

func (h *Handler) listBudgets(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
	bs, err := h.repo.ListBudgets(userID(c), year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bs)
}

func (h *Handler) upsertBudget(c *gin.Context) {
	var b Budget
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.UpsertBudget(userID(c), b)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) deleteBudget(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.repo.DeleteBudget(userID(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ---- Recurring ----

func (h *Handler) listRecurring(c *gin.Context) {
	rs, err := h.repo.ListRecurring(userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rs)
}

func (h *Handler) createRecurring(c *gin.Context) {
	var r Recurring
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.CreateRecurring(userID(c), r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) updateRecurring(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var r Recurring
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.UpdateRecurring(userID(c), id, r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) deleteRecurring(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.repo.DeleteRecurring(userID(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) processRecurring(c *gin.Context) {
	n, err := h.repo.ProcessDue(userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"processed": n})
}

// ---- Stats ----

func (h *Handler) summary(c *gin.Context) {
	start := c.Query("startDate")
	end := c.Query("endDate")
	s, err := h.repo.Summary(userID(c), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) categoryBreakdown(c *gin.Context) {
	txType := c.DefaultQuery("type", "expense")
	start := c.Query("startDate")
	end := c.Query("endDate")
	stats, err := h.repo.CategoryBreakdown(userID(c), txType, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) dailyTrend(c *gin.Context) {
	start := c.Query("startDate")
	end := c.Query("endDate")
	stats, err := h.repo.DailyTrend(userID(c), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ---- Trips ----

func (h *Handler) listTrips(c *gin.Context) {
	trips, err := h.repo.ListTrips(userID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, trips)
}

func (h *Handler) createTrip(c *gin.Context) {
	var t Trip
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.CreateTrip(userID(c), t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) updateTrip(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var t Trip
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.UpdateTrip(userID(c), id, t)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) deleteTrip(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.repo.DeleteTrip(userID(c), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
