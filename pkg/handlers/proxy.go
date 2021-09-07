package handlers

import (
	"net"
	"net/http"
	"regexp"
	"strings"
)

var (
	xForwardedFor    = http.CanonicalHeaderKey("X-Forwarded-For")
	xForwardedHost   = http.CanonicalHeaderKey("X-Forwarded-Host")
	xForwardedPort   = http.CanonicalHeaderKey("X-Forwarded-Port")
	xForwardedProto  = http.CanonicalHeaderKey("X-Forwarded-Proto")
	xForwardedScheme = http.CanonicalHeaderKey("X-Forwarded-Scheme")
	xRealIP          = http.CanonicalHeaderKey("X-Real-IP")
)

var (
	forwarded  = http.CanonicalHeaderKey("Forwarded")
	forRegex   = regexp.MustCompile(`(?i)(?:for=)([^(;|,| )]+)(.*)`)
	protoRegex = regexp.MustCompile(`(?i)^(;|,| )+(?:proto=)(https|http)`)
)

func GetSourceScheme(r *http.Request) string {
	var scheme string

	if proto := r.Header.Get(xForwardedProto); proto != "" {
		scheme = strings.ToLower(proto)
	} else if proto = r.Header.Get(xForwardedScheme); proto != "" {
		scheme = strings.ToLower(proto)
	} else if proto := r.Header.Get(forwarded); proto != "" {
		if match := forRegex.FindStringSubmatch(proto); len(match) > 1 {
			if match = protoRegex.FindStringSubmatch(match[2]); len(match) > 1 {
				scheme = strings.ToLower(match[2])
			}
		}
	}

	return scheme
}

func GetSourceIP(r *http.Request) string {
	var addr string

	if fwd := r.Header.Get(xForwardedFor); fwd != "" {
		s := strings.Index(fwd, ", ")
		if s == -1 {
			s = len(fwd)
		}
		addr = fwd[:s]
	} else if fwd := r.Header.Get(xRealIP); fwd != "" {
		addr = fwd
	} else if fwd := r.Header.Get(forwarded); fwd != "" {
		if match := forRegex.FindStringSubmatch(fwd); len(match) > 1 {
			addr = strings.Trim(match[1], `"`)
		}
	}

	if addr != "" {
		return addr
	}

	addr, _, _ = net.SplitHostPort(r.RemoteAddr)
	return addr
}
