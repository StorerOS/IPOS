// +build !windows

package disk

import (
	"os"
	"syscall"
)

func IsRootDisk(diskPath string) (bool, error) {
	rootDisk := false
	diskInfo, err := os.Stat(diskPath)
	if err != nil {
		return false, err
	}
	rootInfo, err := os.Stat("/")
	if err != nil {
		return false, err
	}
	diskStat, diskStatOK := diskInfo.Sys().(*syscall.Stat_t)
	rootStat, rootStatOK := rootInfo.Sys().(*syscall.Stat_t)
	if diskStatOK && rootStatOK {
		if diskStat.Dev == rootStat.Dev {
			rootDisk = true
		}
	}
	return rootDisk, nil
}
