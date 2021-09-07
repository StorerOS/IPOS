package signer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	streamingSignAlgorithm = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
	streamingPayloadHdr    = "AWS4-HMAC-SHA256-PAYLOAD"
	emptySHA256            = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	payloadChunkSize       = 64 * 1024
	chunkSigConstLen       = 17
	signatureStrLen        = 64
	crlfLen                = 2
)

var ignoredStreamingHeaders = map[string]bool{
	"Authorization": true,
	"User-Agent":    true,
	"Content-Type":  true,
}

func getSignedChunkLength(chunkDataSize int64) int64 {
	return int64(len(fmt.Sprintf("%x", chunkDataSize))) +
		chunkSigConstLen +
		signatureStrLen +
		crlfLen +
		chunkDataSize +
		crlfLen
}

func getStreamLength(dataLen, chunkSize int64) int64 {
	if dataLen <= 0 {
		return 0
	}

	chunksCount := int64(dataLen / chunkSize)
	remainingBytes := int64(dataLen % chunkSize)
	streamLen := int64(0)
	streamLen += chunksCount * getSignedChunkLength(chunkSize)
	if remainingBytes > 0 {
		streamLen += getSignedChunkLength(remainingBytes)
	}
	streamLen += getSignedChunkLength(0)
	return streamLen
}

func buildChunkStringToSign(t time.Time, region, previousSig string, chunkData []byte) string {
	stringToSignParts := []string{
		streamingPayloadHdr,
		t.Format(iso8601DateFormat),
		getScope(region, t, ServiceTypeS3),
		previousSig,
		emptySHA256,
		hex.EncodeToString(sum256(chunkData)),
	}

	return strings.Join(stringToSignParts, "\n")
}

func prepareStreamingRequest(req *http.Request, sessionToken string, dataLen int64, timestamp time.Time) {
	req.Header.Set("X-Amz-Content-Sha256", streamingSignAlgorithm)
	if sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", sessionToken)
	}

	req.Header.Set("X-Amz-Date", timestamp.Format(iso8601DateFormat))

	req.ContentLength = getStreamLength(dataLen, int64(payloadChunkSize))
	req.Header.Set("x-amz-decoded-content-length", strconv.FormatInt(dataLen, 10))
}

func buildChunkHeader(chunkLen int64, signature string) []byte {
	return []byte(strconv.FormatInt(chunkLen, 16) + ";chunk-signature=" + signature + "\r\n")
}

func buildChunkSignature(chunkData []byte, reqTime time.Time, region,
	previousSignature, secretAccessKey string) string {
	chunkStringToSign := buildChunkStringToSign(reqTime, region,
		previousSignature, chunkData)
	signingKey := getSigningKey(secretAccessKey, region, reqTime, ServiceTypeS3)
	return getSignature(signingKey, chunkStringToSign)
}

func (s *StreamingReader) setSeedSignature(req *http.Request) {
	canonicalRequest := getCanonicalRequest(*req, ignoredStreamingHeaders, getHashedPayload(*req))

	stringToSign := getStringToSignV4(s.reqTime, s.region, canonicalRequest, ServiceTypeS3)

	signingKey := getSigningKey(s.secretAccessKey, s.region, s.reqTime, ServiceTypeS3)

	s.seedSignature = getSignature(signingKey, stringToSign)
}

type StreamingReader struct {
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
	region          string
	prevSignature   string
	seedSignature   string
	contentLen      int64
	baseReadCloser  io.ReadCloser
	bytesRead       int64
	buf             bytes.Buffer
	chunkBuf        []byte
	chunkBufLen     int
	done            bool
	reqTime         time.Time
	chunkNum        int
	totalChunks     int
	lastChunkSize   int
}

func (s *StreamingReader) signChunk(chunkLen int) {
	signature := buildChunkSignature(s.chunkBuf[:chunkLen], s.reqTime,
		s.region, s.prevSignature, s.secretAccessKey)

	s.prevSignature = signature

	chunkHdr := buildChunkHeader(int64(chunkLen), signature)
	s.buf.Write(chunkHdr)

	s.buf.Write(s.chunkBuf[:chunkLen])

	s.buf.Write([]byte("\r\n"))

	s.chunkBufLen = 0
	s.chunkNum++
}

func (s *StreamingReader) setStreamingAuthHeader(req *http.Request) {
	credential := GetCredential(s.accessKeyID, s.region, s.reqTime, ServiceTypeS3)
	authParts := []string{
		signV4Algorithm + " Credential=" + credential,
		"SignedHeaders=" + getSignedHeaders(*req, ignoredStreamingHeaders),
		"Signature=" + s.seedSignature,
	}

	auth := strings.Join(authParts, ",")
	req.Header.Set("Authorization", auth)
}

func StreamingSignV4(req *http.Request, accessKeyID, secretAccessKey, sessionToken,
	region string, dataLen int64, reqTime time.Time) *http.Request {
	prepareStreamingRequest(req, sessionToken, dataLen, reqTime)

	if req.Body == nil {
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("")))
	}

	stReader := &StreamingReader{
		baseReadCloser:  req.Body,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		sessionToken:    sessionToken,
		region:          region,
		reqTime:         reqTime,
		chunkBuf:        make([]byte, payloadChunkSize),
		contentLen:      dataLen,
		chunkNum:        1,
		totalChunks:     int((dataLen+payloadChunkSize-1)/payloadChunkSize) + 1,
		lastChunkSize:   int(dataLen % payloadChunkSize),
	}

	stReader.setSeedSignature(req)

	stReader.setStreamingAuthHeader(req)

	stReader.prevSignature = stReader.seedSignature
	req.Body = stReader

	return req
}

func (s *StreamingReader) Read(buf []byte) (int, error) {
	switch {
	case s.done:
	case s.buf.Len() < len(buf):
		s.chunkBufLen = 0
		for {
			n1, err := s.baseReadCloser.Read(s.chunkBuf[s.chunkBufLen:])
			if n1 > 0 {
				s.chunkBufLen += n1
				s.bytesRead += int64(n1)

				if s.chunkBufLen == payloadChunkSize ||
					(s.chunkNum == s.totalChunks-1 &&
						s.chunkBufLen == s.lastChunkSize) {
					s.signChunk(s.chunkBufLen)
					break
				}
			}
			if err != nil {
				if err == io.EOF {
					s.done = true

					if s.bytesRead != s.contentLen {
						return 0, fmt.Errorf("http: ContentLength=%d with Body length %d", s.contentLen, s.bytesRead)
					}

					s.signChunk(0)
					break
				}
				return 0, err
			}

		}
	}
	return s.buf.Read(buf)
}

func (s *StreamingReader) Close() error {
	return s.baseReadCloser.Close()
}
