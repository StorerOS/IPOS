package cmd

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/storeros/ipos/cmd/ipos/crypto"
	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/version"
)

func mustGetRequestID(t time.Time) string {
	return fmt.Sprintf("%X", t.UnixNano())
}

func setCommonHeaders(w http.ResponseWriter) {
	w.Header().Set(xhttp.ServerInfo, "IPOS/"+version.Version)
	if region := globalServerRegion; region != "" {
		w.Header().Set(xhttp.AmzBucketRegion, region)
	}
	w.Header().Set(xhttp.AcceptRanges, "bytes")

	crypto.RemoveSensitiveHeaders(w.Header())
}

func encodeResponse(response interface{}) []byte {
	var bytesBuffer bytes.Buffer
	bytesBuffer.WriteString(xml.Header)
	e := xml.NewEncoder(&bytesBuffer)
	e.Encode(response)
	return bytesBuffer.Bytes()
}

func encodeResponseJSON(response interface{}) []byte {
	var bytesBuffer bytes.Buffer
	e := json.NewEncoder(&bytesBuffer)
	e.Encode(response)
	return bytesBuffer.Bytes()
}

func setObjectHeaders(w http.ResponseWriter, objInfo ObjectInfo, rs *HTTPRangeSpec) (err error) {
	setCommonHeaders(w)

	lastModified := objInfo.ModTime.UTC().Format(http.TimeFormat)
	w.Header().Set(xhttp.LastModified, lastModified)

	if objInfo.ETag != "" {
		w.Header()[xhttp.ETag] = []string{"\"" + objInfo.ETag + "\""}
	}

	if objInfo.ContentType != "" {
		w.Header().Set(xhttp.ContentType, objInfo.ContentType)
	}

	if objInfo.ContentEncoding != "" {
		w.Header().Set(xhttp.ContentEncoding, objInfo.ContentEncoding)
	}

	if !objInfo.Expires.IsZero() {
		w.Header().Set(xhttp.Expires, objInfo.Expires.UTC().Format(http.TimeFormat))
	}

	tags, _ := url.ParseQuery(objInfo.UserTags)
	tagCount := len(tags)
	if tagCount != 0 {
		w.Header().Set(xhttp.AmzTagCount, strconv.Itoa(tagCount))
	}

	for k, v := range objInfo.UserDefined {
		if HasPrefix(k, ReservedMetadataPrefix) {
			continue
		}
		w.Header().Set(k, v)
	}

	var totalObjectSize int64
	switch {
	case crypto.IsEncrypted(objInfo.UserDefined):
		totalObjectSize, err = objInfo.DecryptedSize()
		if err != nil {
			return err
		}
	case objInfo.IsCompressed():
		totalObjectSize = objInfo.GetActualSize()
		if totalObjectSize < 0 {
			return errInvalidDecompressedSize
		}
	default:
		totalObjectSize = objInfo.Size
	}

	start, rangeLen, err := rs.GetOffsetLength(totalObjectSize)
	if err != nil {
		return err
	}

	w.Header().Set(xhttp.ContentLength, strconv.FormatInt(rangeLen, 10))
	if rs != nil {
		contentRange := fmt.Sprintf("bytes %d-%d/%d", start, start+rangeLen-1, totalObjectSize)
		w.Header().Set(xhttp.ContentRange, contentRange)
	}

	return nil
}
