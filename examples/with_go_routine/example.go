package main

import (
	"github.com/braintree/manners"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var GracefulServer *manners.GracefulServer

func main() {
	router := http.NewServeMux()
	router.Handle("/", indexHandler())

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		GracefulServer.Close()
		close(idleConnsClosed)
	}()

	GracefulServer = manners.NewWithServer(&http.Server{
		Addr:    ":8080",
		Handler: router,
	})

	log.Fatal(GracefulServer.ListenAndServe())

	<-idleConnsClosed
}

func indexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5*time.Second)

		GracefulServer.StartRoutine()
		go func() {
			defer GracefulServer.FinishRoutine()
			time.Sleep(5*time.Second)
			log.Print("Go routine done")
		}()

		w.Write([]byte("Hello Go routine"))
	})
}
