package credentials

type Static struct {
	Value
}

func NewStaticV2(id, secret, token string) *Credentials {
	return NewStatic(id, secret, token, SignatureV2)
}

func NewStaticV4(id, secret, token string) *Credentials {
	return NewStatic(id, secret, token, SignatureV4)
}

func NewStatic(id, secret, token string, signerType SignatureType) *Credentials {
	return New(&Static{
		Value: Value{
			AccessKeyID:     id,
			SecretAccessKey: secret,
			SessionToken:    token,
			SignerType:      signerType,
		},
	})
}

func (s *Static) Retrieve() (Value, error) {
	if s.AccessKeyID == "" || s.SecretAccessKey == "" {
		return Value{SignerType: SignatureAnonymous}, nil
	}
	return s.Value, nil
}

func (s *Static) IsExpired() bool {
	return false
}
