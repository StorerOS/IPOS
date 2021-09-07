package lsync

import (
	"context"
	"sync"
	"time"
)

const (
	WRITELOCK = -1 + iota
	NOLOCKS
	READLOCKS
)

type LRWMutex struct {
	id     string
	source string
	state  int64
	m      sync.Mutex
	ctx    context.Context
}

func NewLRWMutex(ctx context.Context) *LRWMutex {
	return &LRWMutex{ctx: ctx}
}

func (lm *LRWMutex) Lock() {

	isWriteLock := true
	lm.lockLoop(lm.id, lm.source, time.Duration(1<<63-1), isWriteLock)
}

func (lm *LRWMutex) GetLock(id string, source string, timeout time.Duration) (locked bool) {

	isWriteLock := true
	return lm.lockLoop(id, source, timeout, isWriteLock)
}

func (lm *LRWMutex) RLock() {

	isWriteLock := false
	lm.lockLoop(lm.id, lm.source, time.Duration(1<<63-1), isWriteLock)
}

func (lm *LRWMutex) GetRLock(id string, source string, timeout time.Duration) (locked bool) {

	isWriteLock := false
	return lm.lockLoop(id, source, timeout, isWriteLock)
}

func (lm *LRWMutex) lockLoop(id, source string, timeout time.Duration, isWriteLock bool) bool {
	doneCh, start := make(chan struct{}), time.Now().UTC()
	defer close(doneCh)

	for range newRetryTimerSimple(doneCh) {
		select {
		case <-lm.ctx.Done():
			break
		default:
		}

		var success bool
		{
			lm.m.Lock()

			lm.id = id
			lm.source = source

			if isWriteLock {
				if lm.state == NOLOCKS {
					lm.state = WRITELOCK
					success = true
				}
			} else {
				if lm.state != WRITELOCK {
					lm.state++
					success = true
				}
			}

			lm.m.Unlock()
		}
		if success {
			return true
		}
		if time.Now().UTC().Sub(start) >= timeout {
			break
		}

	}
	return false
}

func (lm *LRWMutex) Unlock() {

	isWriteLock := true
	success := lm.unlock(isWriteLock)
	if !success {
		panic("Trying to Unlock() while no Lock() is active")
	}
}

func (lm *LRWMutex) RUnlock() {

	isWriteLock := false
	success := lm.unlock(isWriteLock)
	if !success {
		panic("Trying to RUnlock() while no RLock() is active")
	}
}

func (lm *LRWMutex) unlock(isWriteLock bool) (unlocked bool) {
	lm.m.Lock()

	if isWriteLock {
		if lm.state == WRITELOCK {
			lm.state = NOLOCKS
			unlocked = true
		}
	} else {
		if lm.state == WRITELOCK || lm.state == NOLOCKS {
			unlocked = false
		} else {
			lm.state--
			unlocked = true
		}
	}

	lm.m.Unlock()
	return unlocked
}

func (lm *LRWMutex) ForceUnlock() {
	lm.m.Lock()
	lm.state = NOLOCKS
	lm.m.Unlock()
}

func (lm *LRWMutex) DRLocker() sync.Locker {
	return (*drlocker)(lm)
}

type drlocker LRWMutex

func (dr *drlocker) Lock()   { (*LRWMutex)(dr).RLock() }
func (dr *drlocker) Unlock() { (*LRWMutex)(dr).RUnlock() }
