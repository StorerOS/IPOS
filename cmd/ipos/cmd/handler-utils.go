package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/auth"
	"github.com/storeros/ipos/pkg/bucket/object/tagging"
	"github.com/storeros/ipos/pkg/handlers"
)

const (
	copyDirective    = "COPY"
	replaceDirective = "REPLACE"
)

func parseLocationConstraint(r *http.Request) (location string, s3Error APIErrorCode) {
	locationConstraint := createBucketLocationConfiguration{}
	err := xmlDecoder(r.Body, &locationConstraint, r.ContentLength)
	if err != nil && r.ContentLength != 0 {
		logger.LogIf(GlobalContext, err)
		return "", ErrMalformedXML
	}
	location = locationConstraint.Location
	if location == "" {
		location = globalServerRegion
	}
	return location, ErrNone
}

func isValidLocation(location string) bool {
	return globalServerRegion == "" || globalServerRegion == location
}

var supportedHeaders = []string{
	"content-type",
	"cache-control",
	"content-language",
	"content-encoding",
	"content-disposition",
	xhttp.AmzStorageClass,
	xhttp.AmzObjectTagging,
	"expires",
}

func isDirectiveValid(v string) bool {
	return isDirectiveCopy(v) || isDirectiveReplace(v)
}

func isDirectiveCopy(value string) bool {
	return value == copyDirective || value == ""
}

func isDirectiveReplace(value string) bool {
	return value == replaceDirective
}

var userMetadataKeyPrefixes = []string{
	"X-Amz-Meta-",
	"X-IPOS-Meta-",
}

func extractMetadata(ctx context.Context, r *http.Request) (metadata map[string]string, err error) {
	query := r.URL.Query()
	header := r.Header
	metadata = make(map[string]string)
	err = extractMetadataFromMap(ctx, query, metadata)
	if err != nil {
		return nil, err
	}

	err = extractMetadataFromMap(ctx, header, metadata)
	if err != nil {
		return nil, err
	}

	if _, ok := metadata["content-type"]; !ok {
		metadata["content-type"] = "application/octet-stream"
	}
	return metadata, nil
}

func extractMetadataFromMap(ctx context.Context, v map[string][]string, m map[string]string) error {
	if v == nil {
		logger.LogIf(ctx, errInvalidArgument)
		return errInvalidArgument
	}
	for _, supportedHeader := range supportedHeaders {
		if value, ok := v[http.CanonicalHeaderKey(supportedHeader)]; ok {
			m[supportedHeader] = value[0]
		} else if value, ok := v[supportedHeader]; ok {
			m[supportedHeader] = value[0]
		}
	}
	for key := range v {
		for _, prefix := range userMetadataKeyPrefixes {
			if !strings.HasPrefix(strings.ToLower(key), strings.ToLower(prefix)) {
				continue
			}
			value, ok := v[key]
			if ok {
				m[key] = strings.Join(value, ",")
				break
			}
		}
	}
	return nil
}

func extractTags(ctx context.Context, tags string) (string, error) {
	if tags != "" {
		tagging, err := tagging.FromString(tags)
		if err != nil {
			return "", err
		}
		if err := tagging.Validate(); err != nil {
			return "", err
		}
		return tagging.String(), nil
	}
	return "", nil
}

func getRedirectPostRawQuery(objInfo ObjectInfo) string {
	redirectValues := make(url.Values)
	redirectValues.Set("bucket", objInfo.Bucket)
	redirectValues.Set("key", objInfo.Name)
	redirectValues.Set("etag", "\""+objInfo.ETag+"\"")
	return redirectValues.Encode()
}

func getReqAccessCred(r *http.Request, region string) (cred auth.Credentials) {
	cred, _, _ = getReqAccessKeyV4(r, region, serviceS3)
	return cred
}

func extractReqParams(r *http.Request) map[string]string {
	if r == nil {
		return nil
	}

	region := globalServerRegion
	cred := getReqAccessCred(r, region)

	return map[string]string{
		"region":          region,
		"accessKey":       cred.AccessKey,
		"sourceIPAddress": handlers.GetSourceIP(r),
	}
}

func extractRespElements(w http.ResponseWriter) map[string]string {
	return map[string]string{
		"requestId":      w.Header().Get(xhttp.AmzRequestID),
		"content-length": w.Header().Get(xhttp.ContentLength),
	}
}

func trimAwsChunkedContentEncoding(contentEnc string) (trimmedContentEnc string) {
	if contentEnc == "" {
		return contentEnc
	}
	var newEncs []string
	for _, enc := range strings.Split(contentEnc, ",") {
		if enc != streamingContentEncoding {
			newEncs = append(newEncs, enc)
		}
	}
	return strings.Join(newEncs, ",")
}

