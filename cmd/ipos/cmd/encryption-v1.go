package cmd

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/storeros/ipos/cmd/ipos/crypto"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/encrypt"
	sha256 "github.com/storeros/ipos/pkg/sha256-simd"
	"github.com/storeros/ipos/pkg/sio"
)

var (
	errEncryptedObject             = errors.New("The object was stored using a form of SSE")
	errInvalidSSEParameters        = errors.New("The SSE-C key for key-rotation is not correct")
	errKMSNotConfigured            = errors.New("KMS not configured for a server side encrypted object")
	errObjectTampered              = errors.New("The requested object was modified and may be compromised")
	errInvalidEncryptionParameters = errors.New("The encryption parameters are not applicable to this object")
)

const (
	SSECustomerKeySize = 32

	SSEIVSize = 32

	SSEDAREPackageBlockSize = 64 * 1024

	SSEDAREPackageMetaSize = 32
)

func ParseSSECopyCustomerRequest(h http.Header, metadata map[string]string) (key []byte, err error) {
	if crypto.S3.IsEncrypted(metadata) && crypto.SSECopy.IsRequested(h) {
		return nil, crypto.ErrIncompatibleEncryptionMethod
	}
	k, err := crypto.SSECopy.ParseHTTP(h)
	return k[:], err
}

func ParseSSECustomerRequest(r *http.Request) (key []byte, err error) {
	return ParseSSECustomerHeader(r.Header)
}

func ParseSSECustomerHeader(header http.Header) (key []byte, err error) {
	if crypto.S3.IsRequested(header) && crypto.SSEC.IsRequested(header) {
		return key, crypto.ErrIncompatibleEncryptionMethod
	}

	k, err := crypto.SSEC.ParseHTTP(header)
	return k[:], err
}

func rotateKey(oldKey []byte, newKey []byte, bucket, object string, metadata map[string]string) error {
	switch {
	default:
		return errObjectTampered
	case crypto.SSEC.IsEncrypted(metadata):
		sealedKey, err := crypto.SSEC.ParseMetadata(metadata)
		if err != nil {
			return err
		}

		var objectKey crypto.ObjectKey
		var extKey [32]byte
		copy(extKey[:], oldKey)
		if err = objectKey.Unseal(extKey, sealedKey, crypto.SSEC.String(), bucket, object); err != nil {
			if subtle.ConstantTimeCompare(oldKey, newKey) == 1 {
				return errInvalidSSEParameters
			}
			return crypto.ErrInvalidCustomerKey

		}
		if subtle.ConstantTimeCompare(oldKey, newKey) == 1 && sealedKey.Algorithm == crypto.SealAlgorithm {
			return nil
		}
		copy(extKey[:], newKey)
		sealedKey = objectKey.Seal(extKey, sealedKey.IV, crypto.SSEC.String(), bucket, object)
		crypto.SSEC.CreateMetadata(metadata, sealedKey)
		return nil
	case crypto.S3.IsEncrypted(metadata):
		if GlobalKMS == nil {
			return errKMSNotConfigured
		}
		keyID, kmsKey, sealedKey, err := crypto.S3.ParseMetadata(metadata)
		if err != nil {
			return err
		}
		oldKey, err := GlobalKMS.UnsealKey(keyID, kmsKey, crypto.Context{bucket: path.Join(bucket, object)})
		if err != nil {
			return err
		}
		var objectKey crypto.ObjectKey
		if err = objectKey.Unseal(oldKey, sealedKey, crypto.S3.String(), bucket, object); err != nil {
			return err
		}

		newKey, encKey, err := GlobalKMS.GenerateKey(GlobalKMS.KeyID(), crypto.Context{bucket: path.Join(bucket, object)})
		if err != nil {
			return err
		}
		sealedKey = objectKey.Seal(newKey, crypto.GenerateIV(rand.Reader), crypto.S3.String(), bucket, object)
		crypto.S3.CreateMetadata(metadata, GlobalKMS.KeyID(), encKey, sealedKey)
		return nil
	}
}

