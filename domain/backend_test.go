package domain

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBackend_SetAlive(t *testing.T) {
	b := NewBackend("http://localhost:8080")

	// Default should be true
	if !b.IsAlive() {
		t.Error("New backend should be alive by default")
	}

	// Test SetAlive
	b.SetAlive(false)
	if b.IsAlive() {
		t.Error("Backend should be dead after SetAlive(false)")
	}
}

func TestBackend_CheckHealth(t *testing.T) {
	// 1. Create a dummy server that answers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	// 2. Point our backend to the dummy server
	backendUrl, _ := url.Parse(server.URL)
	b := NewBackend(backendUrl.String())

	// 3. Should return true
	if !b.CheckHealth() {
		t.Errorf("Backend should be healthy connecting to %s", server.URL)
	}

	// 4. Test a fake port (should fail)
	deadBackend := NewBackend("http://localhost:9999")
	if deadBackend.CheckHealth() {
		t.Error("Backend should be unhealthy for dead port")
	}
}

func TestReverseProxy_Success(t *testing.T) {
	// start a dummy backend that returns 202 and a body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
		w.Write([]byte("ok-backend"))
	}))
	defer server.Close()

	// create backend pointing to the dummy server
	b := NewBackend(server.URL)

	// perform a proxy request
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	b.ReverseProxy.ServeHTTP(rr, req)

	// verify response forwarded from backend
	if rr.Code != 202 {
		t.Errorf("expected status 202, got %d", rr.Code)
	}
	if rr.Body.String() != "ok-backend" {
		t.Errorf("expected body %q, got %q", "ok-backend", rr.Body.String())
	}
}

func TestReverseProxy_ErrorHandler(t *testing.T) {
	// create backend pointing to an unreachable port to trigger ErrorHandler
	b := NewBackend("http://localhost:9999")

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	b.ReverseProxy.ServeHTTP(rr, req)

	// ErrorHandler should write 503 and the specific message
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}
	expectedBody := "Backend unavailable"
	if rr.Body.String() != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, rr.Body.String())
	}
}
