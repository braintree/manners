package manners

import (
  "os"
	"os/signal"
	"sync"
)

var (
	ShutDownChannel chan os.Signal
	shutdownHandler func()
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
	shutdownHandler()
}

func CloseOnShutdown(listener *GracefulListener) {
  shutdownHandler = func() {
    listener.Close()
  }
}

func ExcecuteOnShutDown(f func()) {
  shutdownHandler = f
}
