package manners

import (
	"net"
	"net/http"
  "errors"
)

func ListenAndServe(addr string, handler http.Handler) error {
  listener, err := NewListener(addr)
  if err != nil { return err }
  CloseOnShutdown(listener)
  err = GracefullyServe(listener, handler)
  return err
}

func NewListener(addr string) (*GracefulListener, error) {
	baseListener, err := net.Listen("tcp", addr)
	if err != nil { return nil, err }
	listener := GracefulListener{baseListener, true}
  return &listener, nil
}

func GracefullyServe(listener *GracefulListener, handler http.Handler) error {
	server := http.Server{Handler: handler}
  go WaitForSignal()
  err := server.Serve(listener)
	if err == nil {
    return nil
  } else if err.Error() == "The server is shutting down." {
    return nil
  }
  return err
}

type GracefulListener struct {
	net.Listener
	open bool
}

func (this *GracefulListener) Accept() (net.Conn, error) {
	conn, err := this.Listener.Accept()
  if err != nil {
		if !this.open {
			WaitForFinish()
      err = errors.New("The server is shutting down.")
		}
		return nil, err
	}
	StartRoutine()
	return GracefulConnection{conn}, nil
}

func (this *GracefulListener) Close() error {
	if !this.open {
		return nil
	}
	this.open = false
	err := this.Listener.Close()
	return err
}

type GracefulConnection struct {
	net.Conn
}

func (this GracefulConnection) Close() error {
	err := this.Conn.Close()
	FinishRoutine()
	return err
}
