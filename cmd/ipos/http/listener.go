package http

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

type acceptResult struct {
	conn net.Conn
	err  error
}

type httpListener struct {
	mutex               sync.Mutex
	tcpListeners        []*net.TCPListener
	acceptCh            chan acceptResult
	doneCh              chan struct{}
	tcpKeepAliveTimeout time.Duration
}

func isRoutineNetErr(err error) bool {
	if err == nil {
		return false
	}
	if nErr, ok := err.(*net.OpError); ok {
		if syscallErr, ok := nErr.Err.(*os.SyscallError); ok {
			if errno, ok := syscallErr.Err.(syscall.Errno); ok {
				return errno == syscall.ECONNRESET
			}
		}
		return nErr.Timeout()
	}
	return err == io.EOF || err.Error() == "EOF"
}

func (listener *httpListener) start() {
	listener.acceptCh = make(chan acceptResult)
	listener.doneCh = make(chan struct{})

	send := func(result acceptResult, doneCh <-chan struct{}) bool {
		select {
		case listener.acceptCh <- result:
			return true
		case <-doneCh:
			if result.conn != nil {
				result.conn.Close()
			}
			return false
		}
	}

	handleConn := func(tcpConn *net.TCPConn, doneCh <-chan struct{}) {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(listener.tcpKeepAliveTimeout)

		send(acceptResult{tcpConn, nil}, doneCh)
	}

	handleListener := func(tcpListener *net.TCPListener, doneCh <-chan struct{}) {
		for {
			tcpConn, err := tcpListener.AcceptTCP()
			if err != nil {
				if !send(acceptResult{nil, err}, doneCh) {
					return
				}
			} else {
				go handleConn(tcpConn, doneCh)
			}
		}
	}

	for _, tcpListener := range listener.tcpListeners {
		go handleListener(tcpListener, listener.doneCh)
	}
}

func (listener *httpListener) Accept() (conn net.Conn, err error) {
	result, ok := <-listener.acceptCh
	if ok {
		return result.conn, result.err
	}

	return nil, syscall.EINVAL
}

func (listener *httpListener) Close() (err error) {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()
	if listener.doneCh == nil {
		return syscall.EINVAL
	}

	for i := range listener.tcpListeners {
		listener.tcpListeners[i].Close()
	}
	close(listener.doneCh)

	listener.doneCh = nil
	return nil
}

func (listener *httpListener) Addr() (addr net.Addr) {
	addr = listener.tcpListeners[0].Addr()
	if len(listener.tcpListeners) == 1 {
		return addr
	}

	tcpAddr := addr.(*net.TCPAddr)
	if ip := net.ParseIP("0.0.0.0"); ip != nil {
		tcpAddr.IP = ip
	}

	addr = tcpAddr
	return addr
}

func (listener *httpListener) Addrs() (addrs []net.Addr) {
	for i := range listener.tcpListeners {
		addrs = append(addrs, listener.tcpListeners[i].Addr())
	}

	return addrs
}

func newHTTPListener(serverAddrs []string,
	tcpKeepAliveTimeout time.Duration) (listener *httpListener, err error) {

	var tcpListeners []*net.TCPListener

	defer func() {
		if err == nil {
			return
		}

		for _, tcpListener := range tcpListeners {
			tcpListener.Close()
		}
	}()

	for _, serverAddr := range serverAddrs {
		var l net.Listener
		if l, err = listen("tcp", serverAddr); err != nil {
			if l, err = fallbackListen("tcp", serverAddr); err != nil {
				return nil, err
			}
		}

		tcpListener, ok := l.(*net.TCPListener)
		if !ok {
			return nil, fmt.Errorf("unexpected listener type found %v, expected net.TCPListener", l)
		}

		tcpListeners = append(tcpListeners, tcpListener)
	}

	listener = &httpListener{
		tcpListeners:        tcpListeners,
		tcpKeepAliveTimeout: tcpKeepAliveTimeout,
	}
	listener.start()

	return listener, nil
}
