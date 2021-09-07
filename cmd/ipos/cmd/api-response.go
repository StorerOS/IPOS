package cmd

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/handlers"
)

const (
	timeFormatAMZLong = "2006-01-02T15:04:05.000Z"
	maxObjectList     = 10000
	maxUploadsList    = 10000
	maxPartsList      = 10000
)

type LocationResponse struct {
	XMLName  xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ LocationConstraint" json:"-"`
	Location string   `xml:",chardata"`
}

type ListVersionsResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListVersionsResult" json:"-"`

	Name      string
	Prefix    string
	KeyMarker string

	NextKeyMarker string `xml:"NextKeyMarker,omitempty"`

	NextVersionIDMarker string `xml:"NextVersionIdMarker"`

	VersionIDMarker string `xml:"VersionIdMarker"`

	MaxKeys     int
	Delimiter   string
	IsTruncated bool

	CommonPrefixes []CommonPrefix
	Versions       []ObjectVersion

	EncodingType string `xml:"EncodingType,omitempty"`
}

type ListObjectsResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListBucketResult" json:"-"`

	Name   string
	Prefix string
	Marker string

	NextMarker string `xml:"NextMarker,omitempty"`

	MaxKeys     int
	Delimiter   string
	IsTruncated bool

	Contents       []Object
	CommonPrefixes []CommonPrefix

	EncodingType string `xml:"EncodingType,omitempty"`
}

type ListObjectsV2Response struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListBucketResult" json:"-"`

	Name                  string
	Prefix                string
	StartAfter            string `xml:"StartAfter,omitempty"`
	ContinuationToken     string `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string `xml:"NextContinuationToken,omitempty"`

	KeyCount    int
	MaxKeys     int
	Delimiter   string
	IsTruncated bool

	Contents       []Object
	CommonPrefixes []CommonPrefix

	EncodingType string `xml:"EncodingType,omitempty"`
}

type Part struct {
	PartNumber   int
	LastModified string
	ETag         string
	Size         int64
}

type ListPartsResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListPartsResult" json:"-"`

	Bucket   string
	Key      string
	UploadID string `xml:"UploadId"`

	Initiator Initiator
	Owner     Owner

	StorageClass string

	PartNumberMarker     int
	NextPartNumberMarker int
	MaxParts             int
	IsTruncated          bool

	Parts []Part `xml:"Part"`
}

type ListMultipartUploadsResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListMultipartUploadsResult" json:"-"`

	Bucket             string
	KeyMarker          string
	UploadIDMarker     string `xml:"UploadIdMarker"`
	NextKeyMarker      string
	NextUploadIDMarker string `xml:"NextUploadIdMarker"`
	Delimiter          string
	Prefix             string
	EncodingType       string `xml:"EncodingType,omitempty"`
	MaxUploads         int
	IsTruncated        bool

	Uploads []Upload `xml:"Upload"`

	CommonPrefixes []CommonPrefix
}

type ListBucketsResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListAllMyBucketsResult" json:"-"`

	Owner Owner

	Buckets struct {
		Buckets []Bucket `xml:"Bucket"`
	}
}

type Upload struct {
	Key          string
	UploadID     string `xml:"UploadId"`
	Initiator    Initiator
	Owner        Owner
	StorageClass string
	Initiated    string
}

type CommonPrefix struct {
	Prefix string
}

type Bucket struct {
	Name         string
	CreationDate string
}

type ObjectVersion struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ Version" json:"-"`
	Object
	VersionID string `xml:"VersionId"`
	IsLatest  bool
}

type StringMap map[string]string

func (s StringMap) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	tokens := []xml.Token{start}

	for key, value := range s {
		t := xml.StartElement{}
		t.Name = xml.Name{
			Space: "",
			Local: key,
		}
		tokens = append(tokens, t, xml.CharData(value), xml.EndElement{Name: t.Name})
	}

	tokens = append(tokens, xml.EndElement{
		Name: start.Name,
	})

	for _, t := range tokens {
		if err := e.EncodeToken(t); err != nil {
			return err
		}
	}

	return e.Flush()
}

type Object struct {
	Key          string
	LastModified string
	ETag         string
	Size         int64

	Owner Owner

	StorageClass string

	UserMetadata StringMap `xml:"UserMetadata,omitempty"`
}

type CopyObjectResponse struct {
	XMLName      xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CopyObjectResult" json:"-"`
	LastModified string
	ETag         string
}

type CopyObjectPartResponse struct {
	XMLName      xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CopyPartResult" json:"-"`
	LastModified string
	ETag         string
}

type Initiator Owner

