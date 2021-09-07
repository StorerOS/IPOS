package jwt

import (
	"crypto"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"
	jsoniter "github.com/json-iterator/go"
)

type SigningMethodHMAC struct {
	Name string
	Hash crypto.Hash
}

var (
	SigningMethodHS256 *SigningMethodHMAC
	SigningMethodHS384 *SigningMethodHMAC
	SigningMethodHS512 *SigningMethodHMAC
)

var (
	base64BufPool sync.Pool
	hmacSigners   []*SigningMethodHMAC
)

func init() {
	base64BufPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 8192)
			return &buf
		},
	}

	hmacSigners = []*SigningMethodHMAC{
		{"HS256", crypto.SHA256},
		{"HS384", crypto.SHA384},
		{"HS512", crypto.SHA512},
	}
}

type StandardClaims struct {
	AccessKey string `json:"accessKey,omitempty"`
	jwtgo.StandardClaims
}

type MapClaims struct {
	AccessKey string `json:"accessKey,omitempty"`
	jwtgo.MapClaims
}

func NewStandardClaims() *StandardClaims {
	return &StandardClaims{}
}

func (c *StandardClaims) SetIssuer(issuer string) {
	c.Issuer = issuer
}

func (c *StandardClaims) SetAudience(aud string) {
	c.Audience = aud
}

func (c *StandardClaims) SetExpiry(t time.Time) {
	c.ExpiresAt = t.Unix()
}

func (c *StandardClaims) SetAccessKey(accessKey string) {
	c.Subject = accessKey
	c.AccessKey = accessKey
}

func (c *StandardClaims) Valid() error {
	if err := c.StandardClaims.Valid(); err != nil {
		return err
	}

	if c.AccessKey == "" && c.Subject == "" {
		return jwtgo.NewValidationError("accessKey/sub missing",
			jwtgo.ValidationErrorClaimsInvalid)
	}

	return nil
}

func NewMapClaims() *MapClaims {
	return &MapClaims{MapClaims: jwtgo.MapClaims{}}
}

func (c *MapClaims) Lookup(key string) (value string, ok bool) {
	var vinterface interface{}
	vinterface, ok = c.MapClaims[key]
	if ok {
		value, ok = vinterface.(string)
	}
	return
}

func (c *MapClaims) SetExpiry(t time.Time) {
	c.MapClaims["exp"] = t.Unix()
}

func (c *MapClaims) SetAccessKey(accessKey string) {
	c.MapClaims["sub"] = accessKey
	c.MapClaims["accessKey"] = accessKey
}

func (c *MapClaims) Valid() error {
	if err := c.MapClaims.Valid(); err != nil {
		return err
	}

	if c.AccessKey == "" {
		return jwtgo.NewValidationError("accessKey/sub missing",
			jwtgo.ValidationErrorClaimsInvalid)
	}

	return nil
}

func (c *MapClaims) Map() map[string]interface{} {
	return c.MapClaims
}

func (c *MapClaims) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.MapClaims)
}

