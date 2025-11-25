package stats

import (
	"encoding/json"
	"net/http"
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
	// uptime tracking
	CreatedAt   time.Time `json:"-"`
	LastChecked time.Time `json:"-"`
	UpDuration  int64     `json:"up_duration_ns"`
	Alive       bool      `json:"-"`
}

type Snapshot struct {
	URL            string        `json:"url"`
	Requests       int64         `json:"requests"`
	AvgLatencyMs   float64       `json:"avg_latency_ms"`
	StatusCounts   map[int]int64 `json:"status_counts"`
	FailureRatePct float64       `json:"failure_rate_pct"`
	UptimePct      float64       `json:"uptime_pct"`
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
		now := time.Now()
		stats[url] = &backendStats{URL: url, StatusCounts: map[int]int64{}, PortCounts: map[string]int64{}, CreatedAt: now, LastChecked: now, Alive: false}
	}
}

// RecordHealth updates uptime info for a backend based on a health check.
func RecordHealth(url string, alive bool) {
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
	now := time.Now()
	if bs.CreatedAt.IsZero() {
		bs.CreatedAt = now
	}
	if bs.LastChecked.IsZero() {
		bs.LastChecked = now
		bs.Alive = alive
		bs.mutex.Unlock()
		return
	}
	// time since last check
	delta := now.Sub(bs.LastChecked)
	if bs.Alive {
		bs.UpDuration += int64(delta)
	}
	bs.LastChecked = now
	bs.Alive = alive
	bs.mutex.Unlock()
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
	now := time.Now()
	for k, v := range stats {
		v.mutex.Lock()
		var avg float64
		if v.Requests > 0 {
			avg = float64(v.TotalLatency) / float64(v.Requests) / 1e6
		}
		// (port most-frequent removed from snapshot; PortCounts still collected)
		scopy := map[int]int64{}
		var failures int64
		for s, c := range v.StatusCounts {
			scopy[s] = c
			if s >= 400 {
				failures += c
			}
		}
		// compute uptime percent
		up := time.Duration(v.UpDuration)
		// if currently alive, add time since last checked
		if v.Alive && !v.LastChecked.IsZero() {
			up += now.Sub(v.LastChecked)
		}
		var uptimePct float64
		if !v.CreatedAt.IsZero() {
			total := now.Sub(v.CreatedAt)
			if total > 0 {
				uptimePct = float64(up) / float64(total) * 100.0
			}
		}
		var failurePct float64
		if v.Requests > 0 {
			failurePct = float64(failures) / float64(v.Requests) * 100.0
		}
		out[k] = Snapshot{
			URL:            v.URL,
			Requests:       v.Requests,
			AvgLatencyMs:   avg,
			StatusCounts:   scopy,
			FailureRatePct: failurePct,
			UptimePct:      uptimePct,
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
