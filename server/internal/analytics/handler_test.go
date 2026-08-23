package analytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeRepository struct {
	summaryResult        Summary
	activeSessionsResult int
	summaryStart         time.Time
	summaryEnd           time.Time
}

func (r *fakeRepository) InsertEvent(string, string, string, string, string, string, string, string, *int64) error {
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
