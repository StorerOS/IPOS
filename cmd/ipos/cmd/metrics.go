package cmd

import (
	"net/http"

	"go.uber.org/atomic"
)

type RequestStats struct {
	Get  atomic.Uint64 `json:"Get"`
	Head atomic.Uint64 `json:"Head"`
	Put  atomic.Uint64 `json:"Put"`
	Post atomic.Uint64 `json:"Post"`
}

type Metrics struct {
	bytesReceived atomic.Uint64
	bytesSent     atomic.Uint64
	requestStats  RequestStats
}

func (s *Metrics) IncBytesReceived(n uint64) {
	s.bytesReceived.Add(n)
}

func (s *Metrics) GetBytesReceived() uint64 {
	return s.bytesReceived.Load()
}

func (s *Metrics) IncBytesSent(n uint64) {
	s.bytesSent.Add(n)
}

func (s *Metrics) GetBytesSent() uint64 {
	return s.bytesSent.Load()
}

func (s *Metrics) IncRequests(method string) {
	if method == http.MethodGet {
		s.requestStats.Get.Add(1)
	} else if method == http.MethodHead {
		s.requestStats.Head.Add(1)
	} else if method == http.MethodPut {
		s.requestStats.Put.Add(1)
	} else if method == http.MethodPost {
		s.requestStats.Post.Add(1)
	}
}

func (s *Metrics) GetRequests() RequestStats {
	return s.requestStats
}

func NewMetrics() *Metrics {
	return &Metrics{}
}