func newEncryptMetadata(key []byte, bucket, object string, metadata map[string]string, sseS3 bool) (crypto.ObjectKey, error) {
	var sealedKey crypto.SealedKey
	if sseS3 {
		if GlobalKMS == nil {
			return crypto.ObjectKey{}, errKMSNotConfigured
		}
		key, encKey, err := GlobalKMS.GenerateKey(GlobalKMS.KeyID(), crypto.Context{bucket: path.Join(bucket, object)})
		if err != nil {
			return crypto.ObjectKey{}, err
		}

		objectKey := crypto.GenerateKey(key, rand.Reader)
		sealedKey = objectKey.Seal(key, crypto.GenerateIV(rand.Reader), crypto.S3.String(), bucket, object)
		crypto.S3.CreateMetadata(metadata, GlobalKMS.KeyID(), encKey, sealedKey)
		return objectKey, nil
	}
	var extKey [32]byte
	copy(extKey[:], key)
	objectKey := crypto.GenerateKey(extKey, rand.Reader)
	sealedKey = objectKey.Seal(extKey, crypto.GenerateIV(rand.Reader), crypto.SSEC.String(), bucket, object)
	crypto.SSEC.CreateMetadata(metadata, sealedKey)
	return objectKey, nil
}

func newEncryptReader(content io.Reader, key []byte, bucket, object string, metadata map[string]string, sseS3 bool) (io.Reader, crypto.ObjectKey, error) {
	objectEncryptionKey, err := newEncryptMetadata(key, bucket, object, metadata, sseS3)
	if err != nil {
		return nil, crypto.ObjectKey{}, err
	}

	reader, err := sio.EncryptReader(content, sio.Config{Key: objectEncryptionKey[:], MinVersion: sio.Version20})
	if err != nil {
		return nil, crypto.ObjectKey{}, crypto.ErrInvalidCustomerKey
	}

	return reader, objectEncryptionKey, nil
}

func setEncryptionMetadata(r *http.Request, bucket, object string, metadata map[string]string) (err error) {
	var (
		key []byte
	)
	if crypto.SSEC.IsRequested(r.Header) {
		key, err = ParseSSECustomerRequest(r)
		if err != nil {
			return
		}
	}
	_, err = newEncryptMetadata(key, bucket, object, metadata, crypto.S3.IsRequested(r.Header))
	return
}

func EncryptRequest(content io.Reader, r *http.Request, bucket, object string, metadata map[string]string) (io.Reader, crypto.ObjectKey, error) {
	if crypto.S3.IsRequested(r.Header) && crypto.SSEC.IsRequested(r.Header) {
		return nil, crypto.ObjectKey{}, crypto.ErrIncompatibleEncryptionMethod
	}
	if r.ContentLength > encryptBufferThreshold {
		content = bufio.NewReaderSize(content, encryptBufferSize)
	}

	var key []byte
	if crypto.SSEC.IsRequested(r.Header) {
		var err error
		key, err = ParseSSECustomerRequest(r)
		if err != nil {
			return nil, crypto.ObjectKey{}, err
		}
	}
	return newEncryptReader(content, key, bucket, object, metadata, crypto.S3.IsRequested(r.Header))
}

func DecryptCopyRequest(client io.Writer, r *http.Request, bucket, object string, metadata map[string]string) (io.WriteCloser, error) {
	var (
		key []byte
		err error
	)
	if crypto.SSECopy.IsRequested(r.Header) {
		key, err = ParseSSECopyCustomerRequest(r.Header, metadata)
		if err != nil {
			return nil, err
		}
	}
	return newDecryptWriter(client, key, bucket, object, 0, metadata)
}

func decryptObjectInfo(key []byte, bucket, object string, metadata map[string]string) ([]byte, error) {
	switch {
	default:
		return nil, errObjectTampered
	case crypto.S3.IsEncrypted(metadata):
		if GlobalKMS == nil {
			return nil, errKMSNotConfigured
		}
		keyID, kmsKey, sealedKey, err := crypto.S3.ParseMetadata(metadata)

		if err != nil {
			return nil, err
		}
		extKey, err := GlobalKMS.UnsealKey(keyID, kmsKey, crypto.Context{bucket: path.Join(bucket, object)})
		if err != nil {
			return nil, err
		}
		var objectKey crypto.ObjectKey
		if err = objectKey.Unseal(extKey, sealedKey, crypto.S3.String(), bucket, object); err != nil {
			return nil, err
		}
		return objectKey[:], nil
	case crypto.SSEC.IsEncrypted(metadata):
		var extKey [32]byte
		copy(extKey[:], key)
		sealedKey, err := crypto.SSEC.ParseMetadata(metadata)
		if err != nil {
			return nil, err
		}
		var objectKey crypto.ObjectKey
		if err = objectKey.Unseal(extKey, sealedKey, crypto.SSEC.String(), bucket, object); err != nil {
			return nil, err
		}
		return objectKey[:], nil
	}
}

