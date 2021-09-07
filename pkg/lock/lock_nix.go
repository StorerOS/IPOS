// +build !windows,!plan9,!solaris

package lock

import (
	"os"
	"syscall"
)

func lockedOpenFile(path string, flag int, perm os.FileMode, lockType int) (*LockedFile, error) {
	switch flag {
	case syscall.O_RDONLY:
		lockType |= syscall.LOCK_SH
	case syscall.O_WRONLY:
		fallthrough
	case syscall.O_RDWR:
		fallthrough
	case syscall.O_WRONLY | syscall.O_CREAT:
		fallthrough
	case syscall.O_RDWR | syscall.O_CREAT:
		lockType |= syscall.LOCK_EX
	default:
		return nil, &os.PathError{
			Op:   "open",
			Path: path,
			Err:  syscall.EINVAL,
		}
	}

	f, err := os.OpenFile(path, flag|syscall.O_SYNC, perm)
	if err != nil {
		return nil, err
	}

	if err = syscall.Flock(int(f.Fd()), lockType); err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK {
			err = ErrAlreadyLocked
		}
		return nil, err
	}

	st, err := os.Stat(path)
	if err != nil {
		f.Close()
		return nil, err
	}

	if st.IsDir() {
		f.Close()
		return nil, &os.PathError{
			Op:   "open",
			Path: path,
			Err:  syscall.EISDIR,
		}
	}

	return &LockedFile{File: f}, nil
}

func TryLockedOpenFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	return lockedOpenFile(path, flag, perm, syscall.LOCK_NB)
}

func LockedOpenFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	return lockedOpenFile(path, flag, perm, 0)
}

func Open(path string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flag, perm)
}
