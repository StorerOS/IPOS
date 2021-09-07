package cmd

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/handlers"
	trace "github.com/storeros/ipos/pkg/trace"
)

type recordRequest struct {
	io.Reader
	logBody   bool
	buf       bytes.Buffer
	headers   http.Header
	bytesRead int
}

func (r *recordRequest) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.bytesRead += n

	if r.logBody {
		r.buf.Write(p[:n])
	}
	if err != nil {
		return n, err
	}
	return n, err
}
func (r *recordRequest) Size() int {
	sz := r.bytesRead
	for k, v := range r.headers {
		sz += len(k) + len(v)
	}
	return sz
}

func (r *recordRequest) Data() []byte {
	if r.logBody {
		return r.buf.Bytes()
	}
	return logger.BodyPlaceHolder
}

func getOpName(name string) (op string) {
	op = strings.TrimPrefix(name, "github.com/storeros/ipos/cmd.")
	op = strings.TrimSuffix(op, "Handler-fm")
	op = strings.Replace(op, "objectAPIHandlers", "s3", 1)
	op = strings.Replace(op, "webAPIHandlers", "webui", 1)
	op = strings.Replace(op, "adminAPIHandlers", "admin", 1)
	op = strings.Replace(op, "(*storageRESTServer)", "internal", 1)
	op = strings.Replace(op, "(*peerRESTServer)", "internal", 1)
	op = strings.Replace(op, "(*lockRESTServer)", "internal", 1)
	op = strings.Replace(op, "(*stsAPIHandlers)", "sts", 1)
	op = strings.Replace(op, "LivenessCheckHandler", "healthcheck", 1)
	op = strings.Replace(op, "ReadinessCheckHandler", "healthcheck", 1)
	op = strings.Replace(op, "-fm", "", 1)
	return op
}

func Trace(f http.HandlerFunc, logBody bool, w http.ResponseWriter, r *http.Request) trace.Info {
	name := getOpName(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())

	reqHeaders := r.Header.Clone()
	reqHeaders.Set("Host", r.Host)
	if len(r.TransferEncoding) == 0 {
		reqHeaders.Set("Content-Length", strconv.Itoa(int(r.ContentLength)))
	}
	for _, enc := range r.TransferEncoding {
		reqHeaders.Add("Transfer-Encoding", enc)
	}

	var reqBodyRecorder *recordRequest
	t := trace.Info{FuncName: name}
	reqBodyRecorder = &recordRequest{Reader: r.Body, logBody: logBody, headers: reqHeaders}
	r.Body = ioutil.NopCloser(reqBodyRecorder)
	t.NodeName = r.Host
	if host, _, err := net.SplitHostPort(t.NodeName); err == nil {
		t.NodeName = host
	}

	rw := logger.NewResponseWriter(w)
	rw.LogBody = logBody
	f(rw, r)

	rq := trace.RequestInfo{
		Time:     time.Now().UTC(),
		Method:   r.Method,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
		Client:   handlers.GetSourceIP(r),
		Headers:  reqHeaders,
		Body:     reqBodyRecorder.Data(),
	}
	rs := trace.ResponseInfo{
		Time:       time.Now().UTC(),
		Headers:    rw.Header().Clone(),
		StatusCode: rw.StatusCode,
		Body:       rw.Body(),
	}

	if rs.StatusCode == 0 {
		rs.StatusCode = http.StatusOK
	}

	t.ReqInfo = rq
	t.RespInfo = rs

	t.CallStats = trace.CallStats{
		Latency:         rs.Time.Sub(rw.StartTime),
		InputBytes:      reqBodyRecorder.Size(),
		OutputBytes:     rw.Size(),
		TimeToFirstByte: rw.TimeToFirstByte,
	}
	return t
}
