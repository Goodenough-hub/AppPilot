package oj

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

var ErrBusy = errors.New("判题队列已满或提交过于频繁")

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListProblems(admin bool, q, difficulty string) ([]Problem, error) {
	query := `SELECT p.id,p.slug,p.title,p.difficulty,p.tags,p.current_version,p.published
		FROM oj_problems p WHERE ($1 OR p.published) AND ($2 = '' OR p.title ILIKE '%' || $2 || '%' OR p.slug ILIKE '%' || $2 || '%')
		AND ($3 = '' OR p.difficulty = $3) ORDER BY p.id DESC LIMIT 200`
	rows, err := r.db.Query(query, admin, q, difficulty)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Problem{}
	for rows.Next() {
		var p Problem
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Difficulty, pq.Array(&p.Tags), &p.Version, &p.Published); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) GetProblem(key string, admin bool) (*Problem, error) {
	p := &Problem{}
	err := r.db.QueryRow(`SELECT p.id,p.slug,p.title,p.difficulty,p.tags,p.current_version,p.published,
		v.statement,v.sample_input,v.sample_output,v.time_limit_ms,v.memory_limit_mb,
		(SELECT count(*) FROM oj_cases c WHERE c.problem_id=p.id AND c.version=p.current_version)
		FROM oj_problems p JOIN oj_problem_versions v ON v.problem_id=p.id AND v.version=p.current_version
		WHERE (p.slug=$1 OR p.id::text=$1) AND ($2 OR p.published)`, key, admin).
		Scan(&p.ID, &p.Slug, &p.Title, &p.Difficulty, pq.Array(&p.Tags), &p.Version, &p.Published,
			&p.Statement, &p.SampleInput, &p.SampleOutput, &p.TimeLimitMS, &p.MemoryLimitMB, &p.CaseCount)
	if err != nil {
		return nil, err
	}
	if admin {
		rows, err := r.db.Query(`SELECT input,expected_output FROM oj_cases WHERE problem_id=$1 AND version=$2 ORDER BY ordinal`, p.ID, p.Version)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		p.Cases = []TestCase{}
		for rows.Next() {
			var c TestCase
			if err := rows.Scan(&c.Input, &c.ExpectedOutput); err != nil {
				return nil, err
			}
			p.Cases = append(p.Cases, c)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (r *Repository) CreateProblem(in ProblemInput) (int64, error) {
	if err := in.Validate(); err != nil {
		return 0, err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRow(`INSERT INTO oj_problems(slug,title,difficulty,tags) VALUES($1,$2,$3,$4) RETURNING id`, in.Slug, in.Title, in.Difficulty, pq.Array(nonNilTags(in.Tags))).Scan(&id)
	if err != nil {
		return 0, err
	}
	if err = insertVersion(tx, id, 1, in); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (r *Repository) UpdateProblem(id int64, in ProblemInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	err = tx.QueryRow(`SELECT current_version FROM oj_problems WHERE id=$1 FOR UPDATE`, id).Scan(&version)
	if err != nil {
		return err
	}
	version++
	_, err = tx.Exec(`UPDATE oj_problems SET slug=$2,title=$3,difficulty=$4,tags=$5,current_version=$6,updated_at=NOW() WHERE id=$1`, id, in.Slug, in.Title, in.Difficulty, pq.Array(nonNilTags(in.Tags)), version)
	if err != nil {
		return err
	}
	if err = insertVersion(tx, id, version, in); err != nil {
		return err
	}
	return tx.Commit()
}

func insertVersion(tx *sql.Tx, id int64, version int, in ProblemInput) error {
	_, err := tx.Exec(`INSERT INTO oj_problem_versions(problem_id,version,statement,sample_input,sample_output,time_limit_ms,memory_limit_mb)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, id, version, in.Statement, in.SampleInput, in.SampleOutput, in.TimeLimitMS, in.MemoryLimitMB)
	if err != nil {
		return err
	}
	for i, c := range in.Cases {
		if _, err = tx.Exec(`INSERT INTO oj_cases(problem_id,version,ordinal,input,expected_output) VALUES($1,$2,$3,$4,$5)`, id, version, i+1, c.Input, c.ExpectedOutput); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) PublishProblem(id int64, published bool) error {
	result, err := r.db.Exec(`UPDATE oj_problems SET published=$2,updated_at=NOW() WHERE id=$1`, id, published)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CreateSubmission(userID, problemID int64, kind, code, input string) (int64, error) {
	if err := ValidateSource(code, input); err != nil {
		return 0, err
	}
	if kind != "run" && kind != "submit" {
		return 0, errors.New("提交类型无效")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	// Serialize requests by the same user so the rate limit cannot be raced.
	var lockedID int64
	if err = tx.QueryRow(`SELECT id FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&lockedID); err != nil {
		return 0, err
	}
	var active int
	if err = tx.QueryRow(`SELECT count(*) FROM oj_submissions WHERE user_id=$1 AND status IN ('queued','running')`, userID).Scan(&active); err != nil {
		return 0, err
	}
	if active >= 2 {
		return 0, ErrBusy
	}
	var recent bool
	if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM oj_submissions WHERE user_id=$1 AND created_at>NOW()-INTERVAL '5 seconds')`, userID).Scan(&recent); err != nil {
		return 0, err
	}
	if recent {
		return 0, ErrBusy
	}
	var queued int
	if err = tx.QueryRow(`SELECT count(*) FROM oj_jobs WHERE state IN ('queued','running')`).Scan(&queued); err != nil {
		return 0, err
	}
	if queued >= 50 {
		return 0, ErrBusy
	}
	var version int
	if err = tx.QueryRow(`SELECT current_version FROM oj_problems WHERE id=$1 AND published`, problemID).Scan(&version); err != nil {
		return 0, err
	}
	var total int
	if kind == "run" {
		total = 1
	} else if err = tx.QueryRow(`SELECT count(*) FROM oj_cases WHERE problem_id=$1 AND version=$2`, problemID, version).Scan(&total); err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, fmt.Errorf("题目没有测试数据")
	}
	var id int64
	err = tx.QueryRow(`INSERT INTO oj_submissions(user_id,problem_id,version,kind,code,custom_input,total)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, userID, problemID, version, kind, code, input, total).Scan(&id)
	if err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`INSERT INTO oj_jobs(submission_id) VALUES($1)`, id); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

const submissionSelect = `SELECT s.id,s.user_id,s.problem_id,p.title,s.version,s.kind,s.code,s.status,s.verdict,
	s.compile_output,s.stdout,s.time_ms,s.memory_kb,s.passed,s.total,s.created_at
	FROM oj_submissions s JOIN oj_problems p ON p.id=s.problem_id`

func scanSubmission(row interface{ Scan(...any) error }) (*Submission, error) {
	s := &Submission{}
	err := row.Scan(&s.ID, &s.UserID, &s.ProblemID, &s.ProblemTitle, &s.Version, &s.Kind, &s.Code, &s.Status, &s.Verdict,
		&s.CompileOutput, &s.Stdout, &s.TimeMS, &s.MemoryKB, &s.Passed, &s.Total, &s.CreatedAt)
	return s, err
}

func (r *Repository) GetSubmission(id, userID int64) (*Submission, error) {
	return scanSubmission(r.db.QueryRow(submissionSelect+` WHERE s.id=$1 AND s.user_id=$2`, id, userID))
}

func (r *Repository) ListSubmissions(userID int64) ([]Submission, error) {
	rows, err := r.db.Query(submissionSelect+` WHERE s.user_id=$1 AND s.kind='submit' ORDER BY s.id DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Submission{}
	for rows.Next() {
		s, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		s.Code = ""
		out = append(out, *s)
	}
	return out, rows.Err()
}

func IsUnique(err error) bool {
	var pg *pq.Error
	return errors.As(err, &pg) && string(pg.Code) == "23505"
}

func NormalizeQuery(q string) string { return strings.TrimSpace(q) }

func nonNilTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}
