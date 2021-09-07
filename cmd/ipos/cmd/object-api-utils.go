package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/klauspost/compress/s2"
	"github.com/klauspost/readahead"

	"github.com/storeros/ipos/cmd/ipos/crypto"
	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/hash"
	"github.com/storeros/ipos/pkg/ioutil"
	"github.com/storeros/ipos/pkg/s3utils"
	"github.com/storeros/ipos/pkg/wildcard"
)

const (
	iposMetaBucket          = ".ipos.sys"
	mpartMetaPrefix         = "multipart"
	iposMetaMultipartBucket = iposMetaBucket + SlashSeparator + mpartMetaPrefix
	iposMetaTmpBucket       = iposMetaBucket + "/tmp"
	dnsDelimiter            = "."
	compReadAheadSize       = 100 << 20
	compReadAheadBuffers    = 5
	compReadAheadBufSize    = 1 << 20
)

func isIPOSMetaBucketName(bucket string) bool {
	return bucket == iposMetaBucket ||
		bucket == iposMetaMultipartBucket ||
		bucket == iposMetaTmpBucket
}

func IsValidBucketName(bucket string) bool {
	if isIPOSMetaBucketName(bucket) {
		return true
	}
	if len(bucket) < 3 || len(bucket) > 63 {
		return false
	}

	allNumbers := true
	pieces := strings.Split(bucket, dnsDelimiter)
	for _, piece := range pieces {
		if len(piece) == 0 || piece[0] == '-' ||
			piece[len(piece)-1] == '-' {
			return false
		}
		isNotNumber := false
		for i := 0; i < len(piece); i++ {
			switch {
			case (piece[i] >= 'a' && piece[i] <= 'z' ||
				piece[i] == '-'):
				isNotNumber = true
			case piece[i] >= '0' && piece[i] <= '9':
			default:
				return false
			}
		}
		allNumbers = allNumbers && !isNotNumber
	}
	return !(len(pieces) == 4 && allNumbers)
}

func IsValidObjectName(object string) bool {
	if len(object) == 0 {
		return false
	}
	if HasSuffix(object, SlashSeparator) {
		return false
	}
	return IsValidObjectPrefix(object)
}

func IsValidObjectPrefix(object string) bool {
	if !utf8.ValidString(object) {
		return false
	}
	if strings.Contains(object, `//`) {
		return false
	}
	return true
}

func checkObjectNameForLengthAndSlash(bucket, object string) error {
	if len(object) > 1024 {
		return ObjectNameTooLong{
			Bucket: bucket,
			Object: object,
		}
	}
	if HasPrefix(object, SlashSeparator) {
		return ObjectNamePrefixAsSlash{
			Bucket: bucket,
			Object: object,
		}
	}
	return nil
}

const SlashSeparator = "/"

func retainSlash(s string) string {
	return strings.TrimSuffix(s, SlashSeparator) + SlashSeparator
}

func pathsJoinPrefix(prefix string, elem ...string) (paths []string) {
	paths = make([]string, len(elem))
	for i, e := range elem {
		paths[i] = pathJoin(prefix, e)
	}
	return paths
}

func pathJoin(elem ...string) string {
	trailingSlash := ""
	if len(elem) > 0 {
		if HasSuffix(elem[len(elem)-1], SlashSeparator) {
			trailingSlash = SlashSeparator
		}
	}
	return path.Join(elem...) + trailingSlash
}

func mustGetUUID() string {
	u, err := uuid.NewRandom()
	if err != nil {
		logger.CriticalIf(GlobalContext, err)
	}

	return u.String()
}

func getCompleteMultipartMD5(parts []CompletePart) string {
	var finalMD5Bytes []byte
	for _, part := range parts {
		md5Bytes, err := hex.DecodeString(canonicalizeETag(part.ETag))
		if err != nil {
			finalMD5Bytes = append(finalMD5Bytes, []byte(part.ETag)...)
		} else {
			finalMD5Bytes = append(finalMD5Bytes, md5Bytes...)
		}
	}
	s3MD5 := fmt.Sprintf("%s-%d", getMD5Hash(finalMD5Bytes), len(parts))
	return s3MD5
}

func cleanMetadata(metadata map[string]string) map[string]string {
	metadata = removeStandardStorageClass(metadata)
	return cleanMetadataKeys(metadata, "md5Sum", "etag", "expires", xhttp.AmzObjectTagging)
}

