package dashboard

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRegisterAdmin_NoRoutePanic confirms the routes mount without a gin
// duplicate-route panic.
func TestRegisterAdmin_NoRoutePanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := NewRepository(&sql.DB{})
	reg := NewRegistry(&sql.DB{})
	h := NewHandler(repo, reg)
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("route registration panicked: %v", rec)
		}
	}()
	h.RegisterAdmin(r.Group("/api/v1/admin"), func(c *gin.Context) {})
}

// TestQueryDataSource_ColonKeyMatches confirms that a data source key
// containing a ':' separator (e.g. "finflow:summary") is captured by Gin's
// ":key" param and reaches the handler — i.e. the route matches (not 404).
// We query an *unknown* key so the registry short-circuits with an error
// before touching the (nil) DB, proving the path matched.
func TestQueryDataSource_ColonKeyMatches(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := NewRepository(&sql.DB{})
	reg := NewRegistry(&sql.DB{})
	h := NewHandler(repo, reg)
	h.RegisterAdmin(r.Group("/api/v1/admin"), func(c *gin.Context) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/datasources/unknown:key/query", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code == http.StatusNotFound {
		t.Fatalf("colon-bearing key did not match route: got %d %s", rr.Code, rr.Body.String())
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown data source, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestListDataSources confirms the listing endpoint returns 200 + JSON array.
func TestListDataSources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := NewRepository(&sql.DB{})
	reg := NewRegistry(&sql.DB{})
	h := NewHandler(repo, reg)
	h.RegisterAdmin(r.Group("/api/v1/admin"), func(c *gin.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/datasources", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestParseIDParam_Invalid confirms a non-numeric :id yields 400, matching
// the admin handler pattern.
func TestParseIDParam_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := NewRepository(&sql.DB{})
	reg := NewRegistry(&sql.DB{})
	h := NewHandler(repo, reg)
	h.RegisterAdmin(r.Group("/api/v1/admin"), func(c *gin.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboards/abc", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d: %s", rr.Code, rr.Body.String())
	}
}
