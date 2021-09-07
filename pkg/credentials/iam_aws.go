package credentials

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const DefaultExpiryWindow = time.Second * 10 // 10 secs

type IAM struct {
	Expiry

	Client *http.Client

	endpoint string
}

const (
	defaultIAMRoleEndpoint      = "http://169.254.169.254"
	defaultECSRoleEndpoint      = "http://169.254.170.2"
	defaultSTSRoleEndpoint      = "https://sts.amazonaws.com"
	defaultIAMSecurityCredsPath = "/latest/meta-data/iam/security-credentials/"
)

func NewIAM(endpoint string) *Credentials {
	p := &IAM{
		Client: &http.Client{
			Transport: http.DefaultTransport,
		},
		endpoint: endpoint,
	}
	return New(p)
}

func (m *IAM) Retrieve() (Value, error) {
	var roleCreds ec2RoleCredRespBody
	var err error

	endpoint := m.endpoint
	switch {
	case len(os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")) > 0:
		if len(endpoint) == 0 {
			if len(os.Getenv("AWS_REGION")) > 0 {
				endpoint = "https://sts." + os.Getenv("AWS_REGION") + ".amazonaws.com"
			} else {
				endpoint = defaultSTSRoleEndpoint
			}
		}

		creds := &STSWebIdentity{
			Client:          m.Client,
			stsEndpoint:     endpoint,
			roleARN:         os.Getenv("AWS_ROLE_ARN"),
			roleSessionName: os.Getenv("AWS_ROLE_SESSION_NAME"),
			getWebIDTokenExpiry: func() (*WebIdentityToken, error) {
				token, err := ioutil.ReadFile(os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE"))
				if err != nil {
					return nil, err
				}

				return &WebIdentityToken{Token: string(token)}, nil
			},
		}

		return creds.Retrieve()

	case len(os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")) > 0:
		if len(endpoint) == 0 {
			endpoint = fmt.Sprintf("%s%s", defaultECSRoleEndpoint,
				os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"))
		}

		roleCreds, err = getEcsTaskCredentials(m.Client, endpoint)

	case len(os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI")) > 0:
		if len(endpoint) == 0 {
			endpoint = os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI")

			var ok bool
			if ok, err = isLoopback(endpoint); !ok {
				if err == nil {
					err = fmt.Errorf("uri host is not a loopback address: %s", endpoint)
				}
				break
			}
		}

		roleCreds, err = getEcsTaskCredentials(m.Client, endpoint)

	default:
		roleCreds, err = getCredentials(m.Client, endpoint)
	}

	if err != nil {
		return Value{}, err
	}

	m.SetExpiration(roleCreds.Expiration, DefaultExpiryWindow)

	return Value{
		AccessKeyID:     roleCreds.AccessKeyID,
		SecretAccessKey: roleCreds.SecretAccessKey,
		SessionToken:    roleCreds.Token,
		SignerType:      SignatureV4,
	}, nil
}

type ec2RoleCredRespBody struct {
	Expiration      time.Time
	AccessKeyID     string
	SecretAccessKey string
	Token           string

	Code    string
	Message string

	LastUpdated time.Time
	Type        string
}

func getIAMRoleURL(endpoint string) (*url.URL, error) {
	if endpoint == "" {
		endpoint = defaultIAMRoleEndpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	u.Path = defaultIAMSecurityCredsPath
	return u, nil
}

func listRoleNames(client *http.Client, u *url.URL) ([]string, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	credsList := []string{}
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		credsList = append(credsList, s.Text())
	}

	if err := s.Err(); err != nil {
		return nil, err
	}

	return credsList, nil
}

func getEcsTaskCredentials(client *http.Client, endpoint string) (ec2RoleCredRespBody, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return ec2RoleCredRespBody{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return ec2RoleCredRespBody{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ec2RoleCredRespBody{}, errors.New(resp.Status)
	}

	respCreds := ec2RoleCredRespBody{}
	if err := jsoniter.NewDecoder(resp.Body).Decode(&respCreds); err != nil {
		return ec2RoleCredRespBody{}, err
	}

	return respCreds, nil
}

func getCredentials(client *http.Client, endpoint string) (ec2RoleCredRespBody, error) {
	u, err := getIAMRoleURL(endpoint)
	if err != nil {
		return ec2RoleCredRespBody{}, err
	}

	roleNames, err := listRoleNames(client, u)
	if err != nil {
		return ec2RoleCredRespBody{}, err
	}

	if len(roleNames) == 0 {
		return ec2RoleCredRespBody{}, errors.New("No IAM roles attached to this EC2 service")
	}

	roleName := roleNames[0]

	u.Path = path.Join(u.Path, roleName)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return ec2RoleCredRespBody{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return ec2RoleCredRespBody{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ec2RoleCredRespBody{}, errors.New(resp.Status)
	}

	respCreds := ec2RoleCredRespBody{}
	if err := jsoniter.NewDecoder(resp.Body).Decode(&respCreds); err != nil {
		return ec2RoleCredRespBody{}, err
	}

	if respCreds.Code != "Success" {
		return ec2RoleCredRespBody{}, errors.New(respCreds.Message)
	}

	return respCreds, nil
}

func isLoopback(uri string) (bool, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return false, err
	}

	host := u.Hostname()
	if len(host) == 0 {
		return false, fmt.Errorf("can't parse host from uri: %s", uri)
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return false, err
	}
	for _, ip := range ips {
		if !net.ParseIP(ip).IsLoopback() {
			return false, nil
		}
	}

	return true, nil
}
