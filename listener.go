package manners

import (
	"net"
	"net/http"
	"sync/atomic"
)

// NewListener wraps an existing listener for use with
// GracefulServer.
//
// Note that you generally don't need to use this directly as
// GracefulServer will automatically wrap any non-graceful listeners
// supplied to it.
func NewListener(l net.Listener) *GracefulListener {
	return &GracefulListener{l, 1}
}

// A GracefulListener differs from a standard net.Listener in one way: if
// Accept() is called after it is gracefully closed, it returns a
// listenerAlreadyClosed error. The GracefulServer will ignore this
// error.
type GracefulListener struct {
	net.Listener
	open int32
}

// Accept implements the Accept method in the net.Listener interface.
func (l *GracefulListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		if atomic.LoadInt32(&l.open) == 0 {
			err = listenerAlreadyClosed{err}
		}
		return nil, err
	}

	gconn := &gracefulConn{conn, 0}
	return gconn, nil
}

// Close tells the wrapped Listener to stop listening.  It is idempotent.
func (l *GracefulListener) Close() error {
	if atomic.CompareAndSwapInt32(&l.open, 1, 0) {
		err := l.Listener.Close()
		return err
	}
	return nil
}

// A gracefulConn wraps a normal net.Conn and tracks its the
// last known http state.
type gracefulConn struct {
	net.Conn
	lastHTTPState http.ConnState
}

type listenerAlreadyClosed struct {
	error
}
