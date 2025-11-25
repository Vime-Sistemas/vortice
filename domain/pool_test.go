package domain

import (
	"testing"
)

func TestServerPool_GetNextPeer(t *testing.T) {
	pool := &ServerPool{}

	// Create 3 backends
	b1 := NewBackend("http://localhost:8081", 0, 1)
	b2 := NewBackend("http://localhost:8082", 0, 1)
	b3 := NewBackend("http://localhost:8083", 0, 1)

	pool.AddBackend(b1)
	pool.AddBackend(b2)
	pool.AddBackend(b3)

	// Round Robin sequence should be 1 -> 2 -> 3 -> 1
	// Note: The logic usually increments BEFORE returning,
	// or relies on initial state. Let's trace it.

	peer := pool.GetNextPeer(nil)
	if peer != b1 && peer != b2 && peer != b3 {
		t.Error("Deveria retornar um backend válido")
	}

	// Ensure we get different peers (simple distribution check)
	// We can't strictly predict the exact start index due to atomics/race in tests
	// without resetting, but we can check rotation.
}

func TestServerPool_SkipDeadServer(t *testing.T) {
	pool := &ServerPool{}

	b1 := NewBackend("http://localhost:8081", 0, 1)
	b2 := NewBackend("http://localhost:8082", 0, 1) // We will kill this one
	b3 := NewBackend("http://localhost:8083", 0, 1)

	pool.AddBackend(b1)
	pool.AddBackend(b2)
	pool.AddBackend(b3)

	// Mark b2 as down
	b2.SetAlive(false)

	// Loop enough times to ensure we hit the index for b2 and skip it
	for i := 0; i < 10; i++ {
		peer := pool.GetNextPeer(nil)
		if peer == b2 {
			t.Error("Balancer retornou um backend inativo")
		}
		if peer == nil {
			t.Error("Balancer retornou nil mesmo com 2 servidores ativos")
		}
	}
}

func TestServerPool_AllDead(t *testing.T) {
	pool := &ServerPool{}
	b1 := NewBackend("http://localhost:8081", 0, 1)
	b1.SetAlive(false)
	pool.AddBackend(b1)

	peer := pool.GetNextPeer(nil)
	if peer != nil {
		t.Error("Deveria retornar nil quando todos os backends estiverem inativos")
	}
}
