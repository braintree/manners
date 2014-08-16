package manners

import (
	"net"
	"net/http"
	"sync"
)

// interface describing a waitgroup, so unit
// tests can mock out an instrumentable version
type waitgroup interface {
	Add(delta int)
	Done()
	Wait()
}

// Creates a new GracefulServer. The server will begin shutting down when
// a value is passed to the Shutdown channel.
func NewServer() *GracefulServer {
	return &GracefulServer{
		Shutdown: make(chan bool),
		wg:       new(sync.WaitGroup),
	}
}

// A GracefulServer maintains a WaitGroup that counts how many in-flight
// requests the server is handling. When it receives a shutdown signal,
// it stops accepting new requests but does not actually shut down until
// all in-flight requests terminate.
type GracefulServer struct {
	Shutdown        chan bool
	wg              waitgroup
	shutdownHandler func()

	// used by test code
	server *http.Server
	up     chan chan bool
}

// A helper function that emulates the functionality of http.ListenAndServe.
func (s *GracefulServer) ListenAndServe(addr string, handler http.Handler) error {
	oldListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	listener := NewListener(oldListener, s)
	err = s.Serve(listener, handler)
	return err
}

// Similar to http.Serve. The listener passed must wrap a GracefulListener.
func (s *GracefulServer) Serve(listener net.Listener, handler http.Handler) error {
	server := http.Server{Handler: handler}
	s.server = &server
	s.shutdownHandler = func() { listener.Close(); server.SetKeepAlivesEnabled(false) }
	s.listenForShutdown()

	server.ConnState = func(conn net.Conn, newState http.ConnState) {
		gconn := conn.(*GracefulConn)
		switch newState {
		case http.StateNew:
			// new_conn -> StateNew
			s.StartRoutine()

		case http.StateActive:
			// (StateNew, StateIdle) -> StateActive
			if gconn.lastHttpState == http.StateIdle {
				// transitioned from idle back to active
				s.StartRoutine()
			}

		case http.StateIdle:
			// StateActive -> StateIdle
			s.FinishRoutine()

		case http.StateClosed, http.StateHijacked:
			// (StateNew, StateActive, StateIdle) -> (StateClosed, StateHiJacked)
			if gconn.lastHttpState != http.StateIdle {
				// if it was idle it's already been decremented
				s.FinishRoutine()
			}
		}
		gconn.lastHttpState = newState
	}
	// only used by unit tests
	if s.up != nil {
		// notify test that server is up; wait for signal to continue
		c := make(chan bool)
		s.up <- c
		<-c
	}
	err := server.Serve(listener)

	// This block is reached when the server has received a shut down command.
	if err == nil {
		s.wg.Wait()
		return nil
	} else if _, ok := err.(listenerAlreadyClosed); ok {
		s.wg.Wait()
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
