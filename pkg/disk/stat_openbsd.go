// +build openbsd

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
	reservedBlocks := uint64(s.F_bfree) - uint64(s.F_bavail)
	info = Info{
		Total:  uint64(s.F_bsize) * (uint64(s.F_blocks) - reservedBlocks),
		Free:   uint64(s.F_bsize) * uint64(s.F_bavail),
		Files:  uint64(s.F_files),
		Ffree:  uint64(s.F_ffree),
		FSType: getFSType(s.F_fstypename[:]),
	}
	return info, nil
}
