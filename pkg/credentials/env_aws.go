package credentials

import "os"

type EnvAWS struct {
	retrieved bool
}

func NewEnvAWS() *Credentials {
	return New(&EnvAWS{})
}

func (e *EnvAWS) Retrieve() (Value, error) {
	e.retrieved = false

	id := os.Getenv("AWS_ACCESS_KEY_ID")
	if id == "" {
		id = os.Getenv("AWS_ACCESS_KEY")
	}

	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secret == "" {
		secret = os.Getenv("AWS_SECRET_KEY")
	}

	signerType := SignatureV4
	if id == "" || secret == "" {
		signerType = SignatureAnonymous
	}

	e.retrieved = true
	return Value{
		AccessKeyID:     id,
		SecretAccessKey: secret,
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
		SignerType:      signerType,
	}, nil
}

func (e *EnvAWS) IsExpired() bool {
	return !e.retrieved
}
