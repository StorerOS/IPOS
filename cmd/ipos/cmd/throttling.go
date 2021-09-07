package cmd

import (
	"net/http"
	"sync"
	"time"
)

type apiThrottling struct {
	mu      sync.RWMutex
	enabled bool

	requestsDeadline time.Duration
	requestsPool     chan struct{}
}

func (t *apiThrottling) init(max int, deadline time.Duration) {
	if max <= 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.requestsPool = make(chan struct{}, max)
	t.requestsDeadline = deadline
	t.enabled = true
}

func (t *apiThrottling) get() (chan struct{}, <-chan time.Time) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.enabled {
		return nil, nil
	}

	return t.requestsPool, time.NewTimer(t.requestsDeadline).C
}

func maxClients(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool, deadlineTimer := globalAPIThrottling.get()
		if pool == nil {
			f.ServeHTTP(w, r)
			return
		}

		select {
		case pool <- struct{}{}:
			defer func() { <-pool }()
			f.ServeHTTP(w, r)
		case <-deadlineTimer:
			writeErrorResponse(r.Context(), w,
				errorCodes.ToAPIErr(ErrOperationMaxedOut),
				r.URL, guessIsBrowserReq(r))
			return
		case <-r.Context().Done():
			return
		}
	}
}
