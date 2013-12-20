package manners

import (
	"net/http"
	"testing"
	"time"
)

// Tests that the server allows in-flight requests to complete before shutting
// down.
func TestGracefulness(t *testing.T) {
	ready := make(chan bool)
	done := make(chan bool)
	handler := newBlockingHandler(ready, done)
	server := NewServer()

	go func() {
		err := server.ListenAndServe(":7000", handler)
		if err != nil {
			t.Error(err)
		}
	}()

	go func() {
		_, err := http.Get("http://localhost:7000")
		if err != nil {
			t.Error(err)
		}
	}()

	// This will block until the server is inside the handler function.
	<-ready
	server.Shutdown <- true
	select {
	case <-time.After(1e9):
		t.Fatal("Did not receive a value from done; the request did not complete.")
	case done <- true:
		// The handler is being allowed to run to completion; test passes.
	}
}

// Tests that the server begins to shut down when it receives a SIGINT and
// does not accept new requests
func TestShutdown(t *testing.T) {
	handler := newTestHandler()
	server := NewServer()

	go func() {
		err := server.ListenAndServe(":7100", handler)
		if err != nil {
			t.Error(err)
		}
	}()

	server.Shutdown <- true

	_, err := http.Get("http://localhost:7100")
	if err == nil {
		t.Fatal("Did not receive an error when trying to connect to server.")
	}
}
