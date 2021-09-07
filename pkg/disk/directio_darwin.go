package disk

import (
	"os"

	"github.com/ncw/directio"
	"golang.org/x/sys/unix"
)

func OpenFileDirectIO(filePath string, flag int, perm os.FileMode) (*os.File, error) {
	return directio.OpenFile(filePath, flag, perm)
}

func DisableDirectIO(f *os.File) error {
	fd := f.Fd()
	_, err := unix.FcntlInt(fd, unix.F_NOCACHE, 0)
	return err
}

func AlignedBlock(BlockSize int) []byte {
	return directio.AlignedBlock(BlockSize)
}
