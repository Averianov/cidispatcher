package dispatcher

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestDispatcher(t *testing.T) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			fmt.Printf("Critical error in main: %v", recoverErr)
		}
	}()

	d := CreateDispatcher(nil, 2)
	tsk1 := d.AddTask("exampleService1", false, exampleService, []*Task{}, "exampleService1")
	tsk2 := d.AddTask("exampleService2", false, exampleService, []*Task{tsk1}, "exampleService2")
	tsk3 := d.AddTask("exampleService3", false, exampleService, []*Task{tsk2}, "exampleService3")

	go func() {
		tsk3.Start()
		time.Sleep(time.Second * 6)
		tsk2.Stop()
		time.Sleep(time.Second * 6)
		d.RemoveTask(tsk2)
		//tsk = D.Tasks["exampleService1"]
		time.Sleep(time.Second * 10)
		log.Fatal("FATAL") // for look process
	}()

	d.Checking()
}

func exampleService(t *Task) (err error) {
	var name string = fmt.Sprintf("%v", t.Val[0])
	i := 30
	t.Started() // WARNING! Task must be checked as Started from this function (after preparing and befor started)
	for {
		i--
		select {
		case <-t.Ctx.Done():
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
