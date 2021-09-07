package cmd

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"net/http"
	"time"

	humanize "github.com/dustin/go-humanize"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/pkg/auth"
	sha256 "github.com/storeros/ipos/pkg/sha256-simd"
)

const (
	emptySHA256              = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	streamingContentSHA256   = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
	signV4ChunkedAlgorithm   = "AWS4-HMAC-SHA256-PAYLOAD"
	streamingContentEncoding = "aws-chunked"
)

func getChunkSignature(cred auth.Credentials, seedSignature string, region string, date time.Time, hashedChunk string) string {
	stringToSign := signV4ChunkedAlgorithm + "\n" +
		date.Format(iso8601Format) + "\n" +
		getScope(date, region) + "\n" +
		seedSignature + "\n" +
		emptySHA256 + "\n" +
		hashedChunk

	signingKey := getSigningKey(cred.SecretKey, date, region, serviceS3)

	newSignature := getSignature(signingKey, stringToSign)

	return newSignature
}

func calculateSeedSignature(r *http.Request) (cred auth.Credentials, signature string, region string, date time.Time, errCode APIErrorCode) {
	req := *r

	v4Auth := req.Header.Get(xhttp.Authorization)

	signV4Values, errCode := parseSignV4(v4Auth, globalServerRegion, serviceS3)
	if errCode != ErrNone {
		return cred, "", "", time.Time{}, errCode
	}

	payload := streamingContentSHA256

	if payload != req.Header.Get(xhttp.AmzContentSha256) {
		return cred, "", "", time.Time{}, ErrContentSHA256Mismatch
	}

	extractedSignedHeaders, errCode := extractSignedHeaders(signV4Values.SignedHeaders, r)
	if errCode != ErrNone {
		return cred, "", "", time.Time{}, errCode
	}

	cred, _, errCode = checkKeyValid(signV4Values.Credential.accessKey)
	if errCode != ErrNone {
		return cred, "", "", time.Time{}, errCode
	}

	region = signV4Values.Credential.scope.region

	var dateStr string
	if dateStr = req.Header.Get("x-amz-date"); dateStr == "" {
		if dateStr = r.Header.Get("Date"); dateStr == "" {
			return cred, "", "", time.Time{}, ErrMissingDateHeader
		}
	}

	var err error
	date, err = time.Parse(iso8601Format, dateStr)
	if err != nil {
		return cred, "", "", time.Time{}, ErrMalformedDate
	}

	queryStr := req.URL.Query().Encode()

	canonicalRequest := getCanonicalRequest(extractedSignedHeaders, payload, queryStr, req.URL.Path, req.Method)

	stringToSign := getStringToSign(canonicalRequest, date, signV4Values.Credential.getScope())

	signingKey := getSigningKey(cred.SecretKey, signV4Values.Credential.scope.date, region, serviceS3)

	newSignature := getSignature(signingKey, stringToSign)

	if !compareSignatureV4(newSignature, signV4Values.Signature) {
		return cred, "", "", time.Time{}, ErrSignatureDoesNotMatch
	}

	return cred, newSignature, region, date, ErrNone
}

const maxLineLength = 4 * humanize.KiByte

var errLineTooLong = errors.New("header line too long")

var errMalformedEncoding = errors.New("malformed chunked encoding")

func newSignV4ChunkedReader(req *http.Request) (io.ReadCloser, APIErrorCode) {
	cred, seedSignature, region, seedDate, errCode := calculateSeedSignature(req)
	if errCode != ErrNone {
		return nil, errCode
	}

	return &s3ChunkedReader{
		reader:            bufio.NewReader(req.Body),
		cred:              cred,
		seedSignature:     seedSignature,
		seedDate:          seedDate,
		region:            region,
		chunkSHA256Writer: sha256.New(),
		state:             readChunkHeader,
	}, ErrNone
}

type s3ChunkedReader struct {
	reader            *bufio.Reader
	cred              auth.Credentials
	seedSignature     string
	seedDate          time.Time
	region            string
	state             chunkState
	lastChunk         bool
	chunkSignature    string
	chunkSHA256Writer hash.Hash
	n                 uint64
	err               error
}

func (cr *s3ChunkedReader) readS3ChunkHeader() {
	var hexChunkSize, hexChunkSignature []byte
	hexChunkSize, hexChunkSignature, cr.err = readChunkLine(cr.reader)
	if cr.err != nil {
		return
	}
	cr.n, cr.err = parseHexUint(hexChunkSize)
	if cr.err != nil {
		return
	}
	if cr.n == 0 {
		cr.err = io.EOF
	}
	cr.chunkSignature = string(hexChunkSignature)
}

type chunkState int

