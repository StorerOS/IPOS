package net

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type URL url.URL

func (u URL) IsEmpty() bool {
	return u.String() == ""
}

func (u URL) String() string {
	if u.Host != "" {
		host, err := ParseHost(u.Host)
		if err != nil {
			panic(err)
		}
		switch {
		case u.Scheme == "http" && host.Port == 80:
			fallthrough
		case u.Scheme == "https" && host.Port == 443:
			u.Host = host.Name
		}
	}

	uu := url.URL(u)
	return uu.String()
}

func (u URL) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *URL) UnmarshalJSON(data []byte) (err error) {
	var s string
	if err = json.Unmarshal(data, &s); err != nil {
		return err
	}

	if s == "" {
		*u = URL{}
		return nil
	}

	var ru *URL
	if ru, err = ParseURL(s); err != nil {
		return err
	}

	*u = *ru
	return nil
}

func (u URL) DialHTTP(transport *http.Transport) error {
	if transport == nil {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 2 * time.Second,
			}).DialContext,
		}

	}

	var client = &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	return nil
}

func ParseHTTPURL(s string) (u *URL, err error) {
	u, err = ParseURL(s)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	default:
		return nil, fmt.Errorf("unexpected scheme found %s", u.Scheme)
	case "http", "https":
		return u, nil
	}
}

func ParseURL(s string) (u *URL, err error) {
	var uu *url.URL
	if uu, err = url.Parse(s); err != nil {
		return nil, err
	}

	if uu.Hostname() == "" {
		if uu.Scheme != "" {
			return nil, errors.New("scheme appears with empty host")
		}
	} else {
		portStr := uu.Port()
		if portStr == "" {
			switch uu.Scheme {
			case "http":
				portStr = "80"
			case "https":
				portStr = "443"
			}
		}
		if _, err = ParseHost(net.JoinHostPort(uu.Hostname(), portStr)); err != nil {
			return nil, err
		}
	}

	if uu.Path != "" {
		uu.Path = path.Clean(uu.Path)
	}

	if strings.HasSuffix(s, "/") && !strings.HasSuffix(uu.Path, "/") {
		uu.Path += "/"
	}

	v := URL(*uu)
	u = &v
	return u, nil
}

func IsNetworkOrHostDown(err error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(net.Error)
	if ok {
		urlErr, ok := e.(*url.Error)
		if ok {
			switch urlErr.Err.(type) {
			case *net.DNSError, *net.OpError, net.UnknownNetworkError:
				return true
			}
		}
		if e.Timeout() {
			return true
		}
	}
	ok = false
	if strings.Contains(err.Error(), "Connection closed by foreign host") {
		ok = true
	} else if strings.Contains(err.Error(), "TLS handshake timeout") {
		ok = true
	} else if strings.Contains(err.Error(), "i/o timeout") {
		ok = true
	} else if strings.Contains(err.Error(), "connection timed out") {
		ok = true
	} else if strings.Contains(strings.ToLower(err.Error()), "503 service unavailable") {
		ok = true
	}
	return ok
}
