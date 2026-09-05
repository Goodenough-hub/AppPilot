package analytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeRepository struct {
	summaryResult        Summary
	activeSessionsResult int
	summaryStart         time.Time
	summaryEnd           time.Time
	insertedApp          string
	insertedEventType    string
	insertedPath         string
	insertCalls          int
}

func (r *fakeRepository) InsertEvent(app, eventType, path string, _, _, _, _, _ string, _ *int64) error {
	r.insertedApp = app
	r.insertedEventType = eventType
	r.insertedPath = path
	r.insertCalls++
	return nil
}

func (r *fakeRepository) PVAggregate(string, time.Time, time.Time) ([]PVDailyRow, error) {
	return nil, nil
}

func (r *fakeRepository) Summary(_ string, start, end time.Time) (Summary, error) {
	r.summaryStart = start
	r.summaryEnd = end
	return r.summaryResult, nil
}

func (r *fakeRepository) TopPages(string, time.Time, time.Time, int) ([]TopPageRow, error) {
	return nil, nil
}

func (r *fakeRepository) ActiveSessions(string, time.Duration) (int, error) {
	return r.activeSessionsResult, nil
}

func newTestRouter(repo repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(repo).RegisterAdmin(router.Group("/api/v1/admin"))
	return router
}

func newPublicTestRouter(repo repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(repo).RegisterPublic(router.Group("/api/v1/analytics"))
	return router
}

func postTrack(t *testing.T, router *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/track", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func TestTrackAcceptsTypResumePageview(t *testing.T) {
	repo := &fakeRepository{}
	router := newPublicTestRouter(repo)

	response := postTrack(t, router, `{"app":"typresume","eventType":"pageview","path":"/resume/editor"}`)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", response.Code, response.Body.String())
	}
	if repo.insertCalls != 1 {
		t.Fatalf("expected exactly one insert, got %d", repo.insertCalls)
	}
	if repo.insertedApp != "typresume" || repo.insertedEventType != "pageview" || repo.insertedPath != "/resume/editor" {
		t.Fatalf("unexpected event: app=%s type=%s path=%s", repo.insertedApp, repo.insertedEventType, repo.insertedPath)
	}
}

func TestTrackRejectsUnknownApp(t *testing.T) {
	repo := &fakeRepository{}
	router := newPublicTestRouter(repo)

	response := postTrack(t, router, `{"app":"unknown-app","eventType":"pageview","path":"/"}`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	if repo.insertCalls != 0 {
		t.Fatalf("expected no insert for an unknown app, got %d", repo.insertCalls)
	}
}

func TestSummaryReturnsRangeTotals(t *testing.T) {
	repo := &fakeRepository{summaryResult: Summary{PageViews: 12, UniqueSessions: 3}}
	router := newTestRouter(repo)
	start := "2026-08-16T16:00:00Z"
	end := "2026-08-23T02:00:00Z"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/analytics/summary?app=finflow&start="+start+"&end="+end, nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body Summary
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.PageViews != 12 || body.UniqueSessions != 3 {
		t.Fatalf("unexpected summary: %+v", body)
	}
	if repo.summaryStart.Format(time.RFC3339) != start || repo.summaryEnd.Format(time.RFC3339) != end {
		t.Fatalf("unexpected range: %s - %s", repo.summaryStart.Format(time.RFC3339), repo.summaryEnd.Format(time.RFC3339))
	}
}

func TestNullableSessionIDFallsBackForEmptyValue(t *testing.T) {
	if nullableSessionID("") != nil {
		t.Fatal("expected an empty session ID to be stored as NULL")
	}
	if nullableSessionID("session-1") != "session-1" {
		t.Fatal("expected a non-empty session ID to be preserved")
	}
}

func TestRealtimeReturnsActiveSessions(t *testing.T) {
	router := newTestRouter(&fakeRepository{activeSessionsResult: 4})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/analytics/realtime?app=finflow", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if response.Body.String() != `{"activeSessions":4}` {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}
