package hub

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// faviconHTTPClient 抓取页面用的客户端：整体超时 + 限 5 次跳转。
// 注意：该端点面向已登录的 hub 用户，且目标场景就是内网站点，故不做内网 IP 拦截；
// 仅限制 scheme 为 http/https（见 handler 入口校验）。
var faviconHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(_ *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return nil
	},
}

// 页面体读取上限 1MB：<link> 都在 <head>，足够；同时防异常大页面
const maxFaviconPageBytes = 1 << 20

// 单次返回的图标 URL 上限
const maxFaviconResults = 5

var (
	linkTagRe  = regexp.MustCompile(`(?i)<link\b[^>]*>`)
	baseTagRe  = regexp.MustCompile(`(?i)<base\b[^>]*>`)
	hrefAttrRe = regexp.MustCompile(`(?i)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>"']+))`)
	relAttrRe  = regexp.MustCompile(`(?i)\brel\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>"']+))`)
)

// tagAttr 从单个标签串里取属性值（兼容双引号/单引号/无引号三种写法）
func tagAttr(tag string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return ""
}

// linkIconScore：rel 为图标链接时的优先级（越小越优先）；-1 表示不是图标链接
func linkIconScore(rel string) int {
	score := -1
	for _, tok := range strings.Fields(strings.ToLower(rel)) {
		switch tok {
		case "icon": // rel="icon"、rel="shortcut icon"
			return 0
		case "apple-touch-icon":
			score = 1
		}
	}
	return score
}

// parseIconLinks 从 HTML 中收集图标链接；相对地址以 base 解析（<base href> 可覆盖基准）。
// 返回按优先级排序（同级保文档顺序）、去重、截断到上限的绝对 URL 列表。
func parseIconLinks(pageURL *url.URL, html string) []string {
	base := pageURL
	if tag := baseTagRe.FindString(html); tag != "" {
		if h := tagAttr(tag, hrefAttrRe); h != "" {
			if u, err := pageURL.Parse(h); err == nil {
				base = u
			}
		}
	}
	type scored struct {
		u     string
		score int
	}
	var found []scored
	for _, tag := range linkTagRe.FindAllString(html, -1) {
		s := linkIconScore(tagAttr(tag, relAttrRe))
		if s < 0 {
			continue
		}
		href := tagAttr(tag, hrefAttrRe)
		if href == "" {
			continue
		}
		u, err := base.Parse(href) // 兼容绝对地址、协议相对 //cdn/...、相对路径
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			continue
		}
		found = append(found, scored{u.String(), s})
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].score < found[j].score })
	seen := make(map[string]bool, len(found))
	out := make([]string, 0, maxFaviconResults)
	for _, f := range found {
		if seen[f.u] {
			continue
		}
		seen[f.u] = true
		out = append(out, f.u)
		if len(out) >= maxFaviconResults {
			break
		}
	}
	return out
}

// fetchIconLinks 抓取 page 的 HTML 并解析图标链接（以跳转后的最终 URL 为相对基准）
func fetchIconLinks(page *url.URL) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, page.String(), nil)
	if err != nil {
		return nil, err
	}
	// 部分站点按 UA 拦截爬虫，伪装成常规浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	resp, err := faviconHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFaviconPageBytes))
	if err != nil {
		return nil, err
	}
	return parseIconLinks(resp.Request.URL, string(body)), nil
}

// discoverFavicons 抓取 target 解析图标；深路径页面无结果时回退抓站点根再解析一次
// （SPA 深链接通常由 fallback 返回 index.html，但真实 404 的深页面就该试根路径）。
func discoverFavicons(target *url.URL) ([]string, error) {
	icons, err := fetchIconLinks(target)
	if err == nil && len(icons) == 0 && target.EscapedPath() != "" && target.EscapedPath() != "/" {
		root := &url.URL{Scheme: target.Scheme, Host: target.Host}
		icons, err = fetchIconLinks(root)
	}
	return icons, err
}
