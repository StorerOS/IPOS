package cmd

import (
	"context"
	"errors"
	pathutil "path"
	"runtime"
	"sort"
	"strings"
	"sync"

	"fmt"
	"time"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/dsync"
	"github.com/storeros/ipos/pkg/lsync"
)

type RWLocker interface {
	GetLock(timeout *dynamicTimeout) (timedOutErr error)
	Unlock()
	GetRLock(timeout *dynamicTimeout) (timedOutErr error)
	RUnlock()
}

func newNSLock(isDistXL bool) *nsLockMap {
	return &nsLockMap{
		lockMap: make(map[string]*nsLock),
	}
}

type nsLock struct {
	*lsync.LRWMutex
	ref uint
}

type nsLockMap struct {
	lockMap      map[string]*nsLock
	lockMapMutex sync.RWMutex
}

func (n *nsLockMap) lock(ctx context.Context, volume string, path string, lockSource, opsID string, readLock bool, timeout time.Duration) (locked bool) {
	var nsLk *nsLock

	resource := pathJoin(volume, path)

	n.lockMapMutex.Lock()
	nsLk, found := n.lockMap[resource]
	if !found {
		nsLk = &nsLock{
			LRWMutex: lsync.NewLRWMutex(ctx),
			ref:      1,
		}
		n.lockMap[resource] = nsLk
	} else {
		nsLk.ref++
	}
	n.lockMapMutex.Unlock()

	if readLock {
		locked = nsLk.GetRLock(opsID, lockSource, timeout)
	} else {
		locked = nsLk.GetLock(opsID, lockSource, timeout)
	}

	if !locked {

		n.lockMapMutex.Lock()
		nsLk.ref--
		if nsLk.ref == 0 {
			delete(n.lockMap, resource)
		}
		n.lockMapMutex.Unlock()
	}
	return
}

func (n *nsLockMap) unlock(volume string, path string, readLock bool) {
	resource := pathJoin(volume, path)
	n.lockMapMutex.RLock()
	nsLk, found := n.lockMap[resource]
	n.lockMapMutex.RUnlock()
	if !found {
		return
	}
	if readLock {
		nsLk.RUnlock()
	} else {
		nsLk.Unlock()
	}
	n.lockMapMutex.Lock()
	if nsLk.ref == 0 {
		logger.LogIf(GlobalContext, errors.New("Namespace reference count cannot be 0"))
	} else {
		nsLk.ref--
		if nsLk.ref == 0 {
			delete(n.lockMap, resource)
		}
	}
	n.lockMapMutex.Unlock()
}

type distLockInstance struct {
	rwMutex *dsync.DRWMutex
	opsID   string
}

func (di *distLockInstance) GetLock(timeout *dynamicTimeout) (timedOutErr error) {
	lockSource := getSource()
	start := UTCNow()

	if !di.rwMutex.GetLock(di.opsID, lockSource, timeout.Timeout()) {
		timeout.LogFailure()
		return OperationTimedOut{}
	}
	timeout.LogSuccess(UTCNow().Sub(start))
	return nil
}

func (di *distLockInstance) Unlock() {
	di.rwMutex.Unlock()
}

func (di *distLockInstance) GetRLock(timeout *dynamicTimeout) (timedOutErr error) {
	lockSource := getSource()
	start := UTCNow()
	if !di.rwMutex.GetRLock(di.opsID, lockSource, timeout.Timeout()) {
		timeout.LogFailure()
		return OperationTimedOut{}
	}
	timeout.LogSuccess(UTCNow().Sub(start))
	return nil
}

func (di *distLockInstance) RUnlock() {
	di.rwMutex.RUnlock()
}

type localLockInstance struct {
	ctx    context.Context
	ns     *nsLockMap
	volume string
	paths  []string
	opsID  string
}

func (n *nsLockMap) NewNSLock(ctx context.Context, lockersFn func() []dsync.NetLocker, volume string, paths ...string) RWLocker {
	opsID := mustGetUUID()
	sort.Strings(paths)
	return &localLockInstance{ctx, n, volume, paths, opsID}
}

func (li *localLockInstance) GetLock(timeout *dynamicTimeout) (timedOutErr error) {
	lockSource := getSource()
	start := UTCNow()
	readLock := false
	var success []int
	for i, path := range li.paths {
		if !li.ns.lock(li.ctx, li.volume, path, lockSource, li.opsID, readLock, timeout.Timeout()) {
			timeout.LogFailure()
			for _, sint := range success {
				li.ns.unlock(li.volume, li.paths[sint], readLock)
			}
			return OperationTimedOut{}
		}
		success = append(success, i)
	}
	timeout.LogSuccess(UTCNow().Sub(start))
	return
}

func (li *localLockInstance) Unlock() {
	readLock := false
	for _, path := range li.paths {
		li.ns.unlock(li.volume, path, readLock)
	}
}

func (li *localLockInstance) GetRLock(timeout *dynamicTimeout) (timedOutErr error) {
	lockSource := getSource()
	start := UTCNow()
	readLock := true
	var success []int
	for i, path := range li.paths {
		if !li.ns.lock(li.ctx, li.volume, path, lockSource, li.opsID, readLock, timeout.Timeout()) {
			timeout.LogFailure()
			for _, sint := range success {
				li.ns.unlock(li.volume, li.paths[sint], readLock)
			}
			return OperationTimedOut{}
		}
		success = append(success, i)
	}
	timeout.LogSuccess(UTCNow().Sub(start))
	return
}

func (li *localLockInstance) RUnlock() {
	readLock := true
	for _, path := range li.paths {
		li.ns.unlock(li.volume, path, readLock)
	}
}

func getSource() string {
	var funcName string
	pc, filename, lineNum, ok := runtime.Caller(2)
	if ok {
		filename = pathutil.Base(filename)
		funcName = strings.TrimPrefix(runtime.FuncForPC(pc).Name(),
			"github.com/storeros/ipos/cmd.")
	} else {
		filename = "<unknown>"
		lineNum = 0
	}

	return fmt.Sprintf("[%s:%d:%s()]", filename, lineNum, funcName)
}
