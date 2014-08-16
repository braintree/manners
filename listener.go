package manners

import (
	"net"
	"net/http"
	"sync/atomic"
)

func NewListener(l net.Listener, s *GracefulServer) *GracefulListener {
	return &GracefulListener{l, 1}
}

type GracefulConn struct {
	net.Conn
	lastHttpState http.ConnState
}

// A GracefulListener differs from a standard net.Listener in one way: if
// Accept() is called after it is gracefully closed, it returns a
// listenerAlreadyClosed error. The GracefulServer will ignore this
// error.
type GracefulListener struct {
	net.Listener
	open int32
}

func (l *GracefulListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		if atomic.LoadInt32(&l.open) == 0 {
			err = listenerAlreadyClosed{err}
		}
		return nil, err
	}
	gconn := &GracefulConn{conn, 0}
	return gconn, nil
}

func (l *GracefulListener) Close() error {
	if atomic.CompareAndSwapInt32(&l.open, 1, 0) {
		err := l.Listener.Close()
		return err
	}
	return nil
}

type listenerAlreadyClosed struct {
	error
}
