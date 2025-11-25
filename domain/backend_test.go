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
