package main

import (
	"log"
	"net/http"
	"vortice/config"
	"vortice/domain"
)

func main() {
	serverList := config.GetBackends()

	serverPool := &domain.ServerPool{}

	for _, u := range serverList {
		be := domain.NewBackend(u)
		serverPool.AddBackend(be)
	}

	go serverPool.StartHealthCheck()

	port := ":" + config.GetAppPort()
	server := http.Server{
		Addr:    port,
		Handler: serverPool,
	}

	log.Printf("🌀 Vortice iniciado na porta %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
