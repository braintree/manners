package manners

import (
	"net"
	"net/http"
	"testing"
)

type httpInterface interface {
	ListenAndServe() error
	ListenAndServeTLS(certFile, keyFile string) error
	Serve(listener net.Listener) error
}

// Test that the method signatures of the methods we override from net/http/Server
// match those of the original.
func TestInterface(t *testing.T) {
	var original, ours interface{}
	original = &http.Server{}
	ours = &GracefulServer{}
	if _, ok := original.(httpInterface); !ok {
		t.Errorf("httpInterface definition does not match the canonical server!")
	}
	if _, ok := ours.(httpInterface); !ok {
		t.Errorf("GracefulServer does not implement httpInterface")
	}
}
