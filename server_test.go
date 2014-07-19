package manners

import (
	"net"
	"net/http"
	"testing"
	"time"
)

// Tests that the server allows in-flight requests to complete before shutting
// down.
func TestGracefulness(t *testing.T) {
	serverDone := make(chan bool)
	clientDone := make(chan bool)
	clientConnected := make(chan bool)

	server := NewServer()

	// server
	go func() {
		err := server.ListenAndServe("localhost:7000", nil)
		if err != nil {
			t.Error(err)
		}
		serverDone <- true
	}()

	// client
	go func() {
		// give the server a chance to start (ugly, FIXME)
		time.Sleep(1000 * time.Microsecond)

		conn, err := net.Dial("tcp", "localhost:7000")
		if err != nil {
			t.Error(err)
		}
		defer conn.Close()

		// signal that client established connection
		clientConnected <- true

		// give the server a chance to exit ungracefully
		time.Sleep(1000 * time.Microsecond)

		// signal we're about to exit
		clientDone <- true
	}()

	// wait for client to connect
	<-clientConnected

	// green light for server to shutdown
	server.Shutdown <- true

	// let's see who exits first
	select {
	case <-clientDone:
		// client exited first, test passed
		// allow server to be done
		<-serverDone
	case <-serverDone:
		t.Fatal("The request did not complete before server exited")
	}
}

// Tests that the server begins to shut down when told to and does not accept
// new requests
func TestShutdown(t *testing.T) {
	server := NewServer()
	exited := make(chan bool)

	go func() {
		err := server.ListenAndServe("localhost:7100", nil)
		if err != nil {
			t.Error(err)
		}
		exited <- true
	}()

	server.Shutdown <- true

	<-exited
	_, err := http.Get("http://localhost:7100")

	if err == nil {
		t.Fatal("Did not receive an error when trying to connect to server.")
	}
}
