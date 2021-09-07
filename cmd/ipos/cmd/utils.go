package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"

	xhttp "github.com/storeros/ipos/cmd/ipos/http"
	"github.com/storeros/ipos/cmd/ipos/logger"
	"github.com/storeros/ipos/pkg/handlers"
	"github.com/storeros/ipos/pkg/madmin"

	humanize "github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
)

const (
	slashSeparator = "/"
)

func IsErrIgnored(err error, ignoredErrs ...error) bool {
	return IsErr(err, ignoredErrs...)
}

func IsErr(err error, errs ...error) bool {
	for _, exactErr := range errs {
		if errors.Is(err, exactErr) {
			return true
		}
	}
	return false
}

func request2BucketObjectName(r *http.Request) (bucketName, objectName string) {
	path, err := getResource(r.URL.Path, r.Host, globalDomainNames)
	if err != nil {
		logger.CriticalIf(GlobalContext, err)
	}

	return path2BucketObject(path)
}

func path2BucketObjectWithBasePath(basePath, path string) (bucket, prefix string) {
	path = strings.TrimPrefix(path, basePath)
	path = strings.TrimPrefix(path, SlashSeparator)
	m := strings.Index(path, SlashSeparator)
	if m < 0 {
		return path, ""
	}
	return path[:m], path[m+len(SlashSeparator):]
}

func path2BucketObject(s string) (bucket, prefix string) {
	return path2BucketObjectWithBasePath("", s)
}

func getDefaultParityBlocks(drive int) int {
	return drive / 2
}

func getDefaultDataBlocks(drive int) int {
	return drive - getDefaultParityBlocks(drive)
}

func getReadQuorum(drive int) int {
	return getDefaultDataBlocks(drive)
}

func getWriteQuorum(drive int) int {
	return getDefaultDataBlocks(drive) + 1
}

const (
	httpScheme  = "http"
	httpsScheme = "https"
)

func nopCharsetConverter(label string, input io.Reader) (io.Reader, error) {
	return input, nil
}

func xmlDecoder(body io.Reader, v interface{}, size int64) error {
	var lbody io.Reader
	if size > 0 {
		lbody = io.LimitReader(body, size)
	} else {
		lbody = body
	}
	d := xml.NewDecoder(lbody)
	d.CharsetReader = nopCharsetConverter
	return d.Decode(v)
}

func checkValidMD5(h http.Header) ([]byte, error) {
	md5B64, ok := h[xhttp.ContentMD5]
	if ok {
		if md5B64[0] == "" {
			return nil, fmt.Errorf("Content-Md5 header set to empty value")
		}
		return base64.StdEncoding.Strict().DecodeString(md5B64[0])
	}
	return []byte{}, nil
}

func hasContentMD5(h http.Header) bool {
	_, ok := h[xhttp.ContentMD5]
	return ok
}

const (
	globalMaxObjectSize = 5 * humanize.TiByte

	globalMinPartSize = 5 * humanize.MiByte

	globalMaxPartSize = 5 * humanize.GiByte

	globalMaxPartID = 10000

	defaultDialTimeout = 5 * time.Second
)

func isMaxObjectSize(size int64) bool {
	return size > globalMaxObjectSize
}

func isMaxAllowedPartSize(size int64) bool {
	return size > globalMaxPartSize
}

func isMinAllowedPartSize(size int64) bool {
	return size >= globalMinPartSize
}

func isMaxPartID(partID int) bool {
	return partID > globalMaxPartID
}

func contains(slice interface{}, elem interface{}) bool {
	v := reflect.ValueOf(slice)
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			if v.Index(i).Interface() == elem {
				return true
			}
		}
	}
	return false
}

type profilerWrapper struct {
	base   []byte
	stopFn func() ([]byte, error)
	ext    string
}

func (p *profilerWrapper) recordBase(name string, debug int) {
	var buf bytes.Buffer
	p.base = nil
	err := pprof.Lookup(name).WriteTo(&buf, debug)
	if err != nil {
		return
	}
	p.base = buf.Bytes()
}

func (p profilerWrapper) Base() []byte {
	return p.base
}

func (p profilerWrapper) Stop() ([]byte, error) {
	return p.stopFn()
}

func (p profilerWrapper) Extension() string {
	return p.ext
}

