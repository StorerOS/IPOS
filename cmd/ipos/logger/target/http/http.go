package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
)

type Target struct {
	logCh chan interface{}

	endpoint  string
	authToken string
	userAgent string
	logKind   string
	client    http.Client
}

func (h *Target) startHTTPLogger() {
	go func() {
		for entry := range h.logCh {
			logJSON, err := json.Marshal(&entry)
			if err != nil {
				continue
			}

			req, err := http.NewRequest(http.MethodPost, h.endpoint, bytes.NewReader(logJSON))
			if err != nil {
				continue
			}
			req.Header.Set(xhttp.ContentType, "application/json")

			req.Header.Set("User-Agent", h.userAgent)

			if h.authToken != "" {
				req.Header.Set("Authorization", h.authToken)
			}

			resp, err := h.client.Do(req)
			if err != nil {
				h.client.CloseIdleConnections()
				continue
			}

			xhttp.DrainBody(resp.Body)
		}
	}()
}

type Option func(*Target)

func WithEndpoint(endpoint string) Option {
	return func(t *Target) {
		t.endpoint = endpoint
	}
}

func WithLogKind(logKind string) Option {
	return func(t *Target) {
		t.logKind = strings.ToUpper(logKind)
	}
}

func WithUserAgent(userAgent string) Option {
	return func(t *Target) {
		t.userAgent = userAgent
	}
}

func WithAuthToken(authToken string) Option {
	return func(t *Target) {
		t.authToken = authToken
	}
}

func WithTransport(transport *http.Transport) Option {
	return func(t *Target) {
		t.client = http.Client{
			Transport: transport,
		}
	}
}

func New(opts ...Option) *Target {
	h := &Target{
		logCh: make(chan interface{}, 10000),
	}

	for _, opt := range opts {
		opt(h)
	}

	h.startHTTPLogger()
	return h
}

func (h *Target) Send(entry interface{}, errKind string) error {
	if h.logKind != errKind && h.logKind != "ALL" {
		return nil
	}

	select {
	case h.logCh <- entry:
	default:
		return errors.New("log buffer full")
	}

	return nil
}
