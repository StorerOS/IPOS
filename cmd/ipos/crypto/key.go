package crypto

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"path"

	"github.com/storeros/ipos/cmd/ipos/logger"
	sha256 "github.com/storeros/ipos/pkg/sha256-simd"
	"github.com/storeros/ipos/pkg/sio"
)

type ObjectKey [32]byte

func GenerateKey(extKey [32]byte, random io.Reader) (key ObjectKey) {
	if random == nil {
		random = rand.Reader
	}
	var nonce [32]byte
	if _, err := io.ReadFull(random, nonce[:]); err != nil {
		logger.CriticalIf(context.Background(), errOutOfEntropy)
	}
	sha := sha256.New()
	sha.Write(extKey[:])
	sha.Write(nonce[:])
	sha.Sum(key[:0])
	return key
}

func GenerateIV(random io.Reader) (iv [32]byte) {
	if random == nil {
		random = rand.Reader
	}
	if _, err := io.ReadFull(random, iv[:]); err != nil {
		logger.CriticalIf(context.Background(), errOutOfEntropy)
	}
	return iv
}

type SealedKey struct {
	Key       [64]byte
	IV        [32]byte
	Algorithm string
}

func (key ObjectKey) Seal(extKey, iv [32]byte, domain, bucket, object string) SealedKey {
	var (
		sealingKey   [32]byte
		encryptedKey bytes.Buffer
	)
	mac := hmac.New(sha256.New, extKey[:])
	mac.Write(iv[:])
	mac.Write([]byte(domain))
	mac.Write([]byte(SealAlgorithm))
	mac.Write([]byte(path.Join(bucket, object)))
	mac.Sum(sealingKey[:0])
	if n, err := sio.Encrypt(&encryptedKey, bytes.NewReader(key[:]), sio.Config{Key: sealingKey[:]}); n != 64 || err != nil {
		logger.CriticalIf(context.Background(), errors.New("Unable to generate sealed key"))
	}
	sealedKey := SealedKey{
		IV:        iv,
		Algorithm: SealAlgorithm,
	}
	copy(sealedKey.Key[:], encryptedKey.Bytes())
	return sealedKey
}

func (key *ObjectKey) Unseal(extKey [32]byte, sealedKey SealedKey, domain, bucket, object string) error {
	var (
		unsealConfig sio.Config
		decryptedKey bytes.Buffer
	)
	switch sealedKey.Algorithm {
	default:
		return Errorf("The sealing algorithm '%s' is not supported", sealedKey.Algorithm)
	case SealAlgorithm:
		mac := hmac.New(sha256.New, extKey[:])
		mac.Write(sealedKey.IV[:])
		mac.Write([]byte(domain))
		mac.Write([]byte(SealAlgorithm))
		mac.Write([]byte(path.Join(bucket, object)))
		unsealConfig = sio.Config{MinVersion: sio.Version20, Key: mac.Sum(nil)}
	case InsecureSealAlgorithm:
		sha := sha256.New()
		sha.Write(extKey[:])
		sha.Write(sealedKey.IV[:])
		unsealConfig = sio.Config{MinVersion: sio.Version10, Key: sha.Sum(nil)}
	}

	if n, err := sio.Decrypt(&decryptedKey, bytes.NewReader(sealedKey.Key[:]), unsealConfig); n != 32 || err != nil {
		return ErrSecretKeyMismatch
	}
	copy(key[:], decryptedKey.Bytes())
	return nil
}

func (key ObjectKey) DerivePartKey(id uint32) (partKey [32]byte) {
	var bin [4]byte
	binary.LittleEndian.PutUint32(bin[:], id)

	mac := hmac.New(sha256.New, key[:])
	mac.Write(bin[:])
	mac.Sum(partKey[:0])
	return partKey
}

func (key ObjectKey) SealETag(etag []byte) []byte {
	if len(etag) == 0 {
		return etag
	}
	var buffer bytes.Buffer
	mac := hmac.New(sha256.New, key[:])
	mac.Write([]byte("SSE-etag"))
	if _, err := sio.Encrypt(&buffer, bytes.NewReader(etag), sio.Config{Key: mac.Sum(nil)}); err != nil {
		logger.CriticalIf(context.Background(), errors.New("Unable to encrypt ETag using object key"))
	}
	return buffer.Bytes()
}

func (key ObjectKey) UnsealETag(etag []byte) ([]byte, error) {
	if !IsETagSealed(etag) {
		return etag, nil
	}
	var buffer bytes.Buffer
	mac := hmac.New(sha256.New, key[:])
	mac.Write([]byte("SSE-etag"))
	if _, err := sio.Decrypt(&buffer, bytes.NewReader(etag), sio.Config{Key: mac.Sum(nil)}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