func getProfileData() (map[string][]byte, error) {
	globalProfilerMu.Lock()
	defer globalProfilerMu.Unlock()

	if len(globalProfiler) == 0 {
		return nil, errors.New("profiler not enabled")
	}

	dst := make(map[string][]byte, len(globalProfiler))
	for typ, prof := range globalProfiler {
		var err error
		buf, err := prof.Stop()
		delete(globalProfiler, typ)
		if err == nil {
			dst[typ+"."+prof.Extension()] = buf
		}
		buf = prof.Base()
		if len(buf) > 0 {
			dst[typ+"-before"+"."+prof.Extension()] = buf
		}
	}
	return dst, nil
}

func setDefaultProfilerRates() {
	runtime.MemProfileRate = 4096
	runtime.SetMutexProfileFraction(0)
	runtime.SetBlockProfileRate(0)
}

func startProfiler(profilerType string) (iposProfiler, error) {
	var prof profilerWrapper
	prof.ext = "pprof"
	switch madmin.ProfilerType(profilerType) {
	case madmin.ProfilerCPU:
		dirPath, err := ioutil.TempDir("", "profile")
		if err != nil {
			return nil, err
		}
		fn := filepath.Join(dirPath, "cpu.out")
		f, err := os.Create(fn)
		if err != nil {
			return nil, err
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			return nil, err
		}
		prof.stopFn = func() ([]byte, error) {
			pprof.StopCPUProfile()
			err := f.Close()
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(dirPath)
			return ioutil.ReadFile(fn)
		}
	case madmin.ProfilerMEM:
		runtime.GC()
		prof.recordBase("heap", 0)
		prof.stopFn = func() ([]byte, error) {
			runtime.GC()
			var buf bytes.Buffer
			err := pprof.Lookup("heap").WriteTo(&buf, 0)
			return buf.Bytes(), err
		}
	case madmin.ProfilerBlock:
		prof.recordBase("block", 0)
		runtime.SetBlockProfileRate(1)
		prof.stopFn = func() ([]byte, error) {
			var buf bytes.Buffer
			err := pprof.Lookup("block").WriteTo(&buf, 0)
			runtime.SetBlockProfileRate(0)
			return buf.Bytes(), err
		}
	case madmin.ProfilerMutex:
		prof.recordBase("mutex", 0)
		runtime.SetMutexProfileFraction(1)
		prof.stopFn = func() ([]byte, error) {
			var buf bytes.Buffer
			err := pprof.Lookup("mutex").WriteTo(&buf, 0)
			runtime.SetMutexProfileFraction(0)
			return buf.Bytes(), err
		}
	case madmin.ProfilerThreads:
		prof.recordBase("threadcreate", 0)
		prof.stopFn = func() ([]byte, error) {
			var buf bytes.Buffer
			err := pprof.Lookup("threadcreate").WriteTo(&buf, 0)
			return buf.Bytes(), err
		}
	case madmin.ProfilerGoroutines:
		prof.ext = "txt"
		prof.recordBase("goroutine", 1)
		prof.stopFn = func() ([]byte, error) {
			var buf bytes.Buffer
			err := pprof.Lookup("goroutine").WriteTo(&buf, 1)
			return buf.Bytes(), err
		}
	case madmin.ProfilerTrace:
		dirPath, err := ioutil.TempDir("", "profile")
		if err != nil {
			return nil, err
		}
		fn := filepath.Join(dirPath, "trace.out")
		f, err := os.Create(fn)
		if err != nil {
			return nil, err
		}
		err = trace.Start(f)
		if err != nil {
			return nil, err
		}
		prof.ext = "trace"
		prof.stopFn = func() ([]byte, error) {
			trace.Stop()
			err := f.Close()
			if err != nil {
				return nil, err
			}
			defer os.RemoveAll(dirPath)
			return ioutil.ReadFile(fn)
		}
	default:
		return nil, errors.New("profiler type unknown")
	}

	return prof, nil
}

type iposProfiler interface {
	Base() []byte
	Stop() ([]byte, error)
	Extension() string
}

var globalProfiler map[string]iposProfiler
var globalProfilerMu sync.Mutex

func dumpRequest(r *http.Request) string {
	header := r.Header.Clone()
	header.Set("Host", r.Host)
	rawURI := strings.Replace(r.RequestURI, "%", "%%", -1)
	req := struct {
		Method     string      `json:"method"`
		RequestURI string      `json:"reqURI"`
		Header     http.Header `json:"header"`
	}{r.Method, rawURI, header}

	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&req); err != nil {
		return fmt.Sprintf("%#v", req)
	}

	return strings.TrimSpace(buffer.String())
}

