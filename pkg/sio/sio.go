package sio

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"runtime"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/sys/cpu"
)

const (
	Version20 byte = 0x20
	Version10 byte = 0x10
)

const (
	AES_256_GCM byte = iota
	CHACHA20_POLY1305
)

var supportsAES = (cpu.X86.HasAES && cpu.X86.HasPCLMULQDQ) || runtime.GOARCH == "s390x"

const (
	keySize = 32

	headerSize     = 16
	maxPayloadSize = 1 << 16
	tagSize        = 16
	maxPackageSize = headerSize + maxPayloadSize + tagSize

	maxDecryptedSize = 1 << 48
	maxEncryptedSize = maxDecryptedSize + ((headerSize + tagSize) * 1 << 32)
)

var newAesGcm = func(key []byte) (cipher.AEAD, error) {
	aes256, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(aes256)
}

var supportedCiphers = [...]func([]byte) (cipher.AEAD, error){
	AES_256_GCM:       newAesGcm,
	CHACHA20_POLY1305: chacha20poly1305.New,
}

var (
	errUnsupportedVersion = Error{"sio: unsupported version"}
	errUnsupportedCipher  = Error{"sio: unsupported cipher suite"}
	errInvalidPayloadSize = Error{"sio: invalid payload size"}
	errTagMismatch        = Error{"sio: authentication failed"}
	errUnexpectedSize     = Error{"sio: size is too large for DARE"}

	errPackageOutOfOrder = Error{"sio: sequence number mismatch"}

	errNonceMismatch  = Error{"sio: header nonce mismatch"}
	errUnexpectedEOF  = Error{"sio: unexpected EOF"}
	errUnexpectedData = Error{"sio: unexpected data after final package"}
)

type Error struct{ msg string }

func (e Error) Error() string { return e.msg }

type Config struct {
	MinVersion byte

	MaxVersion byte

	CipherSuites []byte

	Key []byte

	SequenceNumber uint32

	Rand io.Reader

	PayloadSize int
}

func EncryptedSize(size uint64) (uint64, error) {
	if size > maxDecryptedSize {
		return 0, errUnexpectedSize
	}

	encSize := (size / maxPayloadSize) * maxPackageSize
	if mod := size % maxPayloadSize; mod > 0 {
		encSize += mod + (headerSize + tagSize)
	}
	return encSize, nil
}

func DecryptedSize(size uint64) (uint64, error) {
	if size > maxEncryptedSize {
		return 0, errUnexpectedSize
	}
	decSize := (size / maxPackageSize) * maxPayloadSize
	if mod := size % maxPackageSize; mod > 0 {
		if mod <= headerSize+tagSize {
			return 0, errors.New("sio: size is not valid")
		}
		decSize += mod - (headerSize + tagSize)
	}
	return decSize, nil
}

func Encrypt(dst io.Writer, src io.Reader, config Config) (n int64, err error) {
	encReader, err := EncryptReader(src, config)
	if err != nil {
		return 0, err
	}
	return io.CopyBuffer(dst, encReader, make([]byte, headerSize+maxPayloadSize+tagSize))
}

func Decrypt(dst io.Writer, src io.Reader, config Config) (n int64, err error) {
	decReader, err := DecryptReader(src, config)
	if err != nil {
		return 0, err
	}
	return io.CopyBuffer(dst, decReader, make([]byte, maxPayloadSize))
}

func EncryptReader(src io.Reader, config Config) (io.Reader, error) {
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	if config.MaxVersion == Version20 {
		return encryptReaderV20(src, &config)
	}
	return encryptReaderV10(src, &config)
}

func DecryptReader(src io.Reader, config Config) (io.Reader, error) {
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	if config.MinVersion == Version10 && config.MaxVersion == Version10 {
		return decryptReaderV10(src, &config)
	}
	if config.MinVersion == Version20 && config.MaxVersion == Version20 {
		return decryptReaderV20(src, &config)
	}
	return decryptReader(src, &config), nil
}

func DecryptReaderAt(src io.ReaderAt, config Config) (io.ReaderAt, error) {
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	if config.MinVersion == Version10 && config.MaxVersion == Version10 {
		return decryptReaderAtV10(src, &config)
	}
	if config.MinVersion == Version20 && config.MaxVersion == Version20 {
		return decryptReaderAtV20(src, &config)
	}
	return decryptReaderAt(src, &config), nil
}

func EncryptWriter(dst io.Writer, config Config) (io.WriteCloser, error) {
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	if config.MaxVersion == Version20 {
		return encryptWriterV20(dst, &config)
	}
	return encryptWriterV10(dst, &config)
}

func DecryptWriter(dst io.Writer, config Config) (io.WriteCloser, error) {
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	if config.MinVersion == Version10 && config.MaxVersion == Version10 {
		return decryptWriterV10(dst, &config)
	}
	if config.MinVersion == Version20 && config.MaxVersion == Version20 {
		return decryptWriterV20(dst, &config)
	}
	return decryptWriter(dst, &config), nil
}

func defaultCipherSuites() []byte {
	if supportsAES {
		return []byte{AES_256_GCM, CHACHA20_POLY1305}
	}
	return []byte{CHACHA20_POLY1305, AES_256_GCM}
}

func setConfigDefaults(config *Config) error {
	if config.MinVersion > Version20 {
		return errors.New("sio: unknown minimum version")
	}
	if config.MaxVersion > Version20 {
		return errors.New("sio: unknown maximum version")
	}
	if len(config.Key) != keySize {
		return errors.New("sio: invalid key size")
	}
	if len(config.CipherSuites) > 2 {
		return errors.New("sio: too many cipher suites")
	}
	for _, c := range config.CipherSuites {
		if int(c) >= len(supportedCiphers) {
			return errors.New("sio: unknown cipher suite")
		}
	}
	if config.PayloadSize > maxPayloadSize {
		return errors.New("sio: payload size is too large")
	}

	if config.MinVersion < Version10 {
		config.MinVersion = Version10
	}
	if config.MaxVersion < Version10 {
		config.MaxVersion = Version20
	}
	if config.MinVersion > config.MaxVersion {
		return errors.New("sio: minimum version cannot be larger than maximum version")
	}
	if len(config.CipherSuites) == 0 {
		config.CipherSuites = defaultCipherSuites()
	}
	if config.Rand == nil {
		config.Rand = rand.Reader
	}
	if config.PayloadSize == 0 {
		config.PayloadSize = maxPayloadSize
	}
	return nil
}
