package manners

import (
	"sync"
)

type waitgroup interface {
	Add(int)
	Done()
	Wait()
}

type testWg struct {
	sync.Mutex
	count      int
	waitCalled chan int
}

func newTestWg() *testWg {
	return &testWg{
		waitCalled: make(chan int, 1),
	}
}

func (wg *testWg) Add(delta int) {
	wg.Lock()
	wg.count++
	wg.Unlock()
}

func (wg *testWg) Done() {
	wg.Lock()
	wg.count--
	wg.Unlock()
}

func (wg *testWg) Wait() {
	wg.Lock()
	wg.waitCalled <- wg.count
	wg.Unlock()
}
