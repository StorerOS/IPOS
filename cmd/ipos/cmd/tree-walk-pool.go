package cmd

import (
	"errors"
	"reflect"
	"sync"
	"time"
)

const (
	globalLookupTimeout = time.Minute * 30
)

type listParams struct {
	bucket    string
	recursive bool
	marker    string
	prefix    string
}

var errWalkAbort = errors.New("treeWalk abort")

type treeWalk struct {
	resultCh   chan TreeWalkResult
	endWalkCh  chan struct{}
	endTimerCh chan<- struct{}
}

type TreeWalkPool struct {
	pool    map[listParams][]treeWalk
	timeOut time.Duration
	lock    *sync.Mutex
}

func NewTreeWalkPool(timeout time.Duration) *TreeWalkPool {
	tPool := &TreeWalkPool{
		pool:    make(map[listParams][]treeWalk),
		timeOut: timeout,
		lock:    &sync.Mutex{},
	}
	return tPool
}

func (t TreeWalkPool) Release(params listParams) (resultCh chan TreeWalkResult, endWalkCh chan struct{}) {
	t.lock.Lock()
	defer t.lock.Unlock()
	walks, ok := t.pool[params]
	if ok {
		if len(walks) > 0 {
			walk := walks[0]
			walks = walks[1:]
			if len(walks) > 0 {
				t.pool[params] = walks
			} else {
				delete(t.pool, params)
			}
			walk.endTimerCh <- struct{}{}
			return walk.resultCh, walk.endWalkCh
		}
	}
	return nil, nil
}

func (t TreeWalkPool) Set(params listParams, resultCh chan TreeWalkResult, endWalkCh chan struct{}) {
	t.lock.Lock()
	defer t.lock.Unlock()

	endTimerCh := make(chan struct{}, 1)
	walkInfo := treeWalk{
		resultCh:   resultCh,
		endWalkCh:  endWalkCh,
		endTimerCh: endTimerCh,
	}
	t.pool[params] = append(t.pool[params], walkInfo)

	go func(endTimerCh <-chan struct{}) {
		select {
		case <-time.After(t.timeOut):
			t.lock.Lock()
			walks, ok := t.pool[params]
			if ok {
				nwalks := walks[:0]
				for _, walk := range walks {
					if !reflect.DeepEqual(walk, walkInfo) {
						nwalks = append(nwalks, walk)
					}
				}
				if len(nwalks) == 0 {
					delete(t.pool, params)
				} else {
					t.pool[params] = nwalks
				}
			}
			close(endWalkCh)
			t.lock.Unlock()
		case <-endTimerCh:
			return
		}
	}(endTimerCh)
}
