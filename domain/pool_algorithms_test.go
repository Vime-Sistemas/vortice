package domain

import (
	"net/http"
	"sync/atomic"
	"testing"
)

func TestLeastConn_SelectsMinConn(t *testing.T) {
	pool := &ServerPool{Algorithm: "least_conn"}

	b1 := NewBackend("http://localhost:8081", 0, 1)
	b2 := NewBackend("http://localhost:8082", 0, 1)
	b3 := NewBackend("http://localhost:8083", 0, 1)

	// set connection counts
	atomic.StoreInt64(&b1.ConnCount, 5)
	atomic.StoreInt64(&b2.ConnCount, 2)
	atomic.StoreInt64(&b3.ConnCount, 10)

	pool.AddBackend(b1)
	pool.AddBackend(b2)
	pool.AddBackend(b3)

	peer := pool.GetNextPeer(nil)
	if peer != b2 {
		t.Fatalf("expected b2 (min connections), got %v", peer)
	}
}

func TestIPHash_ConsistentSelection(t *testing.T) {
	pool := &ServerPool{Algorithm: "ip_hash"}
	b1 := NewBackend("http://localhost:8081", 0, 1)
	b2 := NewBackend("http://localhost:8082", 0, 1)
	b3 := NewBackend("http://localhost:8083", 0, 1)
	pool.AddBackend(b1)
	pool.AddBackend(b2)
	pool.AddBackend(b3)

	r := &http.Request{RemoteAddr: "203.0.113.5:54321"}
	p1 := pool.GetNextPeer(r)
	p2 := pool.GetNextPeer(r)
	if p1 == nil || p2 == nil {
		t.Fatalf("expected non-nil peers for ip_hash, got %v and %v", p1, p2)
	}
	if p1 != p2 {
		t.Fatalf("expected same peer for same IP, got %v and %v", p1, p2)
	}
}

func TestRandom_Deterministic(t *testing.T) {
	pool := &ServerPool{Algorithm: "random"}
	b1 := NewBackend("http://localhost:8081", 0, 1)
	b2 := NewBackend("http://localhost:8082", 0, 1)
	b3 := NewBackend("http://localhost:8083", 0, 1)
	pool.AddBackend(b1)
	pool.AddBackend(b2)
	pool.AddBackend(b3)

	// statistical check: across many trials, random should pick more than one backend
	seen := map[*Backend]bool{}
	for i := 0; i < 100; i++ {
		p := pool.GetNextPeer(nil)
		if p == nil {
			t.Fatalf("expected non-nil peer on trial %d", i)
		}
		seen[p] = true
		if len(seen) > 1 {
			break
		}
	}
	if len(seen) <= 1 {
		t.Fatalf("random algorithm did not select multiple backends across trials; seen=%d", len(seen))
	}
}
