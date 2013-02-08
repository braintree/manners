package main

import (
  "net/http"
  "github.com/braintree/manners"
  "github.com/gorilla/mux"
)

func router() http.Handler {
  r := mux.NewRouter()

  // Add your routes here

  return r
}

func main() {
  handler := router()
  manners.ListenAndServe(handler, ":7000")
}
