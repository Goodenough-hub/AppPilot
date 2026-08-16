package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}

func TestParseIconLinks(t *testing.T) {
	page := mustParse(t, "https://app.example.com/console/home")

	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			"相对 href 按页面 URL 解析",
			`<html><head><link rel="icon" href="/assets/icon.svg"></head></html>`,
			[]string{"https://app.example.com/assets/icon.svg"},
		},
		{
			"协议相对 href 继承页面 scheme",
			`<link rel="icon" href="//cdn.example.com/logo.png">`,
			[]string{"https://cdn.example.com/logo.png"},
		},
		{
			"shortcut icon 旧写法也命中",
			`<link rel="shortcut icon" href="favicon.ico">`,
			[]string{"https://app.example.com/console/favicon.ico"},
		},
		{
			"rel=icon 优先于文档顺序靠前的 apple-touch-icon",
			`<link rel="apple-touch-icon" href="/touch.png"><link rel="icon" href="/icon.svg">`,
			[]string{"https://app.example.com/icon.svg", "https://app.example.com/touch.png"},
		},
		{
			"<base href> 改变相对解析基准",
			`<base href="https://static.example.com/app/"><link rel="icon" href="icon.png">`,
			[]string{"https://static.example.com/app/icon.png"},
		},
		{
			"单引号与无引号属性写法",
			`<link rel='icon' href='/a.png'><link rel=icon href=/b.png>`,
			[]string{"https://app.example.com/a.png", "https://app.example.com/b.png"},
		},
		{
			"非图标 link（stylesheet 等）忽略",
			`<link rel="stylesheet" href="/app.css">`,
			nil,
		},
		{
			"非 http(s) scheme 排除",
			`<link rel="icon" href="data:image/png;base64,xxxx">`,
			nil,
		},
		{
			"重复地址去重",
			`<link rel="icon" href="/icon.png"><link rel="icon" href="/icon.png">`,
			[]string{"https://app.example.com/icon.png"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIconLinks(page, tt.html)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestParseIconLinksCap(t *testing.T) {
	html := ""
	for i := 0; i < maxFaviconResults+3; i++ {
		html += `<link rel="icon" href="/icon` + string(rune('a'+i)) + `.png">`
	}
	got := parseIconLinks(mustParse(t, "https://app.example.com/"), html)
	if len(got) != maxFaviconResults {
		t.Fatalf("expected cap at %d, got %d", maxFaviconResults, len(got))
	}
}

// setupFaviconRouter：favicon 不触库，Handler 传 nil repo 即可
func setupFaviconRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil)
	r := gin.New()
	rg := r.Group("/hub", func(c *gin.Context) {
		c.Set("userID", int64(1))
		c.Next()
	})
	h.Register(rg)
	return r
}

func getFavicon(r *gin.Engine, rawURL string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/hub/favicon?url="+url.QueryEscape(rawURL), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeIcons(t *testing.T, w *httptest.ResponseRecorder) []string {
	t.Helper()
	var body struct {
		Icons []string `json:"icons"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.Icons
}

func TestHandlerFavicon(t *testing.T) {
	r := setupFaviconRouter()

	// 上游站点：深页面 SPA fallback 无 icon，根路径声明 CDN 图标
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if req.URL.Path == "/" {
			_, _ = w.Write([]byte(`<html><head><link rel="icon" type="image/svg+xml" href="https://cdn.example.com/logo_small.png"></head></html>`))
			return
		}
		_, _ = w.Write([]byte(`<html><head><title>SPA fallback</title></head></html>`))
	}))
	defer upstream.Close()

	t.Run("深路径无 icon 时回退站点根再解析", func(t *testing.T) {
		w := getFavicon(r, upstream.URL+"/login?redirect=/console")
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		icons := decodeIcons(t, w)
		if len(icons) != 1 || icons[0] != "https://cdn.example.com/logo_small.png" {
			t.Fatalf("unexpected icons: %v", icons)
		}
	})

	t.Run("页面直接声明 icon", func(t *testing.T) {
		w := getFavicon(r, upstream.URL+"/")
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if icons := decodeIcons(t, w); len(icons) != 1 {
			t.Fatalf("unexpected icons: %v", icons)
		}
	})

	t.Run("无 icon 时返回空数组而非 null", func(t *testing.T) {
		// 根路径也无 icon 的上游
		bare := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`<html><head></head></html>`))
		}))
		defer bare.Close()
		w := getFavicon(r, bare.URL+"/deep/page")
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		icons := decodeIcons(t, w)
		if icons == nil || len(icons) != 0 {
			t.Fatalf("expected empty array, got %v", icons)
		}
	})

	t.Run("上游非 2xx 返回 502", func(t *testing.T) {
		dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer dead.Close()
		w := getFavicon(r, dead.URL+"/")
		if w.Code != http.StatusBadGateway {
			t.Fatalf("expected 502, got %d", w.Code)
		}
	})

	t.Run("非法 url 返回 400", func(t *testing.T) {
		for _, raw := range []string{"", "not-a-url", "ftp://example.com/x", "/relative/path"} {
			if w := getFavicon(r, raw); w.Code != http.StatusBadRequest {
				t.Fatalf("url %q: expected 400, got %d", raw, w.Code)
			}
		}
	})
}
