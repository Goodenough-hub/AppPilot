package oj

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestRepositoryLifecycle(t *testing.T) {
	dsn := os.Getenv("PG_TEST_DSN")
	if dsn == "" {
		t.Skip("PG_TEST_DSN not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	r := NewRepository(db)
	var userA, userB int64
	for i, ptr := range []*int64{&userA, &userB} {
		name := fmt.Sprintf("ojtest_%d_%d", time.Now().UnixNano(), i)
		if err := db.QueryRow(`INSERT INTO users(username,password_hash,role) VALUES($1,'test','user') RETURNING id`, name).Scan(ptr); err != nil {
			t.Fatal(err)
		}
		defer db.Exec(`DELETE FROM users WHERE id=$1`, *ptr)
	}
	p := validProblem()
	p.Slug = fmt.Sprintf("ojtest-%d", time.Now().UnixNano())
	id, err := r.CreateProblem(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = db.Exec(`DELETE FROM oj_submissions WHERE problem_id=$1`, id)
		_, _ = db.Exec(`DELETE FROM oj_problems WHERE id=$1`, id)
	}()
	if _, err := r.GetProblem(p.Slug, false); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("draft visible: %v", err)
	}
	if err := r.PublishProblem(id, true); err != nil {
		t.Fatal(err)
	}
	public, err := r.GetProblem(p.Slug, false)
	if err != nil || public.CaseCount != 1 || len(public.Cases) != 0 {
		t.Fatalf("public leaked cases: %+v %v", public, err)
	}
	admin, err := r.GetProblem(p.Slug, true)
	if err != nil || len(admin.Cases) != 1 {
		t.Fatalf("admin cases: %+v %v", admin, err)
	}
	subID, err := r.CreateSubmission(userA, id, "submit", "int main(){return 0;}", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetSubmission(subID, userB); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("another user read code: %v", err)
	}
	owned, err := r.GetSubmission(subID, userA)
	if err != nil || owned.Code == "" || owned.Version != 1 {
		t.Fatalf("owned submission: %+v %v", owned, err)
	}
	p.Cases[0].ExpectedOutput = "9\n"
	if err := r.UpdateProblem(id, p); err != nil {
		t.Fatal(err)
	}
	var oldExpected string
	if err := db.QueryRow(`SELECT expected_output FROM oj_cases WHERE problem_id=$1 AND version=1 AND ordinal=1`, id).Scan(&oldExpected); err != nil {
		t.Fatal(err)
	}
	if oldExpected != "3\n" {
		t.Fatalf("in-flight case changed: %q", oldExpected)
	}
	newest, err := r.GetProblem(p.Slug, true)
	if err != nil || newest.Version != 2 || newest.Cases[0].ExpectedOutput != "9\n" {
		t.Fatalf("version update: %+v %v", newest, err)
	}
}
