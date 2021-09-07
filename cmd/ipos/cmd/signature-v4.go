package cmd

import (
	"bytes"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/pkg/s3utils"
	sha256 "github.com/storeros/ipos/pkg/sha256-simd"
)

const (
	signV4Algorithm = "AWS4-HMAC-SHA256"
	iso8601Format   = "20060102T150405Z"
	yyyymmdd        = "20060102"
)

type serviceType string

const (
	serviceS3  serviceType = "s3"
	serviceSTS serviceType = "sts"
)

func getCanonicalHeaders(signedHeaders http.Header) string {
	var headers []string
	vals := make(http.Header)
	for k, vv := range signedHeaders {
		headers = append(headers, strings.ToLower(k))
		vals[strings.ToLower(k)] = vv
	}
	sort.Strings(headers)

	var buf bytes.Buffer
	for _, k := range headers {
		buf.WriteString(k)
		buf.WriteByte(':')
		for idx, v := range vals[k] {
			if idx > 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(signV4TrimAll(v))
		}
		buf.WriteByte('\n')
	}
	return buf.String()
}

func getSignedHeaders(signedHeaders http.Header) string {
	var headers []string
	for k := range signedHeaders {
		headers = append(headers, strings.ToLower(k))
	}
	sort.Strings(headers)
	return strings.Join(headers, ";")
}

func getCanonicalRequest(extractedSignedHeaders http.Header, payload, queryStr, urlPath, method string) string {
	rawQuery := strings.Replace(queryStr, "+", "%20", -1)
	encodedPath := s3utils.EncodePath(urlPath)
	canonicalRequest := strings.Join([]string{
		method,
		encodedPath,
		rawQuery,
		getCanonicalHeaders(extractedSignedHeaders),
		getSignedHeaders(extractedSignedHeaders),
		payload,
	}, "\n")
	return canonicalRequest
}

func getScope(t time.Time, region string) string {
	scope := strings.Join([]string{
		t.Format(yyyymmdd),
		region,
		string(serviceS3),
		"aws4_request",
	}, SlashSeparator)
	return scope
}

func getStringToSign(canonicalRequest string, t time.Time, scope string) string {
	stringToSign := signV4Algorithm + "\n" + t.Format(iso8601Format) + "\n"
	stringToSign = stringToSign + scope + "\n"
	canonicalRequestBytes := sha256.Sum256([]byte(canonicalRequest))
	stringToSign = stringToSign + hex.EncodeToString(canonicalRequestBytes[:])
	return stringToSign
}

func getSigningKey(secretKey string, t time.Time, region string, stype serviceType) []byte {
	date := sumHMAC([]byte("AWS4"+secretKey), []byte(t.Format(yyyymmdd)))
	regionBytes := sumHMAC(date, []byte(region))
	service := sumHMAC(regionBytes, []byte(stype))
	signingKey := sumHMAC(service, []byte("aws4_request"))
	return signingKey
}

func getSignature(signingKey []byte, stringToSign string) string {
	return hex.EncodeToString(sumHMAC(signingKey, []byte(stringToSign)))
}

func doesPolicySignatureMatch(formValues http.Header) APIErrorCode {
	return doesPolicySignatureV4Match(formValues)
}

func compareSignatureV4(sig1, sig2 string) bool {
	return subtle.ConstantTimeCompare([]byte(sig1), []byte(sig2)) == 1
}

func doesPolicySignatureV4Match(formValues http.Header) APIErrorCode {
	region := globalServerRegion

	credHeader, err := parseCredentialHeader("Credential="+formValues.Get(xhttp.AmzCredential), region, serviceS3)
	if err != ErrNone {
		return ErrMissingFields
	}

	cred, _, s3Err := checkKeyValid(credHeader.accessKey)
	if s3Err != ErrNone {
		return s3Err
	}

	signingKey := getSigningKey(cred.SecretKey, credHeader.scope.date, credHeader.scope.region, serviceS3)

	newSignature := getSignature(signingKey, formValues.Get("Policy"))

	if !compareSignatureV4(newSignature, formValues.Get(xhttp.AmzSignature)) {
		return ErrSignatureDoesNotMatch
	}

	return ErrNone
}

