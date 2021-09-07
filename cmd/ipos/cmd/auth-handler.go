package cmd

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	xjwt "github.com/storeros/ipos/cmd/ipos/jwt"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/auth"
	objectlock "github.com/storeros/ipos/pkg/bucket/object/lock"
	"github.com/storeros/ipos/pkg/bucket/policy"
	"github.com/storeros/ipos/pkg/hash"
	iampolicy "github.com/storeros/ipos/pkg/iam/policy"
)

func isRequestJWT(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get(xhttp.Authorization), jwtAlgorithm)
}

func isRequestSignatureV4(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get(xhttp.Authorization), signV4Algorithm)
}

func isRequestSignatureV2(r *http.Request) bool {
	return (!strings.HasPrefix(r.Header.Get(xhttp.Authorization), signV4Algorithm) &&
		strings.HasPrefix(r.Header.Get(xhttp.Authorization), signV2Algorithm))
}

func isRequestPresignedSignatureV4(r *http.Request) bool {
	_, ok := r.URL.Query()[xhttp.AmzCredential]
	return ok
}

func isRequestPresignedSignatureV2(r *http.Request) bool {
	_, ok := r.URL.Query()[xhttp.AmzAccessKeyID]
	return ok
}

func isRequestPostPolicySignatureV4(r *http.Request) bool {
	return strings.Contains(r.Header.Get(xhttp.ContentType), "multipart/form-data") &&
		r.Method == http.MethodPost
}

func isRequestSignStreamingV4(r *http.Request) bool {
	return r.Header.Get(xhttp.AmzContentSha256) == streamingContentSHA256 &&
		r.Method == http.MethodPut
}

type authType int

const (
	authTypeUnknown authType = iota
	authTypeAnonymous
	authTypePresigned
	authTypePresignedV2
	authTypePostPolicy
	authTypeStreamingSigned
	authTypeSigned
	authTypeSignedV2
	authTypeJWT
	authTypeSTS
)

func getRequestAuthType(r *http.Request) authType {
	if isRequestSignatureV2(r) {
		return authTypeSignedV2
	} else if isRequestPresignedSignatureV2(r) {
		return authTypePresignedV2
	} else if isRequestSignStreamingV4(r) {
		return authTypeStreamingSigned
	} else if isRequestSignatureV4(r) {
		return authTypeSigned
	} else if isRequestPresignedSignatureV4(r) {
		return authTypePresigned
	} else if isRequestJWT(r) {
		return authTypeJWT
	} else if isRequestPostPolicySignatureV4(r) {
		return authTypePostPolicy
	} else if _, ok := r.URL.Query()[xhttp.Action]; ok {
		return authTypeSTS
	} else if _, ok := r.Header[xhttp.Authorization]; !ok {
		return authTypeAnonymous
	}
	return authTypeUnknown
}

func validateAdminSignature(ctx context.Context, r *http.Request, region string) (auth.Credentials, map[string]interface{}, bool, APIErrorCode) {
	var cred auth.Credentials
	var owner bool
	s3Err := ErrAccessDenied
	if _, ok := r.Header[xhttp.AmzContentSha256]; ok &&
		getRequestAuthType(r) == authTypeSigned && !skipContentSha256Cksum(r) {
		cred, owner, s3Err = getReqAccessKeyV4(r, region, serviceS3)
		if s3Err != ErrNone {
			return cred, nil, owner, s3Err
		}

		s3Err = isReqAuthenticated(ctx, r, region, serviceS3)
	}
	if s3Err != ErrNone {
		reqInfo := (&logger.ReqInfo{}).AppendTags("requestHeaders", dumpRequest(r))
		ctx := logger.SetReqInfo(ctx, reqInfo)
		logger.LogIf(ctx, errors.New(getAPIError(s3Err).Description), logger.Application)
		return cred, nil, owner, s3Err
	}

	claims, s3Err := checkClaimsFromToken(r, cred)
	if s3Err != ErrNone {
		return cred, nil, owner, s3Err
	}

	return cred, claims, owner, ErrNone
}

