package main

import (
  "net/http"
  "manners"
  "github.com/gorilla/mux"
)

func router() http.Handler {
  r := mux.NewRouter()

  // Add your routes here

  return r
}

func main() {
  handler := router()
  manners.ListenAndServe(handler, ":800")
}
