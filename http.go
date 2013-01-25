package manners

import (
	"fmt"
	"net"
	"net/http"
)

func GracefulServer(handler http.Handler, port string) http.Server {
	baseListener, err := net.Listen("tcp", port)
	if err != nil {
		error := fmt.Sprintf("Could not open TCP socket on port %s: %s", port, err.Error())
		panic(error)
	}
	listener := &GracefulListener{baseListener, true}
	ShutDownHandler = func() { listener.Close() }
	WaitForSignal()
	server := http.Server{Handler: handler}
	err = server.Serve(listener)
	if err != nil {
		error := fmt.Sprintf("Could not serve HTTP: %s", err.Error())
		panic(error)
	}
	return server
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
