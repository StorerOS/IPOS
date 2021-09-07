package cmd

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/pkg/auth"
)

const (
	signV2Algorithm = "AWS"
)

var resourceList = []string{
	"acl",
	"delete",
	"lifecycle",
	"location",
	"logging",
	"notification",
	"partNumber",
	"policy",
	"requestPayment",
	"response-cache-control",
	"response-content-disposition",
	"response-content-encoding",
	"response-content-language",
	"response-content-type",
	"response-expires",
	"torrent",
	"uploadId",
	"uploads",
	"versionId",
	"versioning",
	"versions",
	"website",
}

func doesPolicySignatureV2Match(formValues http.Header) APIErrorCode {
	cred := globalActiveCred
	accessKey := formValues.Get(xhttp.AmzAccessKeyID)
	cred, _, s3Err := checkKeyValid(accessKey)
	if s3Err != ErrNone {
		return s3Err
	}
	policy := formValues.Get("Policy")
	signature := formValues.Get(xhttp.AmzSignatureV2)
	if !compareSignatureV2(signature, calculateSignatureV2(policy, cred.SecretKey)) {
		return ErrSignatureDoesNotMatch
	}
	return ErrNone
}

func unescapeQueries(encodedQuery string) (unescapedQueries []string, err error) {
	for _, query := range strings.Split(encodedQuery, "&") {
		var unescapedQuery string
		unescapedQuery, err = url.QueryUnescape(query)
		if err != nil {
			return nil, err
		}
		unescapedQueries = append(unescapedQueries, unescapedQuery)
	}
	return unescapedQueries, nil
}

func doesPresignV2SignatureMatch(r *http.Request) APIErrorCode {
	tokens := strings.SplitN(r.RequestURI, "?", 2)
	encodedResource := tokens[0]
	encodedQuery := ""
	if len(tokens) == 2 {
		encodedQuery = tokens[1]
	}

	var (
		filteredQueries []string
		gotSignature    string
		expires         string
		accessKey       string
		err             error
	)

	var unescapedQueries []string
	unescapedQueries, err = unescapeQueries(encodedQuery)
	if err != nil {
		return ErrInvalidQueryParams
	}

	for _, query := range unescapedQueries {
		keyval := strings.SplitN(query, "=", 2)
		if len(keyval) != 2 {
			return ErrInvalidQueryParams
		}
		switch keyval[0] {
		case xhttp.AmzAccessKeyID:
			accessKey = keyval[1]
		case xhttp.AmzSignatureV2:
			gotSignature = keyval[1]
		case xhttp.Expires:
			expires = keyval[1]
		default:
			filteredQueries = append(filteredQueries, query)
		}
	}

	if accessKey == "" || gotSignature == "" || expires == "" {
		return ErrInvalidQueryParams
	}

	cred, _, s3Err := checkKeyValid(accessKey)
	if s3Err != ErrNone {
		return s3Err
	}

	expiresInt, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		return ErrMalformedExpires
	}

	if expiresInt < UTCNow().Unix() {
		return ErrExpiredPresignRequest
	}

	encodedResource, err = getResource(encodedResource, r.Host, globalDomainNames)
	if err != nil {
		return ErrInvalidRequest
	}

	expectedSignature := preSignatureV2(cred, r.Method, encodedResource, strings.Join(filteredQueries, "&"), r.Header, expires)
	if !compareSignatureV2(gotSignature, expectedSignature) {
		return ErrSignatureDoesNotMatch
	}

	return ErrNone
}

func getReqAccessKeyV2(r *http.Request) (auth.Credentials, bool, APIErrorCode) {
	if accessKey := r.URL.Query().Get(xhttp.AmzAccessKeyID); accessKey != "" {
		return checkKeyValid(accessKey)
	}

	authFields := strings.Split(r.Header.Get(xhttp.Authorization), " ")
	if len(authFields) != 2 {
		return auth.Credentials{}, false, ErrMissingFields
	}

	keySignFields := strings.Split(strings.TrimSpace(authFields[1]), ":")
	if len(keySignFields) != 2 {
		return auth.Credentials{}, false, ErrMissingFields
	}

	return checkKeyValid(keySignFields[0])
}

func validateV2AuthHeader(r *http.Request) (auth.Credentials, APIErrorCode) {
	var cred auth.Credentials
	v2Auth := r.Header.Get(xhttp.Authorization)
	if v2Auth == "" {
		return cred, ErrAuthHeaderEmpty
	}

	if !strings.HasPrefix(v2Auth, signV2Algorithm) {
		return cred, ErrSignatureVersionNotSupported
	}

	cred, _, apiErr := getReqAccessKeyV2(r)
	if apiErr != ErrNone {
		return cred, apiErr
	}

	return cred, ErrNone
}

