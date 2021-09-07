package madmin

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var MaxRetry = 10

const MaxJitter = 1.0

const NoJitter = 0.0

const DefaultRetryUnit = time.Second

const DefaultRetryCap = time.Second * 30

type lockedRandSource struct {
	lk  sync.Mutex
	src rand.Source
}

func (r *lockedRandSource) Int63() (n int64) {
	r.lk.Lock()
	n = r.src.Int63()
	r.lk.Unlock()
	return
}

func (r *lockedRandSource) Seed(seed int64) {
	r.lk.Lock()
	r.src.Seed(seed)
	r.lk.Unlock()
}

func (adm AdminClient) newRetryTimer(ctx context.Context, maxRetry int, unit time.Duration, cap time.Duration, jitter float64) <-chan int {
	attemptCh := make(chan int)

	exponentialBackoffWait := func(attempt int) time.Duration {
		if jitter < NoJitter {
			jitter = NoJitter
		}
		if jitter > MaxJitter {
			jitter = MaxJitter
		}

		sleep := unit * time.Duration(1<<uint(attempt))
		if sleep > cap {
			sleep = cap
		}
		if jitter != NoJitter {
			sleep -= time.Duration(adm.random.Float64() * float64(sleep) * jitter)
		}
		return sleep
	}

	go func() {
		defer close(attemptCh)
		for i := 0; i < maxRetry; i++ {
			select {
			case attemptCh <- i + 1:
			case <-ctx.Done():
				return
			}

			select {
			case <-time.After(exponentialBackoffWait(i)):
			case <-ctx.Done():
				return
			}
		}
	}()
	return attemptCh
}

func isHTTPReqErrorRetryable(err error) bool {
	if err == nil {
		return false
	}
	switch e := err.(type) {
	case *url.Error:
		switch e.Err.(type) {
		case *net.DNSError, *net.OpError, net.UnknownNetworkError:
			return true
		}
		if strings.Contains(err.Error(), "Connection closed by foreign host") {
			return true
		} else if strings.Contains(err.Error(), "net/http: TLS handshake timeout") {
			return true
		} else if strings.Contains(err.Error(), "i/o timeout") {
			return true
		} else if strings.Contains(err.Error(), "connection timed out") {
			return true
		} else if strings.Contains(err.Error(), "net/http: HTTP/1.x transport connection broken") {
			return true
		}
	}
	return false
}

var retryableS3Codes = map[string]struct{}{
	"RequestError":         {},
	"RequestTimeout":       {},
	"Throttling":           {},
	"ThrottlingException":  {},
	"RequestLimitExceeded": {},
	"RequestThrottled":     {},
	"InternalError":        {},
	"SlowDown":             {},
}

func isS3CodeRetryable(s3Code string) (ok bool) {
	_, ok = retryableS3Codes[s3Code]
	return ok
}

var retryableHTTPStatusCodes = map[int]struct{}{
	http.StatusRequestTimeout:      {},
	http.StatusTooManyRequests:     {},
	http.StatusInternalServerError: {},
	http.StatusBadGateway:          {},
	http.StatusServiceUnavailable:  {},
}

func isHTTPStatusRetryable(httpStatusCode int) (ok bool) {
	_, ok = retryableHTTPStatusCodes[httpStatusCode]
	return ok
}