type Owner struct {
	ID          string
	DisplayName string
}

type InitiateMultipartUploadResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ InitiateMultipartUploadResult" json:"-"`

	Bucket   string
	Key      string
	UploadID string `xml:"UploadId"`
}

type CompleteMultipartUploadResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ CompleteMultipartUploadResult" json:"-"`

	Location string
	Bucket   string
	Key      string
	ETag     string
}

type DeleteError struct {
	Code    string
	Message string
	Key     string
}

type DeleteObjectsResponse struct {
	XMLName xml.Name `xml:"http://s3.amazonaws.com/doc/2006-03-01/ DeleteResult" json:"-"`

	DeletedObjects []ObjectIdentifier `xml:"Deleted,omitempty"`

	Errors []DeleteError `xml:"Error,omitempty"`
}

type PostResponse struct {
	Bucket   string
	Key      string
	ETag     string
	Location string
}

func getURLScheme(tls bool) string {
	if tls {
		return httpsScheme
	}
	return httpScheme
}

func getObjectLocation(r *http.Request, domains []string, bucket, object string) string {
	if r.Host == "" {
		return path.Clean(r.URL.Path)
	}
	proto := handlers.GetSourceScheme(r)
	if proto == "" {
		proto = getURLScheme(globalIsSSL)
	}
	u := &url.URL{
		Host:   r.Host,
		Path:   path.Join(SlashSeparator, bucket, object),
		Scheme: proto,
	}
	for _, domain := range domains {
		if strings.Contains(r.Host, domain) {
			u.Host = bucket + "." + r.Host
			u.Path = path.Join(SlashSeparator, object)
			break
		}
	}
	return u.String()
}

func generateListBucketsResponse(buckets []BucketInfo) ListBucketsResponse {
	var listbuckets []Bucket
	var data = ListBucketsResponse{}
	var owner = Owner{}

	owner.ID = globalIPOSDefaultOwnerID
	for _, bucket := range buckets {
		var listbucket = Bucket{}
		listbucket.Name = bucket.Name
		listbucket.CreationDate = bucket.Created.UTC().Format(timeFormatAMZLong)
		listbuckets = append(listbuckets, listbucket)
	}

	data.Owner = owner
	data.Buckets.Buckets = listbuckets

	return data
}

func generateListVersionsResponse(bucket, prefix, marker, delimiter, encodingType string, maxKeys int, resp ListObjectsInfo) ListVersionsResponse {
	var versions []ObjectVersion
	var prefixes []CommonPrefix
	var owner = Owner{}
	var data = ListVersionsResponse{}

	owner.ID = globalIPOSDefaultOwnerID
	for _, object := range resp.Objects {
		var content = ObjectVersion{}
		if object.Name == "" {
			continue
		}
		content.Key = s3EncodeName(object.Name, encodingType)
		content.LastModified = object.ModTime.UTC().Format(timeFormatAMZLong)
		if object.ETag != "" {
			content.ETag = "\"" + object.ETag + "\""
		}
		content.Size = object.Size
		if object.StorageClass != "" {
			content.StorageClass = object.StorageClass
		} else {
			content.StorageClass = globalIPOSDefaultStorageClass
		}

		content.Owner = owner
		content.VersionID = "null"
		content.IsLatest = true
		versions = append(versions, content)
	}
	data.Name = bucket
	data.Versions = versions

	data.EncodingType = encodingType
	data.Prefix = s3EncodeName(prefix, encodingType)
	data.KeyMarker = s3EncodeName(marker, encodingType)
	data.Delimiter = s3EncodeName(delimiter, encodingType)
	data.MaxKeys = maxKeys

	data.NextKeyMarker = s3EncodeName(resp.NextMarker, encodingType)
	data.IsTruncated = resp.IsTruncated

	for _, prefix := range resp.Prefixes {
		var prefixItem = CommonPrefix{}
		prefixItem.Prefix = s3EncodeName(prefix, encodingType)
		prefixes = append(prefixes, prefixItem)
	}
	data.CommonPrefixes = prefixes
	return data
}

