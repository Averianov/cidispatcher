package main

import (
	"github.com/Averianov/cidispatcher/build/raw/worker1/service"
	"github.com/Averianov/cidispatcher/wrapper"
)

const (
	Name = "worker1"
)

func main() {
	wpr, err := wrapper.CreateWrapper(Name, 4, 1)
	if err != nil {
		panic(err.Error())
	}
	defer wpr.RegularStop()

	service.Srv(wpr)

	//wpr.StartService("worker2") // инициировать запуск worker2 через master_sock
	//wpr.StopService(wpr.Name)   // "должен быть остановлен" - чтобы повторно не запускался
}
