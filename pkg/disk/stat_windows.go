// +build windows

package disk

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	GetDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
	GetDiskFreeSpace   = kernel32.NewProc("GetDiskFreeSpaceW")
)

func GetInfo(path string) (info Info, err error) {
	if _, err = os.Stat(path); err != nil {
		return Info{}, err
	}

	lpFreeBytesAvailable := int64(0)
	lpTotalNumberOfBytes := int64(0)
	lpTotalNumberOfFreeBytes := int64(0)

	_, _, _ = GetDiskFreeSpaceEx.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&lpFreeBytesAvailable)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfBytes)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfFreeBytes)))
	info = Info{}
	info.Total = uint64(lpTotalNumberOfBytes)
	info.Free = uint64(lpFreeBytesAvailable)
	info.FSType = getFSType(path)

	lpSectorsPerCluster := uint32(0)
	lpBytesPerSector := uint32(0)
	lpNumberOfFreeClusters := uint32(0)
	lpTotalNumberOfClusters := uint32(0)

	_, _, _ = GetDiskFreeSpace.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&lpSectorsPerCluster)),
		uintptr(unsafe.Pointer(&lpBytesPerSector)),
		uintptr(unsafe.Pointer(&lpNumberOfFreeClusters)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfClusters)))

	info.Files = uint64(lpTotalNumberOfClusters)
	info.Ffree = uint64(lpNumberOfFreeClusters)

	return info, nil
}
