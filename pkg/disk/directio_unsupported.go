// +build !linux,!netbsd,!freebsd,!darwin

package disk

import (
	"os"
)

func OpenFileDirectIO(filePath string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(filePath, flag, perm)
}

func DisableDirectIO(f *os.File) error {
	return nil
}

func AlignedBlock(BlockSize int) []byte {
	return make([]byte, BlockSize)
}
