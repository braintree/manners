package manners

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

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

// Tests that the server allows in-flight requests to complete
// before shutting down.
func TestGracefulness(t *testing.T) {
	server := newServer()
	wg := newTestWg()
	server.wg = wg
	listener, exitchan := startServer(t, server, nil)

	client := newClient(listener.Addr(), false)
	client.Run()

	// wait for client to connect, but don't let it send the request yet
	if err := <-client.connected; err != nil {
		t.Fatal("Client failed to connect to server", err)
	}

	server.Close()

	waiting := <-wg.waitCalled
	if waiting < 1 {
		t.Errorf("Expected the waitgroup to equal 1 at shutdown; actually %d", waiting)
	}

	// allow the client to finish sending the request and make sure the server exits after
	// (client will be in connected but idle state at that point)
	client.sendrequest <- true
	close(client.sendrequest)
	if err := <-exitchan; err != nil {
		t.Error("Unexpected error during shutdown", err)
	}
}

// Test the state machine in isolation without a network connection
func TestStateTransitions(t *testing.T) {
	for _, test := range stateTests {
		fmt.Println("Starting test ", fmtstates(test.states))
		server := newServer()
		wg := newTestWg()
		server.wg = wg
		startServer(t, server, nil)

		conn := &gracefulConn{nil, 0}
		for _, newState := range test.states {
			server.ConnState(conn, newState)
		}

		server.Close()
		waiting := <-wg.waitCalled
		if waiting != test.finalWgCount {
			t.Errorf("%s - Waitcount should be %d, got %d", fmtstates(test.states), test.finalWgCount, waiting)
		}

	}
}

// Test that a connection is closed upon reaching an idle state iff the server
// is shutting down.
func TestCloseOnIdle(t *testing.T) {
	server := newServer()
	wg := newTestWg()
	server.wg = wg
	fl := newFakeListener()
	runner := func() error {
		return server.Serve(fl)
	}

	startGenericServer(t, server, nil, runner)

	fconn := &fakeConn{}
	conn := &gracefulConn{fconn, http.StateActive}

	// Change to idle state while server is not closing; Close should not be called
	server.ConnState(conn, http.StateIdle)
	if conn.lastHTTPState != http.StateIdle {
		t.Errorf("State was not changed to idle")
	}
	if fconn.closeCalled {
		t.Error("Close was called unexpected")
	}

	// push back to active state
	conn.lastHTTPState = http.StateActive
	server.Close()
	// race?

	// wait until the server calls Close() on the listener
	// by that point the atomic closing variable will have been updated, avoiding a race.
	<-fl.closeCalled

	server.ConnState(conn, http.StateIdle)
	if conn.lastHTTPState != http.StateIdle {
		t.Error("State was not changed to idle")
	}
	if !fconn.closeCalled {
		t.Error("Close was not called")
	}

	fl.acceptRelease <- true
}

func waitForState(t *testing.T, waiter chan http.ConnState, state http.ConnState, errmsg string) {
	for {
		select {
		case ns := <-waiter:
			if ns == state {
				return
			}
		case <-time.After(time.Second):
			t.Fatal(errmsg)
		}
	}
}

// Test that a request moving from active->idle->active using an actual
// network connection still results in a corect shutdown
func TestStateTransitionActiveIdleActive(t *testing.T) {
	server := newServer()
	wg := newTestWg()
	statechanged := make(chan http.ConnState)
	server.wg = wg
	listener, exitchan := startServer(t, server, statechanged)

	client := newClient(listener.Addr(), false)
	client.Run()

	// wait for client to connect, but don't let it send the request
	if err := <-client.connected; err != nil {
		t.Fatal("Client failed to connect to server", err)
	}

	for i := 0; i < 2; i++ {
		client.sendrequest <- true
		waitForState(t, statechanged, http.StateActive, "Client failed to reach active state")
		<-client.idle
		client.idlerelease <- true
		waitForState(t, statechanged, http.StateIdle, "Client failed to reach idle state")
	}

	// client is now in an idle state

	server.Close()
	waiting := <-wg.waitCalled
	if waiting != 0 {
		t.Errorf("Waitcount should be zero, got %d", waiting)
	}

	if err := <-exitchan; err != nil {
		t.Error("Unexpected error during shutdown", err)
	}
}

