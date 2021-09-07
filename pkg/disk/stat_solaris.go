// +build solaris

package disk

import (
	"golang.org/x/sys/unix"
)

func GetInfo(path string) (info Info, err error) {
	s := unix.Statvfs_t{}
	if err = unix.Statvfs(path, &s); err != nil {
		return Info{}, err
	}
	reservedBlocks := uint64(s.Bfree) - uint64(s.Bavail)
	info = Info{
		Total:  uint64(s.Frsize) * (uint64(s.Blocks) - reservedBlocks),
		Free:   uint64(s.Frsize) * uint64(s.Bavail),
		Files:  uint64(s.Files),
		Ffree:  uint64(s.Ffree),
		FSType: getFSType(s.Fstr[:]),
	}
	return info, nil
}
