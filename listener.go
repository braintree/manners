package manners

import "net"

func NewListener(l net.Listener, s *GracefulServer) *GracefulListener {
	return &GracefulListener{l, true, s}
}

// A GracefulListener differs from a standard net.Listener in one way: if
// Accept() is called after it is gracefully closed, it returns a
// listenerAlreadyClosed error. The GracefulServer will ignore this
// error.
type GracefulListener struct {
	net.Listener
	open   bool
	server *GracefulServer
}

func (l *GracefulListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		if !l.open {
			err = listenerAlreadyClosed{err}
		}
		return nil, err
	}
	return conn, nil
}

func (l *GracefulListener) Close() error {
	if !l.open {
		return nil
	}
	l.open = false
	err := l.Listener.Close()
	return err
}

type listenerAlreadyClosed struct {
	error
}
