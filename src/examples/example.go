package main

import (
  "net/http"
  "manners"
  "time"
)

// Our slow  handler implements http.Handler. But the requests take so long to handle!
type SlowHandler struct {}

// Writes "We made it!" onto the request after 10 seconds
func (this *SlowHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
  time.Sleep(10e9)
  response.Write([]byte("We made it!"))
}

// Boot up the slow handler, but let the well-mannered server serve it.
func main() {
  slowHandler := &SlowHandler{}
  manners.ListenAndServe(slowHandler, ":8000")
}
