package service

import (
	"time"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
	"github.com/Averianov/ciutils"
)

// ### Work #################################################################
func Srv(wpr *wrapper.Wrapper) {
	var err error
	defer sl.L.Warning("[%s] End task by timeout", wpr.Name)

	var i int = 0
	for {
		select {
		case <-wpr.StopChan:
			sl.L.Warning("[%s] Stopping from Channel", wpr.Name)
			return
		default:
			err = wpr.SendToService("logger", wpr.Name, ciutils.IntToStr(i))
			if err != nil {
				time.Sleep(5 * time.Second)
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
