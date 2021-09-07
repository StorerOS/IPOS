package dsync

type LockArgs struct {
	UID string

	Resources []string

	Source string
}

type NetLocker interface {
	RLock(args LockArgs) (bool, error)

	Lock(args LockArgs) (bool, error)

	RUnlock(args LockArgs) (bool, error)

	Unlock(args LockArgs) (bool, error)

	Expired(args LockArgs) (bool, error)

	String() string

	Close() error

	IsOnline() bool
}
