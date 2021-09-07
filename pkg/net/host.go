package net

import (
	"encoding/json"
	"errors"
	"net"
	"regexp"
	"strings"
)

var hostLabelRegexp = regexp.MustCompile("^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$")

type Host struct {
	Name      string
	Port      Port
	IsPortSet bool
}

func (host Host) IsEmpty() bool {
	return host.Name == ""
}

func (host Host) String() string {
	if !host.IsPortSet {
		return host.Name
	}

	return net.JoinHostPort(host.Name, host.Port.String())
}

func (host Host) Equal(compHost Host) bool {
	return host.String() == compHost.String()
}

func (host Host) MarshalJSON() ([]byte, error) {
	return json.Marshal(host.String())
}

func (host *Host) UnmarshalJSON(data []byte) (err error) {
	var s string
	if err = json.Unmarshal(data, &s); err != nil {
		return err
	}

	if s == "" {
		*host = Host{}
		return nil
	}

	var h *Host
	if h, err = ParseHost(s); err != nil {
		return err
	}

	*host = *h
	return nil
}

func ParseHost(s string) (*Host, error) {
	if s == "" {
		return nil, errors.New("invalid argument")
	}
	isValidHost := func(host string) bool {
		if host == "" {
			return true
		}

		if ip := net.ParseIP(host); ip != nil {
			return true
		}

		if len(host) < 1 || len(host) > 253 {
			return false
		}

		for _, label := range strings.Split(host, ".") {
			if len(label) < 1 || len(label) > 63 {
				return false
			}

			if !hostLabelRegexp.MatchString(label) {
				return false
			}
		}

		return true
	}

	var port Port
	var isPortSet bool
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		if !strings.Contains(err.Error(), "missing port in address") {
			return nil, err
		}
		host = s
	} else {
		if port, err = ParsePort(portStr); err != nil {
			return nil, err
		}

		isPortSet = true
	}

	if host != "" {
		host, err = trimIPv6(host)
		if err != nil {
			return nil, err
		}
	}

	trimmedHost := host

	if i := strings.LastIndex(trimmedHost, "%"); i > -1 {
		trimmedHost = trimmedHost[:i]
	}

	if !isValidHost(trimmedHost) {
		return nil, errors.New("invalid hostname")
	}

	return &Host{
		Name:      host,
		Port:      port,
		IsPortSet: isPortSet,
	}, nil
}

func trimIPv6(host string) (string, error) {
	if host[len(host)-1] == ']' {
		if host[0] != '[' {
			return "", errors.New("missing '[' in host")
		}
		return host[1:][:len(host)-2], nil
	}
	return host, nil
}
