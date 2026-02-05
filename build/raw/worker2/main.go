package main

import (
	"fmt"
	"time"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

const (
	Name = "worker2"
	logger = "logger"
)

// В конце останавливает себя и логгер
func main() {
	wpr, err := wrapper.CreateWrapper(Name)
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		//wpr.StopService(wpr.Name) // stop self service - disable autostart after shutdown
		wpr.RegularStop()
	}()

	
	defer sl.L.Warning("[%s] End task by timeout", wpr.Name)
	var i int = 0
	for {
		select {
		case <-wpr.StopChan:
			sl.L.Warning("[%s] Stopping from Channel", wpr.Name)
			return
		default:
			err = wpr.SendToService(logger, fmt.Sprintf("%s:msg-%d", wpr.Name, i))
			if err != nil {
				sl.L.Warning(err.Error())
				continue
			}
			if i == 10 {
				fmt.Printf("[%s] Stopping by timeout\n", wpr.Name)
				//wpr.StopService("worker1")
				//wpr.StopService(logger)
				wpr.StartService("worker3")
				return
			}

			time.Sleep(3 * time.Second)
			i++
		}
	}
}