func doesPresignedSignatureMatch(hashedPayload string, r *http.Request, region string, stype serviceType) APIErrorCode {
	req := *r

	pSignValues, err := parsePreSignV4(req.URL.Query(), region, stype)
	if err != ErrNone {
		return err
	}

	cred, _, s3Err := checkKeyValid(pSignValues.Credential.accessKey)
	if s3Err != ErrNone {
		return s3Err
	}

	extractedSignedHeaders, errCode := extractSignedHeaders(pSignValues.SignedHeaders, r)
	if errCode != ErrNone {
		return errCode
	}

	if pSignValues.Date.After(UTCNow().Add(globalMaxSkewTime)) {
		return ErrRequestNotReadyYet
	}

	if UTCNow().Sub(pSignValues.Date) > pSignValues.Expires {
		return ErrExpiredPresignRequest
	}

	t := pSignValues.Date
	expireSeconds := int(pSignValues.Expires / time.Second)

	query := make(url.Values)
	clntHashedPayload := req.URL.Query().Get(xhttp.AmzContentSha256)
	if clntHashedPayload != "" {
		query.Set(xhttp.AmzContentSha256, hashedPayload)
	}

	token := req.URL.Query().Get(xhttp.AmzSecurityToken)
	if token != "" {
		query.Set(xhttp.AmzSecurityToken, cred.SessionToken)
	}

	query.Set(xhttp.AmzAlgorithm, signV4Algorithm)

	query.Set(xhttp.AmzDate, t.Format(iso8601Format))
	query.Set(xhttp.AmzExpires, strconv.Itoa(expireSeconds))
	query.Set(xhttp.AmzSignedHeaders, getSignedHeaders(extractedSignedHeaders))
	query.Set(xhttp.AmzCredential, cred.AccessKey+SlashSeparator+pSignValues.Credential.getScope())

	for k, v := range req.URL.Query() {
		key := strings.ToLower(k)

		if strings.Contains(key, "x-amz-meta-") {
			query.Set(k, v[0])
			continue
		}

		if strings.Contains(key, "x-amz-server-side-") {
			query.Set(k, v[0])
			continue
		}

		if strings.HasPrefix(key, "x-amz") {
			continue
		}
		query[k] = v
	}

	encodedQuery := query.Encode()

	if req.URL.Query().Get(xhttp.AmzDate) != query.Get(xhttp.AmzDate) {
		return ErrSignatureDoesNotMatch
	}
	if req.URL.Query().Get(xhttp.AmzExpires) != query.Get(xhttp.AmzExpires) {
		return ErrSignatureDoesNotMatch
	}
	if req.URL.Query().Get(xhttp.AmzSignedHeaders) != query.Get(xhttp.AmzSignedHeaders) {
		return ErrSignatureDoesNotMatch
	}
	if req.URL.Query().Get(xhttp.AmzCredential) != query.Get(xhttp.AmzCredential) {
		return ErrSignatureDoesNotMatch
	}
	if clntHashedPayload != "" && clntHashedPayload != query.Get(xhttp.AmzContentSha256) {
		return ErrContentSHA256Mismatch
	}
	if token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(cred.SessionToken)) != 1 {
		return ErrInvalidToken
	}

	presignedCanonicalReq := getCanonicalRequest(extractedSignedHeaders, hashedPayload, encodedQuery, req.URL.Path, req.Method)

	presignedStringToSign := getStringToSign(presignedCanonicalReq, t, pSignValues.Credential.getScope())

	presignedSigningKey := getSigningKey(cred.SecretKey, pSignValues.Credential.scope.date,
		pSignValues.Credential.scope.region, stype)

	newSignature := getSignature(presignedSigningKey, presignedStringToSign)

	if !compareSignatureV4(req.URL.Query().Get(xhttp.AmzSignature), newSignature) {
		return ErrSignatureDoesNotMatch
	}
	return ErrNone
}

func doesSignatureMatch(hashedPayload string, r *http.Request, region string, stype serviceType) APIErrorCode {
	req := *r

	v4Auth := req.Header.Get(xhttp.Authorization)

	signV4Values, err := parseSignV4(v4Auth, region, stype)
	if err != ErrNone {
		return err
	}

	extractedSignedHeaders, errCode := extractSignedHeaders(signV4Values.SignedHeaders, r)
	if errCode != ErrNone {
		return errCode
	}

	cred, _, s3Err := checkKeyValid(signV4Values.Credential.accessKey)
	if s3Err != ErrNone {
		return s3Err
	}

	var date string
	if date = req.Header.Get(xhttp.AmzDate); date == "" {
		if date = r.Header.Get(xhttp.Date); date == "" {
			return ErrMissingDateHeader
		}
	}

	t, e := time.Parse(iso8601Format, date)
	if e != nil {
		return ErrMalformedDate
	}

	queryStr := req.URL.Query().Encode()

	canonicalRequest := getCanonicalRequest(extractedSignedHeaders, hashedPayload, queryStr, req.URL.Path, req.Method)

	stringToSign := getStringToSign(canonicalRequest, t, signV4Values.Credential.getScope())

	signingKey := getSigningKey(cred.SecretKey, signV4Values.Credential.scope.date,
		signV4Values.Credential.scope.region, stype)

	newSignature := getSignature(signingKey, stringToSign)

	if !compareSignatureV4(newSignature, signV4Values.Signature) {
		return ErrSignatureDoesNotMatch
	}

	return ErrNone
}
