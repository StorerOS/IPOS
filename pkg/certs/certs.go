package certs

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rjeczalik/notify"
)

type Certs struct {
	sync.RWMutex
	certFile string
	keyFile  string
	loadCert LoadX509KeyPairFunc

	cert tls.Certificate

	e chan notify.EventInfo
}

type LoadX509KeyPairFunc func(certFile, keyFile string) (tls.Certificate, error)

func New(certFile, keyFile string, loadCert LoadX509KeyPairFunc) (*Certs, error) {
	certFileIsLink, err := checkSymlink(certFile)
	if err != nil {
		return nil, err
	}
	keyFileIsLink, err := checkSymlink(keyFile)
	if err != nil {
		return nil, err
	}
	c := &Certs{
		certFile: certFile,
		keyFile:  keyFile,
		loadCert: loadCert,
		e:        make(chan notify.EventInfo, 1),
	}

	if certFileIsLink && keyFileIsLink {
		if err := c.watchSymlinks(); err != nil {
			return nil, err
		}
	} else {
		if err := c.watch(); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func checkSymlink(file string) (bool, error) {
	st, err := os.Lstat(file)
	if err != nil {
		return false, err
	}
	return st.Mode()&os.ModeSymlink == os.ModeSymlink, nil
}

func (c *Certs) watchSymlinks() (err error) {
	c.Lock()
	c.cert, err = c.loadCert(c.certFile, c.keyFile)
	c.Unlock()
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-c.e:
				return
			case <-time.After(24 * time.Hour):
				cert, cerr := c.loadCert(c.certFile, c.keyFile)
				if cerr != nil {
					continue
				}
				c.Lock()
				c.cert = cert
				c.Unlock()
			}
		}
	}()
	return nil
}

func (c *Certs) watch() (err error) {
	defer func() {
		if err != nil {
			notify.Stop(c.e)
		}
	}()

	if err = notify.Watch(filepath.Dir(c.certFile), c.e, eventWrite...); err != nil {
		return err
	}
	if err = notify.Watch(filepath.Dir(c.keyFile), c.e, eventWrite...); err != nil {
		return err
	}
	c.Lock()
	c.cert, err = c.loadCert(c.certFile, c.keyFile)
	c.Unlock()
	if err != nil {
		return err
	}
	go c.run()
	return nil
}

func (c *Certs) run() {
	for event := range c.e {
		base := filepath.Base(event.Path())
		if isWriteEvent(event.Event()) {
			certChanged := base == filepath.Base(c.certFile)
			keyChanged := base == filepath.Base(c.keyFile)
			if certChanged || keyChanged {
				cert, err := c.loadCert(c.certFile, c.keyFile)
				if err != nil {
					continue
				}
				c.Lock()
				c.cert = cert
				c.Unlock()
			}
		}
	}
}

type GetCertificateFunc func(hello *tls.ClientHelloInfo) (*tls.Certificate, error)

func (c *Certs) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.RLock()
	defer c.RUnlock()
	return &c.cert, nil
}

func (c *Certs) Stop() {
	if c != nil {
		notify.Stop(c.e)
	}
}