func removeStandardStorageClass(metadata map[string]string) map[string]string {
	if metadata[xhttp.AmzStorageClass] == "STANDARD" {
		delete(metadata, xhttp.AmzStorageClass)
	}
	return metadata
}

func cleanMetadataKeys(metadata map[string]string, keyNames ...string) map[string]string {
	var newMeta = make(map[string]string)
	for k, v := range metadata {
		if contains(keyNames, k) {
			continue
		}
		newMeta[k] = v
	}
	return newMeta
}

func extractETag(metadata map[string]string) string {
	etag, ok := metadata["md5Sum"]
	if !ok {
		etag = metadata["etag"]
	}
	return etag
}

func HasPrefix(s string, prefix string) bool {
	if runtime.GOOS == globalWindowsOSName {
		return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
	}
	return strings.HasPrefix(s, prefix)
}

func HasSuffix(s string, suffix string) bool {
	if runtime.GOOS == globalWindowsOSName {
		return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
	}
	return strings.HasSuffix(s, suffix)
}

func isStringEqual(s1 string, s2 string) bool {
	if runtime.GOOS == globalWindowsOSName {
		return strings.EqualFold(s1, s2)
	}
	return s1 == s2
}

func isReservedOrInvalidBucket(bucketEntry string, strict bool) bool {
	bucketEntry = strings.TrimSuffix(bucketEntry, SlashSeparator)
	if strict {
		if err := s3utils.CheckValidBucketNameStrict(bucketEntry); err != nil {
			return true
		}
	} else {
		if err := s3utils.CheckValidBucketName(bucketEntry); err != nil {
			return true
		}
	}
	return isIPOSMetaBucket(bucketEntry) || isIPOSReservedBucket(bucketEntry)
}

func isIPOSMetaBucket(bucketName string) bool {
	return bucketName == iposMetaBucket
}

func isIPOSReservedBucket(bucketName string) bool {
	return bucketName == iposReservedBucket
}

func (o ObjectInfo) IsCompressed() bool {
	_, ok := o.UserDefined[ReservedMetadataPrefix+"compression"]
	return ok
}

func (o ObjectInfo) IsCompressedOK() (bool, error) {
	scheme, ok := o.UserDefined[ReservedMetadataPrefix+"compression"]
	if !ok {
		return false, nil
	}
	if crypto.IsEncrypted(o.UserDefined) {
		return true, fmt.Errorf("compression %q and encryption enabled on same object", scheme)
	}
	switch scheme {
	case compressionAlgorithmV1, compressionAlgorithmV2:
		return true, nil
	}
	return true, fmt.Errorf("unknown compression scheme: %s", scheme)
}

func (o ObjectInfo) GetActualSize() int64 {
	metadata := o.UserDefined
	sizeStr, ok := metadata[ReservedMetadataPrefix+"actual-size"]
	if ok {
		size, err := strconv.ParseInt(sizeStr, 10, 64)
		if err == nil {
			return size
		}
	}
	return -1
}

func hasStringSuffixInSlice(str string, list []string) bool {
	str = strings.ToLower(str)
	for _, v := range list {
		if strings.HasSuffix(str, strings.ToLower(v)) {
			return true
		}
	}
	return false
}

func hasPattern(patterns []string, matchStr string) bool {
	for _, pattern := range patterns {
		if ok := wildcard.MatchSimple(pattern, matchStr); ok {
			return true
		}
	}
	return false
}

func getPartFile(entries []string, partNumber int, etag string) string {
	for _, entry := range entries {
		if strings.HasPrefix(entry, fmt.Sprintf("%.5d.%s.", partNumber, etag)) {
			return entry
		}
	}
	return ""
}

type byBucketName []BucketInfo

func (d byBucketName) Len() int           { return len(d) }
func (d byBucketName) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d byBucketName) Less(i, j int) bool { return d[i].Name < d[j].Name }

type GetObjectReader struct {
	ObjInfo ObjectInfo
	pReader io.Reader

	cleanUpFns []func()
	opts       ObjectOptions
	once       sync.Once
}

