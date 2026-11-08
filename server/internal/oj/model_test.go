package oj

import "testing"

func validProblem() ProblemInput {
	return ProblemInput{
		Slug: "a-plus-b", Title: "A+B", Difficulty: "easy", Statement: "读入两个整数并输出和。",
		SampleInput: "1 2\n", SampleOutput: "3\n", TimeLimitMS: 1000, MemoryLimitMB: 128,
		Cases: []TestCase{{Input: "1 2\n", ExpectedOutput: "3\n"}},
	}
}

func TestProblemValidation(t *testing.T) {
	p := validProblem()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Cases = nil
	if err := p.Validate(); err == nil {
		t.Fatal("需要测试数据")
	}
	p = validProblem()
	p.Slug = "A+B"
	if err := p.Validate(); err == nil {
		t.Fatal("题目标识应拒绝大写和加号")
	}
	p = validProblem()
	p.MemoryLimitMB = 4096
	if err := p.Validate(); err == nil {
		t.Fatal("内存必须受限")
	}
}

func TestSourceValidation(t *testing.T) {
	if err := ValidateSource("int main(){}", ""); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSource(" \n ", ""); err == nil {
		t.Fatal("空代码应被拒绝")
	}
}