func validateFormFieldSize(ctx context.Context, formValues http.Header) error {
	for k := range formValues {
		if int64(len(formValues.Get(k))) > maxFormFieldSize {
			logger.LogIf(ctx, errSizeUnexpected)
			return errSizeUnexpected
		}
	}

	return nil
}

func extractPostPolicyFormValues(ctx context.Context, form *multipart.Form) (filePart io.ReadCloser, fileName string, fileSize int64, formValues http.Header, err error) {
	fileName = ""

	formValues = make(http.Header)
	for k, v := range form.Value {
		formValues[http.CanonicalHeaderKey(k)] = v
	}

	if err = validateFormFieldSize(ctx, formValues); err != nil {
		return nil, "", 0, nil, err
	}

	if len(form.File) == 0 {
		var b = &bytes.Buffer{}
		for _, v := range formValues["File"] {
			b.WriteString(v)
		}
		fileSize = int64(b.Len())
		filePart = ioutil.NopCloser(b)
		return filePart, fileName, fileSize, formValues, nil
	}

	for k, v := range form.File {
		canonicalFormName := http.CanonicalHeaderKey(k)
		if canonicalFormName == "File" {
			if len(v) == 0 {
				logger.LogIf(ctx, errInvalidArgument)
				return nil, "", 0, nil, errInvalidArgument
			}
			fileHeader := v[0]
			fileName = fileHeader.Filename
			filePart, err = fileHeader.Open()
			if err != nil {
				logger.LogIf(ctx, err)
				return nil, "", 0, nil, err
			}
			fileSize, err = filePart.(io.Seeker).Seek(0, 2)
			if err != nil {
				logger.LogIf(ctx, err)
				return nil, "", 0, nil, err
			}
			_, err = filePart.(io.Seeker).Seek(0, 0)
			if err != nil {
				logger.LogIf(ctx, err)
				return nil, "", 0, nil, err
			}
			break
		}
	}
	return filePart, fileName, fileSize, formValues, nil
}

func httpTraceAll(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !globalHTTPTrace.HasSubscribers() {
			f.ServeHTTP(w, r)
			return
		}
		trace := Trace(f, true, w, r)
		globalHTTPTrace.Publish(trace)
	}
}

func httpTraceHdrs(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !globalHTTPTrace.HasSubscribers() {
			f.ServeHTTP(w, r)
			return
		}
		trace := Trace(f, false, w, r)
		globalHTTPTrace.Publish(trace)
	}
}

func collectAPIStats(api string, f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isS3Request := !strings.HasPrefix(r.URL.Path, iposReservedBucketPath)

		tBefore := UTCNow()

		apiStatsWriter := &recordAPIStats{ResponseWriter: w, TTFB: tBefore, isS3Request: isS3Request}

		if isS3Request {
			globalHTTPStats.currentS3Requests.Inc(api)
		}

		f.ServeHTTP(apiStatsWriter, r)

		if isS3Request {
			globalHTTPStats.currentS3Requests.Dec(api)
		}

		tAfter := apiStatsWriter.TTFB

		durationSecs := tAfter.Sub(tBefore).Seconds()

		globalHTTPStats.updateStats(api, r, apiStatsWriter, durationSecs)
	}
}

func getResource(path string, host string, domains []string) (string, error) {
	if len(domains) == 0 {
		return path, nil
	}
	if strings.Contains(host, ":") {
		var err error
		if host, _, err = net.SplitHostPort(host); err != nil {
			reqInfo := (&logger.ReqInfo{}).AppendTags("host", host)
			reqInfo.AppendTags("path", path)
			ctx := logger.SetReqInfo(GlobalContext, reqInfo)
			logger.LogIf(ctx, err)
			return "", err
		}
	}
	for _, domain := range domains {
		if !strings.HasSuffix(host, "."+domain) {
			continue
		}
		bucket := strings.TrimSuffix(host, "."+domain)
		return SlashSeparator + pathJoin(bucket, path), nil
	}
	return path, nil
}

var regexVersion = regexp.MustCompile(`(\w\d+)`)

func extractAPIVersion(r *http.Request) string {
	return regexVersion.FindString(r.URL.Path)
}

func errorResponseHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	default:
		desc := fmt.Sprintf("Unknown API request at %s", r.URL.Path)
		writeErrorResponse(r.Context(), w, APIError{
			Code:           "XIPOSUnknownAPIRequest",
			Description:    desc,
			HTTPStatusCode: http.StatusBadRequest,
		}, r.URL, guessIsBrowserReq(r))
	}
}

func getHostName(r *http.Request) (hostName string) {
	return r.Host
}
