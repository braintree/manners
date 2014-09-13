package manners

import (
	"net"
	"net/http"
	"sync"
)

var defaultServer *GracefulServer

// NewWithServer wraps an existing http.Server object and returns a GracefulServer
// that supports all of the original Server operations.
func NewWithServer(s *http.Server) *GracefulServer {
	return &GracefulServer{
		Server:   s,
		shutdown: make(chan struct{}),
		wg:       new(sync.WaitGroup),
	}
}

// ListenAndServe provides a graceful version of function provided by the net/http package.
func ListenAndServe(addr string, handler http.Handler) error {
	defaultServer = NewWithServer(&http.Server{Addr: addr, Handler: handler})
	return defaultServer.ListenAndServe()
}

// ListenAndServeTLS provides a graceful version of function provided by the net/http package.
func ListenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler) error {
	defaultServer = NewWithServer(&http.Server{Addr: addr, Handler: handler})
	return defaultServer.ListenAndServeTLS(certFile, keyFile)
}

// Serve provides a graceful version of function provided by the net/http package.
func Serve(l net.Listener, handler http.Handler) error {
	defaultServer := NewWithServer(&http.Server{Handler: handler})
	return defaultServer.Serve(l)
}
