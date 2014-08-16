package manners

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// an inefficient replica of a waitgroup that can be introspected
type testWg struct {
	sync.Mutex
	count      int
	waitCalled chan int
}

func newTestWg() *testWg {
	return &testWg{
		waitCalled: make(chan int, 1),
	}
}

func (wg *testWg) Add(delta int) {
	wg.Lock()
	wg.count++
	wg.Unlock()
}

func (wg *testWg) Done() {
	wg.Lock()
	wg.count--
	wg.Unlock()
}

func (wg *testWg) Wait() {
	wg.Lock()
	wg.waitCalled <- wg.count
	wg.Unlock()
}

// a simple step-controllable http client
type client struct {
	connected   chan error
	sendrequest chan bool
	idle        chan bool
	idlerelease chan bool
	closed      chan bool
}

func (c *client) Run() {
	go func() {
		conn, err := net.Dial("tcp", "localhost:7000")
		c.connected <- err
		for <-c.sendrequest {
			defer conn.Close()
			conn.Write([]byte("GET / HTTP/1.1\nHost: localhost:8000\n\n"))
			// Read response; no content
			scanner := bufio.NewScanner(conn)
			for scanner.Scan() {
				// our null handler doesn't send a body, so we know the request is
				// done when we reach the blank line after the headers
				if scanner.Text() == "" {
					break
				}
			}
			c.idle <- true
			<-c.idlerelease
		}
		conn.Close()
		ioutil.ReadAll(conn)
		c.closed <- true
	}()
}

func newClient() *client {
	return &client{
		connected:   make(chan error),
		sendrequest: make(chan bool),
		idle:        make(chan bool),
		idlerelease: make(chan bool),
		closed:      make(chan bool),
	}
}

// a handler that returns 200 ok with no body
var nullHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

func startServer(t *testing.T, server *GracefulServer, statechanged chan http.ConnState) chan error {
	server.up = make(chan chan bool)
	exitchan := make(chan error)

	go func() {
		err := server.ListenAndServe("localhost:7000", nullHandler)
		if err != nil {
			exitchan <- err
		} else {
			exitchan <- nil
		}
	}()

	// wait for server socket to be bound
	select {
	case c := <-server.up:
		// all good
		if statechanged != nil {
			// Wrap the ConnState handler with something that will notify
			// the statechanged channel when a state change happens
			cs := server.server.ConnState
			server.server.ConnState = func(conn net.Conn, newState http.ConnState) {
				cs(conn, newState)
				s := conn.(*GracefulConn).lastHttpState
				statechanged <- s
			}
		}
		c <- true

	case err := <-exitchan:
		// all bad
		t.Fatal("Server failed to start", err)
	}
	return exitchan
}

// Tests that the server allows in-flight requests to complete
// before shutting down.
func TestGracefulness(t *testing.T) {
	server := NewServer()
	wg := newTestWg()
	server.wg = wg
	exitchan := startServer(t, server, nil)

	client := newClient()
	client.Run()

	// wait for client to connect, but don't let it send the request yet
	if err := <-client.connected; err != nil {
		t.Fatal("Client failed to connect to server", err)
	}

	server.Shutdown <- true

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

var stateTests = []struct {
	states       []http.ConnState
	finalWgCount int
}{
	{[]http.ConnState{http.StateNew, http.StateActive}, 1},
	{[]http.ConnState{http.StateNew, http.StateClosed}, 0},
	{[]http.ConnState{http.StateNew, http.StateActive, http.StateClosed}, 0},
	{[]http.ConnState{http.StateNew, http.StateActive, http.StateHijacked}, 0},
	{[]http.ConnState{http.StateNew, http.StateActive, http.StateIdle}, 0},
	{[]http.ConnState{http.StateNew, http.StateActive, http.StateIdle, http.StateActive}, 1},
	{[]http.ConnState{http.StateNew, http.StateActive, http.StateIdle, http.StateActive, http.StateIdle}, 0},
	{[]http.ConnState{http.StateNew, http.StateActive, http.StateIdle, http.StateActive, http.StateClosed}, 0},
	{[]http.ConnState{http.StateNew, http.StateActive, http.StateIdle, http.StateActive, http.StateIdle, http.StateClosed}, 0},
}

func fmtstates(states []http.ConnState) string {
	names := make([]string, len(states))
	for i, s := range states {
		names[i] = s.String()
	}
	return strings.Join(names, " -> ")
}

// Test the state machine in isolation without a network connection
func TestStateTransitions(t *testing.T) {
	for _, test := range stateTests {
		fmt.Println("Starting test ", fmtstates(test.states))
		server := NewServer()
		wg := newTestWg()
		server.wg = wg
		startServer(t, server, nil)

		conn := &GracefulConn{nil, 0}
		for _, newState := range test.states {
			server.server.ConnState(conn, newState)
		}

		server.Shutdown <- true
		waiting := <-wg.waitCalled
		if waiting != test.finalWgCount {
			t.Errorf("%s - Waitcount should be %d, got %d", fmtstates(test.states), test.finalWgCount, waiting)
		}

	}
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
func TestStateTransitioActiveIdleActiven(t *testing.T) {
	server := NewServer()
	wg := newTestWg()
	statechanged := make(chan http.ConnState)
	server.wg = wg
	exitchan := startServer(t, server, statechanged)

	client := newClient()
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

	server.Shutdown <- true
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
	server := NewServer()
	wg := newTestWg()
	statechanged := make(chan http.ConnState)
	server.wg = wg
	exitchan := startServer(t, server, statechanged)

	client := newClient()
	client.Run()

	// wait for client to connect, but don't let it send the request
	if err := <-client.connected; err != nil {
		t.Fatal("Client failed to connect to server", err)
	}

	client.sendrequest <- true
	waitForState(t, statechanged, http.StateActive, "Client failed to reach active state")

	<-client.idle
	client.idlerelease <- true
	waitForState(t, statechanged, http.StateIdle, "Client failed to reach idle state")

	// client is now in an idle state
	close(client.sendrequest)
	<-client.closed
	waitForState(t, statechanged, http.StateClosed, "Client failed to reach closed state")

	server.Shutdown <- true
	waiting := <-wg.waitCalled
	if waiting != 0 {
		t.Errorf("Waitcount should be zero, got %d", waiting)
	}

	if err := <-exitchan; err != nil {
		t.Error("Unexpected error during shutdown", err)
	}
}

// Tests that the server begins to shut down when told to and does not accept
// new requests once shutdown has begun
func TestShutdown(t *testing.T) {
	server := NewServer()
	wg := newTestWg()
	server.wg = wg
	exitchan := startServer(t, server, nil)

	client1 := newClient()
	client1.Run()

	// wait for client1 to connect
	if err := <-client1.connected; err != nil {
		t.Fatal("Client failed to connect to server", err)
	}

	// start the shutdown; once it hits waitgroup.Wait()
	// the listener should of been closed, though client1 is still connected
	server.Shutdown <- true

	waiting := <-wg.waitCalled
	if waiting != 1 {
		t.Errorf("Waitcount should be one, got %d", waiting)
	}

	// should get connection refused at this point
	client2 := newClient()
	client2.Run()

	if err := <-client2.connected; err == nil {
		t.Fatal("client2 connected when it should of received connection refused")
	}

	// let client1 finish so the server can exit
	close(client1.sendrequest) // don't bother sending an actual request

	<-exitchan
}
