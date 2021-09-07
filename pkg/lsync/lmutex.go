package lsync

import (
	"context"
	"sync/atomic"
	"time"
)

type LMutex struct {
	id     string
	source string
	state  int64
	ctx    context.Context
}

func NewLMutex(ctx context.Context) *LMutex {
	return &LMutex{ctx: ctx}
}

func (lm *LMutex) Lock() {
	lm.lockLoop(lm.id, lm.source, time.Duration(1<<63-1))
}

func (lm *LMutex) GetLock(id, source string, timeout time.Duration) (locked bool) {
	return lm.lockLoop(id, source, timeout)
}

func (lm *LMutex) lockLoop(id, source string, timeout time.Duration) bool {
	doneCh, start := make(chan struct{}), time.Now().UTC()
	defer close(doneCh)

	for range newRetryTimerSimple(doneCh) {
		select {
		case <-lm.ctx.Done():
			break
		default:
		}

		if atomic.CompareAndSwapInt64(&lm.state, NOLOCKS, WRITELOCK) {
			lm.id = id
			lm.source = source
			return true
		} else if time.Now().UTC().Sub(start) >= timeout {
			break
		}
	}
	return false
}

func (lm *LMutex) Unlock() {
	if !atomic.CompareAndSwapInt64(&lm.state, WRITELOCK, NOLOCKS) {
		panic("Trying to Unlock() while no Lock() is active")
	}
}
