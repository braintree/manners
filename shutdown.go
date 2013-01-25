package manners

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	ShutDownHandler func()
	ShutDownChannel = make(chan os.Signal)
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
	signal.Notify(ShutDownChannel, syscall.SIGINT)
	<-ShutDownChannel
	ShutDownHandler()
}
