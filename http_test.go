package manners

import (
	"net/http"
	"syscall"
	"testing"
	"time"
  "strings"
  "fmt"
)

type SlowHandler struct{}

func (this *SlowHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	time.Sleep(5 * 1000)
	response.Write([]byte("We made it!"))
}

func TestTheServerShutsDownGracefully(*testing.T) {
	slowHandler := &SlowHandler{}
	gracefulServer := GracefulServer(slowHandler, ":8000")
	err := gracefulServer.ListenAndServe()
	if err != nil {
		panic(err)
	}

  responseWriter := NewResponseWriter()
	request, err := http.NewRequest("GET", "/", strings.NewReader("foo"))
  if err != nil {
		panic(err)
	}

  go slowHandler.ServeHTTP(responseWriter, request)
  ShutDownChannel <- syscall.SIGINT
  fmt.Println(responseWriter.Content)
}