func generateListObjectsV1Response(bucket, prefix, marker, delimiter, encodingType string, maxKeys int, resp ListObjectsInfo) ListObjectsResponse {
	var contents []Object
	var prefixes []CommonPrefix
	var owner = Owner{}
	var data = ListObjectsResponse{}

	owner.ID = globalIPOSDefaultOwnerID
	for _, object := range resp.Objects {
		var content = Object{}
		if object.Name == "" {
			continue
		}
		content.Key = s3EncodeName(object.Name, encodingType)
		content.LastModified = object.ModTime.UTC().Format(timeFormatAMZLong)
		if object.ETag != "" {
			content.ETag = "\"" + object.ETag + "\""
		}
		content.Size = object.Size
		if object.StorageClass != "" {
			content.StorageClass = object.StorageClass
		} else {
			content.StorageClass = globalIPOSDefaultStorageClass
		}
		content.Owner = owner
		contents = append(contents, content)
	}
	data.Name = bucket
	data.Contents = contents

	data.EncodingType = encodingType
	data.Prefix = s3EncodeName(prefix, encodingType)
	data.Marker = s3EncodeName(marker, encodingType)
	data.Delimiter = s3EncodeName(delimiter, encodingType)
	data.MaxKeys = maxKeys

	data.NextMarker = s3EncodeName(resp.NextMarker, encodingType)
	data.IsTruncated = resp.IsTruncated
	for _, prefix := range resp.Prefixes {
		var prefixItem = CommonPrefix{}
		prefixItem.Prefix = s3EncodeName(prefix, encodingType)
		prefixes = append(prefixes, prefixItem)
	}
	data.CommonPrefixes = prefixes
	return data
}

func generateListObjectsV2Response(bucket, prefix, token, nextToken, startAfter, delimiter, encodingType string, fetchOwner, isTruncated bool, maxKeys int, objects []ObjectInfo, prefixes []string, metadata bool) ListObjectsV2Response {
	var contents []Object
	var commonPrefixes []CommonPrefix
	var owner = Owner{}
	var data = ListObjectsV2Response{}

	if fetchOwner {
		owner.ID = globalIPOSDefaultOwnerID
	}

	for _, object := range objects {
		var content = Object{}
		if object.Name == "" {
			continue
		}
		content.Key = s3EncodeName(object.Name, encodingType)
		content.LastModified = object.ModTime.UTC().Format(timeFormatAMZLong)
		if object.ETag != "" {
			content.ETag = "\"" + object.ETag + "\""
		}
		content.Size = object.Size
		if object.StorageClass != "" {
			content.StorageClass = object.StorageClass
		} else {
			content.StorageClass = globalIPOSDefaultStorageClass
		}
		content.Owner = owner
		if metadata {
			content.UserMetadata = make(StringMap)
			for k, v := range CleanIPOSInternalMetadataKeys(object.UserDefined) {
				if HasPrefix(k, ReservedMetadataPrefix) {
					continue
				}
				content.UserMetadata[k] = v
			}
		}
		contents = append(contents, content)
	}
	data.Name = bucket
	data.Contents = contents

	data.EncodingType = encodingType
	data.StartAfter = s3EncodeName(startAfter, encodingType)
	data.Delimiter = s3EncodeName(delimiter, encodingType)
	data.Prefix = s3EncodeName(prefix, encodingType)
	data.MaxKeys = maxKeys
	data.ContinuationToken = base64.StdEncoding.EncodeToString([]byte(token))
	data.NextContinuationToken = base64.StdEncoding.EncodeToString([]byte(nextToken))
	data.IsTruncated = isTruncated
	for _, prefix := range prefixes {
		var prefixItem = CommonPrefix{}
		prefixItem.Prefix = s3EncodeName(prefix, encodingType)
		commonPrefixes = append(commonPrefixes, prefixItem)
	}
	data.CommonPrefixes = commonPrefixes
	data.KeyCount = len(data.Contents) + len(data.CommonPrefixes)
	return data
}

func generateCopyObjectResponse(etag string, lastModified time.Time) CopyObjectResponse {
	return CopyObjectResponse{
		ETag:         "\"" + etag + "\"",
		LastModified: lastModified.UTC().Format(timeFormatAMZLong),
	}
}

func generateCopyObjectPartResponse(etag string, lastModified time.Time) CopyObjectPartResponse {
	return CopyObjectPartResponse{
		ETag:         "\"" + etag + "\"",
		LastModified: lastModified.UTC().Format(timeFormatAMZLong),
	}
}

func generateInitiateMultipartUploadResponse(bucket, key, uploadID string) InitiateMultipartUploadResponse {
	return InitiateMultipartUploadResponse{
		Bucket:   bucket,
		Key:      key,
		UploadID: uploadID,
	}
}

func generateCompleteMultpartUploadResponse(bucket, key, location, etag string) CompleteMultipartUploadResponse {
	return CompleteMultipartUploadResponse{
		Location: location,
		Bucket:   bucket,
		Key:      key,
		ETag:     etag,
	}
}

