package manners

import (
	"net"
	"net/http"
	"sync"
)

// Creates a new GracefulServer. The server will begin shutting down when
// a value is passed to the Shutdown channel.
func NewServer() *GracefulServer {
	return &GracefulServer{
		Shutdown: make(chan bool),
	}
}

// A GracefulServer maintains a WaitGroup that counts how many in-flight
// requests the server is handling. When it receives a shutdown signal,
// it stops accepting new requests but does not actually shut down until
// all in-flight requests terminate.
type GracefulServer struct {
	Shutdown        chan bool
	wg              sync.WaitGroup
	shutdownHandler func()
}

// A helper function that emulates the functionality of http.ListenAndServe.
func (s *GracefulServer) ListenAndServe(addr string, handler http.Handler) error {
	oldListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	listener := &GracefulListener{oldListener, true, s}
	err = s.Serve(listener, handler)
	return err
}

// Similar to http.Serve. The listener passed must wrap a GracefulListener.
func (s *GracefulServer) Serve(listener net.Listener, handler http.Handler) error {
	s.shutdownHandler = func() { listener.Close() }
	s.listenForShutdown()
	server := http.Server{Handler: handler}
	err := server.Serve(listener)
	if err == nil {
		return nil
	} else if _, ok := err.(listenerAlreadyClosed); ok {
		return nil
	}
	return err
}

// Increments the server's WaitGroup. Use this if a web request starts more
// goroutines and these goroutines are not guaranteed to finish before the
// request.
func (s *GracefulServer) StartRoutine() {
	s.wg.Add(1)
}

// Decrement the server's WaitGroup. Used this to complement StartRoutine().
func (s *GracefulServer) FinishRoutine() {
	s.wg.Done()
}

func (s *GracefulServer) listenForShutdown() {
	go func() {
		<-s.Shutdown
		s.shutdownHandler()
	}()
}
