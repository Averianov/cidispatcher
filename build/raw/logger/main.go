package main

import (
	"time"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

var (
	Name = "logger"
)

// RadioKat implementation
var rk = func(sender, key string, value any) {
	sl.L.Info("[%s] GOT {sender: %s value: %s}", Name, sender, value)
}

func main() {
	wrapper.RadioKat = rk
	wrapper.CreateWrapper(Name, -1, -1)

	//### Work #################################################################
	for {
		time.Sleep(5 * time.Second)
		sl.L.Debug("do any service work")
	}
}
