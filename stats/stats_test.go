package stats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRecordAndSnapshot(t *testing.T) {
	RegisterBackend("http://localhost:8081")
	Record("http://localhost:8081", 50*time.Millisecond, 200)
	Record("http://localhost:8081", 70*time.Millisecond, 500)
	snap := SnapshotAll()
	bs, ok := snap["http://localhost:8081"]
	if !ok {
		t.Fatalf("missing snapshot for backend")
	}
	if bs.Requests != 2 {
		t.Fatalf("expected 2 requests, got %d", bs.Requests)
	}
	if bs.StatusCounts[200] != 1 || bs.StatusCounts[500] != 1 {
		t.Fatalf("unexpected status counts: %v", bs.StatusCounts)
	}
	// Expect failure rate ~= 50% (1 failure out of 2)
	if bs.FailureRatePct < 49.0 || bs.FailureRatePct > 51.0 {
		t.Fatalf("expected failure rate ~50%%, got %.2f", bs.FailureRatePct)
	}
	// Uptime percentage should be between 0 and 100
	if bs.UptimePct < 0.0 || bs.UptimePct > 100.0 {
		t.Fatalf("unexpected uptime percent: %.2f", bs.UptimePct)
	}
}

func TestHandlerOutput(t *testing.T) {
	RegisterBackend("http://localhost:8089")
	Record("http://localhost:8089", 10*time.Millisecond, 200)

	rr := httptest.NewRecorder()
	h := Handler()
	h.ServeHTTP(rr, &http.Request{})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out map[string]Snapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	bs, ok := out["http://localhost:8089"]
	if !ok {
		t.Fatalf("expected backend in output")
	}
	if bs.Requests != 1 {
		t.Fatalf("expected 1 request, got %d", bs.Requests)
	}
	if bs.FailureRatePct < 0.0 || bs.FailureRatePct > 100.0 {
		t.Fatalf("unexpected failure rate: %.2f", bs.FailureRatePct)
	}
}
