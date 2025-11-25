package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Vime-Sistemas/vortice/config"
	"github.com/Vime-Sistemas/vortice/domain"
	"github.com/Vime-Sistemas/vortice/stats"
)

// replEnabled toggles special REPL-aware log output.
var replEnabled = false
var replPrompt = "vortice> "

type replWriter struct{}

func (rw replWriter) Write(p []byte) (int, error) {
	// In interactive mode, ensure the log message starts on a new line
	if replEnabled {
		_, _ = os.Stderr.Write([]byte("\n"))
	}
	// write original log bytes to stderr
	n, err := os.Stderr.Write(p)
	if err != nil {
		return n, err
	}
	// ensure newline after the log message
	if len(p) == 0 || p[len(p)-1] != '\n' {
		_, _ = os.Stderr.Write([]byte("\n"))
	}
	// if REPL is enabled, reprint prompt to stdout so user can continue typing
	if replEnabled {
		_, _ = os.Stdout.Write([]byte(replPrompt))
	}
	return n, nil
}

// NewProxyPool creates a ServerPool configured with the provided backend URLs,
// algorithm, per-backend rate limits and optional ip-hash header.
func NewProxyPool(serverList []string, algo string, per [][2]int, ipHashHeader string) *domain.ServerPool {
	pool := &domain.ServerPool{Algorithm: algo, IPHashHeader: ipHashHeader}
	for i, u := range serverList {
		// validate URL
		if _, err := url.Parse(u); err != nil {
			continue
		}
		rps := 0
		burst := 1
		if i < len(per) {
			rps = per[i][0]
			burst = per[i][1]
		}
		be := domain.NewBackend(u, rps, burst)
		pool.AddBackend(be)
	}
	return pool
}

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
		// If forced, ignore configured BACKEND_URLS and use only local backends
		if config.GetLocalBackendForce() {
			log.Println("LOCAL_BACKEND_FORCE=true: backend local irá substituir BACKEND_URLS configurado")
			serverList = []string{}
		}

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
					log.Printf("backend local em %d parado: %v", port, err)
				}
			}(p)

			// only append if not already present
			if !seen[url] {
				serverList = append(serverList, url)
				seen[url] = true
				started++
			}
		}
		log.Printf("Backend local iniciados: %d, porta inicial: %d, tentados: %d", started, startPort, cnt)
	}

	rand.Seed(time.Now().UnixNano())

	// create a ServerPool via helper so callers/tests can reuse it
	per := config.GetPerBackendRateLimits(len(serverList))
	serverPool := NewProxyPool(serverList, config.GetLBAlgorithm(), per, config.GetIPHashHeader())
	log.Printf("Backends: %v", serverList)
	// run an initial health check so we know status immediately
	serverPool.HealthCheck()
	if len(serverList) == 0 {
		log.Println("Aviso: nenhum backend configurado; o proxy responderá 503 até que backends sejam adicionados.")
	}

	go serverPool.StartHealthCheck()

	port := ":" + config.GetAppPort()
	// create a mux to expose stats endpoint and the proxy
	mux := http.NewServeMux()
	mux.Handle("/", serverPool)
	mux.Handle("/stats", stats.Handler())

	server := http.Server{
		Addr:    port,
		Handler: mux,
	}

	log.Printf("🌀 Vortice iniciado na porta %s", port)
	interactive := strings.ToLower(os.Getenv("INTERACTIVE")) == "true"
	// if interactive, replace the default logger output so log lines don't
	// clobber the REPL prompt: the custom writer will reprint the prompt
	// after every log write when interactive mode is active.
	if interactive {
		replEnabled = true
		log.SetOutput(replWriter{})
	}
	if interactive {
		// run HTTP server in background and keep REPL in foreground
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("server error: %v", err)
			}
		}()
		runInteractive(serverPool)
		return
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// runInteractive runs a simple REPL allowing commands to inspect stats/backends.
func runInteractive(serverPool *domain.ServerPool) {
	// ASCII header
	fmt.Println("========================================")
	fmt.Println(" Vortice - console interativo")
	fmt.Println(" Comandos: stats | backends | watch <s> | help | exit")
	fmt.Println("========================================")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(replPrompt)
		if !scanner.Scan() {
			fmt.Println("\nEOF — saindo")
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])
		switch cmd {
		case "help":
			fmt.Println("Comandos:")
			fmt.Println("  stats         - mostrar snapshot de estatísticas como tabela")
			fmt.Println("  backends      - listar backends configurados")
			fmt.Println("  watch <secs>  - atualizar estatísticas a cada <secs> segundos (ctrl+C para parar)")
			fmt.Println("  exit          - sair da console interativa")
		case "backends":
			urls := serverPool.BackendURLs()
			if len(urls) == 0 {
				fmt.Println("(no backends configured)")
				continue
			}
			for i, u := range urls {
				fmt.Printf("%d. %s\n", i+1, u)
			}
		case "stats":
			printStatsTable()
		case "watch":
			secs := 2
			if len(parts) > 1 {
				if v, err := fmt.Sscanf(parts[1], "%d", &secs); err != nil || v == 0 {
					secs = 2
				}
			}
			for {
				printStatsTable()
				time.Sleep(time.Duration(secs) * time.Second)
			}
		case "exit", "quit":
			fmt.Println("Saindo...")
			return
		default:
			fmt.Printf("Comando desconhecido: %s (use 'help')\n", cmd)
		}
	}
}

func printStatsTable() {
	snap := stats.SnapshotAll()
	if len(snap) == 0 {
		fmt.Println("(no stats available)")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "URL\tREQS\tAVG_MS\tFAIL%\tUPTIME%\tSTATUS_COUNTS")
	// deterministic order
	urls := make([]string, 0, len(snap))
	for u := range snap {
		urls = append(urls, u)
	}
	// sort for deterministic output
	// lightweight sort
	for i := 0; i < len(urls)-1; i++ {
		for j := i + 1; j < len(urls); j++ {
			if urls[i] > urls[j] {
				urls[i], urls[j] = urls[j], urls[i]
			}
		}
	}
	for _, u := range urls {
		s := snap[u]
		// format status counts compactly
		scs := ""
		first := true
		for k, v := range s.StatusCounts {
			if !first {
				scs += ","
			}
			scs += fmt.Sprintf("%d=%d", k, v)
			first = false
		}
		fmt.Fprintf(w, "%s\t%d\t%.2f\t%.2f\t%.2f\t%s\n", s.URL, s.Requests, s.AvgLatencyMs, s.FailureRatePct, s.UptimePct, scs)
	}
	_ = w.Flush()
}
