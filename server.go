package manners

import (
	"net"
	"net/http"
	"sync"
)

func NewServer() *GracefulServer {
  return &GracefulServer{
    shutdown: make(chan bool),
  }
}

type GracefulServer struct {
	wg              sync.WaitGroup
	shutdown        chan bool
	shutdownHandler func() error
}

func (s *GracefulServer) ListenAndServe(addr string, handler http.Handler) error {
	oldListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	listener := &GracefulListener{oldListener, true, s}
	err = s.Serve(listener, handler)
	return err
}

func (s *GracefulServer) Serve(listener *GracefulListener, handler http.Handler) error {
	s.shutdownHandler = func() error { return listener.Close() }
  s.WaitForShutdown()
	server := http.Server{Handler: handler}
	err := server.Serve(listener)
	if err == nil {
		return nil
	} else if _, ok := err.(listenerAlreadyClosed); ok {
		return nil
	}
	return err
}

func (s *GracefulServer) StartRoutine() {
	s.wg.Add(1)
}

func (s *GracefulServer) FinishRoutine() {
	s.wg.Done()
}

func (s *GracefulServer) WaitForShutdown() chan error {
  errs := make(chan error)
  go func() {
    <-s.shutdown
    err := s.shutdownHandler()
    errs <-err
  }()
  return errs
}