const (
	readChunkHeader chunkState = iota
	readChunkTrailer
	readChunk
	verifyChunk
	eofChunk
)

func (cs chunkState) String() string {
	stateString := ""
	switch cs {
	case readChunkHeader:
		stateString = "readChunkHeader"
	case readChunkTrailer:
		stateString = "readChunkTrailer"
	case readChunk:
		stateString = "readChunk"
	case verifyChunk:
		stateString = "verifyChunk"
	case eofChunk:
		stateString = "eofChunk"

	}
	return stateString
}

func (cr *s3ChunkedReader) Close() (err error) {
	return nil
}

func (cr *s3ChunkedReader) Read(buf []byte) (n int, err error) {
	for {
		switch cr.state {
		case readChunkHeader:
			cr.readS3ChunkHeader()
			if cr.n == 0 && cr.err == io.EOF {
				cr.state = readChunkTrailer
				cr.lastChunk = true
				continue
			}
			if cr.err != nil {
				return 0, cr.err
			}
			cr.state = readChunk
		case readChunkTrailer:
			cr.err = readCRLF(cr.reader)
			if cr.err != nil {
				return 0, errMalformedEncoding
			}
			cr.state = verifyChunk
		case readChunk:
			if len(buf) == 0 {
				return n, nil
			}
			rbuf := buf
			if uint64(len(rbuf)) > cr.n {
				rbuf = rbuf[:cr.n]
			}
			var n0 int
			n0, cr.err = cr.reader.Read(rbuf)
			if cr.err != nil {
				if cr.err == io.EOF {
					cr.err = io.ErrUnexpectedEOF
				}
				return 0, cr.err
			}

			cr.chunkSHA256Writer.Write(rbuf[:n0])
			n += n0
			buf = buf[n0:]
			cr.n -= uint64(n0)

			if cr.n == 0 {
				cr.state = readChunkTrailer
				continue
			}
		case verifyChunk:
			hashedChunk := hex.EncodeToString(cr.chunkSHA256Writer.Sum(nil))
			newSignature := getChunkSignature(cr.cred, cr.seedSignature, cr.region, cr.seedDate, hashedChunk)
			if !compareSignatureV4(cr.chunkSignature, newSignature) {
				cr.err = errSignatureMismatch
				return 0, cr.err
			}
			cr.seedSignature = newSignature
			cr.chunkSHA256Writer.Reset()
			if cr.lastChunk {
				cr.state = eofChunk
			} else {
				cr.state = readChunkHeader
			}
		case eofChunk:
			return n, io.EOF
		}
	}
}

func readCRLF(reader io.Reader) error {
	buf := make([]byte, 2)
	_, err := io.ReadFull(reader, buf[:2])
	if err != nil {
		return err
	}
	if buf[0] != '\r' || buf[1] != '\n' {
		return errMalformedEncoding
	}
	return nil
}

func readChunkLine(b *bufio.Reader) ([]byte, []byte, error) {
	buf, err := b.ReadSlice('\n')
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		} else if err == bufio.ErrBufferFull {
			err = errLineTooLong
		}
		return nil, nil, err
	}
	if len(buf) >= maxLineLength {
		return nil, nil, errLineTooLong
	}
	hexChunkSize, hexChunkSignature := parseS3ChunkExtension(buf)
	return hexChunkSize, hexChunkSignature, nil
}

func trimTrailingWhitespace(b []byte) []byte {
	for len(b) > 0 && isASCIISpace(b[len(b)-1]) {
		b = b[:len(b)-1]
	}
	return b
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

const s3ChunkSignatureStr = ";chunk-signature="

func parseS3ChunkExtension(buf []byte) ([]byte, []byte) {
	buf = trimTrailingWhitespace(buf)
	semi := bytes.Index(buf, []byte(s3ChunkSignatureStr))
	if semi == -1 {
		return buf, nil
	}
	return buf[:semi], parseChunkSignature(buf[semi:])
}

func parseChunkSignature(chunk []byte) []byte {
	chunkSplits := bytes.SplitN(chunk, []byte(s3ChunkSignatureStr), 2)
	return chunkSplits[1]
}

func parseHexUint(v []byte) (n uint64, err error) {
	for i, b := range v {
		switch {
		case '0' <= b && b <= '9':
			b = b - '0'
		case 'a' <= b && b <= 'f':
			b = b - 'a' + 10
		case 'A' <= b && b <= 'F':
			b = b - 'A' + 10
		default:
			return 0, errors.New("invalid byte in chunk length")
		}
		if i == 16 {
			return 0, errors.New("http chunk length too large")
		}
		n <<= 4
		n |= uint64(b)
	}
	return
}
