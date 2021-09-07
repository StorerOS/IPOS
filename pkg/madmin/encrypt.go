package madmin

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"

	"github.com/secure-io/sio-go"
	"github.com/secure-io/sio-go/sioutil"
	"golang.org/x/crypto/argon2"
)

func EncryptData(password string, data []byte) ([]byte, error) {
	salt := sioutil.MustRandom(32)

	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	var (
		id     byte
		err    error
		stream *sio.Stream
	)
	if sioutil.NativeAES() {
		id = aesGcm
		stream, err = sio.AES_256_GCM.Stream(key)
	} else {
		id = c20p1305
		stream, err = sio.ChaCha20Poly1305.Stream(key)
	}
	if err != nil {
		return nil, err
	}
	nonce := sioutil.MustRandom(stream.NonceSize())

	cLen := int64(len(salt)+1+len(nonce)+len(data)) + stream.Overhead(int64(len(data)))
	ciphertext := bytes.NewBuffer(make([]byte, 0, cLen))

	ciphertext.Write(salt)
	ciphertext.WriteByte(id)
	ciphertext.Write(nonce)

	w := stream.EncryptWriter(ciphertext, nonce, nil)
	if _, err = w.Write(data); err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	return ciphertext.Bytes(), nil
}

var ErrMaliciousData = sio.NotAuthentic

func DecryptData(password string, data io.Reader) ([]byte, error) {
	var (
		salt  [32]byte
		id    [1]byte
		nonce [8]byte
	)

	if _, err := io.ReadFull(data, salt[:]); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(data, id[:]); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(data, nonce[:]); err != nil {
		return nil, err
	}

	key := argon2.IDKey([]byte(password), salt[:], 1, 64*1024, 4, 32)
	var (
		err    error
		stream *sio.Stream
	)
	switch id[0] {
	case aesGcm:
		stream, err = sio.AES_256_GCM.Stream(key)
	case c20p1305:
		stream, err = sio.ChaCha20Poly1305.Stream(key)
	default:
		err = errors.New("madmin: invalid AEAD algorithm ID")
	}
	if err != nil {
		return nil, err
	}

	enBytes, err := ioutil.ReadAll(stream.DecryptReader(data, nonce[:], nil))
	if err != nil {
		if err == sio.NotAuthentic {
			return enBytes, ErrMaliciousData
		}
	}
	return enBytes, err
}

const (
	aesGcm   = 0x00
	c20p1305 = 0x01
)
