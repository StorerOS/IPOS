// +build linux

package disk

import (
	"fmt"
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
		Total:  uint64(s.Frsize) * (uint64(s.Blocks) - reservedBlocks),
		Free:   uint64(s.Frsize) * uint64(s.Bavail),
		Files:  uint64(s.Files),
		Ffree:  uint64(s.Ffree),
		FSType: getFSType(int64(s.Type)),
	}
	if info.Free > info.Total {
		return info, fmt.Errorf("detected free space (%d) > total disk space (%d), fs corruption at (%s). please run 'fsck'", info.Free, info.Total, path)
	}
	return info, nil
}
