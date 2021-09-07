package dsync

import (
	"context"
	"errors"
	golog "log"
	"math/rand"
	"os"
	"sync"
	"time"
)

var dsyncLog bool

func init() {
	dsyncLog = os.Getenv("IPOS_DSYNC_TRACE") == "1"
	rand.Seed(time.Now().UnixNano())
}

func log(msg ...interface{}) {
	if dsyncLog {
		golog.Println(msg...)
	}
}

const DRWMutexAcquireTimeout = 1 * time.Second
const drwMutexInfinite = time.Duration(1<<63 - 1)

type DRWMutex struct {
	Names        []string
	writeLocks   []string
	readersLocks [][]string
	m            sync.Mutex
	clnt         *Dsync
	ctx          context.Context
}

type Granted struct {
	index   int
	lockUID string
}

func (g *Granted) isLocked() bool {
	return isLocked(g.lockUID)
}

func isLocked(uid string) bool {
	return len(uid) > 0
}

func NewDRWMutex(ctx context.Context, clnt *Dsync, names ...string) *DRWMutex {
	return &DRWMutex{
		writeLocks: make([]string, len(clnt.GetLockersFn())),
		Names:      names,
		clnt:       clnt,
		ctx:        ctx,
	}
}

func (dm *DRWMutex) Lock(id, source string) {

	isReadLock := false
	dm.lockBlocking(drwMutexInfinite, id, source, isReadLock)
}

func (dm *DRWMutex) GetLock(id, source string, timeout time.Duration) (locked bool) {

	isReadLock := false
	return dm.lockBlocking(timeout, id, source, isReadLock)
}

func (dm *DRWMutex) RLock(id, source string) {

	isReadLock := true
	dm.lockBlocking(drwMutexInfinite, id, source, isReadLock)
}

func (dm *DRWMutex) GetRLock(id, source string, timeout time.Duration) (locked bool) {

	isReadLock := true
	return dm.lockBlocking(timeout, id, source, isReadLock)
}

func (dm *DRWMutex) lockBlocking(timeout time.Duration, id, source string, isReadLock bool) (locked bool) {
	doneCh, start := make(chan struct{}), time.Now().UTC()
	defer close(doneCh)

	restClnts := dm.clnt.GetLockersFn()

	for range newRetryTimerSimple(doneCh) {
		select {
		case <-dm.ctx.Done():
			return
		default:
		}

		locks := make([]string, len(restClnts))

		success := lock(dm.clnt, &locks, id, source, isReadLock, dm.Names...)
		if success {
			dm.m.Lock()

			if isReadLock {
				dm.readersLocks = append(dm.readersLocks, make([]string, len(restClnts)))
				copy(dm.readersLocks[len(dm.readersLocks)-1], locks[:])
			} else {
				copy(dm.writeLocks, locks[:])
			}

			dm.m.Unlock()
			return true
		}
		if time.Now().UTC().Sub(start) >= timeout {
			break
		}
	}
	return false
}

