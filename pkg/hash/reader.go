package hash

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
	"io"

	sha256 "github.com/storeros/ipos/pkg/sha256-simd"
)

var errNestedReader = errors.New("Nesting of Reader detected, not allowed")

type Reader struct {
	src        io.Reader
	size       int64
	actualSize int64

	md5sum, sha256sum   []byte
	md5Hash, sha256Hash hash.Hash
}

func NewReader(src io.Reader, size int64, md5Hex, sha256Hex string, actualSize int64, strictCompat bool) (*Reader, error) {
	if _, ok := src.(*Reader); ok {
		return nil, errNestedReader
	}

	sha256sum, err := hex.DecodeString(sha256Hex)
	if err != nil {
		return nil, SHA256Mismatch{}
	}

	md5sum, err := hex.DecodeString(md5Hex)
	if err != nil {
		return nil, BadDigest{}
	}

	var sha256Hash hash.Hash
	if len(sha256sum) != 0 {
		sha256Hash = sha256.New()
	}
	var md5Hash hash.Hash
	if strictCompat {
		md5Hash = md5.New()
	} else if len(md5sum) != 0 {
		md5Hash = md5.New()
	}
	if size >= 0 {
		src = io.LimitReader(src, size)
	}
	return &Reader{
		md5sum:     md5sum,
		sha256sum:  sha256sum,
		src:        src,
		size:       size,
		md5Hash:    md5Hash,
		sha256Hash: sha256Hash,
		actualSize: actualSize,
	}, nil
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.src.Read(p)
	if n > 0 {
		if r.md5Hash != nil {
			r.md5Hash.Write(p[:n])
		}
		if r.sha256Hash != nil {
			r.sha256Hash.Write(p[:n])
		}
	}

	if err == io.EOF {
		if cerr := r.Verify(); cerr != nil {
			return 0, cerr
		}
	}

	return
}

func (r *Reader) Size() int64 { return r.size }

func (r *Reader) ActualSize() int64 { return r.actualSize }

func (r *Reader) MD5() []byte {
	return r.md5sum
}

func (r *Reader) MD5Current() []byte {
	if r.md5Hash != nil {
		return r.md5Hash.Sum(nil)
	}
	return nil
}

func (r *Reader) SHA256() []byte {
	return r.sha256sum
}

func (r *Reader) MD5HexString() string {
	return hex.EncodeToString(r.md5sum)
}

func (r *Reader) MD5Base64String() string {
	return base64.StdEncoding.EncodeToString(r.md5sum)
}

func (r *Reader) SHA256HexString() string {
	return hex.EncodeToString(r.sha256sum)
}

func (r *Reader) Verify() error {
	if r.sha256Hash != nil && len(r.sha256sum) > 0 {
		if sum := r.sha256Hash.Sum(nil); !bytes.Equal(r.sha256sum, sum) {
			return SHA256Mismatch{hex.EncodeToString(r.sha256sum), hex.EncodeToString(sum)}
		}
	}
	if r.md5Hash != nil && len(r.md5sum) > 0 {
		if sum := r.md5Hash.Sum(nil); !bytes.Equal(r.md5sum, sum) {
			return BadDigest{hex.EncodeToString(r.md5sum), hex.EncodeToString(sum)}
		}
	}
	return nil
}