func doesSignV2Match(r *http.Request) APIErrorCode {
	v2Auth := r.Header.Get(xhttp.Authorization)
	cred, apiError := validateV2AuthHeader(r)
	if apiError != ErrNone {
		return apiError
	}

	tokens := strings.SplitN(r.RequestURI, "?", 2)
	encodedResource := tokens[0]
	encodedQuery := ""
	if len(tokens) == 2 {
		encodedQuery = tokens[1]
	}

	unescapedQueries, err := unescapeQueries(encodedQuery)
	if err != nil {
		return ErrInvalidQueryParams
	}

	encodedResource, err = getResource(encodedResource, r.Host, globalDomainNames)
	if err != nil {
		return ErrInvalidRequest
	}

	prefix := fmt.Sprintf("%s %s:", signV2Algorithm, cred.AccessKey)
	if !strings.HasPrefix(v2Auth, prefix) {
		return ErrSignatureDoesNotMatch
	}
	v2Auth = v2Auth[len(prefix):]
	expectedAuth := signatureV2(cred, r.Method, encodedResource, strings.Join(unescapedQueries, "&"), r.Header)
	if !compareSignatureV2(v2Auth, expectedAuth) {
		return ErrSignatureDoesNotMatch
	}
	return ErrNone
}

func calculateSignatureV2(stringToSign string, secret string) string {
	hm := hmac.New(sha1.New, []byte(secret))
	hm.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(hm.Sum(nil))
}

func preSignatureV2(cred auth.Credentials, method string, encodedResource string, encodedQuery string, headers http.Header, expires string) string {
	stringToSign := getStringToSignV2(method, encodedResource, encodedQuery, headers, expires)
	return calculateSignatureV2(stringToSign, cred.SecretKey)
}

func signatureV2(cred auth.Credentials, method string, encodedResource string, encodedQuery string, headers http.Header) string {
	stringToSign := getStringToSignV2(method, encodedResource, encodedQuery, headers, "")
	signature := calculateSignatureV2(stringToSign, cred.SecretKey)
	return signature
}

func compareSignatureV2(sig1, sig2 string) bool {
	signature1, err := base64.StdEncoding.DecodeString(sig1)
	if err != nil {
		return false
	}
	signature2, err := base64.StdEncoding.DecodeString(sig2)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(signature1, signature2) == 1
}

func canonicalizedAmzHeadersV2(headers http.Header) string {
	var keys []string
	keyval := make(map[string]string)
	for key := range headers {
		lkey := strings.ToLower(key)
		if !strings.HasPrefix(lkey, "x-amz-") {
			continue
		}
		keys = append(keys, lkey)
		keyval[lkey] = strings.Join(headers[key], ",")
	}
	sort.Strings(keys)
	var canonicalHeaders []string
	for _, key := range keys {
		canonicalHeaders = append(canonicalHeaders, key+":"+keyval[key])
	}
	return strings.Join(canonicalHeaders, "\n")
}

func canonicalizedResourceV2(encodedResource, encodedQuery string) string {
	queries := strings.Split(encodedQuery, "&")
	keyval := make(map[string]string)
	for _, query := range queries {
		key := query
		val := ""
		index := strings.Index(query, "=")
		if index != -1 {
			key = query[:index]
			val = query[index+1:]
		}
		keyval[key] = val
	}

	var canonicalQueries []string
	for _, key := range resourceList {
		val, ok := keyval[key]
		if !ok {
			continue
		}
		if val == "" {
			canonicalQueries = append(canonicalQueries, key)
			continue
		}
		canonicalQueries = append(canonicalQueries, key+"="+val)
	}

	canonicalQuery := strings.Join(canonicalQueries, "&")
	if canonicalQuery != "" {
		return encodedResource + "?" + canonicalQuery
	}
	return encodedResource
}

func getStringToSignV2(method string, encodedResource, encodedQuery string, headers http.Header, expires string) string {
	canonicalHeaders := canonicalizedAmzHeadersV2(headers)
	if len(canonicalHeaders) > 0 {
		canonicalHeaders += "\n"
	}

	date := expires
	if date == "" {
		date = headers.Get(xhttp.Date)
	}

	stringToSign := strings.Join([]string{
		method,
		headers.Get(xhttp.ContentMD5),
		headers.Get(xhttp.ContentType),
		date,
		canonicalHeaders,
	}, "\n")

	return stringToSign + canonicalizedResourceV2(encodedResource, encodedQuery)
}
