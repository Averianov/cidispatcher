package dispatcher

import (
	//"fmt"
	"flag"
	"time"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

func ManualSender() {
	ch := flag.String("ch", wrapper.MASTER, "channel")
	key := flag.String("key", wrapper.STATUS, "key")
	msg := flag.String("m", wrapper.GETINFO, "message")
	l := flag.Int("l", 3, "log level")
	flag.Parse()

	wpr := wrapper.CreateWrapper(wrapper.SENDER, int32(*l), 0)

	go func() {
		_, sender, key, value, err := wpr.ReadGroup()
		if err != nil {
			sl.L.Warning("[%s]%s", wpr.Name, err.Error())
			//panic(fmt.Sprintf("[sender] %s", err.Error()))
			return
		}
		sl.L.Info("\n[response from %s] %s-%v", sender, key, value)
	}()

	time.Sleep(1 * time.Second)

	err := wpr.SendToService(*ch, *key, *msg)
	if err != nil {
		sl.L.Warning("[%s]%s", wpr.Name, err.Error())
		//panic(fmt.Sprintf("[sender] %s", err.Error()))
		return
	}

	time.Sleep(1 * time.Second)
}