func newDecryptWriter(client io.Writer, key []byte, bucket, object string, seqNumber uint32, metadata map[string]string) (io.WriteCloser, error) {
	objectEncryptionKey, err := decryptObjectInfo(key, bucket, object, metadata)
	if err != nil {
		return nil, err
	}
	return newDecryptWriterWithObjectKey(client, objectEncryptionKey, seqNumber, metadata)
}

func newDecryptWriterWithObjectKey(client io.Writer, objectEncryptionKey []byte, seqNumber uint32, metadata map[string]string) (io.WriteCloser, error) {
	writer, err := sio.DecryptWriter(client, sio.Config{
		Key:            objectEncryptionKey,
		SequenceNumber: seqNumber,
	})
	if err != nil {
		return nil, crypto.ErrInvalidCustomerKey
	}
	delete(metadata, crypto.SSEIV)
	delete(metadata, crypto.SSESealAlgorithm)
	delete(metadata, crypto.SSECSealedKey)
	delete(metadata, crypto.SSEMultipart)
	delete(metadata, crypto.S3SealedKey)
	delete(metadata, crypto.S3KMSSealedKey)
	delete(metadata, crypto.S3KMSKeyID)
	return writer, nil
}

func DecryptRequestWithSequenceNumberR(client io.Reader, h http.Header, bucket, object string, seqNumber uint32, metadata map[string]string) (io.Reader, error) {
	if crypto.S3.IsEncrypted(metadata) {
		return newDecryptReader(client, nil, bucket, object, seqNumber, metadata)
	}

	key, err := ParseSSECustomerHeader(h)
	if err != nil {
		return nil, err
	}
	return newDecryptReader(client, key, bucket, object, seqNumber, metadata)
}

func DecryptCopyRequestR(client io.Reader, h http.Header, bucket, object string, seqNumber uint32, metadata map[string]string) (io.Reader, error) {
	var (
		key []byte
		err error
	)
	if crypto.SSECopy.IsRequested(h) {
		key, err = ParseSSECopyCustomerRequest(h, metadata)
		if err != nil {
			return nil, err
		}
	}
	return newDecryptReader(client, key, bucket, object, seqNumber, metadata)
}

func newDecryptReader(client io.Reader, key []byte, bucket, object string, seqNumber uint32, metadata map[string]string) (io.Reader, error) {
	objectEncryptionKey, err := decryptObjectInfo(key, bucket, object, metadata)
	if err != nil {
		return nil, err
	}
	return newDecryptReaderWithObjectKey(client, objectEncryptionKey, seqNumber)
}

func newDecryptReaderWithObjectKey(client io.Reader, objectEncryptionKey []byte, seqNumber uint32) (io.Reader, error) {
	reader, err := sio.DecryptReader(client, sio.Config{
		Key:            objectEncryptionKey,
		SequenceNumber: seqNumber,
	})
	if err != nil {
		return nil, crypto.ErrInvalidCustomerKey
	}
	return reader, nil
}

