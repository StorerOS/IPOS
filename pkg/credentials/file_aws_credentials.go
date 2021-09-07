package credentials

import (
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	ini "gopkg.in/ini.v1"
)

type FileAWSCredentials struct {
	Filename string

	Profile string

	retrieved bool
}

func NewFileAWSCredentials(filename string, profile string) *Credentials {
	return New(&FileAWSCredentials{
		Filename: filename,
		Profile:  profile,
	})
}

func (p *FileAWSCredentials) Retrieve() (Value, error) {
	if p.Filename == "" {
		p.Filename = os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
		if p.Filename == "" {
			homeDir, err := homedir.Dir()
			if err != nil {
				return Value{}, err
			}
			p.Filename = filepath.Join(homeDir, ".aws", "credentials")
		}
	}
	if p.Profile == "" {
		p.Profile = os.Getenv("AWS_PROFILE")
		if p.Profile == "" {
			p.Profile = "default"
		}
	}

	p.retrieved = false

	iniProfile, err := loadProfile(p.Filename, p.Profile)
	if err != nil {
		return Value{}, err
	}

	id := iniProfile.Key("aws_access_key_id")

	secret := iniProfile.Key("aws_secret_access_key")

	token := iniProfile.Key("aws_session_token")

	p.retrieved = true
	return Value{
		AccessKeyID:     id.String(),
		SecretAccessKey: secret.String(),
		SessionToken:    token.String(),
		SignerType:      SignatureV4,
	}, nil
}

func (p *FileAWSCredentials) IsExpired() bool {
	return !p.retrieved
}

func loadProfile(filename, profile string) (*ini.Section, error) {
	config, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}
	iniProfile, err := config.GetSection(profile)
	if err != nil {
		return nil, err
	}
	return iniProfile, nil
}
