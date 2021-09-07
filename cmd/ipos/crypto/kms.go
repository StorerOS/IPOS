package crypto

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/storeros/ipos/cmd/ipos/logger"
	sha256 "github.com/storeros/ipos/pkg/sha256-simd"
	"github.com/storeros/ipos/pkg/sio"
)

type Context map[string]string

func (c Context) WriteTo(w io.Writer) (n int64, err error) {
	sortedKeys := make(sort.StringSlice, 0, len(c))
	for k := range c {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Sort(sortedKeys)

	nn, err := io.WriteString(w, "{")
	if err != nil {
		return n + int64(nn), err
	}
	n += int64(nn)
	for i, k := range sortedKeys {
		s := fmt.Sprintf("\"%s\":\"%s\",", k, c[k])
		if i == len(sortedKeys)-1 {
			s = s[:len(s)-1]
		}

		nn, err = io.WriteString(w, s)
		if err != nil {
			return n + int64(nn), err
		}
		n += int64(nn)
	}
	nn, err = io.WriteString(w, "}")
	return n + int64(nn), err
}

type KMS interface {
	KeyID() string

	GenerateKey(keyID string, context Context) (key [32]byte, sealedKey []byte, err error)

	UnsealKey(keyID string, sealedKey []byte, context Context) (key [32]byte, err error)

	UpdateKey(keyID string, sealedKey []byte, context Context) (rotatedKey []byte, err error)

	Info() (kmsInfo KMSInfo)
}

type masterKeyKMS struct {
	keyID     string
	masterKey [32]byte
}

type KMSInfo struct {
	Endpoint string
	Name     string
	AuthType string
}

func NewMasterKey(keyID string, key [32]byte) KMS { return &masterKeyKMS{keyID: keyID, masterKey: key} }

func (kms *masterKeyKMS) KeyID() string {
	return kms.keyID
}

func (kms *masterKeyKMS) GenerateKey(keyID string, ctx Context) (key [32]byte, sealedKey []byte, err error) {
	if _, err = io.ReadFull(rand.Reader, key[:]); err != nil {
		logger.CriticalIf(context.Background(), errOutOfEntropy)
	}

	var (
		buffer     bytes.Buffer
		derivedKey = kms.deriveKey(keyID, ctx)
	)
	if n, err := sio.Encrypt(&buffer, bytes.NewReader(key[:]), sio.Config{Key: derivedKey[:]}); err != nil || n != 64 {
		logger.CriticalIf(context.Background(), errors.New("KMS: unable to encrypt data key"))
	}
	sealedKey = buffer.Bytes()
	return key, sealedKey, nil
}

func (kms *masterKeyKMS) Info() (info KMSInfo) {
	return KMSInfo{
		Endpoint: "",
		Name:     "",
		AuthType: "master-key",
	}
}

func (kms *masterKeyKMS) UnsealKey(keyID string, sealedKey []byte, ctx Context) (key [32]byte, err error) {
	var (
		buffer     bytes.Buffer
		derivedKey = kms.deriveKey(keyID, ctx)
	)
	if n, err := sio.Decrypt(&buffer, bytes.NewReader(sealedKey), sio.Config{Key: derivedKey[:]}); err != nil || n != 32 {
		return key, err
	}
	copy(key[:], buffer.Bytes())
	return key, nil
}

func (kms *masterKeyKMS) UpdateKey(keyID string, sealedKey []byte, ctx Context) ([]byte, error) {
	if _, err := kms.UnsealKey(keyID, sealedKey, ctx); err != nil {
		return nil, err
	}
	return sealedKey, nil
}

func (kms *masterKeyKMS) deriveKey(keyID string, context Context) (key [32]byte) {
	if context == nil {
		context = Context{}
	}
	mac := hmac.New(sha256.New, kms.masterKey[:])
	mac.Write([]byte(keyID))
	context.WriteTo(mac)
	mac.Sum(key[:0])
	return key
}
