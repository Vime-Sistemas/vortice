package main

import (
	"log"
	"net/http"

	"github.com/Vime-Sistemas/vortice/domain"
)

func main() {
	// Example: programmatically create a server pool and add a backend
	pool := &domain.ServerPool{Algorithm: "round_robin"}
	b := domain.NewBackend("http://localhost:8081", 0, 1)
	pool.AddBackend(b)

	// start background health checks
	go pool.StartHealthCheck()

	// serve the proxy on :8080
	log.Println("Starting example proxy on :8080")
	log.Fatal(http.ListenAndServe(":8080", pool))
}
