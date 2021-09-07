package cmd

import (
	"net/http"
	"path"
	"strings"

	"github.com/gorilla/mux"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	objectlock "github.com/storeros/ipos/pkg/bucket/object/lock"
	"github.com/storeros/ipos/pkg/bucket/policy"
	iampolicy "github.com/storeros/ipos/pkg/iam/policy"
)

const (
	objectLockConfig                  = "object-lock.xml"
	bucketObjectLockEnabledConfigFile = "object-lock-enabled.json"
	bucketObjectLockEnabledConfig     = `{"x-amz-bucket-object-lock-enabled":true}`
)

func (api objectAPIHandlers) GetBucketLocationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "GetBucketLocation")

	defer logger.AuditLog(w, r, "GetBucketLocation", mustGetClaimsFromToken(r))

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}

	if s3Error := checkRequestAuthType(ctx, r, policy.GetBucketLocationAction, bucket, ""); s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	getBucketInfo := objectAPI.GetBucketInfo

	if _, err := getBucketInfo(ctx, bucket); err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	encodedSuccessResponse := encodeResponse(LocationResponse{})
	region := globalServerRegion
	if region != globalIPOSDefaultRegion {
		encodedSuccessResponse = encodeResponse(LocationResponse{
			Location: region,
		})
	}

	writeSuccessResponseXML(w, encodedSuccessResponse)
}

func (api objectAPIHandlers) ListBucketsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "ListBuckets")

	defer logger.AuditLog(w, r, "ListBuckets", mustGetClaimsFromToken(r))

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}

	listBuckets := objectAPI.ListBuckets

	accessKey, owner, s3Error := checkRequestAuthTypeToAccessKey(ctx, r, policy.ListAllMyBucketsAction, "", "")
	if s3Error != ErrNone && s3Error != ErrAccessDenied {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	var bucketsInfo []BucketInfo
	var err error
	bucketsInfo, err = listBuckets(ctx)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	if s3Error == ErrAccessDenied {
		r.Header.Set("prefix", "")

		r.Header.Set("delimiter", SlashSeparator)

		claims, _ := getClaimsFromToken(r, getSessionToken(r))
		n := 0
		for _, bucketInfo := range bucketsInfo {
			if globalIAMSys.IsAllowed(iampolicy.Args{
				AccountName:     accessKey,
				Action:          iampolicy.ListBucketAction,
				BucketName:      bucketInfo.Name,
				ConditionValues: getConditionValues(r, "", accessKey, claims),
				IsOwner:         owner,
				ObjectName:      "",
				Claims:          claims,
			}) {
				bucketsInfo[n] = bucketInfo
				n++
			}
		}
		bucketsInfo = bucketsInfo[:n]
		if len(bucketsInfo) == 0 {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	response := generateListBucketsResponse(bucketsInfo)
	encodedSuccessResponse := encodeResponse(response)

	writeSuccessResponseXML(w, encodedSuccessResponse)
}

func (api objectAPIHandlers) DeleteMultipleObjectsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "DeleteMultipleObjects")

	defer logger.AuditLog(w, r, "DeleteMultipleObjects", mustGetClaimsFromToken(r))

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}

	if _, ok := r.Header[xhttp.ContentMD5]; !ok {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMissingContentMD5), r.URL, guessIsBrowserReq(r))
		return
	}

	if r.ContentLength <= 0 {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMissingContentLength), r.URL, guessIsBrowserReq(r))
		return
	}

	const maxBodySize = 2 * 100000 * 1024

	deleteObjects := &DeleteObjectsRequest{}
	if err := xmlDecoder(r.Body, deleteObjects, maxBodySize); err != nil {
		logger.LogIf(ctx, err, logger.Application)
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	deleteObjectsFn := objectAPI.DeleteObjects
	var objectsToDelete = map[string]int{}
	getObjectInfoFn := objectAPI.GetObjectInfo
	var dErrs = make([]APIErrorCode, len(deleteObjects.Objects))

	for index, object := range deleteObjects.Objects {
		if dErrs[index] = checkRequestAuthType(ctx, r, policy.DeleteObjectAction, bucket, object.ObjectName); dErrs[index] != ErrNone {
			if dErrs[index] == ErrSignatureDoesNotMatch || dErrs[index] == ErrInvalidAccessKeyID {
				writeErrorResponse(ctx, w, errorCodes.ToAPIErr(dErrs[index]), r.URL, guessIsBrowserReq(r))
				return
			}
			continue
		}

		if _, ok := globalBucketObjectLockConfig.Get(bucket); ok {
			if err := enforceRetentionBypassForDelete(ctx, r, bucket, object.ObjectName, getObjectInfoFn); err != ErrNone {
				dErrs[index] = err
				continue
			}
		}

		if _, ok := objectsToDelete[object.ObjectName]; !ok {
			objectsToDelete[object.ObjectName] = index
		}
	}

	toNames := func(input map[string]int) (output []string) {
		output = make([]string, len(input))
		idx := 0
		for name := range input {
			output[idx] = name
			idx++
		}
		return
	}

	deleteList := toNames(objectsToDelete)
	errs, err := deleteObjectsFn(ctx, bucket, deleteList)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	for i, objName := range deleteList {
		dIdx := objectsToDelete[objName]
		dErrs[dIdx] = toAPIErrorCode(ctx, errs[i])
	}

	var deletedObjects []ObjectIdentifier
	var deleteErrors []DeleteError
	for index, errCode := range dErrs {
		object := deleteObjects.Objects[index]
		if errCode == ErrNone || errCode == ErrNoSuchKey {
			deletedObjects = append(deletedObjects, object)
			continue
		}
		apiErr := getAPIError(errCode)
		deleteErrors = append(deleteErrors, DeleteError{
			Code:    apiErr.Code,
			Message: apiErr.Description,
			Key:     object.ObjectName,
		})
	}

	response := generateMultiDeleteResponse(deleteObjects.Quiet, deletedObjects, deleteErrors)
	encodedSuccessResponse := encodeResponse(response)

	writeSuccessResponseXML(w, encodedSuccessResponse)
}

