package ioutil

import (
	"io"
	"os"

	"github.com/storeros/ipos/pkg/lock"
)

func AppendFile(dst string, src string) error {
	appendFile, err := lock.Open(dst, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer appendFile.Close()

	srcFile, err := lock.Open(src, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	var buf = make([]byte, defaultAppendBufferSize)
	_, err = io.CopyBuffer(appendFile, srcFile, buf)
	return err
}
