package dispatcher

import (
	"context"
	"fmt"
	"log"
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

	D = CreateDispatcher(nil, 1)
	tsk1 := D.AddTask("exampleService1", STOP, exampleService, []*Task{}, "exampleService1")
	tsk2 := D.AddTask("exampleService2", STOP, exampleService, []*Task{tsk1}, "exampleService2")
	tsk3 := D.AddTask("exampleService3", STOP, exampleService, []*Task{tsk2}, "exampleService3")

	go func() {
		tsk3.Start()
		time.Sleep(time.Second * 6)
		tsk2.Stop()
		time.Sleep(time.Second * 6)
		D.RemoveTask(tsk2)
		//tsk = D.Tasks["exampleService1"]
		time.Sleep(time.Second * 10)
		log.Fatal("FATAL") // for look process
	}()

	D.Start()
}

func exampleService(ctx context.Context, val ...interface{}) (err error) {
	var name string = fmt.Sprintf("%v", val[0])
	i := 30
	for {
		i--
		select {
		case <-ctx.Done():
			err = fmt.Errorf("got done \n")
			return
		default:
			if i == 0 {
				//fmt.Printf("division by zero %v \n", (1 / i)) // when do panic
				return
			}
			fmt.Printf("test %s - %v\n", name, i)
			time.Sleep(time.Second)
		}
	}
}
