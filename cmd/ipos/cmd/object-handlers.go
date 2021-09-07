package cmd

import (
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/storeros/ipos/cmd/ipos/crypto"
	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	objectlock "github.com/storeros/ipos/pkg/bucket/object/lock"
	"github.com/storeros/ipos/pkg/bucket/policy"
	"github.com/storeros/ipos/pkg/hash"
	iampolicy "github.com/storeros/ipos/pkg/iam/policy"
	"github.com/storeros/ipos/pkg/ioutil"
)

var supportedHeadGetReqParams = map[string]string{
	"response-expires":             xhttp.Expires,
	"response-content-type":        xhttp.ContentType,
	"response-cache-control":       xhttp.CacheControl,
	"response-content-encoding":    xhttp.ContentEncoding,
	"response-content-language":    xhttp.ContentLanguage,
	"response-content-disposition": xhttp.ContentDisposition,
}

const (
	compressionAlgorithmV1 = "golang/snappy/LZ77"
	compressionAlgorithmV2 = "klauspost/compress/s2"

	encryptBufferThreshold = 1 << 20
	encryptBufferSize      = 1 << 20
)

func setHeadGetRespHeaders(w http.ResponseWriter, reqParams url.Values) {
	for k, v := range reqParams {
		if header, ok := supportedHeadGetReqParams[strings.ToLower(k)]; ok {
			w.Header()[header] = v
		}
	}
}

