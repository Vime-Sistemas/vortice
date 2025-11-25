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
	if bs.MostFamousPort == "" {
		t.Fatalf("expected a most famous port, got empty")
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
	if _, ok := out["http://localhost:8089"]; !ok {
		t.Fatalf("expected backend in output")
	}
}