func NewGetObjectReaderFromReader(r io.Reader, oi ObjectInfo, opts ObjectOptions, cleanupFns ...func()) (*GetObjectReader, error) {
	if opts.CheckCopyPrecondFn != nil {
		if ok := opts.CheckCopyPrecondFn(oi, ""); ok {
			for i := len(cleanupFns) - 1; i >= 0; i-- {
				cleanupFns[i]()
			}
			return nil, PreConditionFailed{}
		}
	}
	return &GetObjectReader{
		ObjInfo:    oi,
		pReader:    r,
		cleanUpFns: cleanupFns,
		opts:       opts,
	}, nil
}

type ObjReaderFn func(inputReader io.Reader, h http.Header, pcfn CheckCopyPreconditionFn, cleanupFns ...func()) (r *GetObjectReader, err error)

func NewGetObjectReader(rs *HTTPRangeSpec, oi ObjectInfo, opts ObjectOptions, cleanUpFns ...func()) (
	fn ObjReaderFn, off, length int64, err error) {

	defer func() {
		if err != nil {
			for i := len(cleanUpFns) - 1; i >= 0; i-- {
				cleanUpFns[i]()
			}
		}
	}()

	isEncrypted := crypto.IsEncrypted(oi.UserDefined)
	isCompressed, err := oi.IsCompressedOK()
	if err != nil {
		return nil, 0, 0, err
	}

	var skipLen int64
	switch {
	case isEncrypted:
		var seqNumber uint32
		var partStart int
		off, length, skipLen, seqNumber, partStart, err = oi.GetDecryptedRange(rs)
		if err != nil {
			return nil, 0, 0, err
		}
		var decSize int64
		decSize, err = oi.DecryptedSize()
		if err != nil {
			return nil, 0, 0, err
		}
		var decRangeLength int64
		decRangeLength, err = rs.GetLength(decSize)
		if err != nil {
			return nil, 0, 0, err
		}

		fn = func(inputReader io.Reader, h http.Header, pcfn CheckCopyPreconditionFn, cFns ...func()) (r *GetObjectReader, err error) {
			copySource := h.Get(crypto.SSECopyAlgorithm) != ""

			cFns = append(cleanUpFns, cFns...)
			var decReader io.Reader
			decReader, err = DecryptBlocksRequestR(inputReader, h,
				off, length, seqNumber, partStart, oi, copySource)
			if err != nil {
				for i := len(cFns) - 1; i >= 0; i-- {
					cFns[i]()
				}
				return nil, err
			}
			encETag := oi.ETag
			oi.ETag = getDecryptedETag(h, oi, copySource)

			if opts.CheckCopyPrecondFn != nil {
				if ok := opts.CheckCopyPrecondFn(oi, encETag); ok {
					for i := len(cFns) - 1; i >= 0; i-- {
						cFns[i]()
					}
					return nil, PreConditionFailed{}
				}
			}

			decReader = io.LimitReader(ioutil.NewSkipReader(decReader, skipLen), decRangeLength)

			r = &GetObjectReader{
				ObjInfo:    oi,
				pReader:    decReader,
				cleanUpFns: cFns,
				opts:       opts,
			}
			return r, nil
		}
	case isCompressed:
		actualSize := oi.GetActualSize()
		if actualSize < 0 {
			return nil, 0, 0, errInvalidDecompressedSize
		}
		off, length = int64(0), oi.Size
		decOff, decLength := int64(0), actualSize
		fn = func(inputReader io.Reader, _ http.Header, pcfn CheckCopyPreconditionFn, cFns ...func()) (r *GetObjectReader, err error) {
			cFns = append(cleanUpFns, cFns...)
			if opts.CheckCopyPrecondFn != nil {
				if ok := opts.CheckCopyPrecondFn(oi, ""); ok {
					for i := len(cFns) - 1; i >= 0; i-- {
						cFns[i]()
					}
					return nil, PreConditionFailed{}
				}
			}
			s2Reader := s2.NewReader(inputReader)
			err = s2Reader.Skip(decOff)
			if err != nil {
				for i := len(cFns) - 1; i >= 0; i-- {
					cFns[i]()
				}
				return nil, err
			}

			decReader := io.LimitReader(s2Reader, decLength)
			if decLength > compReadAheadSize {
				rah, err := readahead.NewReaderSize(decReader, compReadAheadBuffers, compReadAheadBufSize)
				if err == nil {
					decReader = rah
					cFns = append(cFns, func() {
						rah.Close()
					})
				}
			}
			oi.Size = decLength

			r = &GetObjectReader{
				ObjInfo:    oi,
				pReader:    decReader,
				cleanUpFns: cFns,
				opts:       opts,
			}
			return r, nil
		}

	default:
		off, length, err = rs.GetOffsetLength(oi.Size)
		if err != nil {
			return nil, 0, 0, err
		}
		fn = func(inputReader io.Reader, _ http.Header, pcfn CheckCopyPreconditionFn, cFns ...func()) (r *GetObjectReader, err error) {
			cFns = append(cleanUpFns, cFns...)
			if opts.CheckCopyPrecondFn != nil {
				if ok := opts.CheckCopyPrecondFn(oi, ""); ok {
					for i := len(cFns) - 1; i >= 0; i-- {
						cFns[i]()
					}
					return nil, PreConditionFailed{}
				}
			}
			r = &GetObjectReader{
				ObjInfo:    oi,
				pReader:    inputReader,
				cleanUpFns: cFns,
				opts:       opts,
			}
			return r, nil
		}
	}
	return fn, off, length, nil
}