func checkAdminRequestAuthType(ctx context.Context, r *http.Request, action iampolicy.AdminAction, region string) (auth.Credentials, APIErrorCode) {
	cred, claims, owner, s3Err := validateAdminSignature(ctx, r, region)
	if s3Err != ErrNone {
		return cred, s3Err
	}
	if globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     cred.AccessKey,
		Action:          iampolicy.Action(action),
		ConditionValues: getConditionValues(r, "", cred.AccessKey, claims),
		IsOwner:         owner,
		Claims:          claims,
	}) {
		return cred, ErrNone
	}

	return cred, ErrAccessDenied
}

func getSessionToken(r *http.Request) (token string) {
	token = r.Header.Get(xhttp.AmzSecurityToken)
	if token != "" {
		return token
	}
	return r.URL.Query().Get(xhttp.AmzSecurityToken)
}

func mustGetClaimsFromToken(r *http.Request) map[string]interface{} {
	claims, _ := getClaimsFromToken(r, getSessionToken(r))
	return claims
}

func getClaimsFromToken(r *http.Request, token string) (map[string]interface{}, error) {
	claims := xjwt.NewMapClaims()
	if token == "" {
		return claims.Map(), nil
	}

	stsTokenCallback := func(claims *xjwt.MapClaims) ([]byte, error) {
		return []byte(globalActiveCred.SecretKey), nil
	}

	if err := xjwt.ParseWithClaims(token, claims, stsTokenCallback); err != nil {
		return nil, errAuthentication
	}

	return claims.Map(), nil
}

func checkClaimsFromToken(r *http.Request, cred auth.Credentials) (map[string]interface{}, APIErrorCode) {
	token := getSessionToken(r)
	if token != "" && cred.AccessKey == "" {
		return nil, ErrNoAccessKey
	}
	if cred.IsServiceAccount() && token == "" {
		token = cred.SessionToken
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(cred.SessionToken)) != 1 {
		return nil, ErrInvalidToken
	}
	claims, err := getClaimsFromToken(r, token)
	if err != nil {
		return nil, toAPIErrorCode(r.Context(), err)
	}
	return claims, ErrNone
}

func checkRequestAuthType(ctx context.Context, r *http.Request, action policy.Action, bucketName, objectName string) (s3Err APIErrorCode) {
	_, _, s3Err = checkRequestAuthTypeToAccessKey(ctx, r, action, bucketName, objectName)
	return s3Err
}

func checkRequestAuthTypeToAccessKey(ctx context.Context, r *http.Request, action policy.Action, bucketName, objectName string) (accessKey string, owner bool, s3Err APIErrorCode) {
	var cred auth.Credentials
	switch getRequestAuthType(r) {
	case authTypeUnknown, authTypeStreamingSigned:
		return accessKey, owner, ErrSignatureVersionNotSupported
	case authTypeSigned:
		region := globalServerRegion
		switch action {
		case policy.GetBucketLocationAction, policy.ListAllMyBucketsAction:
			region = ""
		}
		if s3Err = isReqAuthenticated(ctx, r, region, serviceS3); s3Err != ErrNone {
			return accessKey, owner, s3Err
		}
		cred, owner, s3Err = getReqAccessKeyV4(r, region, serviceS3)
	}
	if s3Err != ErrNone {
		return accessKey, owner, s3Err
	}

	var claims map[string]interface{}
	claims, s3Err = checkClaimsFromToken(r, cred)
	if s3Err != ErrNone {
		return accessKey, owner, s3Err
	}

	var locationConstraint string
	if action == policy.CreateBucketAction {
		payload, err := ioutil.ReadAll(io.LimitReader(r.Body, maxLocationConstraintSize))
		if err != nil {
			logger.LogIf(ctx, err, logger.Application)
			return accessKey, owner, ErrMalformedXML
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(payload))

		var s3Error APIErrorCode
		locationConstraint, s3Error = parseLocationConstraint(r)
		if s3Error != ErrNone {
			return accessKey, owner, s3Error
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(payload))
	}

	if cred.AccessKey == "" {
		if globalPolicySys.IsAllowed(policy.Args{
			AccountName:     cred.AccessKey,
			Action:          action,
			BucketName:      bucketName,
			ConditionValues: getConditionValues(r, locationConstraint, "", nil),
			IsOwner:         false,
			ObjectName:      objectName,
		}) {
			return cred.AccessKey, owner, ErrNone
		}
		return cred.AccessKey, owner, ErrAccessDenied
	}
	if globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     cred.AccessKey,
		Action:          iampolicy.Action(action),
		BucketName:      bucketName,
		ConditionValues: getConditionValues(r, "", cred.AccessKey, claims),
		ObjectName:      objectName,
		IsOwner:         owner,
		Claims:          claims,
	}) {
		return cred.AccessKey, owner, ErrNone
	}
	return cred.AccessKey, owner, ErrAccessDenied
}