func isFile(path string) bool {
	if fi, err := os.Stat(path); err == nil {
		return fi.Mode().IsRegular()
	}

	return false
}

func UTCNow() time.Time {
	return time.Now().UTC()
}

func GenETag() string {
	return ToS3ETag(getMD5Hash([]byte(mustGetUUID())))
}

func ToS3ETag(etag string) string {
	etag = canonicalizeETag(etag)

	if !strings.HasSuffix(etag, "-1") {
		etag += "-1"
	}

	return etag
}

type dialContext func(ctx context.Context, network, address string) (net.Conn, error)

func newCustomDialContext(dialTimeout, dialKeepAlive time.Duration) dialContext {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: dialKeepAlive,
		}

		return dialer.DialContext(ctx, network, addr)
	}
}

func newCustomHTTPTransport(tlsConfig *tls.Config, dialTimeout time.Duration) func() *http.Transport {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           newCustomDialContext(dialTimeout, 15*time.Second),
		MaxIdleConnsPerHost:   16,
		MaxIdleConns:          16,
		IdleConnTimeout:       1 * time.Minute,
		ResponseHeaderTimeout: 3 * time.Minute,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
		TLSClientConfig:       tlsConfig,
		DisableCompression:    true,
	}
	return func() *http.Transport {
		return tr
	}
}

func NewGatewayHTTPTransport() *http.Transport {
	tr := newCustomHTTPTransport(&tls.Config{
		RootCAs: globalRootCAs,
	}, defaultDialTimeout)()
	tr.ResponseHeaderTimeout = 1 * time.Minute

	tr.MaxConnsPerHost = 256
	tr.MaxIdleConnsPerHost = 16
	tr.MaxIdleConns = 256
	return tr
}

func jsonLoad(r io.ReadSeeker, data interface{}) error {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return json.NewDecoder(r).Decode(data)
}

func jsonSave(f interface {
	io.WriteSeeker
	Truncate(int64) error
}, data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err = f.Truncate(0); err != nil {
		return err
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err = f.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func ceilFrac(numerator, denominator int64) (ceil int64) {
	if denominator == 0 {
		return
	}
	if denominator < 0 {
		numerator = -numerator
		denominator = -denominator
	}
	ceil = numerator / denominator
	if numerator > 0 && numerator%denominator != 0 {
		ceil++
	}
	return
}

func newContext(r *http.Request, w http.ResponseWriter, api string) context.Context {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	object, err := url.PathUnescape(vars["object"])
	if err != nil {
		object = vars["object"]
	}
	prefix, err := url.QueryUnescape(vars["prefix"])
	if err != nil {
		prefix = vars["prefix"]
	}
	if prefix != "" {
		object = prefix
	}
	reqInfo := &logger.ReqInfo{
		DeploymentID: globalDeploymentID,
		RequestID:    w.Header().Get(xhttp.AmzRequestID),
		RemoteHost:   handlers.GetSourceIP(r),
		Host:         getHostName(r),
		UserAgent:    r.UserAgent(),
		API:          api,
		BucketName:   bucket,
		ObjectName:   object,
	}
	return logger.SetReqInfo(r.Context(), reqInfo)
}

func restQueries(keys ...string) []string {
	var accumulator []string
	for _, key := range keys {
		accumulator = append(accumulator, key, "{"+key+":.*}")
	}
	return accumulator
}

func reverseStringSlice(input []string) {
	for left, right := 0, len(input)-1; left < right; left, right = left+1, right-1 {
		input[left], input[right] = input[right], input[left]
	}
}

func lcp(l []string) string {
	switch len(l) {
	case 0:
		return ""
	case 1:
		return l[0]
	}
	min, max := l[0], l[0]
	for _, s := range l[1:] {
		switch {
		case s < min:
			min = s
		case s > max:
			max = s
		}
	}
	for i := 0; i < len(min) && i < len(max); i++ {
		if min[i] != max[i] {
			return min[:i]
		}
	}
	return min
}

func iamPolicyClaimNameOpenID() string {
	return "claim_prefix" + "claim_name"
}

func iamPolicyClaimNameSA() string {
	return "sa-policy"
}
