package manners

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	waitGroup       = sync.WaitGroup{}
	shutDownHandler func()
)

func StartRoutine() {
	waitGroup.Add(1)
}

func FinishRoutine() {
	waitGroup.Done()
}

func waitForFinish() {
	waitGroup.Wait()
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	<-ch
	shutDownHandler()
}