func reqSignatureV4Verify(r *http.Request, region string, stype serviceType) (s3Error APIErrorCode) {
	sha256sum := getContentSha256Cksum(r, stype)
	switch {
	case isRequestSignatureV4(r):
		return doesSignatureMatch(sha256sum, r, region, stype)
	case isRequestPresignedSignatureV4(r):
		return doesPresignedSignatureMatch(sha256sum, r, region, stype)
	default:
		return ErrAccessDenied
	}
}

func isReqAuthenticated(ctx context.Context, r *http.Request, region string, stype serviceType) (s3Error APIErrorCode) {
	if errCode := reqSignatureV4Verify(r, region, stype); errCode != ErrNone {
		return errCode
	}

	var (
		err                       error
		contentMD5, contentSHA256 []byte
	)

	contentMD5, err = checkValidMD5(r.Header)
	if err != nil {
		return ErrInvalidDigest
	}

	if skipSHA256 := skipContentSha256Cksum(r); !skipSHA256 && isRequestPresignedSignatureV4(r) {
		if sha256Sum, ok := r.URL.Query()[xhttp.AmzContentSha256]; ok && len(sha256Sum) > 0 {
			contentSHA256, err = hex.DecodeString(sha256Sum[0])
			if err != nil {
				return ErrContentSHA256Mismatch
			}
		}
	} else if _, ok := r.Header[xhttp.AmzContentSha256]; !skipSHA256 && ok {
		contentSHA256, err = hex.DecodeString(r.Header.Get(xhttp.AmzContentSha256))
		if err != nil || len(contentSHA256) == 0 {
			return ErrContentSHA256Mismatch
		}
	}

	reader, err := hash.NewReader(r.Body, -1, hex.EncodeToString(contentMD5),
		hex.EncodeToString(contentSHA256), -1, globalCLIContext.StrictS3Compat)
	if err != nil {
		return toAPIErrorCode(ctx, err)
	}
	r.Body = ioutil.NopCloser(reader)
	return ErrNone
}

type authHandler struct {
	handler http.Handler
}

func setAuthHandler(h http.Handler) http.Handler {
	return authHandler{h}
}

var supportedS3AuthTypes = map[authType]struct{}{
	authTypeSigned:          {},
	authTypeStreamingSigned: {},
}

func isSupportedS3AuthType(aType authType) bool {
	_, ok := supportedS3AuthTypes[aType]
	return ok
}

func (a authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	aType := getRequestAuthType(r)
	if isSupportedS3AuthType(aType) {
		a.handler.ServeHTTP(w, r)
		return
	} else if aType == authTypeJWT {
		if _, _, authErr := webRequestAuthenticate(r); authErr != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		a.handler.ServeHTTP(w, r)
		return
	} else if aType == authTypeSTS {
		a.handler.ServeHTTP(w, r)
		return
	}
	writeErrorResponse(r.Context(), w, errorCodes.ToAPIErr(ErrSignatureVersionNotSupported), r.URL, guessIsBrowserReq(r))
}

func validateSignature(atype authType, r *http.Request) (auth.Credentials, bool, map[string]interface{}, APIErrorCode) {
	var cred auth.Credentials
	var owner bool
	var s3Err APIErrorCode
	switch atype {
	case authTypeUnknown, authTypeStreamingSigned:
		return cred, owner, nil, ErrSignatureVersionNotSupported
	case authTypeSigned:
		region := globalServerRegion
		if s3Err = isReqAuthenticated(GlobalContext, r, region, serviceS3); s3Err != ErrNone {
			return cred, owner, nil, s3Err
		}
		cred, owner, s3Err = getReqAccessKeyV4(r, region, serviceS3)
	}
	if s3Err != ErrNone {
		return cred, owner, nil, s3Err
	}

	claims, s3Err := checkClaimsFromToken(r, cred)
	if s3Err != ErrNone {
		return cred, owner, nil, s3Err
	}

	return cred, owner, claims, ErrNone
}