func ParseWithStandardClaims(tokenStr string, claims *StandardClaims, key []byte) error {
	if key == nil {
		return jwtgo.NewValidationError("no key was provided.", jwtgo.ValidationErrorUnverifiable)
	}

	bufp := base64BufPool.Get().(*[]byte)
	defer base64BufPool.Put(bufp)

	signer, err := ParseUnverifiedStandardClaims(tokenStr, claims, *bufp)
	if err != nil {
		return err
	}

	i := strings.LastIndex(tokenStr, ".")
	if i < 0 {
		return jwtgo.ErrSignatureInvalid
	}

	n, err := base64Decode(tokenStr[i+1:], *bufp)
	if err != nil {
		return err
	}

	hasher := hmac.New(signer.Hash.New, key)
	hasher.Write([]byte(tokenStr[:i]))
	if !hmac.Equal((*bufp)[:n], hasher.Sum(nil)) {
		return jwtgo.ErrSignatureInvalid
	}

	if claims.AccessKey == "" && claims.Subject == "" {
		return jwtgo.NewValidationError("accessKey/sub missing",
			jwtgo.ValidationErrorClaimsInvalid)
	}

	return claims.Valid()
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

func ParseUnverifiedStandardClaims(tokenString string, claims *StandardClaims, buf []byte) (*SigningMethodHMAC, error) {
	if strings.Count(tokenString, ".") != 2 {
		return nil, jwtgo.ErrSignatureInvalid
	}

	i := strings.Index(tokenString, ".")
	j := strings.LastIndex(tokenString, ".")

	n, err := base64Decode(tokenString[:i], buf)
	if err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	var header = jwtHeader{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err = json.Unmarshal(buf[:n], &header); err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	n, err = base64Decode(tokenString[i+1:j], buf)
	if err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	if err = json.Unmarshal(buf[:n], claims); err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	for _, signer := range hmacSigners {
		if header.Algorithm == signer.Name {
			return signer, nil
		}
	}

	return nil, jwtgo.NewValidationError(fmt.Sprintf("signing method (%s) is unavailable.", header.Algorithm),
		jwtgo.ValidationErrorUnverifiable)
}

func ParseWithClaims(tokenStr string, claims *MapClaims, fn func(*MapClaims) ([]byte, error)) error {
	if fn == nil {
		return jwtgo.NewValidationError("no Keyfunc was provided.", jwtgo.ValidationErrorUnverifiable)
	}

	bufp := base64BufPool.Get().(*[]byte)
	defer base64BufPool.Put(bufp)

	signer, err := ParseUnverifiedMapClaims(tokenStr, claims, *bufp)
	if err != nil {
		return err
	}

	i := strings.LastIndex(tokenStr, ".")
	if i < 0 {
		return jwtgo.ErrSignatureInvalid
	}

	n, err := base64Decode(tokenStr[i+1:], *bufp)
	if err != nil {
		return err
	}

	var ok bool
	claims.AccessKey, ok = claims.Lookup("accessKey")
	if !ok {
		claims.AccessKey, ok = claims.Lookup("sub")
		if !ok {
			return jwtgo.NewValidationError("accessKey/sub missing",
				jwtgo.ValidationErrorClaimsInvalid)
		}
	}

	key, err := fn(claims)
	if err != nil {
		return err
	}

	hasher := hmac.New(signer.Hash.New, key)
	hasher.Write([]byte(tokenStr[:i]))
	if !hmac.Equal((*bufp)[:n], hasher.Sum(nil)) {
		return jwtgo.ErrSignatureInvalid
	}

	return claims.Valid()
}

func base64Decode(s string, buf []byte) (int, error) {
	return base64.RawURLEncoding.Decode(buf, []byte(s))
}

func ParseUnverifiedMapClaims(tokenString string, claims *MapClaims, buf []byte) (*SigningMethodHMAC, error) {
	if strings.Count(tokenString, ".") != 2 {
		return nil, jwtgo.ErrSignatureInvalid
	}

	i := strings.Index(tokenString, ".")
	j := strings.LastIndex(tokenString, ".")

	n, err := base64Decode(tokenString[:i], buf)
	if err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	var header = jwtHeader{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	if err = json.Unmarshal(buf[:n], &header); err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	n, err = base64Decode(tokenString[i+1:j], buf)
	if err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	if err = json.Unmarshal(buf[:n], &claims.MapClaims); err != nil {
		return nil, &jwtgo.ValidationError{Inner: err, Errors: jwtgo.ValidationErrorMalformed}
	}

	for _, signer := range hmacSigners {
		if header.Algorithm == signer.Name {
			return signer, nil
		}
	}

	return nil, jwtgo.NewValidationError(fmt.Sprintf("signing method (%s) is unavailable.", header.Algorithm),
		jwtgo.ValidationErrorUnverifiable)
}
