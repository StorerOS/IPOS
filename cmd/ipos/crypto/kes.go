package crypto

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type KesConfig struct {
	Enabled bool

	Endpoint string

	KeyFile string

	CertFile string

	CAPath string

	DefaultKeyID string

	Transport *http.Transport
}

func (k KesConfig) Verify() (err error) {
	switch {
	case k.Endpoint == "":
		err = Errorf("crypto: missing kes endpoint")
	case k.CertFile == "":
		err = Errorf("crypto: missing cert file")
	case k.KeyFile == "":
		err = Errorf("crypto: missing key file")
	case k.DefaultKeyID == "":
		err = Errorf("crypto: missing default key id")
	}
	return err
}

type kesService struct {
	client *kesClient

	endpoint     string
	defaultKeyID string
}

func NewKes(cfg KesConfig) (KMS, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	certPool, err := loadCACertificates(cfg.CAPath)
	if err != nil {
		return nil, err
	}
	cfg.Transport.TLSClientConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
	}
	return &kesService{
		client: &kesClient{
			addr: cfg.Endpoint,
			httpClient: http.Client{
				Transport: cfg.Transport,
			},
		},
		endpoint:     cfg.Endpoint,
		defaultKeyID: cfg.DefaultKeyID,
	}, nil
}

func (kes *kesService) KeyID() string {
	return kes.defaultKeyID
}

func (kes *kesService) Info() KMSInfo {
	return KMSInfo{
		Endpoint: kes.endpoint,
		Name:     kes.KeyID(),
		AuthType: "TLS",
	}
}

func (kes *kesService) GenerateKey(keyID string, ctx Context) (key [32]byte, sealedKey []byte, err error) {
	var context bytes.Buffer
	ctx.WriteTo(&context)

	var plainKey []byte
	plainKey, sealedKey, err = kes.client.GenerateDataKey(keyID, context.Bytes())
	if err != nil {
		return key, nil, err
	}
	if len(plainKey) != len(key) {
		return key, nil, Errorf("crypto: received invalid plaintext key size from KMS")
	}
	copy(key[:], plainKey)
	return key, sealedKey, nil
}

func (kes *kesService) UnsealKey(keyID string, sealedKey []byte, ctx Context) (key [32]byte, err error) {
	var context bytes.Buffer
	ctx.WriteTo(&context)

	var plainKey []byte
	plainKey, err = kes.client.DecryptDataKey(keyID, sealedKey, context.Bytes())
	if err != nil {
		return key, err
	}
	if len(plainKey) != len(key) {
		return key, Errorf("crypto: received invalid plaintext key size from KMS")
	}
	copy(key[:], plainKey)
	return key, nil
}

func (kes *kesService) UpdateKey(keyID string, sealedKey []byte, ctx Context) ([]byte, error) {
	_, err := kes.UnsealKey(keyID, sealedKey, ctx)
	if err != nil {
		return nil, err
	}

	return sealedKey, nil
}

type kesClient struct {
	addr       string
	httpClient http.Client
}

func (c *kesClient) GenerateDataKey(name string, context []byte) ([]byte, []byte, error) {
	type Request struct {
		Context []byte `json:"context"`
	}
	body, err := json.Marshal(Request{
		Context: context,
	})
	if err != nil {
		return nil, nil, err
	}

	url := fmt.Sprintf("%s/v1/key/generate/%s", c.addr, url.PathEscape(name))
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, c.parseErrorResponse(resp)
	}
	defer resp.Body.Close()

	type Response struct {
		Plaintext  []byte `json:"plaintext"`
		Ciphertext []byte `json:"ciphertext"`
	}
	const limit = 1 << 20
	var response Response
	if err = json.NewDecoder(io.LimitReader(resp.Body, limit)).Decode(&response); err != nil {
		return nil, nil, err
	}
	return response.Plaintext, response.Ciphertext, nil
}

func (c *kesClient) DecryptDataKey(name string, ciphertext, context []byte) ([]byte, error) {
	type Request struct {
		Ciphertext []byte `json:"ciphertext"`
		Context    []byte `json:"context"`
	}
	body, err := json.Marshal(Request{
		Ciphertext: ciphertext,
		Context:    context,
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/key/decrypt/%s", c.addr, url.PathEscape(name))
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}
	defer resp.Body.Close()

	type Response struct {
		Plaintext []byte `json:"plaintext"`
	}
	const limit = 32 * 1024
	var response Response
	if err = json.NewDecoder(io.LimitReader(resp.Body, limit)).Decode(&response); err != nil {
		return nil, err
	}
	return response.Plaintext, nil
}

func (c *kesClient) parseErrorResponse(resp *http.Response) error {
	if resp.Body == nil {
		return nil
	}
	defer resp.Body.Close()

	const limit = 32 * 1024
	var errMsg strings.Builder
	if _, err := io.Copy(&errMsg, io.LimitReader(resp.Body, limit)); err != nil {
		return err
	}
	return Errorf("%s: %s", http.StatusText(resp.StatusCode), errMsg.String())
}

func loadCACertificates(path string) (*x509.CertPool, error) {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if path == "" {
		return rootCAs, nil
	}

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return rootCAs, nil
		}
		return nil, Errorf("crypto: cannot open '%s': %v", path, err)
	}

	if !stat.IsDir() {
		cert, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if !rootCAs.AppendCertsFromPEM(cert) {
			return nil, Errorf("crypto: '%s' is not a valid PEM-encoded certificate", path)
		}
		return rootCAs, nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		cert, err := ioutil.ReadFile(filepath.Join(path, file.Name()))
		if err != nil {
			continue
		}
		rootCAs.AppendCertsFromPEM(cert)
	}
	return rootCAs, nil

}
