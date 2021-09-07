package credentials

import (
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	sha256 "github.com/storeros/ipos/pkg/sha256-simd"
	"github.com/storeros/ipos/pkg/signer"
)

type AssumeRoleResponse struct {
	XMLName xml.Name `xml:"https://sts.amazonaws.com/doc/2011-06-15/ AssumeRoleResponse" json:"-"`

	Result           AssumeRoleResult `xml:"AssumeRoleResult"`
	ResponseMetadata struct {
		RequestID string `xml:"RequestId,omitempty"`
	} `xml:"ResponseMetadata,omitempty"`
}

type AssumeRoleResult struct {
	AssumedRoleUser AssumedRoleUser `xml:",omitempty"`

	Credentials struct {
		AccessKey    string    `xml:"AccessKeyId" json:"accessKey,omitempty"`
		SecretKey    string    `xml:"SecretAccessKey" json:"secretKey,omitempty"`
		Expiration   time.Time `xml:"Expiration" json:"expiration,omitempty"`
		SessionToken string    `xml:"SessionToken" json:"sessionToken,omitempty"`
	} `xml:",omitempty"`

	PackedPolicySize int `xml:",omitempty"`
}

type STSAssumeRole struct {
	Expiry

	Client *http.Client

	STSEndpoint string

	Options STSAssumeRoleOptions
}

type STSAssumeRoleOptions struct {
	AccessKey string
	SecretKey string

	Location        string
	DurationSeconds int

	RoleARN         string
	RoleSessionName string
}

func NewSTSAssumeRole(stsEndpoint string, opts STSAssumeRoleOptions) (*Credentials, error) {
	if stsEndpoint == "" {
		return nil, errors.New("STS endpoint cannot be empty")
	}
	if opts.AccessKey == "" || opts.SecretKey == "" {
		return nil, errors.New("AssumeRole credentials access/secretkey is mandatory")
	}
	return New(&STSAssumeRole{
		Client: &http.Client{
			Transport: http.DefaultTransport,
		},
		STSEndpoint: stsEndpoint,
		Options:     opts,
	}), nil
}

const defaultDurationSeconds = 3600

func closeResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

func getAssumeRoleCredentials(clnt *http.Client, endpoint string, opts STSAssumeRoleOptions) (AssumeRoleResponse, error) {
	v := url.Values{}
	v.Set("Action", "AssumeRole")
	v.Set("Version", "2011-06-15")
	if opts.RoleARN != "" {
		v.Set("RoleArn", opts.RoleARN)
	}
	if opts.RoleSessionName != "" {
		v.Set("RoleSessionName", opts.RoleSessionName)
	}
	if opts.DurationSeconds > defaultDurationSeconds {
		v.Set("DurationSeconds", strconv.Itoa(opts.DurationSeconds))
	} else {
		v.Set("DurationSeconds", strconv.Itoa(defaultDurationSeconds))
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return AssumeRoleResponse{}, err
	}
	u.Path = "/"

	postBody := strings.NewReader(v.Encode())
	hash := sha256.New()
	if _, err = io.Copy(hash, postBody); err != nil {
		return AssumeRoleResponse{}, err
	}
	postBody.Seek(0, 0)

	req, err := http.NewRequest("POST", u.String(), postBody)
	if err != nil {
		return AssumeRoleResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(hash.Sum(nil)))
	req = signer.SignV4STS(*req, opts.AccessKey, opts.SecretKey, opts.Location)

	resp, err := clnt.Do(req)
	if err != nil {
		return AssumeRoleResponse{}, err
	}
	defer closeResponse(resp)
	if resp.StatusCode != http.StatusOK {
		return AssumeRoleResponse{}, errors.New(resp.Status)
	}

	a := AssumeRoleResponse{}
	if err = xml.NewDecoder(resp.Body).Decode(&a); err != nil {
		return AssumeRoleResponse{}, err
	}
	return a, nil
}

func (m *STSAssumeRole) Retrieve() (Value, error) {
	a, err := getAssumeRoleCredentials(m.Client, m.STSEndpoint, m.Options)
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
