// +build solaris

package lock

import (
	"os"
	"syscall"
)

func lockedOpenFile(path string, flag int, perm os.FileMode, rlockType int) (*LockedFile, error) {
	var lockType int16
	switch flag {
	case syscall.O_RDONLY:
		lockType = syscall.F_RDLCK
	case syscall.O_WRONLY:
		fallthrough
	case syscall.O_RDWR:
		fallthrough
	case syscall.O_WRONLY | syscall.O_CREAT:
		fallthrough
	case syscall.O_RDWR | syscall.O_CREAT:
		lockType = syscall.F_WRLCK
	default:
		return nil, &os.PathError{
			Op:   "open",
			Path: path,
			Err:  syscall.EINVAL,
		}
	}

	var lock = syscall.Flock_t{
		Start:  0,
		Len:    0,
		Pid:    0,
		Type:   lockType,
		Whence: 0,
	}

	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}

	if err = syscall.FcntlFlock(f.Fd(), rlockType, &lock); err != nil {
		f.Close()
		if err == syscall.EAGAIN {
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

	return &LockedFile{f}, nil
}

func TryLockedOpenFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	return lockedOpenFile(path, flag, perm, syscall.F_SETLK)
}

func LockedOpenFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	return lockedOpenFile(path, flag, perm, syscall.F_SETLKW)
}

func Open(path string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(path, flag, perm)
}
