package credentials

import "strings"

type SignatureType int

const (
	SignatureDefault SignatureType = iota
	SignatureV4
	SignatureV2
	SignatureV4Streaming
	SignatureAnonymous
)

func (s SignatureType) IsV2() bool {
	return s == SignatureV2
}

func (s SignatureType) IsV4() bool {
	return s == SignatureV4 || s == SignatureDefault
}

func (s SignatureType) IsStreamingV4() bool {
	return s == SignatureV4Streaming
}

func (s SignatureType) IsAnonymous() bool {
	return s == SignatureAnonymous
}

func (s SignatureType) String() string {
	if s.IsV2() {
		return "S3v2"
	} else if s.IsV4() {
		return "S3v4"
	} else if s.IsStreamingV4() {
		return "S3v4Streaming"
	}
	return "Anonymous"
}

func parseSignatureType(str string) SignatureType {
	if strings.EqualFold(str, "S3v4") {
		return SignatureV4
	} else if strings.EqualFold(str, "S3v2") {
		return SignatureV2
	} else if strings.EqualFold(str, "S3v4Streaming") {
		return SignatureV4Streaming
	}
	return SignatureAnonymous
}
