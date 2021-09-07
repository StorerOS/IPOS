package cmd

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/storeros/ipos/cmd/ipos/crypto"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/bucket/policy"
)

func validateListObjectsArgs(marker, delimiter, encodingType string, maxKeys int) APIErrorCode {
	if maxKeys < 0 {
		return ErrInvalidMaxKeys
	}

	if encodingType != "" {
		if strings.ToLower(encodingType) != "url" {
			return ErrInvalidEncodingMethod
		}
	}

	return ErrNone
}

func (api objectAPIHandlers) ListObjectsV2Handler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "ListObjectsV2")

	defer logger.AuditLog(w, r, "ListObjectsV2", mustGetClaimsFromToken(r))

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}

	if s3Error := checkRequestAuthType(ctx, r, policy.ListBucketAction, bucket, ""); s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	urlValues := r.URL.Query()

	prefix, token, startAfter, delimiter, fetchOwner, maxKeys, encodingType, errCode := getListObjectsV2Args(urlValues)
	if errCode != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(errCode), r.URL, guessIsBrowserReq(r))
		return
	}

	if s3Error := validateListObjectsArgs(token, delimiter, encodingType, maxKeys); s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	listObjectsV2 := objectAPI.ListObjectsV2

	listObjectsV2Info, err := listObjectsV2(ctx, bucket, prefix, token, delimiter, maxKeys, fetchOwner, startAfter)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	for i := range listObjectsV2Info.Objects {
		var actualSize int64
		if listObjectsV2Info.Objects[i].IsCompressed() {
			actualSize = listObjectsV2Info.Objects[i].GetActualSize()
			if actualSize < 0 {
				writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidDecompressedSize), r.URL, guessIsBrowserReq(r))
				return
			}
			listObjectsV2Info.Objects[i].Size = actualSize
		} else if crypto.IsEncrypted(listObjectsV2Info.Objects[i].UserDefined) {
			listObjectsV2Info.Objects[i].ETag = getDecryptedETag(r.Header, listObjectsV2Info.Objects[i], false)
			listObjectsV2Info.Objects[i].Size, err = listObjectsV2Info.Objects[i].DecryptedSize()
			if err != nil {
				writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
				return
			}
		}
	}

	response := generateListObjectsV2Response(bucket, prefix, token,
		listObjectsV2Info.NextContinuationToken, startAfter,
		delimiter, encodingType, fetchOwner, listObjectsV2Info.IsTruncated,
		maxKeys, listObjectsV2Info.Objects, listObjectsV2Info.Prefixes, false)

	writeSuccessResponseXML(w, encodeResponse(response))
}

func (api objectAPIHandlers) ListObjectsV1Handler(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(r, w, "ListObjectsV1")

	defer logger.AuditLog(w, r, "ListObjectsV1", mustGetClaimsFromToken(r))

	vars := mux.Vars(r)
	bucket := vars["bucket"]

	objectAPI := api.ObjectAPI()
	if objectAPI == nil {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrServerNotInitialized), r.URL, guessIsBrowserReq(r))
		return
	}

	if s3Error := checkRequestAuthType(ctx, r, policy.ListBucketAction, bucket, ""); s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	prefix, marker, delimiter, maxKeys, encodingType, s3Error := getListObjectsV1Args(r.URL.Query())
	if s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	if s3Error := validateListObjectsArgs(marker, delimiter, encodingType, maxKeys); s3Error != ErrNone {
		writeErrorResponse(ctx, w, errorCodes.ToAPIErr(s3Error), r.URL, guessIsBrowserReq(r))
		return
	}

	listObjects := objectAPI.ListObjects

	listObjectsInfo, err := listObjects(ctx, bucket, prefix, marker, delimiter, maxKeys)
	if err != nil {
		writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
		return
	}

	for i := range listObjectsInfo.Objects {
		var actualSize int64
		if listObjectsInfo.Objects[i].IsCompressed() {
			actualSize = listObjectsInfo.Objects[i].GetActualSize()
			if actualSize < 0 {
				writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrInvalidDecompressedSize), r.URL, guessIsBrowserReq(r))
				return
			}
			listObjectsInfo.Objects[i].Size = actualSize
		} else if crypto.IsEncrypted(listObjectsInfo.Objects[i].UserDefined) {
			listObjectsInfo.Objects[i].ETag = getDecryptedETag(r.Header, listObjectsInfo.Objects[i], false)
			listObjectsInfo.Objects[i].Size, err = listObjectsInfo.Objects[i].DecryptedSize()
			if err != nil {
				writeErrorResponse(ctx, w, toAPIError(ctx, err), r.URL, guessIsBrowserReq(r))
				return
			}
		}
	}
	response := generateListObjectsV1Response(bucket, prefix, marker, delimiter, encodingType, maxKeys, listObjectsInfo)

	writeSuccessResponseXML(w, encodeResponse(response))
}
