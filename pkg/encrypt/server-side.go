package encrypt

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/crypto/argon2"
)

const (
	sseGenericHeader = "X-Amz-Server-Side-Encryption"

	sseKmsKeyID = sseGenericHeader + "-Aws-Kms-Key-Id"

	sseEncryptionContext = sseGenericHeader + "-Encryption-Context"

	sseCustomerAlgorithm = sseGenericHeader + "-Customer-Algorithm"

	sseCustomerKey = sseGenericHeader + "-Customer-Key"

	sseCustomerKeyMD5 = sseGenericHeader + "-Customer-Key-MD5"

	sseCopyCustomerAlgorithm = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Algorithm"

	sseCopyCustomerKey = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key"

	sseCopyCustomerKeyMD5 = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key-MD5"
)

type PBKDF func(password, salt []byte) ServerSide

var DefaultPBKDF PBKDF = func(password, salt []byte) ServerSide {
	sse := ssec{}
	copy(sse[:], argon2.IDKey(password, salt, 1, 64*1024, 4, 32))
	return sse
}

type Type string

const (
	SSEC Type = "SSE-C"

	KMS Type = "KMS"

	S3 Type = "S3"
)

type ServerSide interface {
	Type() Type

	Marshal(h http.Header)
}

func NewSSE() ServerSide { return s3{} }

func NewSSEKMS(keyID string, context interface{}) (ServerSide, error) {
	if context == nil {
		return kms{key: keyID, hasContext: false}, nil
	}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	serializedContext, err := json.Marshal(context)
	if err != nil {
		return nil, err
	}
	return kms{key: keyID, context: serializedContext, hasContext: true}, nil
}

func NewSSEC(key []byte) (ServerSide, error) {
	if len(key) != 32 {
		return nil, errors.New("encrypt: SSE-C key must be 256 bit long")
	}
	sse := ssec{}
	copy(sse[:], key)
	return sse, nil
}

func SSE(sse ServerSide) ServerSide {
	if sse == nil || sse.Type() != SSEC {
		return sse
	}
	if sse, ok := sse.(ssecCopy); ok {
		return ssec(sse)
	}
	return sse
}

func SSECopy(sse ServerSide) ServerSide {
	if sse == nil || sse.Type() != SSEC {
		return sse
	}
	if sse, ok := sse.(ssec); ok {
		return ssecCopy(sse)
	}
	return sse
}

type ssec [32]byte

func (s ssec) Type() Type { return SSEC }

func (s ssec) Marshal(h http.Header) {
	keyMD5 := md5.Sum(s[:])
	h.Set(sseCustomerAlgorithm, "AES256")
	h.Set(sseCustomerKey, base64.StdEncoding.EncodeToString(s[:]))
	h.Set(sseCustomerKeyMD5, base64.StdEncoding.EncodeToString(keyMD5[:]))
}

type ssecCopy [32]byte

func (s ssecCopy) Type() Type { return SSEC }

func (s ssecCopy) Marshal(h http.Header) {
	keyMD5 := md5.Sum(s[:])
	h.Set(sseCopyCustomerAlgorithm, "AES256")
	h.Set(sseCopyCustomerKey, base64.StdEncoding.EncodeToString(s[:]))
	h.Set(sseCopyCustomerKeyMD5, base64.StdEncoding.EncodeToString(keyMD5[:]))
}

type s3 struct{}

func (s s3) Type() Type { return S3 }

func (s s3) Marshal(h http.Header) { h.Set(sseGenericHeader, "AES256") }

type kms struct {
	key        string
	context    []byte
	hasContext bool
}

func (s kms) Type() Type { return KMS }

func (s kms) Marshal(h http.Header) {
	h.Set(sseGenericHeader, "aws:kms")
	if s.key != "" {
		h.Set(sseKmsKeyID, s.key)
	}
	if s.hasContext {
		h.Set(sseEncryptionContext, base64.StdEncoding.EncodeToString(s.context))
	}
}