func lock(ds *Dsync, locks *[]string, id, source string, isReadLock bool, lockNames ...string) bool {

	restClnts := ds.GetLockersFn()

	ch := make(chan Granted, len(restClnts))
	defer close(ch)

	var wg sync.WaitGroup
	for index, c := range restClnts {

		wg.Add(1)
		go func(index int, isReadLock bool, c NetLocker) {
			defer wg.Done()

			g := Granted{index: index}
			if c == nil {
				ch <- g
				return
			}

			args := LockArgs{
				UID:       id,
				Resources: lockNames,
				Source:    source,
			}

			var locked bool
			var err error
			if isReadLock {
				if locked, err = c.RLock(args); err != nil {
					log("Unable to call RLock", err)
				}
			} else {
				if locked, err = c.Lock(args); err != nil {
					log("Unable to call Lock", err)
				}
			}

			if locked {
				g.lockUID = args.UID
			}

			ch <- g

		}(index, isReadLock, c)
	}

	quorum := false

	wg.Add(1)
	go func(isReadLock bool) {
		i, locksFailed := 0, 0
		done := false
		timeout := time.After(DRWMutexAcquireTimeout)

		dquorumReads := (len(restClnts) + 1) / 2
		dquorum := dquorumReads + 1

		for ; i < len(restClnts); i++ {

			select {
			case grant := <-ch:
				if grant.isLocked() {
					(*locks)[grant.index] = grant.lockUID
				} else {
					locksFailed++
					if !isReadLock && locksFailed > len(restClnts)-dquorum ||
						isReadLock && locksFailed > len(restClnts)-dquorumReads {
						done = true
						i++
						releaseAll(ds, locks, isReadLock, restClnts, lockNames...)
					}
				}
			case <-timeout:
				done = true
				if !quorumMet(locks, isReadLock, dquorum, dquorumReads) {
					releaseAll(ds, locks, isReadLock, restClnts, lockNames...)
				}
			}

			if done {
				break
			}
		}

		quorum = quorumMet(locks, isReadLock, dquorum, dquorumReads)

		wg.Done()

		for ; i < len(restClnts); i++ {
			grantToBeReleased := <-ch
			if grantToBeReleased.isLocked() {
				sendRelease(ds, restClnts[grantToBeReleased.index],
					grantToBeReleased.lockUID, isReadLock, lockNames...)
			}
		}
	}(isReadLock)

	wg.Wait()

	return quorum
}

func quorumMet(locks *[]string, isReadLock bool, quorum, quorumReads int) bool {

	count := 0
	for _, uid := range *locks {
		if isLocked(uid) {
			count++
		}
	}

	var metQuorum bool
	if isReadLock {
		metQuorum = count >= quorumReads
	} else {
		metQuorum = count >= quorum
	}

	return metQuorum
}

func releaseAll(ds *Dsync, locks *[]string, isReadLock bool, restClnts []NetLocker, lockNames ...string) {
	for lock := range restClnts {
		if isLocked((*locks)[lock]) {
			sendRelease(ds, restClnts[lock], (*locks)[lock], isReadLock, lockNames...)
			(*locks)[lock] = ""
		}
	}
}

func (dm *DRWMutex) Unlock() {
	restClnts := dm.clnt.GetLockersFn()
	locks := make([]string, len(restClnts))

	{
		dm.m.Lock()
		defer dm.m.Unlock()

		lockFound := false
		for _, uid := range dm.writeLocks {
			if isLocked(uid) {
				lockFound = true
				break
			}
		}
		if !lockFound {
			panic("Trying to Unlock() while no Lock() is active")
		}

		copy(locks, dm.writeLocks[:])
		dm.writeLocks = make([]string, len(restClnts))
	}

	isReadLock := false
	unlock(dm.clnt, locks, isReadLock, restClnts, dm.Names...)
}

func (dm *DRWMutex) RUnlock() {
	restClnts := dm.clnt.GetLockersFn()

	locks := make([]string, len(restClnts))
	{
		dm.m.Lock()
		defer dm.m.Unlock()
		if len(dm.readersLocks) == 0 {
			panic("Trying to RUnlock() while no RLock() is active")
		}
		copy(locks, dm.readersLocks[0][:])
		dm.readersLocks = dm.readersLocks[1:]
	}

	isReadLock := true
	unlock(dm.clnt, locks, isReadLock, restClnts, dm.Names...)
}

func unlock(ds *Dsync, locks []string, isReadLock bool, restClnts []NetLocker, names ...string) {

	for index, c := range restClnts {

		if isLocked(locks[index]) {
			sendRelease(ds, c, locks[index], isReadLock, names...)
		}
	}
}

func sendRelease(ds *Dsync, c NetLocker, uid string, isReadLock bool, names ...string) {
	if c == nil {
		log("Unable to call RUnlock", errors.New("netLocker is offline"))
		return
	}

	args := LockArgs{
		UID:       uid,
		Resources: names,
	}
	if isReadLock {
		if _, err := c.RUnlock(args); err != nil {
			log("Unable to call RUnlock", err)
		}
	} else {
		if _, err := c.Unlock(args); err != nil {
			log("Unable to call Unlock", err)
		}
	}
}
