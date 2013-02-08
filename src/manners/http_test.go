package manners

import (
  "os"
  "net/http"
	"syscall"
	"testing"
	"time"
  "strings"
)

var testChan chan string

type handlerStub struct{}

func (this *handlerStub) ServeHTTP(response http.ResponseWriter, request *http.Request) {
  time.Sleep(3e9)
  testChan <- "Request finished serving"
}

func serveOrPanic(addr string, handler http.Handler, errors chan error) {
  err := ListenAndServe(addr, handler)
  if err != nil {
    errors <- err
  }
}

func getOrPanic(addr string, errors chan error) {
  _, err := http.Get(addr)
  if err != nil {
    errors <- err
  }
}

// Test that the server finishes handling requests after being told to shut down.
func TestGracefulness(T *testing.T) {
  handler := &handlerStub{}
  ShutDownChannel = make(chan os.Signal, 1)
  testChan = make(chan string, 1)
  serviceErrors := make(chan error, 1)
  getErrors := make(chan error, 1)
  go serveOrPanic(":7000", handler, serviceErrors)
  go getOrPanic("http://localhost:7000", getErrors)
  ShutDownChannel <- syscall.SIGINT
  /*err := <-getErrors*/
  /*panic(err)*/
  <-testChan
}

// Test that the server does not accept a new request after being told to shut down.
func TestShutdown(T *testing.T) {
  handler := &handlerStub{}
  ShutDownChannel = make(chan os.Signal, 1)
  testChan = make(chan string, 1)
  go ListenAndServe("8000", handler)
  ShutDownChannel <- syscall.SIGINT
  _, err := http.Get("http://localhost:8000")
  if err == nil {
    T.Error("Did not get error when trying to get at closed server.")
  } else if !strings.Contains(err.Error(), "connection refused") {
    T.Error("Connection was not refused after server shut down")
  }
}
