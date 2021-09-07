package cmd

import (
	"io"
	"net/http"
	"time"
)

type recordTrafficRequest struct {
	io.ReadCloser
	isS3Request bool
}

func (r *recordTrafficRequest) Read(p []byte) (n int, err error) {
	n, err = r.ReadCloser.Read(p)
	return n, err
}

type recordTrafficResponse struct {
	http.ResponseWriter
	isS3Request bool
}

func (r *recordTrafficResponse) Write(p []byte) (n int, err error) {
	n, err = r.ResponseWriter.Write(p)
	return n, err
}

func (r *recordTrafficResponse) Flush() {
	r.ResponseWriter.(http.Flusher).Flush()
}

type recordAPIStats struct {
	http.ResponseWriter
	TTFB           time.Time
	firstByteRead  bool
	respStatusCode int
	isS3Request    bool
}

func (r *recordAPIStats) WriteHeader(i int) {
	r.respStatusCode = i
	r.ResponseWriter.WriteHeader(i)
}

func (r *recordAPIStats) Write(p []byte) (n int, err error) {
	if !r.firstByteRead {
		r.TTFB = UTCNow()
		r.firstByteRead = true
	}
	return r.ResponseWriter.Write(p)
}

func (r *recordAPIStats) Flush() {
	r.ResponseWriter.(http.Flusher).Flush()
}
