// +build windows

package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modkernel32    = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx = modkernel32.NewProc("LockFileEx")
)

const (
	lockFileExclusiveLock   = 2
	lockFileFailImmediately = 1

	errLockViolation syscall.Errno = 0x21
)

func lockedOpenFile(path string, flag int, perm os.FileMode, lockType uint32) (*LockedFile, error) {
	f, err := Open(path, flag, perm)
	if err != nil {
		return nil, err
	}

	if err = lockFile(syscall.Handle(f.Fd()), lockType); err != nil {
		f.Close()
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
	var lockType uint32 = lockFileFailImmediately | lockFileExclusiveLock
	switch flag {
	case syscall.O_RDONLY:
		lockType = lockFileFailImmediately | 0
	}
	return lockedOpenFile(path, flag, perm, lockType)
}

func LockedOpenFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	var lockType uint32 = lockFileExclusiveLock
	switch flag {
	case syscall.O_RDONLY:
		lockType = 0
	}
	return lockedOpenFile(path, flag, perm, lockType)
}

func fixLongPath(path string) string {
	if len(path) < 248 {
		return path
	}

	if len(path) >= 2 && path[:2] == `\\` {
		return path
	}
	if !filepath.IsAbs(path) {
		return path
	}

	const prefix = `\\?`

	pathbuf := make([]byte, len(prefix)+len(path)+len(`\`))
	copy(pathbuf, prefix)
	n := len(path)
	r, w := 0, len(prefix)
	for r < n {
		switch {
		case os.IsPathSeparator(path[r]):
			r++
		case path[r] == '.' && (r+1 == n || os.IsPathSeparator(path[r+1])):
			r++
		case r+1 < n && path[r] == '.' && path[r+1] == '.' && (r+2 == n || os.IsPathSeparator(path[r+2])):
			return path
		default:
			pathbuf[w] = '\\'
			w++
			for ; r < n && !os.IsPathSeparator(path[r]); r++ {
				pathbuf[w] = path[r]
				w++
			}
		}
	}
	if w == len(`\\?\c:`) {
		pathbuf[w] = '\\'
		w++
	}
	return string(pathbuf[:w])
}

func Open(path string, flag int, perm os.FileMode) (*os.File, error) {
	if path == "" {
		return nil, syscall.ERROR_FILE_NOT_FOUND
	}

	pathp, err := syscall.UTF16PtrFromString(fixLongPath(path))
	if err != nil {
		return nil, err
	}

	var access uint32
	switch flag {
	case syscall.O_RDONLY:
		access = syscall.GENERIC_READ
	case syscall.O_WRONLY:
		access = syscall.GENERIC_WRITE
	case syscall.O_RDWR:
		fallthrough
	case syscall.O_RDWR | syscall.O_CREAT:
		fallthrough
	case syscall.O_WRONLY | syscall.O_CREAT:
		access = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	case syscall.O_WRONLY | syscall.O_CREAT | syscall.O_APPEND:
		access = syscall.FILE_APPEND_DATA
	default:
		return nil, fmt.Errorf("Unsupported flag (%d)", flag)
	}

	var createflag uint32
	switch {
	case flag&syscall.O_CREAT == syscall.O_CREAT:
		createflag = syscall.OPEN_ALWAYS
	default:
		createflag = syscall.OPEN_EXISTING
	}

	shareflag := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)
	accessAttr := uint32(syscall.FILE_ATTRIBUTE_NORMAL | 0x80000000)

	fd, err := syscall.CreateFile(pathp, access, shareflag, nil, createflag, accessAttr, 0)
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), path), nil
}

func lockFile(fd syscall.Handle, flags uint32) error {
	if fd == syscall.InvalidHandle {
		return nil
	}

	err := lockFileEx(fd, flags, 1, 0, &syscall.Overlapped{})
	if err == nil {
		return nil
	} else if err.Error() == "The process cannot access the file because another process has locked a portion of the file." {
		return ErrAlreadyLocked
	} else if err != errLockViolation {
		return err
	}

	return nil
}

func lockFileEx(h syscall.Handle, flags, locklow, lockhigh uint32, ol *syscall.Overlapped) (err error) {
	var reserved = uint32(0)
	r1, _, e1 := syscall.Syscall6(procLockFileEx.Addr(), 6, uintptr(h), uintptr(flags),
		uintptr(reserved), uintptr(locklow), uintptr(lockhigh), uintptr(unsafe.Pointer(ol)))
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}
