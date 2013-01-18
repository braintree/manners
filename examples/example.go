package main

import (
  "net/http"
  "time"
  "github.com/braintreeps/manners"
)

// Our slow  handler implements http.Handler. But the requests take so long to handle!
type SlowHandler struct {}

// Writes "We made it!" onto the request after 10 seconds
func (this *SlowHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
  time.Sleep(10 * 1000)
  response.Write([]byte("We made it!"))
}

// Boot up the slow handler, but let the well-mannered server serve it.
func main() {
  slowHandler := SlowHandler{}
  gracefulServer := manners.GracefulServer(slowHandler, ":8000")
  err := gracefulServer.ListenAndServe()
  if err != nil {
    panic(err)
  }
}
