package main

import (
	"github.com/Averianov/cidispatcher/build/raw/worker1/service"
	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

const (
	Name = "worker1"
)

func main() {

	wpr := wrapper.CreateWrapper(Name, -1, -1)

	// RadioKat implementation
	wrapper.RadioKat = func(sender, value string) {
		switch value {
		case "stop":
			sl.L.Info("[%s] try stop parent process", wpr.Name)
			wpr.StopChan <- struct{}{} // stop parent process
		default:
			sl.L.Info("[%s] GOT: from %s: %s", wpr.Name, sender, value)
		}
	}
	
	service.Srv(wpr)
	//wpr.StartService("worker2") // инициировать запуск worker2 через master_sock
	//wpr.StopService(wpr.Name)   // "должен быть остановлен" - чтобы повторно не запускался
}
