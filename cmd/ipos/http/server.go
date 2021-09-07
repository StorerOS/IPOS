package http

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	humanize "github.com/dustin/go-humanize"

	"github.com/storeros/ipos/pkg/certs"
	"github.com/storeros/ipos/pkg/set"
)

const (
	serverShutdownPoll = 500 * time.Millisecond

	DefaultShutdownTimeout = 5 * time.Second

	DefaultTCPKeepAliveTimeout = 30 * time.Second

	DefaultMaxHeaderBytes = 1 * humanize.MiByte
)

type Server struct {
	http.Server
	Addrs               []string
	ShutdownTimeout     time.Duration
	TCPKeepAliveTimeout time.Duration
	listenerMutex       sync.Mutex
	listener            *httpListener
	inShutdown          uint32
	requestCount        int32
}

func (srv *Server) GetRequestCount() int32 {
	return atomic.LoadInt32(&srv.requestCount)
}

func (srv *Server) Start() (err error) {
	var tlsConfig *tls.Config
	if srv.TLSConfig != nil {
		tlsConfig = srv.TLSConfig.Clone()
	}
	handler := srv.Handler

	addrs := set.CreateStringSet(srv.Addrs...).ToSlice()
	tcpKeepAliveTimeout := srv.TCPKeepAliveTimeout

	var listener *httpListener
	listener, err = newHTTPListener(
		addrs,
		tcpKeepAliveTimeout,
	)
	if err != nil {
		return err
	}

	wrappedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadUint32(&srv.inShutdown) != 0 {
			w.Header().Set("Connection", "close")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(http.ErrServerClosed.Error()))
			w.(http.Flusher).Flush()
			return
		}

		atomic.AddInt32(&srv.requestCount, 1)
		defer atomic.AddInt32(&srv.requestCount, -1)

		handler.ServeHTTP(w, r)
	})

	srv.listenerMutex.Lock()
	srv.Handler = wrappedHandler
	srv.listener = listener
	srv.listenerMutex.Unlock()

	if tlsConfig != nil {
		return srv.Server.Serve(tls.NewListener(listener, tlsConfig))
	}
	return srv.Server.Serve(listener)
}

func (srv *Server) Shutdown() error {
	srv.listenerMutex.Lock()
	if srv.listener == nil {
		srv.listenerMutex.Unlock()
		return http.ErrServerClosed
	}
	srv.listenerMutex.Unlock()

	if atomic.AddUint32(&srv.inShutdown, 1) > 1 {
		return http.ErrServerClosed
	}

	srv.listenerMutex.Lock()
	err := srv.listener.Close()
	srv.listenerMutex.Unlock()

	shutdownTimeout := srv.ShutdownTimeout
	shutdownTimer := time.NewTimer(shutdownTimeout)
	ticker := time.NewTicker(serverShutdownPoll)
	defer ticker.Stop()
	for {
		select {
		case <-shutdownTimer.C:
			tmp, err := ioutil.TempFile("", "ipos-goroutines-*.txt")
			if err == nil {
				_ = pprof.Lookup("goroutine").WriteTo(tmp, 1)
				tmp.Close()
				return errors.New("timed out. some connections are still active. doing abnormal shutdown. goroutines written to " + tmp.Name())
			}
			return errors.New("timed out. some connections are still active. doing abnormal shutdown")
		case <-ticker.C:
			if atomic.LoadInt32(&srv.requestCount) <= 0 {
				return err
			}
		}
	}
}

var defaultCipherSuites = []uint16{
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
}

var secureCurves = []tls.CurveID{tls.X25519, tls.CurveP256}

func NewServer(addrs []string, handler http.Handler, getCert certs.GetCertificateFunc) *Server {
	var tlsConfig *tls.Config
	if getCert != nil {
		tlsConfig = &tls.Config{
			PreferServerCipherSuites: true,
			CipherSuites:             defaultCipherSuites,
			CurvePreferences:         secureCurves,
			MinVersion:               tls.VersionTLS12,
			NextProtos:               []string{"http/1.1", "h2"},
		}
		tlsConfig.GetCertificate = getCert
	}

	httpServer := &Server{
		Addrs:               addrs,
		ShutdownTimeout:     DefaultShutdownTimeout,
		TCPKeepAliveTimeout: DefaultTCPKeepAliveTimeout,
	}
	httpServer.Handler = handler
	httpServer.TLSConfig = tlsConfig
	httpServer.MaxHeaderBytes = DefaultMaxHeaderBytes

	return httpServer
}
