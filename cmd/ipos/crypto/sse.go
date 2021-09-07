package crypto

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/ioutil"
	"github.com/storeros/ipos/pkg/sio"
)

const (
	SSEMultipart = "X-IPOS-Internal-Encrypted-Multipart"

	SSEIV = "X-IPOS-Internal-Server-Side-Encryption-Iv"

	SSESealAlgorithm = "X-IPOS-Internal-Server-Side-Encryption-Seal-Algorithm"

	SSECSealedKey = "X-IPOS-Internal-Server-Side-Encryption-Sealed-Key"

	S3SealedKey = "X-IPOS-Internal-Server-Side-Encryption-S3-Sealed-Key"

	S3KMSKeyID = "X-IPOS-Internal-Server-Side-Encryption-S3-Kms-Key-Id"

	S3KMSSealedKey = "X-IPOS-Internal-Server-Side-Encryption-S3-Kms-Sealed-Key"
)

const (
	SealAlgorithm = "DAREv2-HMAC-SHA256"

	InsecureSealAlgorithm = "DARE-SHA256"
)

func (s3) String() string { return "SSE-S3" }

func (sse s3) UnsealObjectKey(kms KMS, metadata map[string]string, bucket, object string) (key ObjectKey, err error) {
	keyID, kmsKey, sealedKey, err := sse.ParseMetadata(metadata)
	if err != nil {
		return
	}
	unsealKey, err := kms.UnsealKey(keyID, kmsKey, Context{bucket: path.Join(bucket, object)})
	if err != nil {
		return
	}
	err = key.Unseal(unsealKey, sealedKey, sse.String(), bucket, object)
	return
}

func (ssec) String() string { return "SSE-C" }

func (sse ssec) UnsealObjectKey(h http.Header, metadata map[string]string, bucket, object string) (key ObjectKey, err error) {
	clientKey, err := sse.ParseHTTP(h)
	if err != nil {
		return
	}
	return unsealObjectKey(clientKey, metadata, bucket, object)
}

func (sse ssecCopy) UnsealObjectKey(h http.Header, metadata map[string]string, bucket, object string) (key ObjectKey, err error) {
	clientKey, err := sse.ParseHTTP(h)
	if err != nil {
		return
	}
	return unsealObjectKey(clientKey, metadata, bucket, object)
}

func unsealObjectKey(clientKey [32]byte, metadata map[string]string, bucket, object string) (key ObjectKey, err error) {
	sealedKey, err := SSEC.ParseMetadata(metadata)
	if err != nil {
		return
	}
	err = key.Unseal(clientKey, sealedKey, SSEC.String(), bucket, object)
	return
}

func EncryptSinglePart(r io.Reader, key ObjectKey) io.Reader {
	r, err := sio.EncryptReader(r, sio.Config{MinVersion: sio.Version20, Key: key[:]})
	if err != nil {
		logger.CriticalIf(context.Background(), errors.New("Unable to encrypt io.Reader using object key"))
	}
	return r
}

func EncryptMultiPart(r io.Reader, partID int, key ObjectKey) io.Reader {
	partKey := key.DerivePartKey(uint32(partID))
	return EncryptSinglePart(r, ObjectKey(partKey))
}

func DecryptSinglePart(w io.Writer, offset, length int64, key ObjectKey) io.WriteCloser {
	const PayloadSize = 1 << 16
	w = ioutil.LimitedWriter(w, offset%PayloadSize, length)

	decWriter, err := sio.DecryptWriter(w, sio.Config{Key: key[:]})
	if err != nil {
		logger.CriticalIf(context.Background(), errors.New("Unable to decrypt io.Writer using object key"))
	}
	return decWriter
}
