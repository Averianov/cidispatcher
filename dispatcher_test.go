package dispatcher

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

var D *Dispatcher

func TestDispatcher(t *testing.T) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			fmt.Printf("Critical error in main: %v", recoverErr)
		}
	}()

	D = CreateDispatcher(nil, 0)
	tsk1 := D.AddTask("exampleService", STOP, exampleService)
	tsk2 := D.AddTask("exampleService2", STOP, exampleService)
	tsk2.Required = append(tsk2.Required, tsk1)

	go func() {
		tsk2.Start()
		time.Sleep(time.Second * 16)
		tsk2.Stop()
		time.Sleep(time.Second * 20)
		D.RemoveTask(tsk2)
		//tsk = D.Tasks["exampleService"]
		tsk2.Start()
		time.Sleep(time.Second * 10)
		os.Exit(0)
	}()

	D.Start()
}

func exampleService(ctx context.Context, val ...interface{}) (err error) {
	i := 10
	for {
		i--
		select {
		case <-ctx.Done():
			fmt.Printf("got done \n")
			return
		default:
			if i == 0 {
				//fmt.Printf("division by zero %v \n", (1 / i)) // when do panic
				return
			}
			fmt.Printf("test %v\n", i)
			time.Sleep(time.Second)
		}
	}
}
