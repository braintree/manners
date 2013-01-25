package manners

import (
  "fmt"
  "os"
	"os/signal"
	"sync"
)

var (
	ShutDownHandler func()
	ShutDownChannel = make(chan os.Signal, 1)
	waitGroup       = sync.WaitGroup{}
)

func StartRoutine() {
	waitGroup.Add(1)
}

func FinishRoutine() {
	waitGroup.Done()
}

func WaitForFinish() {
	waitGroup.Wait()
}

func WaitForSignal() {
  fmt.Println("Waiting for a signal")
	signal.Notify(ShutDownChannel)
	<-ShutDownChannel
  fmt.Println("Caught me a signal")
	ShutDownHandler()
}