func (g *GetObjectReader) Close() error {
	g.once.Do(func() {
		for i := len(g.cleanUpFns) - 1; i >= 0; i-- {
			g.cleanUpFns[i]()
		}
	})
	return nil
}

func (g *GetObjectReader) Read(p []byte) (n int, err error) {
	n, err = g.pReader.Read(p)
	if err != nil {
		g.Close()
	}
	return
}

type SealMD5CurrFn func([]byte) []byte

type PutObjReader struct {
	*hash.Reader
	rawReader *hash.Reader
	sealMD5Fn SealMD5CurrFn
}

func (p *PutObjReader) Size() int64 {
	return p.Reader.Size()
}

func (p *PutObjReader) MD5CurrentHexString() string {
	md5sumCurr := p.rawReader.MD5Current()
	var appendHyphen bool
	if len(md5sumCurr) == 0 {
		md5sumCurr = make([]byte, 16)
		rand.Read(md5sumCurr)
		appendHyphen = true
	}
	if p.sealMD5Fn != nil {
		md5sumCurr = p.sealMD5Fn(md5sumCurr)
	}
	if appendHyphen {
		return hex.EncodeToString(md5sumCurr)[:32] + "-1"
	}
	return hex.EncodeToString(md5sumCurr)
}

func NewPutObjReader(rawReader *hash.Reader, encReader *hash.Reader, key *crypto.ObjectKey) *PutObjReader {
	p := PutObjReader{Reader: rawReader, rawReader: rawReader}

	if key != nil && encReader != nil {
		p.sealMD5Fn = sealETagFn(*key)
		p.Reader = encReader
	}
	return &p
}

func sealETag(encKey crypto.ObjectKey, md5CurrSum []byte) []byte {
	var emptyKey [32]byte
	if bytes.Equal(encKey[:], emptyKey[:]) {
		return md5CurrSum
	}
	return encKey.SealETag(md5CurrSum)
}

func sealETagFn(key crypto.ObjectKey) SealMD5CurrFn {
	fn := func(md5sumcurr []byte) []byte {
		return sealETag(key, md5sumcurr)
	}
	return fn
}

func CleanIPOSInternalMetadataKeys(metadata map[string]string) map[string]string {
	var newMeta = make(map[string]string, len(metadata))
	for k, v := range metadata {
		if strings.HasPrefix(k, "X-Amz-Meta-X-IPOS-Internal-") {
			newMeta[strings.TrimPrefix(k, "X-Amz-Meta-")] = v
		} else {
			newMeta[k] = v
		}
	}
	return newMeta
}

func newS2CompressReader(r io.Reader) io.ReadCloser {
	pr, pw := io.Pipe()
	comp := s2.NewWriter(pw)
	go func() {
		_, err := io.Copy(comp, r)
		if err != nil {
			comp.Close()
			pw.CloseWithError(err)
			return
		}
		err = comp.Close()
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.Close()
	}()
	return pr
}

type detectDisconnect struct {
	io.ReadCloser
	cancelCh <-chan struct{}
}

func (d *detectDisconnect) Read(p []byte) (int, error) {
	select {
	case <-d.cancelCh:
		return 0, io.ErrUnexpectedEOF
	default:
		return d.ReadCloser.Read(p)
	}
}