func DecryptBlocksRequestR(inputReader io.Reader, h http.Header, offset,
	length int64, seqNumber uint32, partStart int, oi ObjectInfo, copySource bool) (
	io.Reader, error) {

	bucket, object := oi.Bucket, oi.Name
	var reader io.Reader
	var err error
	if copySource {
		reader, err = DecryptCopyRequestR(inputReader, h, bucket, object, seqNumber, oi.UserDefined)
	} else {
		reader, err = DecryptRequestWithSequenceNumberR(inputReader, h, bucket, object, seqNumber, oi.UserDefined)
	}
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func DecryptRequestWithSequenceNumber(client io.Writer, r *http.Request, bucket, object string, seqNumber uint32, metadata map[string]string) (io.WriteCloser, error) {
	if crypto.S3.IsEncrypted(metadata) {
		return newDecryptWriter(client, nil, bucket, object, seqNumber, metadata)
	}

	key, err := ParseSSECustomerRequest(r)
	if err != nil {
		return nil, err
	}
	return newDecryptWriter(client, key, bucket, object, seqNumber, metadata)
}

func DecryptRequest(client io.Writer, r *http.Request, bucket, object string, metadata map[string]string) (io.WriteCloser, error) {
	return DecryptRequestWithSequenceNumber(client, r, bucket, object, 0, metadata)
}

type DecryptBlocksReader struct {
	reader         io.Reader
	decrypter      io.Reader
	startSeqNum    uint32
	partIndex      int
	header         http.Header
	bucket, object string
	metadata       map[string]string

	partDecRelOffset, partEncRelOffset int64

	copySource        bool
	customerKeyHeader string
}

func (d *DecryptBlocksReader) Read(p []byte) (int, error) {
	var err error
	var n1 int
	n1, err = d.decrypter.Read(p)
	if err != nil {
		return 0, err
	}
	d.partDecRelOffset += int64(n1)
	return len(p), nil
}

func getEncryptedSinglePartOffsetLength(offset, length int64, objInfo ObjectInfo) (seqNumber uint32, encOffset int64, encLength int64) {
	onePkgSize := int64(SSEDAREPackageBlockSize + SSEDAREPackageMetaSize)

	seqNumber = uint32(offset / SSEDAREPackageBlockSize)
	encOffset = int64(seqNumber) * onePkgSize
	encLength = ((offset+length)/SSEDAREPackageBlockSize)*onePkgSize - encOffset

	if (offset+length)%SSEDAREPackageBlockSize > 0 {
		encLength += onePkgSize
	}

	if encLength+encOffset > objInfo.EncryptedSize() {
		encLength = objInfo.EncryptedSize() - encOffset
	}
	return seqNumber, encOffset, encLength
}

func (o *ObjectInfo) DecryptedSize() (int64, error) {
	if !crypto.IsEncrypted(o.UserDefined) {
		return 0, errors.New("Cannot compute decrypted size of an unencrypted object")
	}

	size, err := sio.DecryptedSize(uint64(o.Size))
	if err != nil {
		err = errObjectTampered
	}
	return int64(size), err
}

func DecryptETag(key crypto.ObjectKey, object ObjectInfo) (string, error) {
	if n := strings.Count(object.ETag, "-"); n > 0 {
		if n != 1 {
			return "", errObjectTampered
		}
		i := strings.IndexByte(object.ETag, '-')
		if len(object.ETag[:i]) != 32 {
			return "", errObjectTampered
		}
		if _, err := hex.DecodeString(object.ETag[:32]); err != nil {
			return "", errObjectTampered
		}
		if _, err := strconv.ParseInt(object.ETag[i+1:], 10, 32); err != nil {
			return "", errObjectTampered
		}
		return object.ETag, nil
	}

	etag, err := hex.DecodeString(object.ETag)
	if err != nil {
		return "", err
	}
	etag, err = key.UnsealETag(etag)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(etag), nil
}

func getDecryptedETag(headers http.Header, objInfo ObjectInfo, copySource bool) (decryptedETag string) {
	var (
		key [32]byte
		err error
	)
	if len(objInfo.ETag) == 32 {
		return objInfo.ETag
	}

	if crypto.IsMultiPart(objInfo.UserDefined) {
		return objInfo.ETag
	}
	if crypto.SSECopy.IsRequested(headers) {
		key, err = crypto.SSECopy.ParseHTTP(headers)
		if err != nil {
			return objInfo.ETag
		}
	}
	if crypto.SSEC.IsEncrypted(objInfo.UserDefined) && !copySource {
		return objInfo.ETag[len(objInfo.ETag)-32:]
	}

	objectEncryptionKey, err := decryptObjectInfo(key[:], objInfo.Bucket, objInfo.Name, objInfo.UserDefined)
	if err != nil {
		return objInfo.ETag
	}
	return tryDecryptETag(objectEncryptionKey, objInfo.ETag, false)
}

func tryDecryptETag(key []byte, encryptedETag string, ssec bool) string {
	if ssec {
		return encryptedETag[len(encryptedETag)-32:]
	}
	var objectKey crypto.ObjectKey
	copy(objectKey[:], key)
	encBytes, err := hex.DecodeString(encryptedETag)
	if err != nil {
		return encryptedETag
	}
	etagBytes, err := objectKey.UnsealETag(encBytes)
	if err != nil {
		return encryptedETag
	}
	return hex.EncodeToString(etagBytes)
}

func (o *ObjectInfo) GetDecryptedRange(rs *HTTPRangeSpec) (encOff, encLength, skipLen int64, seqNumber uint32, partStart int, err error) {
	if !crypto.IsEncrypted(o.UserDefined) {
		err = errors.New("Object is not encrypted")
		return
	}

	if rs == nil {
		return 0, int64(o.Size), 0, 0, 0, nil
	}

	var sizes []int64
	var decObjSize int64
	var partSize uint64
	partSize, err = sio.DecryptedSize(uint64(o.Size))
	if err != nil {
		err = errObjectTampered
		return
	}
	sizes = []int64{int64(partSize)}
	decObjSize = sizes[0]

	var off, length int64
	off, length, err = rs.GetOffsetLength(decObjSize)
	if err != nil {
		return
	}

	var partEnd int
	var cumulativeSum, encCumulativeSum int64
	for i, size := range sizes {
		if off < cumulativeSum+size {
			partStart = i
			break
		}
		cumulativeSum += size
		encPartSize, _ := sio.EncryptedSize(uint64(size))
		encCumulativeSum += int64(encPartSize)
	}

	sseDAREEncPackageBlockSize := int64(SSEDAREPackageBlockSize + SSEDAREPackageMetaSize)
	startPkgNum := (off - cumulativeSum) / SSEDAREPackageBlockSize

	skipLen = (off - cumulativeSum) % SSEDAREPackageBlockSize

	encOff = encCumulativeSum + startPkgNum*sseDAREEncPackageBlockSize
	endOffset := off + length - 1
	for i1, size := range sizes[partStart:] {
		i := partStart + i1
		if endOffset < cumulativeSum+size {
			partEnd = i
			break
		}
		cumulativeSum += size
		encPartSize, _ := sio.EncryptedSize(uint64(size))
		encCumulativeSum += int64(encPartSize)
	}
	endPkgNum := (endOffset - cumulativeSum) / SSEDAREPackageBlockSize
	endEncOffset := encCumulativeSum + (endPkgNum+1)*sseDAREEncPackageBlockSize
	lastPartSize, _ := sio.EncryptedSize(uint64(sizes[partEnd]))
	if endEncOffset > encCumulativeSum+int64(lastPartSize) {
		endEncOffset = encCumulativeSum + int64(lastPartSize)
	}
	encLength = endEncOffset - encOff
	seqNumber = uint32(startPkgNum)
	return encOff, encLength, skipLen, seqNumber, partStart, nil
}

func (o *ObjectInfo) EncryptedSize() int64 {
	size, err := sio.EncryptedSize(uint64(o.Size))
	if err != nil {
		reqInfo := (&logger.ReqInfo{}).AppendTags("size", strconv.FormatUint(size, 10))
		ctx := logger.SetReqInfo(GlobalContext, reqInfo)
		logger.CriticalIf(ctx, err)
	}
	return int64(size)
}

func DecryptCopyObjectInfo(info *ObjectInfo, headers http.Header) (errCode APIErrorCode, encrypted bool) {
	if info.IsDir {
		return ErrNone, false
	}
	if errCode, encrypted = ErrNone, crypto.IsEncrypted(info.UserDefined); !encrypted && crypto.SSECopy.IsRequested(headers) {
		errCode = ErrInvalidEncryptionParameters
	} else if encrypted {
		if (!crypto.SSECopy.IsRequested(headers) && crypto.SSEC.IsEncrypted(info.UserDefined)) ||
			(crypto.SSECopy.IsRequested(headers) && crypto.S3.IsEncrypted(info.UserDefined)) {
			errCode = ErrSSEEncryptedObject
			return
		}
		var err error
		if info.Size, err = info.DecryptedSize(); err != nil {
			errCode = toAPIErrorCode(GlobalContext, err)
		}
	}
	return
}

func DecryptObjectInfo(info *ObjectInfo, headers http.Header) (encrypted bool, err error) {
	if info.IsDir {
		return false, nil
	}
	if crypto.S3.IsRequested(headers) {
		err = errInvalidEncryptionParameters
		return
	}
	if err, encrypted = nil, crypto.IsEncrypted(info.UserDefined); !encrypted && crypto.SSEC.IsRequested(headers) {
		err = errInvalidEncryptionParameters
	} else if encrypted {
		if (crypto.SSEC.IsEncrypted(info.UserDefined) && !crypto.SSEC.IsRequested(headers)) ||
			(crypto.S3.IsEncrypted(info.UserDefined) && crypto.SSEC.IsRequested(headers)) {
			err = errEncryptedObject
			return
		}
		_, err = info.DecryptedSize()

		if crypto.IsEncrypted(info.UserDefined) && !crypto.IsMultiPart(info.UserDefined) {
			info.ETag = getDecryptedETag(headers, *info, false)
		}

	}
	return
}

func deriveClientKey(clientKey [32]byte, bucket, object string) [32]byte {
	var key [32]byte
	mac := hmac.New(sha256.New, clientKey[:])
	mac.Write([]byte(crypto.SSEC.String()))
	mac.Write([]byte(path.Join(bucket, object)))
	mac.Sum(key[:0])
	return key
}

func getDefaultOpts(header http.Header, copySource bool, metadata map[string]string) (opts ObjectOptions, err error) {
	var clientKey [32]byte
	var sse encrypt.ServerSide

	if copySource {
		if crypto.SSECopy.IsRequested(header) {
			clientKey, err = crypto.SSECopy.ParseHTTP(header)
			if err != nil {
				return
			}
			if sse, err = encrypt.NewSSEC(clientKey[:]); err != nil {
				return
			}
			return ObjectOptions{ServerSideEncryption: encrypt.SSECopy(sse), UserDefined: metadata}, nil
		}
		return
	}

	if crypto.SSEC.IsRequested(header) {
		clientKey, err = crypto.SSEC.ParseHTTP(header)
		if err != nil {
			return
		}
		if sse, err = encrypt.NewSSEC(clientKey[:]); err != nil {
			return
		}
		return ObjectOptions{ServerSideEncryption: sse, UserDefined: metadata}, nil
	}
	if crypto.S3.IsRequested(header) || (metadata != nil && crypto.S3.IsEncrypted(metadata)) {
		return ObjectOptions{ServerSideEncryption: encrypt.NewSSE(), UserDefined: metadata}, nil
	}
	return ObjectOptions{UserDefined: metadata}, nil
}

func getOpts(ctx context.Context, r *http.Request, bucket, object string) (ObjectOptions, error) {
	var (
		encryption encrypt.ServerSide
		opts       ObjectOptions
	)

	var partNumber int
	var err error
	if pn := r.URL.Query().Get("partNumber"); pn != "" {
		partNumber, err = strconv.Atoi(pn)
		if err != nil {
			return opts, err
		}
		if partNumber < 0 {
			return opts, errInvalidArgument
		}
	}

	if crypto.SSEC.IsRequested(r.Header) {
		key, err := crypto.SSEC.ParseHTTP(r.Header)
		if err != nil {
			return opts, err
		}
		derivedKey := deriveClientKey(key, bucket, object)
		encryption, err = encrypt.NewSSEC(derivedKey[:])
		logger.CriticalIf(ctx, err)
		return ObjectOptions{ServerSideEncryption: encryption, PartNumber: partNumber}, nil
	}

	opts, err = getDefaultOpts(r.Header, false, nil)
	if err != nil {
		return opts, err
	}
	opts.PartNumber = partNumber
	return opts, nil
}

func putOpts(ctx context.Context, r *http.Request, bucket, object string, metadata map[string]string) (opts ObjectOptions, err error) {
	if crypto.S3.IsRequested(r.Header) || crypto.S3.IsEncrypted(metadata) {
		return ObjectOptions{ServerSideEncryption: encrypt.NewSSE(), UserDefined: metadata}, nil
	}
	if crypto.SSEC.IsRequested(r.Header) {
		opts, err = getOpts(ctx, r, bucket, object)
		opts.UserDefined = metadata
		return
	}
	if crypto.S3KMS.IsRequested(r.Header) {
		keyID, context, err := crypto.S3KMS.ParseHTTP(r.Header)
		if err != nil {
			return ObjectOptions{}, err
		}
		sseKms, err := encrypt.NewSSEKMS(keyID, context)
		if err != nil {
			return ObjectOptions{}, err
		}
		return ObjectOptions{ServerSideEncryption: sseKms, UserDefined: metadata}, nil
	}
	return getDefaultOpts(r.Header, false, metadata)
}

func copyDstOpts(ctx context.Context, r *http.Request, bucket, object string, metadata map[string]string) (opts ObjectOptions, err error) {
	return putOpts(ctx, r, bucket, object, metadata)
}

func copySrcOpts(ctx context.Context, r *http.Request, bucket, object string) (ObjectOptions, error) {
	var (
		ssec encrypt.ServerSide
		opts ObjectOptions
	)

	if crypto.SSECopy.IsRequested(r.Header) {
		key, err := crypto.SSECopy.ParseHTTP(r.Header)
		if err != nil {
			return opts, err
		}
		derivedKey := deriveClientKey(key, bucket, object)
		ssec, err = encrypt.NewSSEC(derivedKey[:])
		if err != nil {
			return opts, err
		}
		return ObjectOptions{ServerSideEncryption: encrypt.SSECopy(ssec)}, nil
	}

	return getDefaultOpts(r.Header, true, nil)
}
