package signer

import (
	"crypto/hmac"
	"net/http"
	"strings"

	"github.com/storeros/ipos/pkg/sha256-simd"
)

const unsignedPayload = "UNSIGNED-PAYLOAD"

func sum256(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}

func sumHMAC(key []byte, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func getHostAddr(req *http.Request) string {
	if req.Host != "" {
		return req.Host
	}
	return req.URL.Host
}

func signV4TrimAll(input string) string {
	return strings.Join(strings.Fields(input), " ")
}
