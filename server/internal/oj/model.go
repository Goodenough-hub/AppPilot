package oj

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type TestCase struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expectedOutput"`
}

type ProblemInput struct {
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Difficulty    string     `json:"difficulty"`
	Tags          []string   `json:"tags"`
	Statement     string     `json:"statement"`
	SampleInput   string     `json:"sampleInput"`
	SampleOutput  string     `json:"sampleOutput"`
	TimeLimitMS   int        `json:"timeLimitMs"`
	MemoryLimitMB int        `json:"memoryLimitMb"`
	Cases         []TestCase `json:"cases"`
}

func (p *ProblemInput) Validate() error {
	if len(p.Slug) > 80 || !slugPattern.MatchString(p.Slug) {
		return errors.New("题目标识只能使用小写字母、数字和连字符")
	}
	if strings.TrimSpace(p.Title) == "" || len(p.Title) > 200 {
		return errors.New("题目标题不能为空且不能超过 200 字节")
	}
	if p.Difficulty != "easy" && p.Difficulty != "medium" && p.Difficulty != "hard" {
		return errors.New("题目难度无效")
	}
	if strings.TrimSpace(p.Statement) == "" || len(p.Statement) > 200_000 {
		return errors.New("题面不能为空且不能超过 200KB")
	}
	if len(p.SampleInput) > 64*1024 || len(p.SampleOutput) > 64*1024 {
		return errors.New("样例不能超过 64KB")
	}
	if p.TimeLimitMS < 100 || p.TimeLimitMS > 2000 || p.MemoryLimitMB < 64 || p.MemoryLimitMB > 256 {
		return errors.New("时间或内存限制超出范围")
	}
	if len(p.Cases) == 0 || len(p.Cases) > 100 {
		return errors.New("需要 1 到 100 组测试数据")
	}
	for _, tc := range p.Cases {
		if len(tc.Input) > 64*1024 || len(tc.ExpectedOutput) > 64*1024 {
			return errors.New("单组测试数据不能超过 64KB")
		}
	}
	if len(p.Tags) > 20 {
		return errors.New("标签不能超过 20 个")
	}
	for _, tag := range p.Tags {
		if strings.TrimSpace(tag) == "" || len(tag) > 32 {
			return errors.New("标签不能为空且不能超过 32 字节")
		}
	}
	return nil
}

type Problem struct {
	ID            int64      `json:"id,string"`
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Difficulty    string     `json:"difficulty"`
	Tags          []string   `json:"tags"`
	Statement     string     `json:"statement,omitempty"`
	SampleInput   string     `json:"sampleInput,omitempty"`
	SampleOutput  string     `json:"sampleOutput,omitempty"`
	TimeLimitMS   int        `json:"timeLimitMs,omitempty"`
	MemoryLimitMB int        `json:"memoryLimitMb,omitempty"`
	Version       int        `json:"version"`
	Published     bool       `json:"published"`
	CaseCount     int        `json:"caseCount,omitempty"`
	Cases         []TestCase `json:"cases,omitempty"`
}

type Submission struct {
	ID            int64     `json:"id,string"`
	UserID        int64     `json:"-"`
	ProblemID     int64     `json:"problemId,string"`
	ProblemTitle  string    `json:"problemTitle"`
	Version       int       `json:"version"`
	Kind          string    `json:"kind"`
	Code          string    `json:"code,omitempty"`
	Status        string    `json:"status"`
	Verdict       *string   `json:"verdict"`
	CompileOutput string    `json:"compileOutput,omitempty"`
	Stdout        string    `json:"stdout,omitempty"`
	TimeMS        int       `json:"timeMs"`
	MemoryKB      int       `json:"memoryKb"`
	Passed        int       `json:"passed"`
	Total         int       `json:"total"`
	CreatedAt     time.Time `json:"createdAt"`
}

func ValidateSource(code, input string) error {
	if strings.TrimSpace(code) == "" || len(code) > 64*1024 {
		return errors.New("代码不能为空且不能超过 64KB")
	}
	if len(input) > 64*1024 {
		return errors.New("自测输入不能超过 64KB")
	}
	return nil
}
