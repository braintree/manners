package manners

import (
	"net/http"
)

type waitgroup interface {
	Add(int)
	Done()
	Wait()
}

func newServer() *GracefulServer {
	return NewWithServer(new(http.Server))
}
