package lock

import (
	"errors"
	"os"
	"sync"
)

var (
	ErrAlreadyLocked = errors.New("file already locked")
)

type RLockedFile struct {
	*LockedFile
	mutex sync.Mutex
	refs  int
}

func (r *RLockedFile) IsClosed() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.refs == 0
}

func (r *RLockedFile) IncLockRef() {
	r.mutex.Lock()
	r.refs++
	r.mutex.Unlock()
}

func (r *RLockedFile) Close() (err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.refs == 0 {
		return os.ErrInvalid
	}

	r.refs--
	if r.refs == 0 {
		err = r.File.Close()
	}

	return err
}

func newRLockedFile(lkFile *LockedFile) (*RLockedFile, error) {
	if lkFile == nil {
		return nil, os.ErrInvalid
	}

	return &RLockedFile{
		LockedFile: lkFile,
		refs:       1,
	}, nil
}

func RLockedOpenFile(path string) (*RLockedFile, error) {
	lkFile, err := LockedOpenFile(path, os.O_RDONLY, 0666)
	if err != nil {
		return nil, err
	}

	return newRLockedFile(lkFile)

}

type LockedFile struct {
	*os.File
}
