package cmd

var errFileNotFound = StorageErr("file not found")

type StorageErr string

func (h StorageErr) Error() string {
	return string(h)
}
