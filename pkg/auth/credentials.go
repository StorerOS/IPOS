package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"
)

const (
	accessKeyMinLen = 3

	accessKeyMaxLen = 20

	secretKeyMinLen = 8

	secretKeyMaxLen = 40

	alphaNumericTable = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	alphaNumericTableLen = byte(len(alphaNumericTable))
)

var (
	ErrInvalidAccessKeyLength = fmt.Errorf("access key must be minimum %v or more characters long", accessKeyMinLen)
	ErrInvalidSecretKeyLength = fmt.Errorf("secret key must be minimum %v or more characters long", secretKeyMinLen)
)

func IsAccessKeyValid(accessKey string) bool {
	return len(accessKey) >= accessKeyMinLen
}

func IsSecretKeyValid(secretKey string) bool {
	return len(secretKey) >= secretKeyMinLen
}

const (
	DefaultAccessKey = "iposadmin"
	DefaultSecretKey = "iposadmin"
)

var (
	DefaultCredentials = Credentials{
		AccessKey: DefaultAccessKey,
		SecretKey: DefaultSecretKey,
	}
)

type Credentials struct {
	AccessKey    string    `xml:"AccessKeyId" json:"accessKey,omitempty"`
	SecretKey    string    `xml:"SecretAccessKey" json:"secretKey,omitempty"`
	Expiration   time.Time `xml:"Expiration" json:"expiration,omitempty"`
	SessionToken string    `xml:"SessionToken" json:"sessionToken,omitempty"`
	Status       string    `xml:"-" json:"status,omitempty"`
	ParentUser   string    `xml:"-" json:"parentUser,omitempty"`
}

func (cred Credentials) String() string {
	var s strings.Builder
	s.WriteString(cred.AccessKey)
	s.WriteString(":")
	s.WriteString(cred.SecretKey)
	if cred.SessionToken != "" {
		s.WriteString("\n")
		s.WriteString(cred.SessionToken)
	}
	if !cred.Expiration.IsZero() && !cred.Expiration.Equal(timeSentinel) {
		s.WriteString("\n")
		s.WriteString(cred.Expiration.String())
	}
	return s.String()
}

func (cred Credentials) IsExpired() bool {
	if cred.Expiration.IsZero() || cred.Expiration.Equal(timeSentinel) {
		return false
	}

	return cred.Expiration.Before(time.Now().UTC())
}

func (cred Credentials) IsTemp() bool {
	return cred.SessionToken != "" && cred.ParentUser == "" && !cred.Expiration.IsZero() && !cred.Expiration.Equal(timeSentinel)
}

func (cred Credentials) IsServiceAccount() bool {
	return cred.ParentUser != "" && (cred.Expiration.IsZero() || cred.Expiration.Equal(timeSentinel))
}

func (cred Credentials) IsValid() bool {
	if cred.Status == "off" {
		return false
	}
	return IsAccessKeyValid(cred.AccessKey) && IsSecretKeyValid(cred.SecretKey) && !cred.IsExpired()
}

func (cred Credentials) Equal(ccred Credentials) bool {
	if !ccred.IsValid() {
		return false
	}
	return (cred.AccessKey == ccred.AccessKey && subtle.ConstantTimeCompare([]byte(cred.SecretKey), []byte(ccred.SecretKey)) == 1 &&
		subtle.ConstantTimeCompare([]byte(cred.SessionToken), []byte(ccred.SessionToken)) == 1)
}

var timeSentinel = time.Unix(0, 0).UTC()

var ErrInvalidDuration = errors.New("invalid token expiry")

func ExpToInt64(expI interface{}) (expAt int64, err error) {
	switch exp := expI.(type) {
	case string:
		expAt, err = strconv.ParseInt(exp, 10, 64)
	case float64:
		expAt, err = int64(exp), nil
	case int64:
		expAt, err = exp, nil
	case int:
		expAt, err = int64(exp), nil
	case uint64:
		expAt, err = int64(exp), nil
	case uint:
		expAt, err = int64(exp), nil
	case json.Number:
		expAt, err = exp.Int64()
	case time.Duration:
		expAt, err = time.Now().UTC().Add(exp).Unix(), nil
	case nil:
		expAt, err = 0, nil
	default:
		expAt, err = 0, ErrInvalidDuration
	}
	if expAt < 0 {
		return 0, ErrInvalidDuration
	}
	return expAt, err
}

func GetNewCredentialsWithMetadata(m map[string]interface{}, tokenSecret string) (cred Credentials, err error) {
	readBytes := func(size int) (data []byte, err error) {
		data = make([]byte, size)
		var n int
		if n, err = rand.Read(data); err != nil {
			return nil, err
		} else if n != size {
			return nil, fmt.Errorf("Not enough data. Expected to read: %v bytes, got: %v bytes", size, n)
		}
		return data, nil
	}

	keyBytes, err := readBytes(accessKeyMaxLen)
	if err != nil {
		return cred, err
	}
	for i := 0; i < accessKeyMaxLen; i++ {
		keyBytes[i] = alphaNumericTable[keyBytes[i]%alphaNumericTableLen]
	}
	cred.AccessKey = string(keyBytes)

	keyBytes, err = readBytes(secretKeyMaxLen)
	if err != nil {
		return cred, err
	}
	cred.SecretKey = strings.Replace(string([]byte(base64.StdEncoding.EncodeToString(keyBytes))[:secretKeyMaxLen]),
		"/", "+", -1)
	cred.Status = "on"

	if tokenSecret == "" {
		cred.Expiration = timeSentinel
		return cred, nil
	}

	expiry, err := ExpToInt64(m["exp"])
	if err != nil {
		return cred, err
	}

	m["accessKey"] = cred.AccessKey
	jwt := jwtgo.NewWithClaims(jwtgo.SigningMethodHS512, jwtgo.MapClaims(m))

	cred.Expiration = time.Unix(expiry, 0).UTC()
	cred.SessionToken, err = jwt.SignedString([]byte(tokenSecret))
	if err != nil {
		return cred, err
	}

	return cred, nil
}

func GetNewCredentials() (cred Credentials, err error) {
	return GetNewCredentialsWithMetadata(map[string]interface{}{}, "")
}

func CreateCredentials(accessKey, secretKey string) (cred Credentials, err error) {
	if !IsAccessKeyValid(accessKey) {
		return cred, ErrInvalidAccessKeyLength
	}
	if !IsSecretKeyValid(secretKey) {
		return cred, ErrInvalidSecretKeyLength
	}
	cred.AccessKey = accessKey
	cred.SecretKey = secretKey
	cred.Expiration = timeSentinel
	cred.Status = "on"
	return cred, nil
}
