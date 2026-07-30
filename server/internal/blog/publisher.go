package blog

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GitFile 是要提交进仓库的文件（Markdown 正文或图片二进制）。
type GitFile struct {
	Path    string
	Content []byte
}

// Publisher 用 GitHub Git Data API 在单个提交里原子写入多个文件，
// 并以非 force 方式更新分支 ref。分支头冲突时刷新一次并重试，
// 仍冲突返回 ErrGitConflict（调用方返回 409）。
// baseURL 与 client 可注入，便于用 httptest 测试。
type Publisher struct {
	repo    string
	branch  string
	token   string
	baseURL string
	client  *http.Client
}

func NewPublisher(repo, branch, token string) *Publisher {
	return &Publisher{
		repo:    repo,
		branch:  branch,
		token:   token,
		baseURL: "https://api.github.com",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Commit 在仓库 {branch} 上追加一个 commit，包含 files 全部内容。
func (p *Publisher) Commit(ctx context.Context, message string, files []GitFile) (string, error) {
	if p.token == "" {
		return "", fmt.Errorf("github token not configured")
	}
	commitSHA, err := p.commitOnce(ctx, message, files)
	if err == nil {
		return commitSHA, nil
	}
	if !errors.Is(err, ErrGitConflict) {
		return "", err
	}
	// 冲突：刷新 base 重试一次
	commitSHA, err2 := p.commitOnce(ctx, message, files)
	if err2 != nil {
		return "", err2
	}
	return commitSHA, nil
}

func (p *Publisher) commitOnce(ctx context.Context, message string, files []GitFile) (string, error) {
	// 1. 取分支 ref 的 commit SHA
	baseCommitSHA, err := p.refHead(ctx)
	if err != nil {
		return "", err
	}
	// 2. 取 base commit 的 tree SHA
	baseTreeSHA, err := p.commitTree(ctx, baseCommitSHA)
	if err != nil {
		return "", err
	}
	// 3. 为每个文件创建 blob
	entries := make([]treeEntry, 0, len(files))
	for _, f := range files {
		sha, err := p.createBlob(ctx, f.Content)
		if err != nil {
			return "", err
		}
		entries = append(entries, treeEntry{Path: f.Path, Mode: "100644", Type: "blob", SHA: sha})
	}
	// 4. 创建 tree
	treeSHA, err := p.createTree(ctx, baseTreeSHA, entries)
	if err != nil {
		return "", err
	}
	// 5. 创建 commit
	commitSHA, err := p.createCommit(ctx, message, treeSHA, baseCommitSHA)
	if err != nil {
		return "", err
	}
	// 6. 非-force 更新 ref
	if err := p.updateRef(ctx, commitSHA); err != nil {
		return "", err
	}
	return commitSHA, nil
}

// ---- GitHub 响应类型 ----

type refResp struct {
	Object struct {
		SHA  string `json:"sha"`
		Type string `json:"type"`
	} `json:"object"`
}

type commitResp struct {
	SHA  string `json:"sha"`
	Tree struct {
		SHA string `json:"sha"`
	} `json:"tree"`
}

type blobResp struct {
	SHA string `json:"sha"`
}

type treeEntry struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type treeReq struct {
	BaseTree string      `json:"base_tree,omitempty"`
	Tree     []treeEntry `json:"tree"`
}

type treeResp struct {
	SHA string `json:"sha"`
}

type commitReq struct {
	Message string   `json:"message"`
	Tree    string   `json:"tree"`
	Parents []string `json:"parents"`
}

type refUpdateReq struct {
	SHA   string `json:"sha"`
	Force bool   `json:"force"`
}

func (p *Publisher) refHead(ctx context.Context) (string, error) {
	var r refResp
	if err := p.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/git/refs/heads/%s", p.repo, p.branch), nil, &r); err != nil {
		return "", err
	}
	return r.Object.SHA, nil
}

func (p *Publisher) commitTree(ctx context.Context, commitSHA string) (string, error) {
	var c commitResp
	if err := p.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/git/commits/%s", p.repo, commitSHA), nil, &c); err != nil {
		return "", err
	}
	return c.Tree.SHA, nil
}

func (p *Publisher) createBlob(ctx context.Context, content []byte) (string, error) {
	body := map[string]string{
		"content":  base64.StdEncoding.EncodeToString(content),
		"encoding": "base64",
	}
	var b blobResp
	if err := p.do(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/git/blobs", p.repo), body, &b); err != nil {
		return "", err
	}
	return b.SHA, nil
}

func (p *Publisher) createTree(ctx context.Context, baseTree string, entries []treeEntry) (string, error) {
	req := treeReq{BaseTree: baseTree, Tree: entries}
	var t treeResp
	if err := p.do(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/git/trees", p.repo), req, &t); err != nil {
		return "", err
	}
	return t.SHA, nil
}

func (p *Publisher) createCommit(ctx context.Context, message, treeSHA, parent string) (string, error) {
	req := commitReq{Message: message, Tree: treeSHA, Parents: []string{parent}}
	var c commitResp
	if err := p.do(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/git/commits", p.repo), req, &c); err != nil {
		return "", err
	}
	return c.SHA, nil
}

func (p *Publisher) updateRef(ctx context.Context, sha string) error {
	body := refUpdateReq{SHA: sha, Force: false}
	err := p.do(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/git/refs/heads/%s", p.repo, p.branch), body, nil)
	if err == nil {
		return nil
	}
	// 非快进冲突：GitHub 返回 422，message 含 "non-fast-forward" 或 "Reference update"
	if isConflict(err) {
		return ErrGitConflict
	}
	return err
}

type apiError struct {
	Status int
	Body   string
}

func (e *apiError) Error() string { return fmt.Sprintf("github api %d: %s", e.Status, e.Body) }

func isConflict(err error) bool {
	var ae *apiError
	if errors.As(err, &ae) && ae.Status == 422 {
		return true
	}
	return false
}

func (p *Publisher) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return &apiError{Status: resp.StatusCode, Body: string(raw)}
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}
