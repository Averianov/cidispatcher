package main

import (
	"time"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

var (
	Name = "logger"
)

func main() {
	wpr, err := wrapper.CreateWrapper(Name, -1, -1)
	if err != nil {
		panic(err.Error())
	}
	defer wpr.RegularStop()

	//### Work #################################################################
	defer sl.L.Warning("[%s] End task by timeout", wpr.Name)

	var i int = 0
	for {
		select {
		case <-wpr.StopChan:
			sl.L.Warning("[%s] Stopping from Channel", wpr.Name)
			return
		default:
			_, msg, err := wpr.ReadGroup(wrapper.DEFAULT_TRYING_COUNT)
			if err != nil {
				time.Sleep(5 * time.Second)
				sl.L.Warning(err.Error())
				continue
			}
			sl.L.Info("[%s] GOT: %s", wpr.Name, msg)
			i++
		}
	}
}
