package signer

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/storeros/ipos/pkg/s3utils"
)

const (
	signV4Algorithm   = "AWS4-HMAC-SHA256"
	iso8601DateFormat = "20060102T150405Z"
	yyyymmdd          = "20060102"
)

const (
	ServiceTypeS3  = "s3"
	ServiceTypeSTS = "sts"
)

var v4IgnoredHeaders = map[string]bool{
	"Authorization": true,
	"User-Agent":    true,
}

func getSigningKey(secret, loc string, t time.Time, serviceType string) []byte {
	date := sumHMAC([]byte("AWS4"+secret), []byte(t.Format(yyyymmdd)))
	location := sumHMAC(date, []byte(loc))
	service := sumHMAC(location, []byte(serviceType))
	signingKey := sumHMAC(service, []byte("aws4_request"))
	return signingKey
}

func getSignature(signingKey []byte, stringToSign string) string {
	return hex.EncodeToString(sumHMAC(signingKey, []byte(stringToSign)))
}

func getScope(location string, t time.Time, serviceType string) string {
	scope := strings.Join([]string{
		t.Format(yyyymmdd),
		location,
		serviceType,
		"aws4_request",
	}, "/")
	return scope
}

func GetCredential(accessKeyID, location string, t time.Time, serviceType string) string {
	scope := getScope(location, t, serviceType)
	return accessKeyID + "/" + scope
}

func getHashedPayload(req http.Request) string {
	hashedPayload := req.Header.Get("X-Amz-Content-Sha256")
	if hashedPayload == "" {
		hashedPayload = unsignedPayload
	}
	return hashedPayload
}

func getCanonicalHeaders(req http.Request, ignoredHeaders map[string]bool) string {
	var headers []string
	vals := make(map[string][]string)
	for k, vv := range req.Header {
		if _, ok := ignoredHeaders[http.CanonicalHeaderKey(k)]; ok {
			continue
		}
		headers = append(headers, strings.ToLower(k))
		vals[strings.ToLower(k)] = vv
	}
	headers = append(headers, "host")
	sort.Strings(headers)

	var buf bytes.Buffer

	for _, k := range headers {
		buf.WriteString(k)
		buf.WriteByte(':')
		switch {
		case k == "host":
			buf.WriteString(getHostAddr(&req))
			fallthrough
		default:
			for idx, v := range vals[k] {
				if idx > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(signV4TrimAll(v))
			}
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

func getSignedHeaders(req http.Request, ignoredHeaders map[string]bool) string {
	var headers []string
	for k := range req.Header {
		if _, ok := ignoredHeaders[http.CanonicalHeaderKey(k)]; ok {
			continue
		}
		headers = append(headers, strings.ToLower(k))
	}
	headers = append(headers, "host")
	sort.Strings(headers)
	return strings.Join(headers, ";")
}

func getCanonicalRequest(req http.Request, ignoredHeaders map[string]bool, hashedPayload string) string {
	req.URL.RawQuery = strings.Replace(req.URL.Query().Encode(), "+", "%20", -1)
	canonicalRequest := strings.Join([]string{
		req.Method,
		s3utils.EncodePath(req.URL.Path),
		req.URL.RawQuery,
		getCanonicalHeaders(req, ignoredHeaders),
		getSignedHeaders(req, ignoredHeaders),
		hashedPayload,
	}, "\n")
	return canonicalRequest
}

func getStringToSignV4(t time.Time, location, canonicalRequest, serviceType string) string {
	stringToSign := signV4Algorithm + "\n" + t.Format(iso8601DateFormat) + "\n"
	stringToSign = stringToSign + getScope(location, t, serviceType) + "\n"
	stringToSign = stringToSign + hex.EncodeToString(sum256([]byte(canonicalRequest)))
	return stringToSign
}

func PreSignV4(req http.Request, accessKeyID, secretAccessKey, sessionToken, location string, expires int64) *http.Request {
	if accessKeyID == "" || secretAccessKey == "" {
		return &req
	}

	t := time.Now().UTC()

	credential := GetCredential(accessKeyID, location, t, ServiceTypeS3)

	signedHeaders := getSignedHeaders(req, v4IgnoredHeaders)

	query := req.URL.Query()
	query.Set("X-Amz-Algorithm", signV4Algorithm)
	query.Set("X-Amz-Date", t.Format(iso8601DateFormat))
	query.Set("X-Amz-Expires", strconv.FormatInt(expires, 10))
	query.Set("X-Amz-SignedHeaders", signedHeaders)
	query.Set("X-Amz-Credential", credential)

	if sessionToken != "" {
		query.Set("X-Amz-Security-Token", sessionToken)
	}
	req.URL.RawQuery = query.Encode()

	canonicalRequest := getCanonicalRequest(req, v4IgnoredHeaders, getHashedPayload(req))

	stringToSign := getStringToSignV4(t, location, canonicalRequest, ServiceTypeS3)

	signingKey := getSigningKey(secretAccessKey, location, t, ServiceTypeS3)

	signature := getSignature(signingKey, stringToSign)

	req.URL.RawQuery += "&X-Amz-Signature=" + signature

	return &req
}

func PostPresignSignatureV4(policyBase64 string, t time.Time, secretAccessKey, location string) string {
	signingkey := getSigningKey(secretAccessKey, location, t, ServiceTypeS3)

	signature := getSignature(signingkey, policyBase64)
	return signature
}

func SignV4STS(req http.Request, accessKeyID, secretAccessKey, location string) *http.Request {
	return signV4(req, accessKeyID, secretAccessKey, "", location, ServiceTypeSTS)
}

func signV4(req http.Request, accessKeyID, secretAccessKey, sessionToken, location, serviceType string) *http.Request {
	if accessKeyID == "" || secretAccessKey == "" {
		return &req
	}

	t := time.Now().UTC()

	req.Header.Set("X-Amz-Date", t.Format(iso8601DateFormat))

	if sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", sessionToken)
	}

	hashedPayload := getHashedPayload(req)
	if serviceType == ServiceTypeSTS {
		req.Header.Del("X-Amz-Content-Sha256")
	}

	canonicalRequest := getCanonicalRequest(req, v4IgnoredHeaders, hashedPayload)

	stringToSign := getStringToSignV4(t, location, canonicalRequest, serviceType)

	signingKey := getSigningKey(secretAccessKey, location, t, serviceType)

	credential := GetCredential(accessKeyID, location, t, serviceType)

	signedHeaders := getSignedHeaders(req, v4IgnoredHeaders)

	signature := getSignature(signingKey, stringToSign)

	parts := []string{
		signV4Algorithm + " Credential=" + credential,
		"SignedHeaders=" + signedHeaders,
		"Signature=" + signature,
	}

	auth := strings.Join(parts, ", ")
	req.Header.Set("Authorization", auth)

	return &req
}

func SignV4(req http.Request, accessKeyID, secretAccessKey, sessionToken, location string) *http.Request {
	return signV4(req, accessKeyID, secretAccessKey, sessionToken, location, ServiceTypeS3)
}
