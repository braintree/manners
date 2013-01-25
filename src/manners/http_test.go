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
	time.Sleep(5e9)
	response.Write([]byte("We made it!"))
}

func TestTheServerShutsDownGracefully(T *testing.T) {
	slowHandler := &SlowHandler{}
	ListenAndServe(slowHandler, ":8000")

  responseWriter := NewResponseWriter()
	request, err := http.NewRequest("GET", "/", strings.NewReader("foo"))
  if err != nil {
		panic(err)
	}

  go slowHandler.ServeHTTP(responseWriter, request)
  fmt.Println("Before call to SIGINT")
  ShutDownChannel <- syscall.SIGINT
  fmt.Println(responseWriter.Content)
  T.Error("false")
}
