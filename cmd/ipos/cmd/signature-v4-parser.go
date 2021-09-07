package cmd

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/storeros/ipos/pkg/auth"
)

type credentialHeader struct {
	accessKey string
	scope     struct {
		date    time.Time
		region  string
		service string
		request string
	}
}

func (c credentialHeader) getScope() string {
	return strings.Join([]string{
		c.scope.date.Format(yyyymmdd),
		c.scope.region,
		c.scope.service,
		c.scope.request,
	}, SlashSeparator)
}

func getReqAccessKeyV4(r *http.Request, region string, stype serviceType) (auth.Credentials, bool, APIErrorCode) {
	ch, err := parseCredentialHeader("Credential="+r.URL.Query().Get("X-Amz-Credential"), region, stype)
	if err != ErrNone {
		v4Auth := strings.TrimPrefix(r.Header.Get("Authorization"), signV4Algorithm)
		authFields := strings.Split(strings.TrimSpace(v4Auth), ",")
		if len(authFields) != 3 {
			return auth.Credentials{}, false, ErrMissingFields
		}
		ch, err = parseCredentialHeader(authFields[0], region, stype)
		if err != ErrNone {
			return auth.Credentials{}, false, err
		}
	}
	return checkKeyValid(ch.accessKey)
}

func parseCredentialHeader(credElement string, region string, stype serviceType) (ch credentialHeader, aec APIErrorCode) {
	creds := strings.SplitN(strings.TrimSpace(credElement), "=", 2)
	if len(creds) != 2 {
		return ch, ErrMissingFields
	}
	if creds[0] != "Credential" {
		return ch, ErrMissingCredTag
	}
	credElements := strings.Split(strings.TrimSpace(creds[1]), SlashSeparator)
	if len(credElements) < 5 {
		return ch, ErrCredMalformed
	}
	accessKey := strings.Join(credElements[:len(credElements)-4], SlashSeparator)
	if !auth.IsAccessKeyValid(accessKey) {
		return ch, ErrInvalidAccessKeyID
	}
	cred := credentialHeader{
		accessKey: accessKey,
	}
	credElements = credElements[len(credElements)-4:]
	var e error
	cred.scope.date, e = time.Parse(yyyymmdd, credElements[0])
	if e != nil {
		return ch, ErrMalformedCredentialDate
	}

	cred.scope.region = credElements[1]
	sRegion := cred.scope.region
	if region == "" {
		region = sRegion
	}
	if !isValidRegion(sRegion, region) {
		return ch, ErrAuthorizationHeaderMalformed

	}
	if credElements[2] != string(stype) {
		switch stype {
		case serviceSTS:
			return ch, ErrInvalidServiceSTS
		}
		return ch, ErrInvalidServiceS3
	}
	cred.scope.service = credElements[2]
	if credElements[3] != "aws4_request" {
		return ch, ErrInvalidRequestVersion
	}
	cred.scope.request = credElements[3]
	return cred, ErrNone
}

func parseSignature(signElement string) (string, APIErrorCode) {
	signFields := strings.Split(strings.TrimSpace(signElement), "=")
	if len(signFields) != 2 {
		return "", ErrMissingFields
	}
	if signFields[0] != "Signature" {
		return "", ErrMissingSignTag
	}
	if signFields[1] == "" {
		return "", ErrMissingFields
	}
	signature := signFields[1]
	return signature, ErrNone
}

func parseSignedHeader(signedHdrElement string) ([]string, APIErrorCode) {
	signedHdrFields := strings.Split(strings.TrimSpace(signedHdrElement), "=")
	if len(signedHdrFields) != 2 {
		return nil, ErrMissingFields
	}
	if signedHdrFields[0] != "SignedHeaders" {
		return nil, ErrMissingSignHeadersTag
	}
	if signedHdrFields[1] == "" {
		return nil, ErrMissingFields
	}
	signedHeaders := strings.Split(signedHdrFields[1], ";")
	return signedHeaders, ErrNone
}

type signValues struct {
	Credential    credentialHeader
	SignedHeaders []string
	Signature     string
}

type preSignValues struct {
	signValues
	Date    time.Time
	Expires time.Duration
}

func doesV4PresignParamsExist(query url.Values) APIErrorCode {
	v4PresignQueryParams := []string{"X-Amz-Algorithm", "X-Amz-Credential", "X-Amz-Signature", "X-Amz-Date", "X-Amz-SignedHeaders", "X-Amz-Expires"}
	for _, v4PresignQueryParam := range v4PresignQueryParams {
		if _, ok := query[v4PresignQueryParam]; !ok {
			return ErrInvalidQueryParams
		}
	}
	return ErrNone
}

func parsePreSignV4(query url.Values, region string, stype serviceType) (psv preSignValues, aec APIErrorCode) {
	err := doesV4PresignParamsExist(query)
	if err != ErrNone {
		return psv, err
	}

	if query.Get("X-Amz-Algorithm") != signV4Algorithm {
		return psv, ErrInvalidQuerySignatureAlgo
	}

	preSignV4Values := preSignValues{}

	preSignV4Values.Credential, err = parseCredentialHeader("Credential="+query.Get("X-Amz-Credential"), region, stype)
	if err != ErrNone {
		return psv, err
	}

	var e error
	preSignV4Values.Date, e = time.Parse(iso8601Format, query.Get("X-Amz-Date"))
	if e != nil {
		return psv, ErrMalformedPresignedDate
	}

	preSignV4Values.Expires, e = time.ParseDuration(query.Get("X-Amz-Expires") + "s")
	if e != nil {
		return psv, ErrMalformedExpires
	}

	if preSignV4Values.Expires < 0 {
		return psv, ErrNegativeExpires
	}

	if preSignV4Values.Expires.Seconds() > 604800 {
		return psv, ErrMaximumExpires
	}

	preSignV4Values.SignedHeaders, err = parseSignedHeader("SignedHeaders=" + query.Get("X-Amz-SignedHeaders"))
	if err != ErrNone {
		return psv, err
	}

	preSignV4Values.Signature, err = parseSignature("Signature=" + query.Get("X-Amz-Signature"))
	if err != ErrNone {
		return psv, err
	}

	return preSignV4Values, ErrNone
}

func parseSignV4(v4Auth string, region string, stype serviceType) (sv signValues, aec APIErrorCode) {
	credElement := strings.TrimPrefix(strings.Split(strings.TrimSpace(v4Auth), ",")[0], signV4Algorithm)
	v4Auth = strings.Replace(v4Auth, " ", "", -1)
	if v4Auth == "" {
		return sv, ErrAuthHeaderEmpty
	}

	if !strings.HasPrefix(v4Auth, signV4Algorithm) {
		return sv, ErrSignatureVersionNotSupported
	}

	v4Auth = strings.TrimPrefix(v4Auth, signV4Algorithm)
	authFields := strings.Split(strings.TrimSpace(v4Auth), ",")
	if len(authFields) != 3 {
		return sv, ErrMissingFields
	}

	signV4Values := signValues{}

	var err APIErrorCode
	signV4Values.Credential, err = parseCredentialHeader(strings.TrimSpace(credElement), region, stype)
	if err != ErrNone {
		return sv, err
	}

	signV4Values.SignedHeaders, err = parseSignedHeader(authFields[1])
	if err != ErrNone {
		return sv, err
	}

	signV4Values.Signature, err = parseSignature(authFields[2])
	if err != ErrNone {
		return sv, err
	}

	return signV4Values, ErrNone
}
