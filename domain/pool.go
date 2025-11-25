package domain

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type ServerPool struct {
	backends []*Backend
	current  uint64
}

func (s *ServerPool) AddBackend(b *Backend) {
	s.backends = append(s.backends, b)
}

func (s *ServerPool) GetNextPeer() *Backend {
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

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

func (s *ServerPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	peer := s.GetNextPeer()

	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Serviço não disponível", http.StatusServiceUnavailable)
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
