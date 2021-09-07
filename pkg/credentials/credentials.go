package credentials

import (
	"sync"
	"time"
)

type Value struct {
	AccessKeyID string

	SecretAccessKey string

	SessionToken string

	SignerType SignatureType
}

type Provider interface {
	Retrieve() (Value, error)

	IsExpired() bool
}

type Expiry struct {
	expiration time.Time

	CurrentTime func() time.Time
}

func (e *Expiry) SetExpiration(expiration time.Time, window time.Duration) {
	e.expiration = expiration
	if window > 0 {
		e.expiration = e.expiration.Add(-window)
	}
}

func (e *Expiry) IsExpired() bool {
	if e.CurrentTime == nil {
		e.CurrentTime = time.Now
	}
	return e.expiration.Before(e.CurrentTime())
}

type Credentials struct {
	sync.Mutex

	creds        Value
	forceRefresh bool
	provider     Provider
}

func New(provider Provider) *Credentials {
	return &Credentials{
		provider:     provider,
		forceRefresh: true,
	}
}

func (c *Credentials) Get() (Value, error) {
	c.Lock()
	defer c.Unlock()

	if c.isExpired() {
		creds, err := c.provider.Retrieve()
		if err != nil {
			return Value{}, err
		}
		c.creds = creds
		c.forceRefresh = false
	}

	return c.creds, nil
}

func (c *Credentials) Expire() {
	c.Lock()
	defer c.Unlock()

	c.forceRefresh = true
}

func (c *Credentials) IsExpired() bool {
	c.Lock()
	defer c.Unlock()

	return c.isExpired()
}

func (c *Credentials) isExpired() bool {
	return c.forceRefresh || c.provider.IsExpired()
}
