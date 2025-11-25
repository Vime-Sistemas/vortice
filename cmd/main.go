package main

import (
	"log"
	"net/http"
	"vortice/domain"
)

func main() {
	serverList := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	serverPool := &domain.ServerPool{}

	for _, u := range serverList {
		be := domain.NewBackend(u)
		serverPool.AddBackend(be)
	}

	go serverPool.StartHealthCheck()

	port := ":8080"
	server := http.Server{
		Addr:    port,
		Handler: serverPool,
	}

	log.Printf("🌀 Vortice Load Balancer started on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
