package cmd

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/storeros/ipos/cmd/ipos/crypto"
	xhttp "github.com/storeros/ipos/cmd/ipos/http"
)

var (
	etagRegex = regexp.MustCompile("\"*?([^\"]*?)\"*?$")
)

func checkCopyObjectPartPreconditions(ctx context.Context, w http.ResponseWriter, r *http.Request, objInfo ObjectInfo, encETag string) bool {
	return checkCopyObjectPreconditions(ctx, w, r, objInfo, encETag)
}

func checkCopyObjectPreconditions(ctx context.Context, w http.ResponseWriter, r *http.Request, objInfo ObjectInfo, encETag string) bool {
	if r.Method != http.MethodPut {
		return false
	}
	if encETag == "" {
		encETag = objInfo.ETag
	}
	if objInfo.ModTime.IsZero() || objInfo.ModTime.Equal(time.Unix(0, 0)) {
		return false
	}

	writeHeaders := func() {
		setCommonHeaders(w)

		w.Header().Set(xhttp.LastModified, objInfo.ModTime.UTC().Format(http.TimeFormat))

		if objInfo.ETag != "" {
			w.Header()[xhttp.ETag] = []string{"\"" + objInfo.ETag + "\""}
		}
	}
	ifModifiedSinceHeader := r.Header.Get(xhttp.AmzCopySourceIfModifiedSince)
	if ifModifiedSinceHeader != "" {
		if givenTime, err := time.Parse(http.TimeFormat, ifModifiedSinceHeader); err == nil {
			if !ifModifiedSince(objInfo.ModTime, givenTime) {
				writeHeaders()
				writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrPreconditionFailed), r.URL, guessIsBrowserReq(r))
				return true
			}
		}
	}

	ifUnmodifiedSinceHeader := r.Header.Get(xhttp.AmzCopySourceIfUnmodifiedSince)
	if ifUnmodifiedSinceHeader != "" {
		if givenTime, err := time.Parse(http.TimeFormat, ifUnmodifiedSinceHeader); err == nil {
			if ifModifiedSince(objInfo.ModTime, givenTime) {
				writeHeaders()
				writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrPreconditionFailed), r.URL, guessIsBrowserReq(r))
				return true
			}
		}
	}

	shouldDecryptEtag := crypto.SSECopy.IsRequested(r.Header) && !crypto.IsMultiPart(objInfo.UserDefined)

	ifMatchETagHeader := r.Header.Get(xhttp.AmzCopySourceIfMatch)
	if ifMatchETagHeader != "" {
		etag := objInfo.ETag
		if shouldDecryptEtag {
			etag = encETag[len(encETag)-32:]
		}
		if objInfo.ETag != "" && !isETagEqual(etag, ifMatchETagHeader) {
			writeHeaders()
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrPreconditionFailed), r.URL, guessIsBrowserReq(r))
			return true
		}
	}

	ifNoneMatchETagHeader := r.Header.Get(xhttp.AmzCopySourceIfNoneMatch)
	if ifNoneMatchETagHeader != "" {
		etag := objInfo.ETag
		if shouldDecryptEtag {
			etag = encETag[len(encETag)-32:]
		}
		if objInfo.ETag != "" && isETagEqual(etag, ifNoneMatchETagHeader) {
			writeHeaders()
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrPreconditionFailed), r.URL, guessIsBrowserReq(r))
			return true
		}
	}
	return false
}

func checkPreconditions(ctx context.Context, w http.ResponseWriter, r *http.Request, objInfo ObjectInfo, opts ObjectOptions) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if objInfo.ModTime.IsZero() || objInfo.ModTime.Equal(time.Unix(0, 0)) {
		return false
	}

	writeHeaders := func() {
		setCommonHeaders(w)

		w.Header().Set(xhttp.LastModified, objInfo.ModTime.UTC().Format(http.TimeFormat))

		if objInfo.ETag != "" {
			w.Header()[xhttp.ETag] = []string{"\"" + objInfo.ETag + "\""}
		}
	}

	ifModifiedSinceHeader := r.Header.Get(xhttp.IfModifiedSince)
	if ifModifiedSinceHeader != "" {
		if givenTime, err := time.Parse(http.TimeFormat, ifModifiedSinceHeader); err == nil {
			if !ifModifiedSince(objInfo.ModTime, givenTime) {
				writeHeaders()
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	ifUnmodifiedSinceHeader := r.Header.Get(xhttp.IfUnmodifiedSince)
	if ifUnmodifiedSinceHeader != "" {
		if givenTime, err := time.Parse(http.TimeFormat, ifUnmodifiedSinceHeader); err == nil {
			if ifModifiedSince(objInfo.ModTime, givenTime) {
				writeHeaders()
				writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrPreconditionFailed), r.URL, guessIsBrowserReq(r))
				return true
			}
		}
	}

	ifMatchETagHeader := r.Header.Get(xhttp.IfMatch)
	if ifMatchETagHeader != "" {
		if !isETagEqual(objInfo.ETag, ifMatchETagHeader) {
			writeHeaders()
			writeErrorResponse(ctx, w, errorCodes.ToAPIErr(ErrPreconditionFailed), r.URL, guessIsBrowserReq(r))
			return true
		}
	}

	ifNoneMatchETagHeader := r.Header.Get(xhttp.IfNoneMatch)
	if ifNoneMatchETagHeader != "" {
		if isETagEqual(objInfo.ETag, ifNoneMatchETagHeader) {
			writeHeaders()
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}

func ifModifiedSince(objTime time.Time, givenTime time.Time) bool {
	return objTime.After(givenTime.Add(1 * time.Second))
}

func canonicalizeETag(etag string) string {
	return etagRegex.ReplaceAllString(etag, "$1")
}

func isETagEqual(left, right string) bool {
	return canonicalizeETag(left) == canonicalizeETag(right)
}

func deleteObject(ctx context.Context, obj ObjectLayer, bucket, object string, r *http.Request) (err error) {
	deleteObject := obj.DeleteObject
	if err = deleteObject(ctx, bucket, object); err != nil {
		return err
	}

	return nil
}
