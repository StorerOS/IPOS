package handlers

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const defaultFlushInterval = time.Duration(100) * time.Millisecond

type Forwarder struct {
	RoundTripper http.RoundTripper
	PassHost     bool
	Logger       func(error)

	rewriter *headerRewriter
}

func NewForwarder(f *Forwarder) *Forwarder {
	f.rewriter = &headerRewriter{}
	if f.RoundTripper == nil {
		f.RoundTripper = http.DefaultTransport
	}

	return f
}

func (f *Forwarder) ServeHTTP(w http.ResponseWriter, inReq *http.Request) {
	outReq := new(http.Request)
	*outReq = *inReq

	revproxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			f.modifyRequest(req, inReq.URL)
		},
		Transport:     f.RoundTripper,
		FlushInterval: defaultFlushInterval,
		ErrorHandler:  f.customErrHandler,
	}
	revproxy.ServeHTTP(w, outReq)
}

func (f *Forwarder) customErrHandler(w http.ResponseWriter, r *http.Request, err error) {
	if f.Logger != nil && err != context.Canceled {
		f.Logger(err)
	}
	w.WriteHeader(http.StatusBadGateway)
}

func (f *Forwarder) getURLFromRequest(req *http.Request) *url.URL {
	u := req.URL
	if req.RequestURI != "" {
		parsedURL, err := url.ParseRequestURI(req.RequestURI)
		if err == nil {
			u = parsedURL
		}
	}
	return u
}

func copyURL(i *url.URL) *url.URL {
	out := *i
	if i.User != nil {
		u := *i.User
		out.User = &u
	}
	return &out
}

func (f *Forwarder) modifyRequest(outReq *http.Request, target *url.URL) {
	outReq.URL = copyURL(outReq.URL)
	outReq.URL.Scheme = target.Scheme
	outReq.URL.Host = target.Host

	u := f.getURLFromRequest(outReq)

	outReq.URL.Path = u.Path
	outReq.URL.RawPath = u.RawPath
	outReq.URL.RawQuery = u.RawQuery
	outReq.RequestURI = ""

	if !f.PassHost {
		outReq.Host = target.Host
	}

	outReq.Proto = "HTTP/1.1"
	outReq.ProtoMajor = 1
	outReq.ProtoMinor = 1

	f.rewriter.Rewrite(outReq)

	if outReq.Method == http.MethodGet {
		quietReq := outReq.WithContext(context.Background())
		*outReq = *quietReq
	}
}

type headerRewriter struct{}

func ipv6fix(clientIP string) string {
	return strings.Split(clientIP, "%")[0]
}

func (rw *headerRewriter) Rewrite(req *http.Request) {
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		clientIP = ipv6fix(clientIP)
		if req.Header.Get(xRealIP) == "" {
			req.Header.Set(xRealIP, clientIP)
		}
	}

	xfProto := req.Header.Get(xForwardedProto)
	if xfProto == "" {
		if req.TLS != nil {
			req.Header.Set(xForwardedProto, "https")
		} else {
			req.Header.Set(xForwardedProto, "http")
		}
	}

	if xfPort := req.Header.Get(xForwardedPort); xfPort == "" {
		req.Header.Set(xForwardedPort, forwardedPort(req))
	}

	if xfHost := req.Header.Get(xForwardedHost); xfHost == "" && req.Host != "" {
		req.Header.Set(xForwardedHost, req.Host)
	}
}

func forwardedPort(req *http.Request) string {
	if req == nil {
		return ""
	}

	if _, port, err := net.SplitHostPort(req.Host); err == nil && port != "" {
		return port
	}

	if req.TLS != nil {
		return "443"
	}

	return "80"
}