func (api objectAPIHandlers) GetObjectHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "GetObject")

	defer logger.AuditLog(w, r, "GetObject", mustGetClaimsFromToken(r))

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}
	if crypto.S3.IsRequested(r.Header) || crypto.S3KMS.IsRequested(r.Header) {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrBadRequest), r.URL, guessIsBrowserReq(r))
		return
	}
	if !api.EncryptionEnabled() && crypto.IsRequested(r.Header) {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrBadRequest), r.URL, guessIsBrowserReq(r))
		return
	}
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object, err := url.PathUnescape(vars["object"])
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	if vid := r.URL.Query().Get("versionId"); vid != "" && vid != "null" {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrNoSuchVersion), r.URL, guessIsBrowserReq(r))
		return
	}

	opts, err := getOpts(ctx, r, bucket, object)
	if err != nil {
		writeErrorResponseHeadersOnly(w, toAPIError(ctx, err))
		return
	}

	if s3Error := checkRequestAuthType(ctx, r, policy.GetObjectAction, bucket, object); s3Error != ErrNone {
		if getRequestAuthType(r) == authTypeAnonymous {
			if globalPolicySys.IsAllowed(policy.Args{
				Action:          policy.ListBucketAction,
				BucketName:      bucket,
				ConditionValues: getConditionValues(r, "", "", nil),
				IsOwner:         false,
			}) {
				getObjectInfo := objectAPI.GetObjectInfo

				_, err = getObjectInfo(ctx, bucket, object, opts)
				if toAPIError(ctx, err).Code == "NoSuchKey" {
					s3Error = ErrNoSuchKey
				}
			}
		}
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	getObjectNInfo := objectAPI.GetObjectNInfo

	var rs *HTTPRangeSpec
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		if rs, err = parseRequestRangeSpec(rangeHeader); err != nil {
			if err == errInvalidRange {
				writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidRange), r.URL, guessIsBrowserReq(r))
				return
			}

			logger.LogIf(ctx, err, logger.Application)
		}
	}

	gr, err := getObjectNInfo(ctx, bucket, object, rs, r.Header, readLock, opts)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}
	defer gr.Close()
	objInfo := gr.ObjInfo

	getRetPerms := checkRequestAuthType(ctx, r, policy.GetObjectRetentionAction, bucket, object)
	legalHoldPerms := checkRequestAuthType(ctx, r, policy.GetObjectLegalHoldAction, bucket, object)

	objInfo.UserDefined = objectlock.FilterObjectLockMetadata(objInfo.UserDefined, getRetPerms != ErrNone, legalHoldPerms != ErrNone)

	if objectAPI.IsEncryptionSupported() {
		objInfo.UserDefined = CleanIPOSInternalMetadataKeys(objInfo.UserDefined)
		if _, err = DecryptObjectInfo(&objInfo, r.Header); err != nil {
			writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	if checkPreconditions(ctx, w, r, objInfo, opts) {
		return
	}

	if objectAPI.IsEncryptionSupported() {
		if crypto.IsEncrypted(objInfo.UserDefined) {
			switch {
			case crypto.S3.IsEncrypted(objInfo.UserDefined):
				w.Header().Set(crypto.SSEHeader, crypto.SSEAlgorithmAES256)
			case crypto.SSEC.IsEncrypted(objInfo.UserDefined):
				w.Header().Set(crypto.SSECAlgorithm, r.Header.Get(crypto.SSECAlgorithm))
				w.Header().Set(crypto.SSECKeyMD5, r.Header.Get(crypto.SSECKeyMD5))
			}
		}
	}

	if err = setObjectHeaders(w, objInfo, rs); err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	setHeadGetRespHeaders(w, r.URL.Query())

	statusCodeWritten := false
	httpWriter := ioutil.WriteOnClose(w)
	if rs != nil {
		statusCodeWritten = true
		w.WriteHeader(http.StatusPartialContent)
	}
	if _, err = io.Copy(httpWriter, gr); err != nil {
		if !httpWriter.HasWritten() && !statusCodeWritten {
			writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		}
		return
	}

	if err = httpWriter.Close(); err != nil {
		if !httpWriter.HasWritten() && !statusCodeWritten {
			writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
			return
		}
	}
}

func (api objectAPIHandlers) PutObjectHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "PutObject")
	defer logger.AuditLog(w, r, "PutObject", mustGetClaimsFromToken(r))

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}
	if crypto.S3KMS.IsRequested(r.Header) && !api.AllowSSEKMS() {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrNotImplemented), r.URL, guessIsBrowserReq(r))
		return
	}
	if !api.EncryptionEnabled() && crypto.IsRequested(r.Header) {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrNotImplemented), r.URL, guessIsBrowserReq(r))
		return
	}
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object, err := url.PathUnescape(vars["object"])
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	r.Body = &detectDisconnect{r.Body, r.Context().Done()}

	if _, ok := r.Header[xhttp.AmzCopySource]; ok {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidCopySource), r.URL, guessIsBrowserReq(r))
		return
	}

	if sc := r.Header.Get(xhttp.AmzStorageClass); sc != "" {
		if !(sc == "rrs" || sc == "standard") {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidStorageClass), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	md5Bytes, err := checkValidMD5(r.Header)
	if err != nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidDigest), r.URL, guessIsBrowserReq(r))
		return
	}

	size := r.ContentLength
	rAuthType := getRequestAuthType(r)
	if rAuthType == authTypeStreamingSigned {
		if sizeStr, ok := r.Header[xhttp.AmzDecodedContentLength]; ok {
			if sizeStr[0] == "" {
				writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMissingContentLength), r.URL, guessIsBrowserReq(r))
				return
			}
			size, err = strconv.ParseInt(sizeStr[0], 10, 64)
			if err != nil {
				writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
				return
			}
		}
	}
	if size == -1 {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrMissingContentLength), r.URL, guessIsBrowserReq(r))
		return
	}

	if isMaxObjectSize(size) {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrEntityTooLarge), r.URL, guessIsBrowserReq(r))
		return
	}

	metadata, err := extractMetadata(ctx, r)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	if tags := r.Header.Get(xhttp.AmzObjectTagging); tags != "" {
		metadata[xhttp.AmzObjectTagging], err = extractTags(ctx, tags)
		if err != nil {
			writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	if rAuthType == authTypeStreamingSigned {
		if contentEncoding, ok := metadata["content-encoding"]; ok {
			contentEncoding = trimAwsChunkedContentEncoding(contentEncoding)
			if contentEncoding != "" {
				metadata["content-encoding"] = contentEncoding
			} else {

				delete(metadata, "content-encoding")
			}
		}
	}

	var (
		md5hex    = hex.EncodeToString(md5Bytes)
		sha256hex = ""
		reader    io.Reader
		s3Err     APIErrorCode
		putObject = objectAPI.PutObject
	)
	reader = r.Body

	if s3Err = isPutActionAllowed(rAuthType, bucket, object, r, iampolicy.PutObjectAction); s3Err != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Err), r.URL, guessIsBrowserReq(r))
		return
	}

	switch rAuthType {
	case authTypeStreamingSigned:
		reader, s3Err = newSignV4ChunkedReader(r)
		if s3Err != ErrNone {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Err), r.URL, guessIsBrowserReq(r))
			return
		}
	case authTypeSigned:
		if s3Err = reqSignatureV4Verify(r, globalServerRegion, serviceS3); s3Err != ErrNone {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Err), r.URL, guessIsBrowserReq(r))
			return
		}
		if !skipContentSha256Cksum(r) {
			sha256hex = getContentSha256Cksum(r, serviceS3)
		}
	}

	if globalAutoEncryption && !crypto.SSEC.IsRequested(r.Header) && !crypto.S3KMS.IsRequested(r.Header) {
		r.Header.Add(crypto.SSEHeader, crypto.SSEAlgorithmAES256)
	}

	actualSize := size
	hashReader, err := hash.NewReader(reader, size, md5hex, sha256hex, actualSize, globalCLIContext.StrictS3Compat)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	rawReader := hashReader
	pReader := NewPutObjReader(rawReader, nil, nil)

	var opts ObjectOptions
	opts, err = putOpts(ctx, r, bucket, object, metadata)
	if err != nil {
		writeErrorResponseHeadersOnly(w, toAPIError(ctx, err))
		return
	}

	retPerms := isPutActionAllowed(getRequestAuthType(r), bucket, object, r, iampolicy.PutObjectRetentionAction)
	holdPerms := isPutActionAllowed(getRequestAuthType(r), bucket, object, r, iampolicy.PutObjectLegalHoldAction)

	getObjectInfo := objectAPI.GetObjectInfo

	retentionMode, retentionDate, legalHold, s3Err := checkPutObjectLockAllowed(ctx, r, bucket, object, getObjectInfo, retPerms, holdPerms)
	if s3Err == ErrNone && retentionMode.Valid() {
		metadata[strings.ToLower(xhttp.AmzObjectLockMode)] = string(retentionMode)
		metadata[strings.ToLower(xhttp.AmzObjectLockRetainUntilDate)] = retentionDate.UTC().Format(time.RFC3339)
	}
	if s3Err == ErrNone && legalHold.Status.Valid() {
		metadata[strings.ToLower(xhttp.AmzObjectLockLegalHold)] = string(legalHold.Status)
	}
	if s3Err != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Err), r.URL, guessIsBrowserReq(r))
		return
	}

	var objectEncryptionKey crypto.ObjectKey
	if objectAPI.IsEncryptionSupported() {
		if crypto.IsRequested(r.Header) && !HasSuffix(object, SlashSeparator) {
			if crypto.SSECopy.IsRequested(r.Header) {
				writeErrorResponse(ctx, w, toAPIError(ctx, errInvalidEncryptionParameters), r.URL, guessIsBrowserReq(r))
				return
			}

			reader, objectEncryptionKey, err = EncryptRequest(hashReader, r, bucket, object, metadata)
			if err != nil {
				writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
				return
			}
			info := ObjectInfo{Size: size}

			hashReader, err = hash.NewReader(reader, info.EncryptedSize(), "", "", size, globalCLIContext.StrictS3Compat)
			if err != nil {
				writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
				return
			}
			pReader = NewPutObjReader(rawReader, hashReader, &objectEncryptionKey)
		}
	}

	crypto.RemoveSensitiveEntries(metadata)

	objInfo, err := putObject(ctx, bucket, object, pReader, opts)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	etag := objInfo.ETag
	switch {
	case objInfo.IsCompressed():
		if !strings.HasSuffix(objInfo.ETag, "-1") {
			etag = objInfo.ETag + "-1"
		}
	case crypto.IsEncrypted(objInfo.UserDefined):
		switch {
		case crypto.S3.IsEncrypted(objInfo.UserDefined):
			w.Header().Set(crypto.SSEHeader, crypto.SSEAlgorithmAES256)
			etag, _ = DecryptETag(objectEncryptionKey, ObjectInfo{ETag: etag})
		case crypto.SSEC.IsEncrypted(objInfo.UserDefined):
			w.Header().Set(crypto.SSECAlgorithm, r.Header.Get(crypto.SSECAlgorithm))
			w.Header().Set(crypto.SSECKeyMD5, r.Header.Get(crypto.SSECKeyMD5))

			if len(etag) >= 32 && strings.Count(etag, "-") != 1 {
				etag = etag[len(etag)-32:]
			}
		}
	}
	w.Header()[xhttp.ETag] = []string{`"` + etag + `"`}
	writeSuccessResponseHeadersOnly(w)
}

func (api objectAPIHandlers) DeleteObjectHandler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "DeleteObject")

	defer logger.AuditLog(w, r, "DeleteObject", mustGetClaimsFromToken(r))

	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object, err := url.PathUnescape(vars["object"])
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}

	if s3Error := checkRequestAuthType(ctx, r, policy.DeleteObjectAction, bucket, object); s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	if vid := r.URL.Query().Get("versionId"); vid != "" && vid != "null" {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrNoSuchVersion), r.URL, guessIsBrowserReq(r))
		return
	}

	getObjectInfo := objectAPI.GetObjectInfo
	apiErr := ErrNone
	if _, ok := globalBucketObjectLockConfig.Get(bucket); ok {
		apiErr = enforceRetentionBypassForDelete(ctx, r, bucket, object, getObjectInfo)
		if apiErr != ErrNone && apiErr != ErrNoSuchKey {
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(apiErr), r.URL, guessIsBrowserReq(r))
			return
		}
	}

	if apiErr == ErrNone {
		if err := deleteObject(ctx, objectAPI, bucket, object, r); err != nil {
			switch err.(type) {
			case BucketNotFound:
				writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
				return
			}
		}
	}

	writeSuccessNoContent(w)
}
