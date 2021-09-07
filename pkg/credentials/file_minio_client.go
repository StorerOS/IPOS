package credentials

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	jsoniter "github.com/json-iterator/go"
	homedir "github.com/mitchellh/go-homedir"
)

type FileIPOSClient struct {
	Filename string

	Alias string

	retrieved bool
}

func NewFileIPOSClient(filename string, alias string) *Credentials {
	return New(&FileIPOSClient{
		Filename: filename,
		Alias:    alias,
	})
}

func (p *FileIPOSClient) Retrieve() (Value, error) {
	if p.Filename == "" {
		if value, ok := os.LookupEnv("IPOS_SHARED_CREDENTIALS_FILE"); ok {
			p.Filename = value
		} else {
			homeDir, err := homedir.Dir()
			if err != nil {
				return Value{}, err
			}
			p.Filename = filepath.Join(homeDir, ".mc", "config.json")
			if runtime.GOOS == "windows" {
				p.Filename = filepath.Join(homeDir, "mc", "config.json")
			}
		}
	}

	if p.Alias == "" {
		p.Alias = os.Getenv("IPOS_ALIAS")
		if p.Alias == "" {
			p.Alias = "s3"
		}
	}

	p.retrieved = false

	hostCfg, err := loadAlias(p.Filename, p.Alias)
	if err != nil {
		return Value{}, err
	}

	p.retrieved = true
	return Value{
		AccessKeyID:     hostCfg.AccessKey,
		SecretAccessKey: hostCfg.SecretKey,
		SignerType:      parseSignatureType(hostCfg.API),
	}, nil
}

func (p *FileIPOSClient) IsExpired() bool {
	return !p.retrieved
}

type hostConfig struct {
	URL       string `json:"url"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	API       string `json:"api"`
}

type config struct {
	Version string                `json:"version"`
	Hosts   map[string]hostConfig `json:"hosts"`
}

func loadAlias(filename, alias string) (hostConfig, error) {
	cfg := &config{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return hostConfig{}, err
	}
	if err = json.Unmarshal(configBytes, cfg); err != nil {
		return hostConfig{}, err
	}
	return cfg.Hosts[alias], nil
}
