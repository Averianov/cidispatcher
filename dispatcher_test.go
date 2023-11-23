package dispatcher

import (
	"fmt"
	"testing"
	"time"
)

func TestDispatcher(t *testing.T) {
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			fmt.Printf("Critical error in main: %v", recoverErr)
		}
	}()

	d := CreateDispatcher(nil, 2) // every 2 seconds dispatcher do checking tasks
	tsk1 := d.AddTask("exampleService1", false, exampleService, []*Task{})
	tsk2 := d.AddTask("exampleService2", false, exampleService, []*Task{tsk1})
	d.AddTask("exampleService3", true, exampleService, []*Task{tsk2})

	go func() { // we can manage tasks from any process who take access to dispatcher
		time.Sleep(time.Second * 6)
		tsk2.Stop()
		time.Sleep(time.Second * 6)
		d.RemoveTaskAndRequired(tsk2)
		//tsk = D.Tasks["exampleService1"]
		time.Sleep(time.Second * 10)
		d.Stop() // for gracefull shutdown application
		time.Sleep(time.Second * 6)
		//log.Fatal("FATAL") // for look process
	}()

	d.Checking()
}

// exampleService is wrapper. It is example how make task_function \n
// template: func(t *Task) error
func exampleService(t *Task) (err error) {

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
			fmt.Printf("test %s - %v\n", t.Name, i)
			time.Sleep(time.Second)
		}
	}
}
