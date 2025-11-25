package domain

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// Backend holds the data for a single server
type Backend struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
	Mux          sync.RWMutex
	Alive        bool
}

// NewBackend creates a new backend instance
func NewBackend(serverUrl string) *Backend {
	u, _ := url.Parse(serverUrl)
	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Backend unavailable"))
	}

	return &Backend{
		URL:          u,
		ReverseProxy: proxy,
		Alive:        true,
	}
}

func (b *Backend) SetAlive(alive bool) {
	b.Mux.Lock()
	b.Alive = alive
	b.Mux.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.Mux.RLock()
	alive := b.Alive
	b.Mux.RUnlock()
	return alive
}

// CheckHealth attempts to dial the server to see if it responds
func (b *Backend) CheckHealth() bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(b.URL.String())
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	// consider healthy any 2xx or 3xx response
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
