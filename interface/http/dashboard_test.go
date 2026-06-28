package httpinterface

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZONO33LHD/anneal/usecase"
)

type fakeDashboard struct {
	view usecase.DashboardView
	err  error
}

func (f fakeDashboard) Snapshot(context.Context) (usecase.DashboardView, error) {
	return f.view, f.err
}

func TestDashboardHandlerRendersHTML(t *testing.T) {
	uc := fakeDashboard{view: usecase.DashboardView{
		ActiveVersion:  "prompt_v2",
		TotalUpdates:   3,
		StateCounts:    []usecase.StateCount{{State: "merged", Count: 2}},
		RecentScores:   []usecase.ScoreRow{{Package: "lodash", Version: "prompt_v1", Status: "final", Total: 80}},
		Improvements:   []usecase.ImprovementRow{{Version: "prompt_v2", Status: "canary", Baseline: 70}},
		AverageScore:   80,
		HasFinalScores: true,
	}}
	h := NewDashboardHandler(DashboardOptions{Dashboard: uc})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type=%q, want text/html", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"prompt_v2", "lodash", "merged", "80.0"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestDashboardHandlerNotFoundForUnknownPath(t *testing.T) {
	h := NewDashboardHandler(DashboardOptions{Dashboard: fakeDashboard{}})
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
}

func TestDashboardHandlerRejectsNonGet(t *testing.T) {
	h := NewDashboardHandler(DashboardOptions{Dashboard: fakeDashboard{}})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want 405", rec.Code)
	}
}
