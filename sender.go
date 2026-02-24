package dispatcher

import (
	//"fmt"
	"time"
	"flag"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

func ManualSender() {
	ch := flag.String("ch", "master", "channel")
	msg := flag.String("m", wrapper.STATUS, "message")
	l := flag.Int("l", 3, "log level")
    flag.Parse()

	wpr := wrapper.CreateWrapper(wrapper.SENDER, int32(*l), 0)

	go func() {
		_, _, value, err := wpr.ReadGroup()
		if err != nil {
			sl.L.Warning("[%s]%s", wpr.Name, err.Error())
			//panic(fmt.Sprintf("[sender] %s", err.Error()))
			return
		}
		sl.L.Info("\n%s", value)
	}()
	
	time.Sleep(1 * time.Second)

	err := wpr.SendToService(*ch, *msg)
	if err != nil {
		sl.L.Warning("[%s]%s", wpr.Name, err.Error())
		//panic(fmt.Sprintf("[sender] %s", err.Error()))
		return
	}

	time.Sleep(1 * time.Second)
}