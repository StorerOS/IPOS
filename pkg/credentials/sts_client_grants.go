package credentials

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type AssumedRoleUser struct {
	Arn           string
	AssumedRoleID string `xml:"AssumeRoleId"`
}

type AssumeRoleWithClientGrantsResponse struct {
	XMLName          xml.Name           `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleWithClientGrantsResponse" json:"-"`
	Result           ClientGrantsResult `xml:"AssumeRoleWithClientGrantsResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

type ClientGrantsResult struct {
	AssumedRoleUser AssumedRoleUser `xml:",omitempty"`
	Audience        string          `xml:",omitempty"`
	Credentials     struct {
		AccessKey    string    `xml:"AccessKeyId" json:"accessKey,omitempty"`
		SecretKey    string    `xml:"SecretAccessKey" json:"secretKey,omitempty"`
		Expiration   time.Time `xml:"Expiration" json:"expiration,omitempty"`
		SessionToken string    `xml:"SessionToken" json:"sessionToken,omitempty"`
	} `xml:",omitempty"`
	PackedPolicySize             int    `xml:",omitempty"`
	Provider                     string `xml:",omitempty"`
	SubjectFromClientGrantsToken string `xml:",omitempty"`
}

type ClientGrantsToken struct {
	Token  string
	Expiry int
}

type STSClientGrants struct {
	Expiry

	Client *http.Client

	stsEndpoint string

	getClientGrantsTokenExpiry func() (*ClientGrantsToken, error)
}

func NewSTSClientGrants(stsEndpoint string, getClientGrantsTokenExpiry func() (*ClientGrantsToken, error)) (*Credentials, error) {
	if stsEndpoint == "" {
		return nil, errors.New("STS endpoint cannot be empty")
	}
	if getClientGrantsTokenExpiry == nil {
		return nil, errors.New("Client grants access token and expiry retrieval function should be defined")
	}
	return New(&STSClientGrants{
		Client: &http.Client{
			Transport: http.DefaultTransport,
		},
		stsEndpoint:                stsEndpoint,
		getClientGrantsTokenExpiry: getClientGrantsTokenExpiry,
	}), nil
}

func getClientGrantsCredentials(clnt *http.Client, endpoint string,
	getClientGrantsTokenExpiry func() (*ClientGrantsToken, error)) (AssumeRoleWithClientGrantsResponse, error) {
	accessToken, err := getClientGrantsTokenExpiry()
	if err != nil {
		return AssumeRoleWithClientGrantsResponse{}, err
	}

	v := url.Values{}
	v.Set("Action", "AssumeRoleWithClientGrants")
	v.Set("Token", accessToken.Token)
	v.Set("DurationSeconds", fmt.Sprintf("%d", accessToken.Expiry))
	v.Set("Version", "2011-06-15")

	u, err := url.Parse(endpoint)
	if err != nil {
		return AssumeRoleWithClientGrantsResponse{}, err
	}
	u.RawQuery = v.Encode()

	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return AssumeRoleWithClientGrantsResponse{}, err
	}
	resp, err := clnt.Do(req)
	if err != nil {
		return AssumeRoleWithClientGrantsResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return AssumeRoleWithClientGrantsResponse{}, errors.New(resp.Status)
	}

	a := AssumeRoleWithClientGrantsResponse{}
	if err = xml.NewDecoder(resp.Body).Decode(&a); err != nil {
		return AssumeRoleWithClientGrantsResponse{}, err
	}
	return a, nil
}

func (m *STSClientGrants) Retrieve() (Value, error) {
	a, err := getClientGrantsCredentials(m.Client, m.stsEndpoint, m.getClientGrantsTokenExpiry)
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
