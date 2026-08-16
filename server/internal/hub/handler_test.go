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

func TestHandlerItemFolderField(t *testing.T) {
	r, _ := setupTestRouter(t)

	// create 带 folder → 201 且响应回显；同时自动登记进 folders
	w := doJSON(r, "POST", "/hub/items", map[string]any{
		"type": "bookmark", "title": "cloud", "folder": "Infini-AI",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create code=%d body=%s", w.Code, w.Body.String())
	}
	var created Item
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.Folder != "Infini-AI" {
		t.Fatalf("created folder = %q", created.Folder)
	}
	w = doJSON(r, "GET", "/hub/folders?type=bookmark", nil)
	var folders []Folder
	_ = json.Unmarshal(w.Body.Bytes(), &folders)
	if len(folders) != 1 || folders[0].Name != "Infini-AI" || folders[0].ItemCount != 1 {
		t.Fatalf("auto upsert via api: %+v", folders)
	}

	// PATCH folder → 200 且回显
	w = doJSON(r, "PATCH", "/hub/items/"+strconv.FormatInt(created.ID, 10), map[string]any{"folder": "other"})
	if w.Code != 200 {
		t.Fatalf("patch folder code=%d body=%s", w.Code, w.Body.String())
	}
	var updated Item
	_ = json.Unmarshal(w.Body.Bytes(), &updated)
	if updated.Folder != "other" {
		t.Fatalf("patched folder = %q", updated.Folder)
	}

	// PATCH folder 超长 → 400
	long := make([]byte, 201)
	w = doJSON(r, "PATCH", "/hub/items/"+strconv.FormatInt(created.ID, 10), map[string]any{"folder": string(long)})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long folder, got %d", w.Code)
	}

	// PATCH icon → 200 且回显；icon 超长 → 400
	w = doJSON(r, "PATCH", "/hub/items/"+strconv.FormatInt(created.ID, 10), map[string]any{"icon": "https://cdn.example.com/f.png"})
	if w.Code != 200 {
		t.Fatalf("patch icon code=%d body=%s", w.Code, w.Body.String())
	}
	var withIcon Item
	_ = json.Unmarshal(w.Body.Bytes(), &withIcon)
	if withIcon.Icon != "https://cdn.example.com/f.png" {
		t.Fatalf("patched icon = %q", withIcon.Icon)
	}
	longIcon := make([]byte, 1001)
	w = doJSON(r, "PATCH", "/hub/items/"+strconv.FormatInt(created.ID, 10), map[string]any{"icon": string(longIcon)})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long icon, got %d", w.Code)
	}
}

func TestHandlerFolders(t *testing.T) {
	r, _ := setupTestRouter(t)

	// POST 新建 → 201
	w := doJSON(r, "POST", "/hub/folders", map[string]any{"type": "bookmark", "name": " Infini-AI "})
	if w.Code != http.StatusCreated {
		t.Fatalf("create folder code=%d body=%s", w.Code, w.Body.String())
	}
	var f Folder
	_ = json.Unmarshal(w.Body.Bytes(), &f)
	if f.Name != "Infini-AI" { // 服务端应 trim
		t.Fatalf("folder name not trimmed: %q", f.Name)
	}

	// 重名 → 409；缺 type → 400
	w = doJSON(r, "POST", "/hub/folders", map[string]any{"type": "bookmark", "name": "Infini-AI"})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	w = doJSON(r, "POST", "/hub/folders", map[string]any{"type": "note", "name": "x"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid type, got %d", w.Code)
	}

	// GET list（type 非法 → 400）
	w = doJSON(r, "GET", "/hub/folders?type=bookmark", nil)
	if w.Code != 200 {
		t.Fatalf("list folders code=%d", w.Code)
	}
	var folders []Folder
	_ = json.Unmarshal(w.Body.Bytes(), &folders)
	if len(folders) != 1 || folders[0].Name != "Infini-AI" {
		t.Fatalf("folders mismatch: %+v", folders)
	}
	w = doJSON(r, "GET", "/hub/folders?type=note", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid type query, got %d", w.Code)
	}

	// 挂两个条目进文件夹，验证 rename 级联
	doJSON(r, "POST", "/hub/items", map[string]any{"type": "bookmark", "title": "A", "folder": "Infini-AI"})
	doJSON(r, "POST", "/hub/items", map[string]any{"type": "bookmark", "title": "B", "folder": "Infini-AI"})
	w = doJSON(r, "PATCH", "/hub/folders/"+strconv.FormatInt(f.ID, 10), map[string]any{"name": "芯穹"})
	if w.Code != 200 {
		t.Fatalf("rename code=%d body=%s", w.Code, w.Body.String())
	}
	w = doJSON(r, "GET", "/hub/items", nil)
	var items []Item
	_ = json.Unmarshal(w.Body.Bytes(), &items)
	for _, it := range items {
		if it.Folder != "芯穹" {
			t.Fatalf("rename cascade failed, item %d folder = %q", it.ID, it.Folder)
		}
	}

	// DELETE → 204，条目回落未分类；再删 → 404
	w = doJSON(r, "DELETE", "/hub/folders/"+strconv.FormatInt(f.ID, 10), nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete folder code=%d", w.Code)
	}
	w = doJSON(r, "GET", "/hub/items", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &items)
	for _, it := range items {
		if it.Folder != "" {
			t.Fatalf("item %d should fall back to uncategorized, got %q", it.ID, it.Folder)
		}
	}
	w = doJSON(r, "DELETE", "/hub/folders/"+strconv.FormatInt(f.ID, 10), nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for re-delete, got %d", w.Code)
	}
}

func TestHandlerReorderItems(t *testing.T) {
	r, _ := setupTestRouter(t)

	// 建 3 条同文件夹书签
	ids := []int64{}
	for _, title := range []string{"A", "B", "C"} {
		w := doJSON(r, "POST", "/hub/items", map[string]any{"type": "bookmark", "title": title, "folder": "F"})
		if w.Code != http.StatusCreated {
			t.Fatalf("seed %s: %d", title, w.Code)
		}
		var it Item
		_ = json.Unmarshal(w.Body.Bytes(), &it)
		ids = append(ids, it.ID)
	}

	// 重排为 [C, A, B] → 204，GET 验证顺序
	w := doJSON(r, "POST", "/hub/items/order", map[string]any{
		"type": "bookmark", "folder": "F", "ids": []int64{ids[2], ids[0], ids[1]},
	})
	if w.Code != http.StatusNoContent {
		t.Fatalf("reorder code=%d body=%s", w.Code, w.Body.String())
	}
	w = doJSON(r, "GET", "/hub/items", nil)
	var items []Item
	_ = json.Unmarshal(w.Body.Bytes(), &items)
	got := []string{}
	for _, it := range items {
		if it.Folder == "F" {
			got = append(got, it.Title)
		}
	}
	if len(got) != 3 || got[0] != "C" || got[1] != "A" || got[2] != "B" {
		t.Fatalf("order after reorder: %v", got)
	}

	// 非法 type → 400；空 ids → 400；不存在的 id → 404
	w = doJSON(r, "POST", "/hub/items/order", map[string]any{"type": "note", "folder": "F", "ids": ids})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid type, got %d", w.Code)
	}
	w = doJSON(r, "POST", "/hub/items/order", map[string]any{"type": "bookmark", "folder": "F", "ids": []int64{}})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty ids, got %d", w.Code)
	}
	w = doJSON(r, "POST", "/hub/items/order", map[string]any{"type": "bookmark", "folder": "F", "ids": []int64{999999}})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing id, got %d", w.Code)
	}
}
