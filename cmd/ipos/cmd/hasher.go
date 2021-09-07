package cmd

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/storeros/ipos/pkg/sha256-simd"
)

func getSHA256Hash(data []byte) string {
	return hex.EncodeToString(getSHA256Sum(data))
}

func getSHA256Sum(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

func getMD5Sum(data []byte) []byte {
	hash := md5.New()
	hash.Write(data)
	return hash.Sum(nil)
}

func getMD5Hash(data []byte) string {
	return hex.EncodeToString(getMD5Sum(data))
}
