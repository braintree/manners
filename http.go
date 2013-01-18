package manners

import (
	"fmt"
	"net"
	"net/http"
)

func GracefulServer(handler http.Handler, port string) http.Server {
	listener := listenAndWaitForShutdown(port)
	server := http.Server{Handler: handler}
	err := server.Serve(listener)
	if err != nil {
		error := fmt.Sprintf("Could not serve HTTP: %s", err.Error())
		panic(error)
	}
	return server
}

func listenAndWaitForShutdown(port string) net.Listener {
	baseListener, err := net.Listen("tcp", port)
	if err != nil {
		error := fmt.Sprintf("Could not open TCP socket on port %s: %s", port, err.Error())
		panic(error)
	}
	listener := &gracefulListener{baseListener, true}
	shutDownHandler = func() { listener.Close() }
	return listener
}

type gracefulListener struct {
	net.Listener
	open bool
}

func (gl *gracefulListener) Accept() (net.Conn, error) {
	conn, err := gl.Listener.Accept()
	if err != nil {
		if !gl.open {
			waitForFinish()
		}
		return nil, err
	}
	StartRoutine()
	return gracefulConnection{conn}, nil
}

func (gl *gracefulListener) Close() error {
	if !gl.open {
		return nil
	}
	gl.open = false
	err := gl.Listener.Close()
	return err
}

type gracefulConnection struct {
	net.Conn
}

func (gc gracefulConnection) Close() error {
	err := gc.Conn.Close()
	FinishRoutine()
	return err
}
