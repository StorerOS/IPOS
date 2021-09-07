// +build darwin freebsd dragonfly

package disk

import (
	"syscall"
)

func GetInfo(path string) (info Info, err error) {
	s := syscall.Statfs_t{}
	err = syscall.Statfs(path, &s)
	if err != nil {
		return Info{}, err
	}
	reservedBlocks := uint64(s.Bfree) - uint64(s.Bavail)
	info = Info{
		Total:  uint64(s.Bsize) * (uint64(s.Blocks) - reservedBlocks),
		Free:   uint64(s.Bsize) * uint64(s.Bavail),
		Files:  uint64(s.Files),
		Ffree:  uint64(s.Ffree),
		FSType: getFSType(s.Fstypename[:]),
	}
	return info, nil
}
