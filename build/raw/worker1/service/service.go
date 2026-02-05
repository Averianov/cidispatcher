package service

import (
	"fmt"
	"time"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

//### Work #################################################################
func Srv(wpr *wrapper.Wrapper) {
	var err error

	// Launch self channel reader
	go func() {
		for {
			_, msg, err := wpr.ReadGroup(wrapper.DEFAULT_TRYING_COUNT)
			if err != nil {
				sl.L.Warning(err.Error())
				if _, ok := <-wpr.StopChan; !ok { // check parent process
					return
				}
				continue
			}

			switch msg {
			case "stop":
				sl.L.Info("[%s] try stop parent process", wpr.Name)
				wpr.StopChan <- struct{}{} // stop parent process
			default:
				sl.L.Info("[%s] GOT: %s - %s", wpr.Name, msg)
			}
		}
	}()

	defer sl.L.Warning("[%s] End task by timeout", wpr.Name)

	var i int = 0
	for {
		select {
		case <-wpr.StopChan:
			sl.L.Warning("[%s] Stopping from Channel", wpr.Name)
			return
		default:
			err = wpr.SendToService("logger", fmt.Sprintf("%s:msg-%d", wpr.Name, i))
			if err != nil {
				sl.L.Warning(err.Error())
				continue
			}
			if i == 10 {
				wpr.StartService("worker2")
				//wpr.StopService("logger")
				return
			}

			time.Sleep(1 * time.Second)
			i++
		}
	}
}
