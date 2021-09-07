// +build !windows

package ioutil

import (
	"io"
	"os"
)

func AppendFile(dst string, src string) error {
	appendFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0666)
	if err != nil {
		return err
	}
	defer appendFile.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	var buf = make([]byte, defaultAppendBufferSize)
	_, err = io.CopyBuffer(appendFile, srcFile, buf)
	return err
}
