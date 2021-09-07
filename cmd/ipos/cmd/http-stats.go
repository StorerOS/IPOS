package cmd

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/atomic"
)

type ConnStats struct {
	totalInputBytes  atomic.Uint64
	totalOutputBytes atomic.Uint64
	s3InputBytes     atomic.Uint64
	s3OutputBytes    atomic.Uint64
}

func (s *ConnStats) incInputBytes(n int) {
	s.totalInputBytes.Add(uint64(n))
}

func (s *ConnStats) incOutputBytes(n int) {
	s.totalOutputBytes.Add(uint64(n))
}

func (s *ConnStats) getTotalInputBytes() uint64 {
	return s.totalInputBytes.Load()
}

func (s *ConnStats) getTotalOutputBytes() uint64 {
	return s.totalOutputBytes.Load()
}

func (s *ConnStats) incS3InputBytes(n int) {
	s.s3InputBytes.Add(uint64(n))
}

func (s *ConnStats) incS3OutputBytes(n int) {
	s.s3OutputBytes.Add(uint64(n))
}

func (s *ConnStats) getS3InputBytes() uint64 {
	return s.s3InputBytes.Load()
}

func (s *ConnStats) getS3OutputBytes() uint64 {
	return s.s3OutputBytes.Load()
}

func newConnStats() *ConnStats {
	return &ConnStats{}
}

type HTTPAPIStats struct {
	apiStats map[string]int
	sync.RWMutex
}

func (stats *HTTPAPIStats) Inc(api string) {
	if stats == nil {
		return
	}
	stats.Lock()
	defer stats.Unlock()
	if stats.apiStats == nil {
		stats.apiStats = make(map[string]int)
	}
	stats.apiStats[api]++
}

func (stats *HTTPAPIStats) Dec(api string) {
	if stats == nil {
		return
	}
	stats.Lock()
	defer stats.Unlock()
	if val, ok := stats.apiStats[api]; ok && val > 0 {
		stats.apiStats[api]--
	}
}

func (stats *HTTPAPIStats) Load() map[string]int {
	stats.Lock()
	defer stats.Unlock()
	var apiStats = make(map[string]int, len(stats.apiStats))
	for k, v := range stats.apiStats {
		apiStats[k] = v
	}
	return apiStats
}

type HTTPStats struct {
	currentS3Requests HTTPAPIStats
	totalS3Requests   HTTPAPIStats
	totalS3Errors     HTTPAPIStats
}

func durationStr(totalDuration, totalCount float64) string {
	return fmt.Sprint(time.Duration(totalDuration/totalCount) * time.Second)
}

func (st *HTTPStats) updateStats(api string, r *http.Request, w *recordAPIStats, durationSecs float64) {
	successReq := (w.respStatusCode >= 200 && w.respStatusCode < 300)

	if w.isS3Request {
		st.totalS3Requests.Inc(api)
		if !successReq && w.respStatusCode != 0 {
			st.totalS3Errors.Inc(api)
		}
	}
}

func newHTTPStats() *HTTPStats {
	return &HTTPStats{}
}
