package dispatcher

import (
	"fmt"
	"time"
	"flag"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

func ManualSender() {
	ch := flag.String("ch", "master", "channel")
	msg := flag.String("m", wrapper.STATUS+" "+wrapper.SENDER, "message")
    flag.Parse()

	wpr, err := wrapper.CreateWrapper(wrapper.SENDER, 3, 0)
	if err != nil {
		panic(fmt.Sprintf("[sender] %s", err.Error()))
	}

	go func() {
		var rmsg string
		_, rmsg, err = wpr.ReadGroup()

		sl.L.Info("[%s]\n%s", wpr.Name, rmsg)
	}()
	
	time.Sleep(1 * time.Second)

	err = wpr.SendToService(*ch, *msg)
	if err != nil {
		panic(fmt.Sprintf("[sender] %s", err.Error()))
	}

	time.Sleep(1 * time.Second)
}