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
			L.Alert(TestDispatcher, "Critical error in main: %v", recoverErr)
		}
	}()

	D = CreateDispatcher(0)
	tsk := D.AddTask("exampleService", STOP, exampleService)
	go func() {
		tsk.Start()
		time.Sleep(time.Second * 6)
		tsk.Stop()
		time.Sleep(time.Second * 10)
		D.RemoveTask(tsk)
		//tsk = D.Tasks["exampleService"]
		tsk.Start()
		time.Sleep(time.Second * 10)
		os.Exit(0)
	}()

	D.Tracker()
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

// func shortenRun() {

// 	var shortenString = func(message string) func() string {
// 		return func() string {
// 			messageSlice := strings.Split(message, " ")
// 			wordLength := len(messageSlice)
// 			if wordLength < 1 {
// 				return "Nothingn Left!"
// 			} else {
// 				messageSlice = messageSlice[:(wordLength - 1)]
// 				message = strings.Join(messageSlice, " ")
// 				return message
// 			}
// 		}
// 	}

// 	myString := shortenString("Welcome to concurrency in Go! ...")
// 	fmt.Println(myString())
// 	fmt.Println(myString())
// 	fmt.Println(myString())
// 	fmt.Println(myString())
// 	fmt.Println(myString())
// 	fmt.Println(myString())
// }
