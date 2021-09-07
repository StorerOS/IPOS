package logger

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/storeros/ipos/cmd/ipos/logger/message/audit"
	"github.com/gorilla/mux"
)

type ResponseWriter struct {
	http.ResponseWriter
	StatusCode      int
	LogBody         bool
	TimeToFirstByte time.Duration
	StartTime       time.Time
	bytesWritten    int
	headers         bytes.Buffer
	body            bytes.Buffer
	headersLogged   bool
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
		StartTime:      time.Now().UTC(),
	}
}

func (lrw *ResponseWriter) Write(p []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(p)
	lrw.bytesWritten += n
	if lrw.TimeToFirstByte == 0 {
		lrw.TimeToFirstByte = time.Now().UTC().Sub(lrw.StartTime)
	}
	if !lrw.headersLogged {
		lrw.writeHeaders(&lrw.headers, http.StatusOK, lrw.Header())
		lrw.headersLogged = true
	}
	if lrw.StatusCode >= http.StatusBadRequest || lrw.LogBody {
		lrw.body.Write(p)
	}
	if err != nil {
		return n, err
	}
	return n, err
}

func (lrw *ResponseWriter) writeHeaders(w io.Writer, statusCode int, headers http.Header) {
	n, _ := fmt.Fprintf(w, "%d %s\n", statusCode, http.StatusText(statusCode))
	lrw.bytesWritten += n
	for k, v := range headers {
		n, _ := fmt.Fprintf(w, "%s: %s\n", k, v[0])
		lrw.bytesWritten += n
	}
}

var BodyPlaceHolder = []byte("<BODY>")

func (lrw *ResponseWriter) Body() []byte {
	if lrw.StatusCode >= http.StatusBadRequest || lrw.LogBody {
		return lrw.body.Bytes()
	}
	return BodyPlaceHolder
}

func (lrw *ResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	if !lrw.headersLogged {
		lrw.writeHeaders(&lrw.headers, code, lrw.ResponseWriter.Header())
		lrw.headersLogged = true
	}
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *ResponseWriter) Flush() {
	lrw.ResponseWriter.(http.Flusher).Flush()
}

func (lrw *ResponseWriter) Size() int {
	return lrw.bytesWritten
}

var AuditTargets = []Target{}

func AddAuditTarget(t Target) {
	AuditTargets = append(AuditTargets, t)
}

func AuditLog(w http.ResponseWriter, r *http.Request, api string, reqClaims map[string]interface{}) {
	var statusCode int
	var timeToResponse time.Duration
	var timeToFirstByte time.Duration
	lrw, ok := w.(*ResponseWriter)
	if ok {
		statusCode = lrw.StatusCode
		timeToResponse = time.Now().UTC().Sub(lrw.StartTime)
		timeToFirstByte = lrw.TimeToFirstByte
	}

	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object, err := url.PathUnescape(vars["object"])
	if err != nil {
		object = vars["object"]
	}

	for _, t := range AuditTargets {
		entry := audit.ToEntry(w, r, reqClaims, globalDeploymentID)
		entry.API.Name = api
		entry.API.Bucket = bucket
		entry.API.Object = object
		entry.API.Status = http.StatusText(statusCode)
		entry.API.StatusCode = statusCode
		entry.API.TimeToFirstByte = timeToFirstByte.String()
		entry.API.TimeToResponse = timeToResponse.String()
		_ = t.Send(entry, string(All))
	}
}
