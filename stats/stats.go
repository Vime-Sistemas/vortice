package stats

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"time"
)

type backendStats struct {
	URL          string           `json:"url"`
	Requests     int64            `json:"requests"`
	TotalLatency int64            `json:"total_latency_ns"`
	StatusCounts map[int]int64    `json:"status_counts"`
	PortCounts   map[string]int64 `json:"port_counts"`
	mutex        sync.Mutex       `json:"-"`
}

type Snapshot struct {
	URL            string        `json:"url"`
	Requests       int64         `json:"requests"`
	AvgLatencyMs   float64       `json:"avg_latency_ms"`
	StatusCounts   map[int]int64 `json:"status_counts"`
	MostFamousPort string        `json:"most_famous_port"`
}

var (
	mu    sync.RWMutex
	stats = map[string]*backendStats{}
)

// RegisterBackend ensures a backend entry exists.
func RegisterBackend(url string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := stats[url]; !ok {
		stats[url] = &backendStats{URL: url, StatusCounts: map[int]int64{}, PortCounts: map[string]int64{}}
	}
}

// Record records a completed request for a backend.
func Record(url string, duration time.Duration, status int) {
	mu.RLock()
	bs, ok := stats[url]
	mu.RUnlock()
	if !ok {
		RegisterBackend(url)
		mu.RLock()
		bs = stats[url]
		mu.RUnlock()
	}
	bs.mutex.Lock()
	bs.Requests++
	bs.TotalLatency += int64(duration)
	bs.StatusCounts[status]++
	// try to extract port from url; naive parse: look for :port at end
	// we'll scan from the right for ':' and take substring
	u := url
	port := ""
	for i := len(u) - 1; i >= 0; i-- {
		if u[i] == ':' {
			port = u[i+1:]
			break
		}
		if u[i] == '/' {
			break
		}
	}
	if port == "" {
		port = "default"
	}
	bs.PortCounts[port]++
	bs.mutex.Unlock()
}

// SnapshotAll returns a snapshot copy of all backend stats.
func SnapshotAll() map[string]Snapshot {
	out := map[string]Snapshot{}
	mu.RLock()
	defer mu.RUnlock()
	for k, v := range stats {
		v.mutex.Lock()
		var avg float64
		if v.Requests > 0 {
			avg = float64(v.TotalLatency) / float64(v.Requests) / 1e6
		}
		// find most famous port
		mf := ""
		max := int64(0)
		// create ordered list to keep deterministic results
		ports := make([]string, 0, len(v.PortCounts))
		for p := range v.PortCounts {
			ports = append(ports, p)
		}
		sort.Strings(ports)
		for _, p := range ports {
			if v.PortCounts[p] > max {
				max = v.PortCounts[p]
				mf = p
			}
		}
		scopy := map[int]int64{}
		for s, c := range v.StatusCounts {
			scopy[s] = c
		}
		out[k] = Snapshot{
			URL:            v.URL,
			Requests:       v.Requests,
			AvgLatencyMs:   avg,
			StatusCounts:   scopy,
			MostFamousPort: mf,
		}
		v.mutex.Unlock()
	}
	return out
}

// Handler returns an http.Handler that serves JSON stats.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snap := SnapshotAll()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(snap)
	})
}
