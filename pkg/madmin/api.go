package madmin

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/storeros/ipos/pkg/credentials"
	"github.com/storeros/ipos/pkg/s3utils"
	"github.com/storeros/ipos/pkg/signer"
	"golang.org/x/net/publicsuffix"
)

type AdminClient struct {
	endpointURL *url.URL

	credsProvider *credentials.Credentials

	appInfo struct {
		appName    string
		appVersion string
	}

	secure bool

	httpClient *http.Client

	random *rand.Rand

	isTraceEnabled bool
	traceOutput    io.Writer
}

const (
	libraryName    = "madmin-go"
	libraryVersion = "0.0.1"

	libraryAdminURLPrefix = "/ipos/admin"
)

const (
	libraryUserAgentPrefix = "IPOS (" + runtime.GOOS + "; " + runtime.GOARCH + ") "
	libraryUserAgent       = libraryUserAgentPrefix + libraryName + "/" + libraryVersion
)

type Options struct {
	Creds  *credentials.Credentials
	Secure bool
}

func New(endpoint string, accessKeyID, secretAccessKey string, secure bool) (*AdminClient, error) {
	creds := credentials.NewStaticV4(accessKeyID, secretAccessKey, "")

	clnt, err := privateNew(endpoint, creds, secure)
	if err != nil {
		return nil, err
	}
	return clnt, nil
}

func NewWithOptions(endpoint string, opts *Options) (*AdminClient, error) {
	clnt, err := privateNew(endpoint, opts.Creds, opts.Secure)
	if err != nil {
		return nil, err
	}
	return clnt, nil
}

func privateNew(endpoint string, creds *credentials.Credentials, secure bool) (*AdminClient, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	endpointURL, err := getEndpointURL(endpoint, secure)
	if err != nil {
		return nil, err
	}

	clnt := new(AdminClient)

	clnt.credsProvider = creds

	clnt.secure = secure

	clnt.endpointURL = endpointURL

	clnt.httpClient = &http.Client{
		Jar:       jar,
		Transport: DefaultTransport(secure),
	}

	clnt.random = rand.New(&lockedRandSource{src: rand.NewSource(time.Now().UTC().UnixNano())})

	return clnt, nil
}

func (adm *AdminClient) SetAppInfo(appName string, appVersion string) {
	if appName != "" && appVersion != "" {
		adm.appInfo.appName = appName
		adm.appInfo.appVersion = appVersion
	}
}

func (adm *AdminClient) SetCustomTransport(customHTTPTransport http.RoundTripper) {
	if adm.httpClient != nil {
		adm.httpClient.Transport = customHTTPTransport
	}
}

func (adm *AdminClient) TraceOn(outputStream io.Writer) {
	if outputStream == nil {
		outputStream = os.Stdout
	}
	adm.traceOutput = outputStream

	adm.isTraceEnabled = true
}

func (adm *AdminClient) TraceOff() {
	adm.isTraceEnabled = false
}

type requestData struct {
	customHeaders http.Header
	queryValues   url.Values
	relPath       string
	content       []byte
}

func (adm AdminClient) filterSignature(req *http.Request) {
	origAuth := req.Header.Get("Authorization")
	regCred := regexp.MustCompile("Credential=([A-Z0-9]+)/")
	newAuth := regCred.ReplaceAllString(origAuth, "Credential=**REDACTED**/")

	regSign := regexp.MustCompile("Signature=([[0-9a-f]+)")
	newAuth = regSign.ReplaceAllString(newAuth, "Signature=**REDACTED**")

	req.Header.Set("Authorization", newAuth)
}

func (adm AdminClient) dumpHTTP(req *http.Request, resp *http.Response) error {
	_, err := fmt.Fprintln(adm.traceOutput, "---------START-HTTP---------")
	if err != nil {
		return err
	}

	adm.filterSignature(req)

	reqTrace, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(adm.traceOutput, string(reqTrace))
	if err != nil {
		return err
	}

	var respTrace []byte

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusPartialContent &&
		resp.StatusCode != http.StatusNoContent {
		respTrace, err = httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
	} else {
		if resp.ContentLength == 0 {
			var buffer bytes.Buffer
			if err = resp.Header.Write(&buffer); err != nil {
				return err
			}
			respTrace = buffer.Bytes()
			respTrace = append(respTrace, []byte("\r\n")...)
		} else {
			respTrace, err = httputil.DumpResponse(resp, false)
			if err != nil {
				return err
			}
		}
	}
	_, err = fmt.Fprint(adm.traceOutput, strings.TrimSuffix(string(respTrace), "\r\n"))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(adm.traceOutput, "---------END-HTTP---------")
	return err
}

