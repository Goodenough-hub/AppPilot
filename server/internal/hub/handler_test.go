package hub

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupTestRouter 挂 handler，人工注入 userID = 1（模拟 AuthRequired 已过）
func setupTestRouter(t *testing.T) (*gin.Engine, *Repository) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)
	repo := NewRepository(db)
	h := NewHandler(repo)
	r := gin.New()
	rg := r.Group("/hub", func(c *gin.Context) {
		c.Set("userID", int64(1))
		c.Set("role", "admin")
		c.Next()
	})
	h.Register(rg)
	return r, repo
}

func doJSON(r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandlerCRUD(t *testing.T) {
	r, _ := setupTestRouter(t)

	// POST create
	w := doJSON(r, "POST", "/hub/items", map[string]any{
		"type": "prompt", "title": "P1", "content": "hello", "tags": []string{"x"},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", w.Code, w.Body.String())
	}
	var created Item
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// GET list
	w = doJSON(r, "GET", "/hub/items", nil)
	if w.Code != 200 {
		t.Fatalf("list code=%d", w.Code)
	}
	var items []Item
	_ = json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) != 1 || items[0].Title != "P1" {
		t.Fatalf("list mismatch: %+v", items)
	}

	// PATCH favorite
	w = doJSON(r, "PATCH", "/hub/items/"+strconv.FormatInt(created.ID, 10), map[string]any{"favorite": true})
	if w.Code != 200 {
		t.Fatalf("patch code=%d body=%s", w.Code, w.Body.String())
	}

	// DELETE
	w = doJSON(r, "DELETE", "/hub/items/"+strconv.FormatInt(created.ID, 10), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete code=%d", w.Code)
	}
}

func TestHandlerCreateValidation(t *testing.T) {
	r, _ := setupTestRouter(t)
	// 空 title
	w := doJSON(r, "POST", "/hub/items", map[string]any{"type": "prompt", "title": ""})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	// 无效 type
	w = doJSON(r, "POST", "/hub/items", map[string]any{"type": "note", "title": "x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlerUpdateInvalidID(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := doJSON(r, "PATCH", "/hub/items/abc", map[string]any{"favorite": true})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d", w.Code)
	}
}

func TestHandlerUpdateNotFound(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := doJSON(r, "PATCH", "/hub/items/999999", map[string]any{"favorite": true})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing id, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerUpdateInvalidType(t *testing.T) {
	r, _ := setupTestRouter(t)
	// 先建一条，再尝试把它 type 改成非法值
	w := doJSON(r, "POST", "/hub/items", map[string]any{"type": "bookmark", "title": "x"})
	if w.Code != http.StatusCreated {
		t.Fatalf("seed: %d", w.Code)
	}
	var created Item
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	w = doJSON(r, "PATCH", "/hub/items/"+strconv.FormatInt(created.ID, 10), map[string]any{"type": "note"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid type patch, got %d", w.Code)
	}
}

func TestHandlerRemoveInvalidID(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := doJSON(r, "DELETE", "/hub/items/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d", w.Code)
	}
}

func TestHandlerRemoveNotFound(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := doJSON(r, "DELETE", "/hub/items/999999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing id, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerExportImport(t *testing.T) {
	r, _ := setupTestRouter(t)

	// 先创建 2 条
	doJSON(r, "POST", "/hub/items", map[string]any{"type": "bookmark", "title": "E1"})
	doJSON(r, "POST", "/hub/items", map[string]any{"type": "prompt", "title": "E2"})

	// export
	w := doJSON(r, "GET", "/hub/export", nil)
	if w.Code != 200 {
		t.Fatalf("export code=%d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("export content-type=%q", got)
	}
	var dump []Item
	if err := json.Unmarshal(w.Body.Bytes(), &dump); err != nil {
		t.Fatalf("export unmarshal: %v", err)
	}
	if len(dump) != 2 {
		t.Fatalf("dump len=%d", len(dump))
	}

	// import replace with different data
	w = doJSON(r, "POST", "/hub/import?mode=replace", []map[string]any{
		{"type": "skill", "title": "I1"},
	})
	if w.Code != 200 {
		t.Fatalf("import code=%d body=%s", w.Code, w.Body.String())
	}
	w = doJSON(r, "GET", "/hub/items", nil)
	var items []Item
	_ = json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) != 1 || items[0].Title != "I1" {
		t.Fatalf("after replace: %+v", items)
	}
}

func TestHandlerImportInvalidMode(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := doJSON(r, "POST", "/hub/import?mode=merge_replace", []map[string]any{
		{"type": "bookmark", "title": "x"},
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid mode, got %d", w.Code)
	}
}

func TestHandlerImportRejectsEmptyReplace(t *testing.T) {
	r, _ := setupTestRouter(t)
	w := doJSON(r, "POST", "/hub/import?mode=replace", []map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty replace, got %d body=%s", w.Code, w.Body.String())
	}
}
