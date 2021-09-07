package dsync

import (
	"math/rand"
	"sync"
	"time"
)

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

const MaxJitter = 1.0

const NoJitter = 0.0

var globalRandomSource = rand.New(&lockedRandSource{
	src: rand.NewSource(time.Now().UTC().UnixNano()),
})

func newRetryTimerWithJitter(unit time.Duration, cap time.Duration, jitter float64, doneCh <-chan struct{}) <-chan int {
	attemptCh := make(chan int)

	if jitter < NoJitter {
		jitter = NoJitter
	}
	if jitter > MaxJitter {
		jitter = MaxJitter
	}

	exponentialBackoffWait := func(attempt int) time.Duration {
		maxAttempt := 30
		if attempt > maxAttempt {
			attempt = maxAttempt
		}
		sleep := unit * time.Duration(1<<uint(attempt))
		if sleep > cap {
			sleep = cap
		}
		if jitter != NoJitter {
			sleep -= time.Duration(globalRandomSource.Float64() * float64(sleep) * jitter)
		}
		return sleep
	}

	go func() {
		defer close(attemptCh)
		nextBackoff := 0
		var timer *time.Timer
		for {
			select {
			case attemptCh <- nextBackoff:
				nextBackoff++
			case <-doneCh:
				return
			}
			timer = time.NewTimer(exponentialBackoffWait(nextBackoff))
			select {
			case <-timer.C:
			case <-doneCh:
				timer.Stop()
				return
			}

		}
	}()

	return attemptCh
}

const (
	defaultRetryUnit = time.Second
	defaultRetryCap  = 1 * time.Second
)

func newRetryTimer(unit time.Duration, cap time.Duration, doneCh <-chan struct{}) <-chan int {
	return newRetryTimerWithJitter(unit, cap, MaxJitter, doneCh)
}

func newRetryTimerSimple(doneCh <-chan struct{}) <-chan int {
	return newRetryTimerWithJitter(defaultRetryUnit, defaultRetryCap, MaxJitter, doneCh)
}
