package cmd

import (
	"bytes"
	"crypto/hmac"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/auth"
	"github.com/storeros/ipos/pkg/sha256-simd"
)

const unsignedPayload = "UNSIGNED-PAYLOAD"

func skipContentSha256Cksum(r *http.Request) bool {
	var (
		v  []string
		ok bool
	)

	if isRequestPresignedSignatureV4(r) {
		v, ok = r.URL.Query()[xhttp.AmzContentSha256]
		if !ok {
			v, ok = r.Header[xhttp.AmzContentSha256]
		}
	} else {
		v, ok = r.Header[xhttp.AmzContentSha256]
	}

	return !(ok && v[0] != unsignedPayload)
}

func getContentSha256Cksum(r *http.Request, stype serviceType) string {
	if stype == serviceSTS {
		payload, err := ioutil.ReadAll(io.LimitReader(r.Body, stsRequestBodyLimit))
		if err != nil {
			logger.CriticalIf(GlobalContext, err)
		}
		sum256 := sha256.New()
		sum256.Write(payload)
		r.Body = ioutil.NopCloser(bytes.NewReader(payload))
		return hex.EncodeToString(sum256.Sum(nil))
	}

	var (
		defaultSha256Cksum string
		v                  []string
		ok                 bool
	)

	if isRequestPresignedSignatureV4(r) {
		defaultSha256Cksum = unsignedPayload
		v, ok = r.URL.Query()[xhttp.AmzContentSha256]
		if !ok {
			v, ok = r.Header[xhttp.AmzContentSha256]
		}
	} else {
		defaultSha256Cksum = emptySHA256
		v, ok = r.Header[xhttp.AmzContentSha256]
	}

	if ok {
		return v[0]
	}

	return defaultSha256Cksum
}

func isValidRegion(reqRegion string, confRegion string) bool {
	if confRegion == "" {
		return true
	}
	if confRegion == "US" {
		confRegion = globalIPOSDefaultRegion
	}
	if reqRegion == "US" {
		reqRegion = globalIPOSDefaultRegion
	}
	return reqRegion == confRegion
}

func checkKeyValid(accessKey string) (auth.Credentials, bool, APIErrorCode) {
	var owner = true
	var cred = globalActiveCred
	if cred.AccessKey != accessKey {
		if globalIAMSys == nil {
			return cred, false, ErrInvalidAccessKeyID
		}
		var ok bool
		if cred, ok = globalIAMSys.GetUser(accessKey); !ok {
			return cred, false, ErrInvalidAccessKeyID
		}
		owner = false
	}
	return cred, owner, ErrNone
}

func sumHMAC(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func extractSignedHeaders(signedHeaders []string, r *http.Request) (http.Header, APIErrorCode) {
	reqHeaders := r.Header
	reqQueries := r.URL.Query()
	if !contains(signedHeaders, "host") {
		return nil, ErrUnsignedHeaders
	}
	extractedSignedHeaders := make(http.Header)
	for _, header := range signedHeaders {
		val, ok := reqHeaders[http.CanonicalHeaderKey(header)]
		if !ok {
			val, ok = reqQueries[header]
		}
		if ok {
			for _, enc := range val {
				extractedSignedHeaders.Add(header, enc)
			}
			continue
		}
		switch header {
		case "expect":
			extractedSignedHeaders.Set(header, "100-continue")
		case "host":
			extractedSignedHeaders.Set(header, r.Host)
		case "transfer-encoding":
			for _, enc := range r.TransferEncoding {
				extractedSignedHeaders.Add(header, enc)
			}
		case "content-length":
			extractedSignedHeaders.Set(header, strconv.FormatInt(r.ContentLength, 10))
		default:
			return nil, ErrUnsignedHeaders
		}
	}
	return extractedSignedHeaders, ErrNone
}

func signV4TrimAll(input string) string {
	return strings.Join(strings.Fields(input), " ")
}
