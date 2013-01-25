package manners

import (
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
	signal.Notify(ShutDownChannel)
	<-ShutDownChannel
	ShutDownHandler()
}
