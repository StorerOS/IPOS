package credentials

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type AssumeRoleWithWebIdentityResponse struct {
	XMLName          xml.Name          `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleWithWebIdentityResponse" json:"-"`
	Result           WebIdentityResult `xml:"AssumeRoleWithWebIdentityResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

type WebIdentityResult struct {
	AssumedRoleUser AssumedRoleUser `xml:",omitempty"`
	Audience        string          `xml:",omitempty"`
	Credentials     struct {
		AccessKey    string    `xml:"AccessKeyId" json:"accessKey,omitempty"`
		SecretKey    string    `xml:"SecretAccessKey" json:"secretKey,omitempty"`
		Expiration   time.Time `xml:"Expiration" json:"expiration,omitempty"`
		SessionToken string    `xml:"SessionToken" json:"sessionToken,omitempty"`
	} `xml:",omitempty"`
	PackedPolicySize            int    `xml:",omitempty"`
	Provider                    string `xml:",omitempty"`
	SubjectFromWebIdentityToken string `xml:",omitempty"`
}

type WebIdentityToken struct {
	Token  string
	Expiry int
}

type STSWebIdentity struct {
	Expiry

	Client *http.Client

	stsEndpoint string

	getWebIDTokenExpiry func() (*WebIdentityToken, error)

	roleARN string

	roleSessionName string
}

func NewSTSWebIdentity(stsEndpoint string, getWebIDTokenExpiry func() (*WebIdentityToken, error)) (*Credentials, error) {
	if stsEndpoint == "" {
		return nil, errors.New("STS endpoint cannot be empty")
	}
	if getWebIDTokenExpiry == nil {
		return nil, errors.New("Web ID token and expiry retrieval function should be defined")
	}
	return New(&STSWebIdentity{
		Client: &http.Client{
			Transport: http.DefaultTransport,
		},
		stsEndpoint:         stsEndpoint,
		getWebIDTokenExpiry: getWebIDTokenExpiry,
	}), nil
}

func getWebIdentityCredentials(clnt *http.Client, endpoint, roleARN, roleSessionName string,
	getWebIDTokenExpiry func() (*WebIdentityToken, error)) (AssumeRoleWithWebIdentityResponse, error) {
	idToken, err := getWebIDTokenExpiry()
	if err != nil {
		return AssumeRoleWithWebIdentityResponse{}, err
	}

	v := url.Values{}
	v.Set("Action", "AssumeRoleWithWebIdentity")
	if len(roleARN) > 0 {
		v.Set("RoleArn", roleARN)

		if len(roleSessionName) == 0 {
			roleSessionName = strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		v.Set("RoleSessionName", roleSessionName)
	}
	v.Set("WebIdentityToken", idToken.Token)
	if idToken.Expiry > 0 {
		v.Set("DurationSeconds", fmt.Sprintf("%d", idToken.Expiry))
	}
	v.Set("Version", "2011-06-15")

	u, err := url.Parse(endpoint)
	if err != nil {
		return AssumeRoleWithWebIdentityResponse{}, err
	}

	u.RawQuery = v.Encode()

	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return AssumeRoleWithWebIdentityResponse{}, err
	}

	resp, err := clnt.Do(req)
	if err != nil {
		return AssumeRoleWithWebIdentityResponse{}, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return AssumeRoleWithWebIdentityResponse{}, errors.New(resp.Status)
	}

	a := AssumeRoleWithWebIdentityResponse{}
	if err = xml.NewDecoder(resp.Body).Decode(&a); err != nil {
		return AssumeRoleWithWebIdentityResponse{}, err
	}

	return a, nil
}

func (m *STSWebIdentity) Retrieve() (Value, error) {
	a, err := getWebIdentityCredentials(m.Client, m.stsEndpoint, m.roleARN, m.roleSessionName, m.getWebIDTokenExpiry)
	if err != nil {
		return Value{}, err
	}

	m.SetExpiration(a.Result.Credentials.Expiration, DefaultExpiryWindow)

	return Value{
		AccessKeyID:     a.Result.Credentials.AccessKey,
		SecretAccessKey: a.Result.Credentials.SecretKey,
		SessionToken:    a.Result.Credentials.SessionToken,
		SignerType:      SignatureV4,
	}, nil
}
