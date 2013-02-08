package manners

import (
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

var testChan chan string

type handlerStub struct{}

func (this *handlerStub) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	time.Sleep(5e9)
	testChan <- "Request finished serving"
}

// Test that the server finishes handling requests after being told to shut down.
func TestGracefulness(T *testing.T) {
	handler := &handlerStub{}
	ShutdownChannel = make(chan os.Signal)
	testChan = make(chan string)
	go ListenAndServe(":7000", handler)
	// Need to ensure that the server boots before sending the request
	time.Sleep(3e9)
	go http.Get("http://localhost:7000")
	// Need to ensure that the request has time to move to the ServeHTTP method
	time.Sleep(3e9)
	ShutdownChannel <- syscall.SIGINT
	select {
	case <-testChan:
	case <-time.After(10e9):
		T.Error("The request did not run completion")
	}
}

// Test that the server does not accept a new request after being told to shut down.
func TestShutdown(T *testing.T) {
	handler := &handlerStub{}
	ShutdownChannel = make(chan os.Signal)
	testChan = make(chan string)
	go ListenAndServe(":7000", handler)
	ShutdownChannel <- syscall.SIGINT
	_, err := http.Get("http://localhost:7000")
	if err == nil {
		T.Error("Did not get error when trying to get at closed server.")
	} else if !strings.Contains(err.Error(), "connection refused") {
		T.Error("Connection was not refused after server shut down")
	}
}

// Test that the server does not accept a new request after being told to shut down
// even if a request is currently being served.
func TestShutdownWithInflightRequest(T *testing.T) {
	handler := &handlerStub{}
	ShutdownChannel = make(chan os.Signal)
	go ListenAndServe(":7000", handler)
	// Need to ensure that the server boots before sending the request
	time.Sleep(3e9)
	go http.Get("http://localhost:7000")
	// Need to ensure that the request has time to move to the ServeHTTP method
	ShutdownChannel <- syscall.SIGINT
	_, err := http.Get("http://localhost:7000")
	if err == nil {
		T.Error("Did not get error when trying to get at closed server.")
	} else if !strings.Contains(err.Error(), "connection refused") {
		T.Error("Connection was not refused after server shut down")
	}
}
