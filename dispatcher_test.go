package dispatcher

import (
	"fmt"
	"testing"
	"time"
)

var D *Dispatcher

func TestDispatcher(t *testing.T) {
	D = CreateDispatcher()
	tsk := D.AddTask("testService", RUN, testService)
	go tsk.Tracker()
	time.Sleep(time.Second * 6)
	//tsk.Stop()
	time.Sleep(time.Second * 14)
}

func testService(down chan struct{}, errch chan error, val ...interface{}) {
	var err error
	defer func() {
		errch <- err
	}()

	var done chan struct{} = make(chan struct{})
	service := func() <-chan struct{} {
		for i := 0; i < 3; i++ {
			fmt.Printf("test %v\n", i)
			time.Sleep(time.Second)
		}
		done <- struct{}{}
		return done
	}

	//go service()

	for {
		select {
		case <-service():
			errch <- nil
			return

		case <-down:
			err = fmt.Errorf("test down signal")
			errch <- err
			return
		}
	}
}