// Test state transitions from new->active->-idle->closed using an actual
// network connection and make sure the waitgroup count is correct at the end.
func TestStateTransitionActiveIdleClosed(t *testing.T) {
	var (
		listener net.Listener
		exitchan chan error
	)

	keyFile, err1 := NewTempFile(localhostKey)
	certFile, err2 := NewTempFile(localhostCert)
	defer keyFile.Unlink()
	defer certFile.Unlink()

	if err1 != nil || err2 != nil {
		t.Fatal("Failed to create temporary files", err1, err2)
	}

	for _, withTLS := range []bool{false, true} {
		server := newServer()
		wg := newTestWg()
		statechanged := make(chan http.ConnState)
		server.wg = wg
		if withTLS {
			listener, exitchan = startTLSServer(t, server, certFile.Name(), keyFile.Name(), statechanged)
		} else {
			listener, exitchan = startServer(t, server, statechanged)
		}

		client := newClient(listener.Addr(), withTLS)
		client.Run()

		// wait for client to connect, but don't let it send the request
		if err := <-client.connected; err != nil {
			t.Fatal("Client failed to connect to server", err)
		}

		client.sendrequest <- true
		waitForState(t, statechanged, http.StateActive, "Client failed to reach active state")

		err := <-client.idle
		if err != nil {
			t.Fatalf("tls=%t unexpected error from client %s", withTLS, err)
		}

		client.idlerelease <- true
		waitForState(t, statechanged, http.StateIdle, "Client failed to reach idle state")

		// client is now in an idle state
		close(client.sendrequest)
		<-client.closed
		waitForState(t, statechanged, http.StateClosed, "Client failed to reach closed state")

		server.Close()
		waiting := <-wg.waitCalled
		if waiting != 0 {
			t.Errorf("Waitcount should be zero, got %d", waiting)
		}

		if err := <-exitchan; err != nil {
			t.Error("Unexpected error during shutdown", err)
		}
	}
}

// Test that supplying a non GracefulListener to Serve works
// correctly (ie. that the listener is wrapped to become graceful)
func TestWrapConnection(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal("Failed to create listener", err)
	}

	s := newServer()
	s.up = make(chan net.Listener)

	var called bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		s.Close() // clean shutdown as soon as handler exits
	})
	s.Handler = handler

	serverr := make(chan error)

	go func() {
		serverr <- s.Serve(l)
	}()

	gl := <-s.up
	if _, ok := gl.(*GracefulListener); !ok {
		t.Fatal("connection was not wrapped into a GracefulListener")
	}

	addr := l.Addr()
	if _, err := http.Get("http://" + addr.String()); err != nil {
		t.Fatal("Get failed", err)
	}

	if err := <-serverr; err != nil {
		t.Fatal("Error from Serve()", err)
	}

	if !called {
		t.Error("Handler was not called")
	}

}

// Tests that the server begins to shut down when told to and does not accept
// new requests once shutdown has begun
func TestShutdown(t *testing.T) {

	server := newServer()
	wg := newTestWg()
	server.wg = wg
	listener, exitchan := startServer(t, server, nil)

	client1 := newClient(listener.Addr(), false)
	client1.Run()

	// wait for client1 to connect
	if err := <-client1.connected; err != nil {
		t.Fatal("Client failed to connect to server", err)
	}

	// start the shutdown; once it hits waitgroup.Wait()
	// the listener should of been closed, though client1 is still connected
	server.Close()

	waiting := <-wg.waitCalled
	if waiting != 1 {
		t.Errorf("Waitcount should be one, got %d", waiting)
	}

	// should get connection refused at this point
	client2 := newClient(listener.Addr(), false)
	client2.Run()

	if err := <-client2.connected; err == nil {
		t.Fatal("client2 connected when it should of received connection refused")
	}

	// let client1 finish so the server can exit
	close(client1.sendrequest) // don't bother sending an actual request

	<-exitchan
}
