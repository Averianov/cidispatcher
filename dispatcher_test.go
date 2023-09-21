package dispatcher

import (
	"context"
	"fmt"
	"testing"
	"time"
)

var D *Dispatcher

func TestDispatcher(t *testing.T) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			L.Alert(TestDispatcher, "Critical error in main: %v", recoverErr)
		}
	}()

	D = CreateDispatcher(0)
	tsk := D.AddTask("exampleService", STOP, exampleService)
	go func() {
		tsk.Start()
		time.Sleep(time.Second * 9)
		tsk.Stop()
		time.Sleep(time.Second * 120)
	}()

	D.Tracker()
}

func exampleService(ctx context.Context, val ...interface{}) (err error) {
	i := 0
	for {
		i++
		select {
		case <-ctx.Done():
			return
		default:
			// if i == 3 { // when do panic
			// 	a := 0
			// 	fmt.Printf("division by zero %v \n", (1 / a))
			// }
			fmt.Printf("test %v\n", i)
			time.Sleep(time.Second)
		}
	}
}