func isPutRetentionAllowed(bucketName, objectName string, retDays int, retDate time.Time, retMode objectlock.RetMode, byPassSet bool, r *http.Request, cred auth.Credentials, owner bool, claims map[string]interface{}) (s3Err APIErrorCode) {
	var retSet bool
	if cred.AccessKey == "" {
		conditions := getConditionValues(r, "", "", nil)
		conditions["object-lock-mode"] = []string{string(retMode)}
		conditions["object-lock-retain-until-date"] = []string{retDate.Format(time.RFC3339)}
		if retDays > 0 {
			conditions["object-lock-remaining-retention-days"] = []string{strconv.Itoa(retDays)}
		}
		if retMode == objectlock.RetGovernance && byPassSet {
			byPassSet = globalPolicySys.IsAllowed(policy.Args{
				AccountName:     cred.AccessKey,
				Action:          policy.Action(policy.BypassGovernanceRetentionAction),
				BucketName:      bucketName,
				ConditionValues: conditions,
				IsOwner:         false,
				ObjectName:      objectName,
			})
		}
		if globalPolicySys.IsAllowed(policy.Args{
			AccountName:     cred.AccessKey,
			Action:          policy.Action(policy.PutObjectRetentionAction),
			BucketName:      bucketName,
			ConditionValues: conditions,
			IsOwner:         false,
			ObjectName:      objectName,
		}) {
			retSet = true
		}
		if byPassSet || retSet {
			return ErrNone
		}
		return ErrAccessDenied
	}

	conditions := getConditionValues(r, "", cred.AccessKey, claims)
	conditions["object-lock-mode"] = []string{string(retMode)}
	conditions["object-lock-retain-until-date"] = []string{retDate.Format(time.RFC3339)}
	if retDays > 0 {
		conditions["object-lock-remaining-retention-days"] = []string{strconv.Itoa(retDays)}
	}
	if retMode == objectlock.RetGovernance && byPassSet {
		byPassSet = globalIAMSys.IsAllowed(iampolicy.Args{
			AccountName:     cred.AccessKey,
			Action:          policy.BypassGovernanceRetentionAction,
			BucketName:      bucketName,
			ObjectName:      objectName,
			ConditionValues: conditions,
			IsOwner:         owner,
			Claims:          claims,
		})
	}
	if globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     cred.AccessKey,
		Action:          policy.PutObjectRetentionAction,
		BucketName:      bucketName,
		ConditionValues: conditions,
		ObjectName:      objectName,
		IsOwner:         owner,
		Claims:          claims,
	}) {
		retSet = true
	}
	if byPassSet || retSet {
		return ErrNone
	}
	return ErrAccessDenied
}

func isPutActionAllowed(atype authType, bucketName, objectName string, r *http.Request, action iampolicy.Action) (s3Err APIErrorCode) {
	var cred auth.Credentials
	var owner bool
	switch atype {
	case authTypeUnknown:
		return ErrSignatureVersionNotSupported
	case authTypeStreamingSigned, authTypeSigned:
		region := globalServerRegion
		cred, owner, s3Err = getReqAccessKeyV4(r, region, serviceS3)
	}
	if s3Err != ErrNone {
		return s3Err
	}

	claims, s3Err := checkClaimsFromToken(r, cred)
	if s3Err != ErrNone {
		return s3Err
	}

	if action == iampolicy.PutObjectRetentionAction &&
		r.Header.Get(xhttp.AmzObjectLockMode) == "" &&
		r.Header.Get(xhttp.AmzObjectLockRetainUntilDate) == "" {
		return ErrNone
	}

	if cred.AccessKey == "" {
		if globalPolicySys.IsAllowed(policy.Args{
			AccountName:     cred.AccessKey,
			Action:          policy.Action(action),
			BucketName:      bucketName,
			ConditionValues: getConditionValues(r, "", "", nil),
			IsOwner:         false,
			ObjectName:      objectName,
		}) {
			return ErrNone
		}
		return ErrAccessDenied
	}

	if globalIAMSys.IsAllowed(iampolicy.Args{
		AccountName:     cred.AccessKey,
		Action:          action,
		BucketName:      bucketName,
		ConditionValues: getConditionValues(r, "", cred.AccessKey, claims),
		ObjectName:      objectName,
		IsOwner:         owner,
		Claims:          claims,
	}) {
		return ErrNone
	}
	return ErrAccessDenied
}
