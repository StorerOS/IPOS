package dir

import (
	"errors"
	"os"
	"path/filepath"
)

func Writable(path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}
	if f, err := os.Create(filepath.Join(path, "._check_writable")); err == nil {
		f.Close()
		os.Remove(f.Name())
	} else {
		return errors.New("'" + path + "' is not writable")
	}
	return nil
}