func (adm AdminClient) do(req *http.Request) (*http.Response, error) {
	resp, err := adm.httpClient.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			if strings.Contains(urlErr.Err.Error(), "EOF") {
				return nil, &url.Error{
					Op:  urlErr.Op,
					URL: urlErr.URL,
					Err: errors.New("Connection closed by foreign host " + urlErr.URL + ". Retry again."),
				}
			}
		}
		return nil, err
	}

	if resp == nil {
		msg := "Response is empty. "
		return nil, ErrInvalidArgument(msg)
	}

	if adm.isTraceEnabled {
		err = adm.dumpHTTP(req, resp)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

var successStatus = []int{
	http.StatusOK,
	http.StatusNoContent,
	http.StatusPartialContent,
}

func (adm AdminClient) executeMethod(ctx context.Context, method string, reqData requestData) (res *http.Response, err error) {
	var reqRetry = MaxRetry

	defer func() {
		if err != nil {
			adm.httpClient.CloseIdleConnections()
		}
	}()

	retryCtx, cancel := context.WithCancel(ctx)

	defer cancel()

	for range adm.newRetryTimer(retryCtx, reqRetry, DefaultRetryUnit, DefaultRetryCap, MaxJitter) {
		var req *http.Request
		req, err = adm.newRequest(method, reqData)
		if err != nil {
			return nil, err
		}

		req = req.WithContext(ctx)

		res, err = adm.do(req)
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				return nil, err
			}
			continue
		}

		for _, httpStatus := range successStatus {
			if httpStatus == res.StatusCode {
				return res, nil
			}
		}

		errBodyBytes, err := ioutil.ReadAll(res.Body)
		closeResponse(res)
		if err != nil {
			return nil, err
		}

		errBodySeeker := bytes.NewReader(errBodyBytes)
		res.Body = ioutil.NopCloser(errBodySeeker)

		errResponse := ToErrorResponse(httpRespToErrorResponse(res))

		errBodySeeker.Seek(0, 0)
		res.Body = ioutil.NopCloser(errBodySeeker)

		if isS3CodeRetryable(errResponse.Code) {
			continue
		}

		if isHTTPStatusRetryable(res.StatusCode) {
			continue
		}

		break
	}

	if e := retryCtx.Err(); e != nil {
		return nil, e
	}

	return res, err
}

func (adm AdminClient) setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", libraryUserAgent)
	if adm.appInfo.appName != "" && adm.appInfo.appVersion != "" {
		req.Header.Set("User-Agent", libraryUserAgent+" "+adm.appInfo.appName+"/"+adm.appInfo.appVersion)
	}
}

func (adm AdminClient) getSecretKey() string {
	value, err := adm.credsProvider.Get()
	if err != nil {
		return ""
	}

	return value.SecretAccessKey
}

func (adm AdminClient) newRequest(method string, reqData requestData) (req *http.Request, err error) {
	if method == "" {
		method = "POST"
	}

	location := ""

	targetURL, err := adm.makeTargetURL(reqData)
	if err != nil {
		return nil, err
	}

	req, err = http.NewRequest(method, targetURL.String(), nil)
	if err != nil {
		return nil, err
	}

	value, err := adm.credsProvider.Get()
	if err != nil {
		return nil, err
	}

	var (
		accessKeyID     = value.AccessKeyID
		secretAccessKey = value.SecretAccessKey
		sessionToken    = value.SessionToken
	)

	adm.setUserAgent(req)
	for k, v := range reqData.customHeaders {
		req.Header.Set(k, v[0])
	}
	if length := len(reqData.content); length > 0 {
		req.ContentLength = int64(length)
	}
	req.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(sum256(reqData.content)))
	req.Body = ioutil.NopCloser(bytes.NewReader(reqData.content))

	req = signer.SignV4(*req, accessKeyID, secretAccessKey, sessionToken, location)
	return req, nil
}

func (adm AdminClient) makeTargetURL(r requestData) (*url.URL, error) {

	host := adm.endpointURL.Host
	scheme := adm.endpointURL.Scheme

	urlStr := scheme + "://" + host + libraryAdminURLPrefix + r.relPath

	if len(r.queryValues) > 0 {
		urlStr = urlStr + "?" + s3utils.QueryEncode(r.queryValues)
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	return u, nil
}