func generateListPartsResponse(partsInfo ListPartsInfo, encodingType string) ListPartsResponse {
	listPartsResponse := ListPartsResponse{}
	listPartsResponse.Bucket = partsInfo.Bucket
	listPartsResponse.Key = s3EncodeName(partsInfo.Object, encodingType)
	listPartsResponse.UploadID = partsInfo.UploadID
	listPartsResponse.StorageClass = globalIPOSDefaultStorageClass
	listPartsResponse.Initiator.ID = globalIPOSDefaultOwnerID
	listPartsResponse.Owner.ID = globalIPOSDefaultOwnerID

	listPartsResponse.MaxParts = partsInfo.MaxParts
	listPartsResponse.PartNumberMarker = partsInfo.PartNumberMarker
	listPartsResponse.IsTruncated = partsInfo.IsTruncated
	listPartsResponse.NextPartNumberMarker = partsInfo.NextPartNumberMarker

	listPartsResponse.Parts = make([]Part, len(partsInfo.Parts))
	for index, part := range partsInfo.Parts {
		newPart := Part{}
		newPart.PartNumber = part.PartNumber
		newPart.ETag = "\"" + part.ETag + "\""
		newPart.Size = part.Size
		newPart.LastModified = part.LastModified.UTC().Format(timeFormatAMZLong)
		listPartsResponse.Parts[index] = newPart
	}
	return listPartsResponse
}

func generateListMultipartUploadsResponse(bucket string, multipartsInfo ListMultipartsInfo, encodingType string) ListMultipartUploadsResponse {
	listMultipartUploadsResponse := ListMultipartUploadsResponse{}
	listMultipartUploadsResponse.Bucket = bucket
	listMultipartUploadsResponse.Delimiter = s3EncodeName(multipartsInfo.Delimiter, encodingType)
	listMultipartUploadsResponse.IsTruncated = multipartsInfo.IsTruncated
	listMultipartUploadsResponse.EncodingType = encodingType
	listMultipartUploadsResponse.Prefix = s3EncodeName(multipartsInfo.Prefix, encodingType)
	listMultipartUploadsResponse.KeyMarker = s3EncodeName(multipartsInfo.KeyMarker, encodingType)
	listMultipartUploadsResponse.NextKeyMarker = s3EncodeName(multipartsInfo.NextKeyMarker, encodingType)
	listMultipartUploadsResponse.MaxUploads = multipartsInfo.MaxUploads
	listMultipartUploadsResponse.NextUploadIDMarker = multipartsInfo.NextUploadIDMarker
	listMultipartUploadsResponse.UploadIDMarker = multipartsInfo.UploadIDMarker
	listMultipartUploadsResponse.CommonPrefixes = make([]CommonPrefix, len(multipartsInfo.CommonPrefixes))
	for index, commonPrefix := range multipartsInfo.CommonPrefixes {
		listMultipartUploadsResponse.CommonPrefixes[index] = CommonPrefix{
			Prefix: s3EncodeName(commonPrefix, encodingType),
		}
	}
	listMultipartUploadsResponse.Uploads = make([]Upload, len(multipartsInfo.Uploads))
	for index, upload := range multipartsInfo.Uploads {
		newUpload := Upload{}
		newUpload.UploadID = upload.UploadID
		newUpload.Key = s3EncodeName(upload.Object, encodingType)
		newUpload.Initiated = upload.Initiated.UTC().Format(timeFormatAMZLong)
		listMultipartUploadsResponse.Uploads[index] = newUpload
	}
	return listMultipartUploadsResponse
}

func generateMultiDeleteResponse(quiet bool, deletedObjects []ObjectIdentifier, errs []DeleteError) DeleteObjectsResponse {
	deleteResp := DeleteObjectsResponse{}
	if !quiet {
		deleteResp.DeletedObjects = deletedObjects
	}
	deleteResp.Errors = errs
	return deleteResp
}

func writeResponse(w http.ResponseWriter, statusCode int, response []byte, mType mimeType) {
	setCommonHeaders(w)
	if mType != mimeNone {
		w.Header().Set(xhttp.ContentType, string(mType))
	}
	w.Header().Set(xhttp.ContentLength, strconv.Itoa(len(response)))
	w.WriteHeader(statusCode)
	if response != nil {
		w.Write(response)
		w.(http.Flusher).Flush()
	}
}

type mimeType string

const (
	mimeNone mimeType = ""
	mimeJSON mimeType = "application/json"
	mimeXML  mimeType = "application/xml"
)

func writeSuccessResponseJSON(w http.ResponseWriter, response []byte) {
	writeResponse(w, http.StatusOK, response, mimeJSON)
}

func writeSuccessResponseXML(w http.ResponseWriter, response []byte) {
	writeResponse(w, http.StatusOK, response, mimeXML)
}

