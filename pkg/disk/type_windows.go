// +build windows

package disk

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	GetVolumeInformation = kernel32.NewProc("GetVolumeInformationW")
)

func getFSType(path string) string {
	var volumeNameSize uint32 = 260
	var nFileSystemNameSize, lpVolumeSerialNumber uint32
	var lpFileSystemFlags, lpMaximumComponentLength uint32
	var lpFileSystemNameBuffer, volumeName [260]byte
	var ps = syscall.StringToUTF16Ptr(filepath.VolumeName(path))

	_, _, _ = GetVolumeInformation.Call(uintptr(unsafe.Pointer(ps)),
		uintptr(unsafe.Pointer(&volumeName)),
		uintptr(volumeNameSize),
		uintptr(unsafe.Pointer(&lpVolumeSerialNumber)),
		uintptr(unsafe.Pointer(&lpMaximumComponentLength)),
		uintptr(unsafe.Pointer(&lpFileSystemFlags)),
		uintptr(unsafe.Pointer(&lpFileSystemNameBuffer)),
		uintptr(unsafe.Pointer(&nFileSystemNameSize)), 0)
	var bytes []byte
	if lpFileSystemNameBuffer[6] == 0 {
		bytes = []byte{lpFileSystemNameBuffer[0], lpFileSystemNameBuffer[2],
			lpFileSystemNameBuffer[4]}
	} else {
		bytes = []byte{lpFileSystemNameBuffer[0], lpFileSystemNameBuffer[2],
			lpFileSystemNameBuffer[4], lpFileSystemNameBuffer[6]}
	}

	return string(bytes)
}
