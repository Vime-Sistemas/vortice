package main

import (
	"log"
	"net/http"
	"vortice/config"
	"vortice/domain"
)

func main() {
	// Load .env files (if present) so GetBackends reads configured values
	if loaded := config.LoadEnv(); loaded != "" {
		log.Printf("Loaded env file: %s", loaded)
	}

	serverList := config.GetBackends()

	serverPool := &domain.ServerPool{}

	for _, u := range serverList {
		be := domain.NewBackend(u)
		serverPool.AddBackend(be)
	}
	log.Printf("Backends configurados: %v", serverList)
	if len(serverList) == 0 {
		log.Println("Aviso: nenhum backend configurado; o proxy responderá 503 até que backends sejam adicionados.")
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
