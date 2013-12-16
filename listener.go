package manners

import (
  "net"
)

// A GracefulListener differs from a standard net.Listener in three ways:
//    1. It increases the server's WaitGroup when it accepts a connection.
//    2. It returns GracefulConnections rather than normal net.Conns.
//    3. If Accept() is called after it is gracefully closed, it returns a 
//       listenerAlreadyClosed error. The GracefulServer will ignore this
//       error.
type GracefulListener struct {
	net.Listener
	open   bool
	server *GracefulServer
}

func (this *GracefulListener) Accept() (net.Conn, error) {
	conn, err := this.Listener.Accept()
	if err != nil {
		if !this.open {
			err = listenerAlreadyClosed{err}
		}
		return nil, err
	}
	this.server.StartRoutine()
	return GracefulConnection{conn, this.server}, nil
}

func (this *GracefulListener) Close() error {
	if !this.open {
		return nil
	}
	this.open = false
	err := this.Listener.Close()
	return err
}

// GracefulConnections are identical to net.Conns except that they decrement
// their parent servers' WaitGroup after closing.
type GracefulConnection struct {
	net.Conn
	server *GracefulServer
}

func (this GracefulConnection) Close() error {
	err := this.Conn.Close()
	this.server.FinishRoutine()
	return err
}

type listenerAlreadyClosed struct {
	error
}
