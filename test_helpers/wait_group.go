package test_helpers

import "sync"

type WaitGroup struct {
	sync.Mutex
	Count        int
	WaitCalled   chan int
	CountChanged chan int
}

func NewWaitGroup() *WaitGroup {
	return &WaitGroup{
		WaitCalled:   make(chan int, 1),
		CountChanged: make(chan int, 1024),
	}
}

func (wg *WaitGroup) Add(delta int) {
	wg.Lock()
	wg.Count++
	wg.CountChanged <- wg.Count
	wg.Unlock()
}

func (wg *WaitGroup) Done() {
	wg.Lock()
	wg.Count--
	wg.CountChanged <- wg.Count
	wg.Unlock()
}

func (wg *WaitGroup) Wait() {
	wg.Lock()
	wg.WaitCalled <- wg.Count
	wg.Unlock()
}
