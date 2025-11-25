package domain

import (
	"hash/fnv"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Vime-Sistemas/vortice/stats"
)

type ServerPool struct {
	backends []*Backend
	current  uint64
	// Algorithm pode ser: "round_robin", "least_conn", "random", "ip_hash"
	Algorithm string
	// IPHashHeader, se não vazio, indica o header a ser usado para ip_hash (ex: X-Forwarded-For)
	IPHashHeader string
}

func (s *ServerPool) AddBackend(b *Backend) {
	s.backends = append(s.backends, b)
	// register backend for stats collection
	stats.RegisterBackend(b.URL.String())
}

func (s *ServerPool) GetNextPeer(r *http.Request) *Backend {
	if len(s.backends) == 0 {
		return nil
	}

	switch strings.ToLower(s.Algorithm) {
	case "least_conn":
		// find alive backend with minimum ConnCount
		var chosen *Backend
		var min int64 = -1
		for _, be := range s.backends {
			if !be.IsAlive() {
				continue
			}
			cnt := atomic.LoadInt64(&be.ConnCount)
			if chosen == nil || cnt < min {
				chosen = be
				min = cnt
			}
		}
		return chosen
	case "random":
		// pick a random alive backend
		alive := make([]*Backend, 0, len(s.backends))
		for _, be := range s.backends {
			if be.IsAlive() {
				alive = append(alive, be)
			}
		}
		if len(alive) == 0 {
			return nil
		}
		return alive[rand.Intn(len(alive))]
	case "ip_hash":
		// hash remote ip or header to pick backend
		var key string
		if s.IPHashHeader != "" {
			if r != nil {
				key = r.Header.Get(s.IPHashHeader)
			}
		} else if r != nil {
			key = r.RemoteAddr
		}
		if key == "" {
			// fallback to round-robin when no key available
			next := s.NextIndex()
			l := len(s.backends) + s.NextIndex()
			for i := next; i < l; i++ {
				idx := i % len(s.backends)
				if s.backends[idx].IsAlive() {
					if i != next {
						atomic.StoreUint64(&s.current, uint64(idx))
					}
					return s.backends[idx]
				}
			}
			return nil
		}
		// if header contains multiple ips (X-Forwarded-For), take first
		if idxc := strings.Index(key, ","); idxc != -1 {
			key = strings.TrimSpace(key[:idxc])
		}
		// remove port if present
		if idx := strings.LastIndex(key, ":"); idx != -1 {
			key = key[:idx]
		}
		h := fnv.New32a()
		h.Write([]byte(key))
		idx := int(h.Sum32()) % len(s.backends)
		// find next alive starting from idx
		for i := 0; i < len(s.backends); i++ {
			j := (idx + i) % len(s.backends)
			if s.backends[j].IsAlive() {
				return s.backends[j]
			}
		}
		return nil
	default:
		// round_robin (default)
		next := s.NextIndex()
		l := len(s.backends) + s.NextIndex()
		for i := next; i < l; i++ {
			idx := i % len(s.backends)
			if s.backends[idx].IsAlive() {
				if i != next {
					atomic.StoreUint64(&s.current, uint64(idx))
				}
				return s.backends[idx]
			}
		}
		return nil
	}
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

func (s *ServerPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	peer := s.GetNextPeer(r)

	if peer == nil {
		http.Error(w, "Serviço não disponível", http.StatusServiceUnavailable)
		return
	}

	// rate limiting per backend
	if peer.Limiter != nil {
		if !peer.Limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
	}

	// increment active connections and ensure decrement after serving
	atomic.AddInt64(&peer.ConnCount, 1)
	defer atomic.AddInt64(&peer.ConnCount, -1)

	// record start time and capture status
	start := time.Now()
	rw := &statusRecorder{ResponseWriter: w, status: 0}
	peer.ReverseProxy.ServeHTTP(rw, r)
	duration := time.Since(start)
	status := rw.status
	if status == 0 {
		status = http.StatusOK
	}
	// record stats
	stats.Record(peer.URL.String(), duration, status)
}

// statusRecorder wraps ResponseWriter to capture status code
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// HealthCheck loops through all backends and updates their status
func (s *ServerPool) HealthCheck() {
	for _, b := range s.backends {
		status := "ativo"
		alive := b.CheckHealth()
		b.SetAlive(alive)
		if !alive {
			status = "inativo"
		}
		log.Printf("%s [%s]", b.URL, status)
	}
}

// StartHealthCheck starts a ticker to run the check every 20 seconds
func (s *ServerPool) StartHealthCheck() {
	t := time.NewTicker(time.Second * 20)
	for {
		select {
		case <-t.C:
			log.Println("Iniciando verificação de saúde...")
			s.HealthCheck()
			log.Println("Verificação de saúde concluída")
		}
	}
}

// BackendURLs returns the list of backend URLs (string form).
func (s *ServerPool) BackendURLs() []string {
	out := make([]string, 0, len(s.backends))
	for _, b := range s.backends {
		out = append(out, b.URL.String())
	}
	return out
}
