package manners

import (
	"net/http"
	"syscall"
	"testing"
	"time"
)

type MyResponseWriter struct {
	header http.Header
	bytes  []byte
}

type SlowHandler struct{}

func (this *SlowHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	time.Sleep(10 * 1000)
	response.Write([]byte("We made it!"))
}

func TestTheServerShutsDownGracefully(*testing.T) {
	slowHandler := &SlowHandler{}
	gracefulServer := GracefulServer(slowHandler, ":8000")
	err := gracefulServer.ListenAndServe()
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest("GET", "/")
	if err != nil {
		panic(err)
	}
	ShutDownChannel <- syscall.SIGINT

}