func (api objectAPIHandlers) PutBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "PutBucket")

	defer logger.AuditLog(w, r, "PutBucket", mustGetClaimsFromToken(r))

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectLockEnabled := false
	if vs, found := r.Header[http.CanonicalHeaderKey("x-amz-bucket-object-lock-enabled")]; found {
		v := strings.ToLower(strings.Join(vs, ""))
		if v != "true" && v != "false" {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidRequest), r.URL, guessIsBrowserReq(r))
			return
		}
		objectLockEnabled = v == "true"
	}

	if s3Error := checkRequestAuthType(ctx, r, policy.CreateBucketAction, bucket, ""); s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	location, s3Error := parseLocationConstraint(r)
	if s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	if !isValidLocation(location) {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidRegion), r.URL, guessIsBrowserReq(r))
		return
	}

	err := objectAPI.MakeBucketWithLocation(ctx, bucket, location)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	if objectLockEnabled {
		configFile := path.Join(bucketConfigPrefix, bucket, bucketObjectLockEnabledConfigFile)
		if err = saveConfig(ctx, objectAPI, configFile, []byte(bucketObjectLockEnabledConfig)); err != nil {
			writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
			return
		}
		globalBucketObjectLockConfig.Set(bucket, objectlock.Retention{})
	}

	w.Header().Set(xhttp.Location, path.Clean(r.URL.Path))

	writeSuccessResponseHeadersOnly(w)
}

func (api objectAPIHandlers) HeadBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "HeadBucket")

	defer logger.AuditLog(w, r, "HeadBucket", mustGetClaimsFromToken(r))

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponseHeadersOnly(w, errorCodes.ToAPIErr(ErrServerNotInitialized))
		return
	}

	if s3Error := checkRequestAuthType(ctx, r, policy.ListBucketAction, bucket, ""); s3Error != ErrNone {
		writeErrorResponseHeadersOnly(w, errorCodes.ToAPIErr(s3Error))
		return
	}

	getBucketInfo := objectAPI.GetBucketInfo

	if _, err := getBucketInfo(ctx, bucket); err != nil {
		writeErrorResponseHeadersOnly(w, toAPIError(ctx, err))
		return
	}

	writeSuccessResponseHeadersOnly(w)
}

func (api objectAPIHandlers) DeleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "DeleteBucket")

	defer logger.AuditLog(w, r, "DeleteBucket", mustGetClaimsFromToken(r))

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}

	forceDelete := false
	if value := r.Header.Get(xhttp.IPOSForceDelete); value != "" {
		switch value {
		case "true":
			forceDelete = true
		case "false":
		default:
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidRequest), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	if forceDelete {
		if s3Error := checkRequestAuthType(ctx, r, policy.ForceDeleteBucketAction, bucket, ""); s3Error != ErrNone {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
			return
		}
	} else {
		if s3Error := checkRequestAuthType(ctx, r, policy.DeleteBucketAction, bucket, ""); s3Error != ErrNone {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	if _, ok := globalBucketObjectLockConfig.Get(bucket); ok && forceDelete {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMethodNotAllowed), r.URL, guessIsBrowserReq(r))
		return
	}

	deleteBucket := objectAPI.DeleteBucket

	if err := deleteBucket(ctx, bucket, forceDelete); err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	writeSuccessNoContent(w)
}
