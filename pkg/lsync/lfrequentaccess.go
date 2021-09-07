package lsync

import (
	"sync"
	"sync/atomic"
)

type LFrequentAccess struct {
	state     atomic.Value
	writeLock sync.Mutex
	locked    bool
}

func NewLFrequentAccess(x interface{}) *LFrequentAccess {
	lm := &LFrequentAccess{}
	lm.state.Store(x)
	return lm
}

func (lm *LFrequentAccess) ReadOnlyAccess() (constReadOnly interface{}) {
	return lm.state.Load()
}

func (lm *LFrequentAccess) LockBeforeSet() (constCurVersion interface{}) {
	lm.writeLock.Lock()
	lm.locked = true
	return lm.state.Load()
}

func (lm *LFrequentAccess) SetNewCopyAndUnlock(newCopy interface{}) {
	if !lm.locked {
		panic("SetNewCopyAndUnlock: locked state is false (did you call LockBeforeSet?)")
	}
	lm.state.Store(newCopy)
	lm.locked = false
	lm.writeLock.Unlock()
}
