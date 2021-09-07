package credentials

type Chain struct {
	Providers []Provider
	curr      Provider
}

func NewChainCredentials(providers []Provider) *Credentials {
	return New(&Chain{
		Providers: append([]Provider{}, providers...),
	})
}

func (c *Chain) Retrieve() (Value, error) {
	for _, p := range c.Providers {
		creds, _ := p.Retrieve()
		if creds.AccessKeyID == "" && creds.SecretAccessKey == "" {
			continue
		}
		c.curr = p
		return creds, nil
	}

	return Value{
		SignerType: SignatureAnonymous,
	}, nil
}

func (c *Chain) IsExpired() bool {
	if c.curr != nil {
		return c.curr.IsExpired()
	}

	return true
}
