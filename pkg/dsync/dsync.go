package dsync

type Dsync struct {
	GetLockersFn func() []NetLocker
}