func writeSuccessNoContent(w http.ResponseWriter) {
	writeResponse(w, http.StatusNoContent, nil, mimeNone)
}

func writeRedirectSeeOther(w http.ResponseWriter, location string) {
	w.Header().Set(xhttp.Location, location)
	writeResponse(w, http.StatusSeeOther, nil, mimeNone)
}

func writeSuccessResponseHeadersOnly(w http.ResponseWriter) {
	writeResponse(w, http.StatusOK, nil, mimeNone)
}

func writeErrorResponse(ctx context.Context, w http.ResponseWriter, err APIError, reqURL *url.URL, browser bool) {
	switch err.Code {
	case "SlowDown", "XIPOSServerNotInitialized", "XIPOSReadQuorum", "XIPOSWriteQuorum":
		w.Header().Set(xhttp.RetryAfter, "120")
	case "AccessDenied":
		if browser && globalBrowserEnabled {
			w.Header().Set(xhttp.Location, iposReservedBucketPath+reqURL.Path)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}

	errorResponse := getAPIErrorResponse(ctx, err, reqURL.Path,
		w.Header().Get(xhttp.AmzRequestID), globalDeploymentID)
	encodedErrorResponse := encodeResponse(errorResponse)
	writeResponse(w, err.HTTPStatusCode, encodedErrorResponse, mimeXML)
}

func writeErrorResponseHeadersOnly(w http.ResponseWriter, err APIError) {
	writeResponse(w, err.HTTPStatusCode, nil, mimeNone)
}

func writeErrorResponseString(ctx context.Context, w http.ResponseWriter, err APIError, reqURL *url.URL) {
	writeResponse(w, err.HTTPStatusCode, []byte(err.Description), mimeNone)
}

func writeErrorResponseJSON(ctx context.Context, w http.ResponseWriter, err APIError, reqURL *url.URL) {
	errorResponse := getAPIErrorResponse(ctx, err, reqURL.Path, w.Header().Get(xhttp.AmzRequestID), globalDeploymentID)
	encodedErrorResponse := encodeResponseJSON(errorResponse)
	writeResponse(w, err.HTTPStatusCode, encodedErrorResponse, mimeJSON)
}

func writeVersionMismatchResponse(ctx context.Context, w http.ResponseWriter, err APIError, reqURL *url.URL, isJSON bool) {
	if isJSON {
		errorResponse := getAPIErrorResponse(ctx, err, reqURL.String(), w.Header().Get(xhttp.AmzRequestID), globalDeploymentID)
		writeResponse(w, err.HTTPStatusCode, encodeResponseJSON(errorResponse), mimeJSON)
	} else {
		writeResponse(w, err.HTTPStatusCode, []byte(err.Description), mimeNone)
	}
}

func writeCustomErrorResponseJSON(ctx context.Context, w http.ResponseWriter, err APIError,
	errBody string, reqURL *url.URL) {

	reqInfo := logger.GetReqInfo(ctx)
	errorResponse := APIErrorResponse{
		Code:       err.Code,
		Message:    errBody,
		Resource:   reqURL.Path,
		BucketName: reqInfo.BucketName,
		Key:        reqInfo.ObjectName,
		RequestID:  w.Header().Get(xhttp.AmzRequestID),
		HostID:     globalDeploymentID,
	}
	encodedErrorResponse := encodeResponseJSON(errorResponse)
	writeResponse(w, err.HTTPStatusCode, encodedErrorResponse, mimeJSON)
}

func writeCustomErrorResponseXML(ctx context.Context, w http.ResponseWriter, err APIError, errBody string, reqURL *url.URL, browser bool) {

	switch err.Code {
	case "SlowDown", "XIPOSServerNotInitialized", "XIPOSReadQuorum", "XIPOSWriteQuorum":
		w.Header().Set(xhttp.RetryAfter, "120")
	case "AccessDenied":
		if browser && globalBrowserEnabled {
			w.Header().Set(xhttp.Location, iposReservedBucketPath+reqURL.Path)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}

	reqInfo := logger.GetReqInfo(ctx)
	errorResponse := APIErrorResponse{
		Code:       err.Code,
		Message:    errBody,
		Resource:   reqURL.Path,
		BucketName: reqInfo.BucketName,
		Key:        reqInfo.ObjectName,
		RequestID:  w.Header().Get(xhttp.AmzRequestID),
		HostID:     globalDeploymentID,
	}

	encodedErrorResponse := encodeResponse(errorResponse)
	writeResponse(w, err.HTTPStatusCode, encodedErrorResponse, mimeXML)
}
