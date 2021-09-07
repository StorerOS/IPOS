package credentials

import "os"

type EnvIPOS struct {
	retrieved bool
}

func NewEnvIPOS() *Credentials {
	return New(&EnvIPOS{})
}

func (e *EnvIPOS) Retrieve() (Value, error) {
	e.retrieved = false

	id := os.Getenv("IPOS_ACCESS_KEY")
	secret := os.Getenv("IPOS_SECRET_KEY")

	signerType := SignatureV4
	if id == "" || secret == "" {
		signerType = SignatureAnonymous
	}

	e.retrieved = true
	return Value{
		AccessKeyID:     id,
		SecretAccessKey: secret,
		SignerType:      signerType,
	}, nil
}

func (e *EnvIPOS) IsExpired() bool {
	return !e.retrieved
}
