package signer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/storeros/ipos/pkg/s3utils"
)

const (
	signV2Algorithm = "AWS"
)

func encodeURL2Path(req *http.Request, virtualHost bool) (path string) {
	if virtualHost {
		reqHost := getHostAddr(req)
		dotPos := strings.Index(reqHost, ".")
		if dotPos > -1 {
			bucketName := reqHost[:dotPos]
			path = "/" + bucketName
			path += req.URL.Path
			path = s3utils.EncodePath(path)
			return
		}
	}
	path = s3utils.EncodePath(req.URL.Path)
	return
}

func PreSignV2(req http.Request, accessKeyID, secretAccessKey string, expires int64, virtualHost bool) *http.Request {
	if accessKeyID == "" || secretAccessKey == "" {
		return &req
	}

	d := time.Now().UTC()

	epochExpires := d.Unix() + expires

	if expiresStr := req.Header.Get("Expires"); expiresStr == "" {
		req.Header.Set("Expires", strconv.FormatInt(epochExpires, 10))
	}

	stringToSign := preStringToSignV2(req, virtualHost)
	hm := hmac.New(sha1.New, []byte(secretAccessKey))
	hm.Write([]byte(stringToSign))

	signature := base64.StdEncoding.EncodeToString(hm.Sum(nil))

	query := req.URL.Query()

	if strings.Contains(getHostAddr(&req), ".storage.googleapis.com") {
		query.Set("GoogleAccessId", accessKeyID)
	} else {
		query.Set("AWSAccessKeyId", accessKeyID)
	}

	query.Set("Expires", strconv.FormatInt(epochExpires, 10))

	req.URL.RawQuery = s3utils.QueryEncode(query)

	req.URL.RawQuery += "&Signature=" + s3utils.EncodePath(signature)

	return &req
}

func PostPresignSignatureV2(policyBase64, secretAccessKey string) string {
	hm := hmac.New(sha1.New, []byte(secretAccessKey))
	hm.Write([]byte(policyBase64))
	signature := base64.StdEncoding.EncodeToString(hm.Sum(nil))
	return signature
}

func SignV2(req http.Request, accessKeyID, secretAccessKey string, virtualHost bool) *http.Request {
	if accessKeyID == "" || secretAccessKey == "" {
		return &req
	}

	d := time.Now().UTC()

	if date := req.Header.Get("Date"); date == "" {
		req.Header.Set("Date", d.Format(http.TimeFormat))
	}

	stringToSign := stringToSignV2(req, virtualHost)
	hm := hmac.New(sha1.New, []byte(secretAccessKey))
	hm.Write([]byte(stringToSign))

	authHeader := new(bytes.Buffer)
	authHeader.WriteString(fmt.Sprintf("%s %s:", signV2Algorithm, accessKeyID))
	encoder := base64.NewEncoder(base64.StdEncoding, authHeader)
	encoder.Write(hm.Sum(nil))
	encoder.Close()

	req.Header.Set("Authorization", authHeader.String())

	return &req
}

func preStringToSignV2(req http.Request, virtualHost bool) string {
	buf := new(bytes.Buffer)

	writePreSignV2Headers(buf, req)

	writeCanonicalizedHeaders(buf, req)

	writeCanonicalizedResource(buf, req, virtualHost)
	return buf.String()
}

func writePreSignV2Headers(buf *bytes.Buffer, req http.Request) {
	buf.WriteString(req.Method + "\n")
	buf.WriteString(req.Header.Get("Content-Md5") + "\n")
	buf.WriteString(req.Header.Get("Content-Type") + "\n")
	buf.WriteString(req.Header.Get("Expires") + "\n")
}

func stringToSignV2(req http.Request, virtualHost bool) string {
	buf := new(bytes.Buffer)

	writeSignV2Headers(buf, req)

	writeCanonicalizedHeaders(buf, req)

	writeCanonicalizedResource(buf, req, virtualHost)
	return buf.String()
}

func writeSignV2Headers(buf *bytes.Buffer, req http.Request) {
	buf.WriteString(req.Method + "\n")
	buf.WriteString(req.Header.Get("Content-Md5") + "\n")
	buf.WriteString(req.Header.Get("Content-Type") + "\n")
	buf.WriteString(req.Header.Get("Date") + "\n")
}

func writeCanonicalizedHeaders(buf *bytes.Buffer, req http.Request) {
	var protoHeaders []string
	vals := make(map[string][]string)
	for k, vv := range req.Header {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-amz") {
			protoHeaders = append(protoHeaders, lk)
			vals[lk] = vv
		}
	}
	sort.Strings(protoHeaders)
	for _, k := range protoHeaders {
		buf.WriteString(k)
		buf.WriteByte(':')
		for idx, v := range vals[k] {
			if idx > 0 {
				buf.WriteByte(',')
			}
			if strings.Contains(v, "\n") {
				buf.WriteString(v)
			} else {
				buf.WriteString(v)
			}
		}
		buf.WriteByte('\n')
	}
}

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

func writeCanonicalizedResource(buf *bytes.Buffer, req http.Request, virtualHost bool) {
	requestURL := req.URL

	buf.WriteString(encodeURL2Path(&req, virtualHost))
	if requestURL.RawQuery != "" {
		var n int
		vals, _ := url.ParseQuery(requestURL.RawQuery)
		for _, resource := range resourceList {
			if vv, ok := vals[resource]; ok && len(vv) > 0 {
				n++
				switch n {
				case 1:
					buf.WriteByte('?')
				default:
					buf.WriteByte('&')
				}
				buf.WriteString(resource)
				if len(vv[0]) > 0 {
					buf.WriteByte('=')
					buf.WriteString(vv[0])
				}
			}
		}
	}
}
