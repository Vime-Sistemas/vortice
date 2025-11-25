package domain

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestRateLimit_Triggers429(t *testing.T) {
	// backend that responds quickly
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	// create pool with one backend with low rate (1 rps, burst 1)
	pool := &ServerPool{Algorithm: "round_robin"}
	b := NewBackend(backend.URL, 1, 1)
	pool.AddBackend(b)

	// start proxy server on ephemeral port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()
	server := &http.Server{Handler: pool}
	go server.Serve(ln)
	defer server.Close()

	addr := "http://" + ln.Addr().String()

	// fire several concurrent requests
	var wg sync.WaitGroup
	n := 10
	results := make(chan int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(addr + "/")
			if err != nil {
				results <- 0
				return
			}
			body, _ := ioutil.ReadAll(resp.Body)
			_ = body
			results <- resp.StatusCode
			resp.Body.Close()
		}()
	}
	wg.Wait()
	close(results)

	var got429 int
	var got200 int
	for code := range results {
		if code == http.StatusTooManyRequests {
			got429++
		} else if code == http.StatusOK {
			got200++
		}
	}

	if got200 == 0 {
		t.Fatalf("expected at least one 200 OK, got none")
	}
	if got429 == 0 {
		t.Fatalf("expected some 429 Too Many Requests due to rate limit, got none")
	}
}
