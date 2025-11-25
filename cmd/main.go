package main

import (
	"fmt"
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

	// If configured, start a set of local backend servers for development/testing
	if config.StartLocalBackendsEnabled() {
		cnt := config.GetLocalBackendCount()
		startPort := config.GetLocalBackendStartPort()

		// build a set of already configured urls to avoid duplicates
		seen := make(map[string]bool)
		for _, u := range serverList {
			seen[u] = true
		}

		started := 0
		for i := 0; i < cnt; i++ {
			p := startPort + i
			url := fmt.Sprintf("http://localhost:%d", p)
			// start a tiny HTTP server that responds on / and can be health-checked
			go func(port int) {
				mux := http.NewServeMux()
				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					_, _ = w.Write([]byte(fmt.Sprintf("ok-backend-%d", port)))
				})
				mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					_, _ = w.Write([]byte("ok"))
				})
				srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
				if err := srv.ListenAndServe(); err != nil {
					log.Printf("local backend on %d stopped: %v", port, err)
				}
			}(p)

			// only append if not already present
			if !seen[url] {
				serverList = append(serverList, url)
				seen[url] = true
				started++
			}
		}
		log.Printf("Started %d local backends starting at %d (attempted %d)", started, startPort, cnt)
	}

	serverPool := &domain.ServerPool{}

	for _, u := range serverList {
		be := domain.NewBackend(u)
		serverPool.AddBackend(be)
	}
	log.Printf("Backends: %v", serverList)
	// run an initial health check so we know status immediately
	serverPool.HealthCheck()
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
