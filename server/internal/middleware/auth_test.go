package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupScopeRouter 挂 AppScopeRequired("hub")，人工注入给定的 appScope（模拟 AuthRequired 已过）
func setupScopeRouter(appScope []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rg := r.Group("/hub", func(c *gin.Context) {
		c.Set("appScope", appScope)
		c.Next()
	}, AppScopeRequired("hub"))
	rg.GET("/items", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestAppScopeRequired(t *testing.T) {
	cases := []struct {
		name     string
		appScope []string
		want     int
	}{
		{"scope 含 hub 放行", []string{"finflow", "hub"}, http.StatusOK},
		{"admin 伪 scope 直通", []string{"finflow", "admin"}, http.StatusOK},
		{"scope 不含 hub 拒绝", []string{"finflow"}, http.StatusForbidden},
		{"空 scope 拒绝", []string{}, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := setupScopeRouter(tc.appScope)
			req := httptest.NewRequest(http.MethodGet, "/hub/items", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d", w.Code, tc.want)
			}
		})
	}
}
