package crypto

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

const SSEHeader = "X-Amz-Server-Side-Encryption"

const (
	SSEKmsID = SSEHeader + "-Aws-Kms-Key-Id"

	SSEKmsContext = SSEHeader + "-Context"
)

const (
	SSECAlgorithm = SSEHeader + "-Customer-Algorithm"

	SSECKey = SSEHeader + "-Customer-Key"

	SSECKeyMD5 = SSEHeader + "-Customer-Key-Md5"
)

const (
	SSECopyAlgorithm = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Algorithm"

	SSECopyKey = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key"

	SSECopyKeyMD5 = "X-Amz-Copy-Source-Server-Side-Encryption-Customer-Key-Md5"
)

const (
	SSEAlgorithmAES256 = "AES256"

	SSEAlgorithmKMS = "aws:kms"
)

func RemoveSensitiveHeaders(h http.Header) {
	h.Del(SSECKey)
	h.Del(SSECopyKey)
}

func IsRequested(h http.Header) bool {
	return S3.IsRequested(h) || SSEC.IsRequested(h) || SSECopy.IsRequested(h) || S3KMS.IsRequested(h)
}

var S3 = s3{}

type s3 struct{}

func (s3) IsRequested(h http.Header) bool {
	_, ok := h[SSEHeader]
	return ok && strings.ToLower(h.Get(SSEHeader)) != SSEAlgorithmKMS
}

func (s3) ParseHTTP(h http.Header) (err error) {
	if h.Get(SSEHeader) != SSEAlgorithmAES256 {
		err = ErrInvalidEncryptionMethod
	}
	return
}

var S3KMS = s3KMS{}

type s3KMS struct{}

func (s3KMS) IsRequested(h http.Header) bool {
	if _, ok := h[SSEKmsID]; ok {
		return true
	}
	if _, ok := h[SSEKmsContext]; ok {
		return true
	}
	if _, ok := h[SSEHeader]; ok {
		return strings.ToUpper(h.Get(SSEHeader)) != SSEAlgorithmAES256
	}
	return false
}

func (s3KMS) ParseHTTP(h http.Header) (string, interface{}, error) {
	algorithm := h.Get(SSEHeader)
	if algorithm != SSEAlgorithmKMS {
		return "", nil, ErrInvalidEncryptionMethod
	}

	contextStr, ok := h[SSEKmsContext]
	if ok {
		var context map[string]interface{}
		if err := json.Unmarshal([]byte(contextStr[0]), &context); err != nil {
			return "", nil, err
		}
		return h.Get(SSEKmsID), context, nil
	}
	return h.Get(SSEKmsID), nil, nil
}

var (
	SSEC = ssec{}

	SSECopy = ssecCopy{}
)

type ssec struct{}
type ssecCopy struct{}

func (ssec) IsRequested(h http.Header) bool {
	if _, ok := h[SSECAlgorithm]; ok {
		return true
	}
	if _, ok := h[SSECKey]; ok {
		return true
	}
	if _, ok := h[SSECKeyMD5]; ok {
		return true
	}
	return false
}

func (ssecCopy) IsRequested(h http.Header) bool {
	if _, ok := h[SSECopyAlgorithm]; ok {
		return true
	}
	if _, ok := h[SSECopyKey]; ok {
		return true
	}
	if _, ok := h[SSECopyKeyMD5]; ok {
		return true
	}
	return false
}

func (ssec) ParseHTTP(h http.Header) (key [32]byte, err error) {
	if h.Get(SSECAlgorithm) != SSEAlgorithmAES256 {
		return key, ErrInvalidCustomerAlgorithm
	}
	if h.Get(SSECKey) == "" {
		return key, ErrMissingCustomerKey
	}
	if h.Get(SSECKeyMD5) == "" {
		return key, ErrMissingCustomerKeyMD5
	}

	clientKey, err := base64.StdEncoding.DecodeString(h.Get(SSECKey))
	if err != nil || len(clientKey) != 32 {
		return key, ErrInvalidCustomerKey
	}
	keyMD5, err := base64.StdEncoding.DecodeString(h.Get(SSECKeyMD5))
	if md5Sum := md5.Sum(clientKey); err != nil || !bytes.Equal(md5Sum[:], keyMD5) {
		return key, ErrCustomerKeyMD5Mismatch
	}
	copy(key[:], clientKey)
	return key, nil
}

func (ssecCopy) ParseHTTP(h http.Header) (key [32]byte, err error) {
	if h.Get(SSECopyAlgorithm) != SSEAlgorithmAES256 {
		return key, ErrInvalidCustomerAlgorithm
	}
	if h.Get(SSECopyKey) == "" {
		return key, ErrMissingCustomerKey
	}
	if h.Get(SSECopyKeyMD5) == "" {
		return key, ErrMissingCustomerKeyMD5
	}

	clientKey, err := base64.StdEncoding.DecodeString(h.Get(SSECopyKey))
	if err != nil || len(clientKey) != 32 {
		return key, ErrInvalidCustomerKey
	}
	keyMD5, err := base64.StdEncoding.DecodeString(h.Get(SSECopyKeyMD5))
	if md5Sum := md5.Sum(clientKey); err != nil || !bytes.Equal(md5Sum[:], keyMD5) {
		return key, ErrCustomerKeyMD5Mismatch
	}
	copy(key[:], clientKey)
	return key, nil
}
