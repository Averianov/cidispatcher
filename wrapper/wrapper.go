package wrapper

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	sl "github.com/Averianov/cisystemlog"
	"github.com/Averianov/ciutils"
	"github.com/redis/go-redis/v9"
)

const (
	NAME           string = "NAME"
	MASTER         string = "MASTER"
	SENDER         string = "SENDER"
	LOG_LEVEL      string = "LOGLEVEL"
	SIZE_LOG_FILE  string = "SIZE_LOG_FILE"
	TIMELOCATION   string = "TIMELOCATION"
	PORT_FILE_PATH string = "./port"

	SEPARATOR string = ":::"

	DEFAULT_TRYING_COUNT int    = 2
	JUST_WAIT            string = "jw"
)

const (
	STATUS   string = "STATUS"
	LAUNCHED string = "LAUNCHED"
	STOPPED  string = "STOPPED"
	START    string = "START"
	STOP     string = "STOP"
	EXIT     string = "EXIT"
)

var (
	Wpr *Wrapper

	// Sender, Value receive from redis response
	RadioKat func(string, string) // function which preparing receive data from redis channel
)

type Wrapper struct {
	Name      string
	RClient   *redis.Client
	PubSub    *redis.PubSub
	StopChan  chan struct{}
	Env       map[string]string
	TimeDelay map[string]int
	NextTry   map[string]int64
}

// CreateWrapper got name current service and logLevel & sizeLogFile for cisystemlog
func CreateWrapper(name string, logLevel int32, sizeLogFile int64) (wpr *Wrapper) {
	var err error
	defer func() {
		if err != nil {
			panic(err.Error())
		}
	}()

	if name == "" {
		err = fmt.Errorf("%s", "missing service name")
		return
	}
	name = strings.ToUpper(name)

	if logLevel < 1 {
		val, ok := os.LookupEnv(LOG_LEVEL)
		if !ok {
			err = fmt.Errorf("[%s] no log level in env", name)
			return
		}
		logLevel = int32(ciutils.StrToInt(val))
		if logLevel == 0 {
			err = fmt.Errorf("[%s] wrong log level in env", name)
			return
		}
	}

	if sizeLogFile < 0 {
		val, ok := os.LookupEnv(SIZE_LOG_FILE)
		if !ok {
			sizeLogFile = 0
		} else {
			sizeLogFile = ciutils.StrToInt64(val)
		}
	}

	sl.CreateLogs(name, "./log/", logLevel, sizeLogFile)

	Wpr = &Wrapper{
		StopChan:  make(chan struct{}),
		Env:       make(map[string]string),
		TimeDelay: make(map[string]int),
		NextTry:   make(map[string]int64),
	}

	if location, ok := os.LookupEnv(TIMELOCATION); ok {
		ciutils.TimeLocation, err = time.LoadLocation(location)
		if err != nil {
			sl.L.Warning("[%s] %s", name, err.Error())
			return
		}
	}

	// Recheck nameing task in process as in dispatcher
	if name != MASTER && name != SENDER && len(os.Environ()) > 0 {
		for _, val := range os.Environ() {
			//sl.L.Debug("[%s] got env: %s", name, val)
			senv := strings.Split(val, "=")
			Wpr.Env[senv[0]] = senv[1]
			sl.L.Debug("[%s] added env: %s=%s", name, senv[0], senv[1])
		}

		val, ok := os.LookupEnv(NAME)
		if !ok || name != val || name != Wpr.Env[NAME] {
			err = fmt.Errorf("service with name \"%s\" not equal naming with started process.", name)
			return
		}
		name = val
	}
	Wpr.Name = name

	_, err = ciutils.MakeSureFileExists(PORT_FILE_PATH)
	if err != nil {
		sl.L.Warning("[%s] %s", name, err.Error())
		return
	}

	var raw []byte
	raw, err = os.ReadFile(PORT_FILE_PATH)
	if err != nil {
		sl.L.Warning("[%s] %s", name, err.Error())
		return
	}
	rport := string(raw)
	sl.L.Debug("[%s] connect to Redis on: %s", name, rport)
	Wpr.RClient = redis.NewClient(&redis.Options{
		Addr:             "localhost:" + rport,
		ReadTimeout:      -1, // Disable network timeout to read
		WriteTimeout:     5 * time.Second,
		DisableIndentity: true,
		DB:               0,
	})

	ctx := context.Background()
	Wpr.PubSub = Wpr.RClient.Subscribe(ctx, name)

	if Wpr.Name != MASTER && Wpr.Name != SENDER {
		Wpr.SendToService(MASTER, LAUNCHED)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGUSR1) // for cooperative shutdown
	go Wpr.RadioKatListner(sig)
	// go func() {
	// 	<-sig
	// 	sl.L.Alert("[%s] Cooperative shutdown (SIGUSR1)", Wpr.Name)
	// 	close(Wpr.StopChan)
	// 	time.Sleep(5 * time.Second)
	// 	os.Exit(0)
	// 	//panic(fmt.Sprintf("[%s] Cooperative shutdown (SIGUSR1)", Wpr.Name)) // temporary ??
	// }()
	return Wpr
}

func (wpr *Wrapper) RegularStop() {
	Wpr.SendToService(MASTER, STOPPED)
	Wpr.PubSub.Close()
}

func (wpr *Wrapper) StartService(serviceName string) (err error) {
	err = wpr.SendToService(MASTER, START+SEPARATOR+serviceName)
	if err != nil {
		sl.L.Warning("[%s] %s", Wpr.Name, err.Error())
	}
	return
}

func (wpr *Wrapper) StopService(serviceName string) (err error) {
	err = wpr.SendToService(MASTER, STOP+SEPARATOR+serviceName)
	if err != nil {
		sl.L.Warning("[%s] %s", Wpr.Name, err.Error())
	}
	return
}

func (wpr *Wrapper) ReadGroup() (channel, sender, value string, err error) {
	var rmsg *redis.Message

	int64Now := ciutils.TimeToInt64(ciutils.Now())
	// sl.L.Debug("[%s] now: %v; NextTry: %v", wpr.Name,
	// 	ciutils.TimeToStringInFormat(ciutils.Int64ToTime(int64Now), "15:04:05"),
	// 	ciutils.TimeToStringInFormat(ciutils.Int64ToTime(wpr.NextTry[wpr.Name]), "15:04:05"))

	if wpr.NextTry[wpr.Name] > int64Now {
		//err = fmt.Errorf("%s", "Too mutch error Receive from redis server; wait to next available try")
		err = fmt.Errorf("%s", JUST_WAIT)
		//sl.L.Debug("[%s] %s", wpr.Name, err.Error())
		return
	}

	rmsg, err = wpr.PubSub.ReceiveMessage(context.Background())
	if err != nil {
		//sl.L.Debug("[%s] Error ReceiveMessage:%s", wpr.Name, err.Error())
		if wpr.TimeDelay[wpr.Name] == 0 {
			wpr.TimeDelay[wpr.Name] = 1
		}
		wpr.TimeDelay[wpr.Name] = wpr.TimeDelay[wpr.Name] * 2
		//sl.L.Debug("[%s] TimeDelay: %v; NextTry: %v", wpr.Name, wpr.TimeDelay[wpr.Name], ciutils.Int64ToTime(wpr.NextTry[wpr.Name]))
		wpr.NextTry[wpr.Name] = ciutils.TimeToInt64(ciutils.Now().Add(time.Duration(wpr.TimeDelay[wpr.Name]) * time.Second))
		return
	}
	wpr.TimeDelay[wpr.Name] = 1

	//sl.L.Debug("[%s] GOT RAW %v", wpr.Name, rmsg)
	channel = rmsg.Channel
	// PREPARING
	raw := strings.Split(rmsg.Payload, SEPARATOR)
	switch len(raw) {
	case 1:
		sl.L.Debug("[%s] Got: %s", wpr.Name, rmsg.Payload)
	case 2:
		sender = raw[0]
		value = raw[1]
		switch strings.ToUpper(value) {
		case STATUS:
			if wpr.Name == MASTER || wpr.Name == SENDER {
				break
			}

			err = Wpr.SendToService(strings.ToUpper(sender), LAUNCHED)
			if err != nil {
				sl.L.Warning("[%s] %s", wpr.Name, err.Error())
				return
			}
			sender = ""
			value = ""
			return
		}
	default:
		sender = raw[0]
		value = strings.Join(raw[1:], SEPARATOR)
		sl.L.Debug("[%s] Got from %s value: %s", wpr.Name, sender, value)
	}
	return
}

func (wpr *Wrapper) SendToService(channelName, value string) (err error) {
	ctx := context.Background()
	channelName = strings.ToUpper(channelName)
	sl.L.Debug("[%s] Send to service %s: %s", wpr.Name, channelName, value)

	int64Now := ciutils.TimeToInt64(ciutils.Now())
	// sl.L.Debug("[%s] now: %v; NextTry: %v", wpr.Name,
	// 	ciutils.TimeToStringInFormat(ciutils.Int64ToTime(int64Now), "15:04:05"),
	// 	ciutils.TimeToStringInFormat(ciutils.Int64ToTime(wpr.NextTry[channelName]), "15:04:05"))

	if wpr.NextTry[channelName] > int64Now {
		//err = fmt.Errorf("[%s] Too mutch error Send to %s, wait to next available try", wpr.Name, channelName)
		err = fmt.Errorf("%s", JUST_WAIT)
		//sl.L.Debug("[%s] %s", wpr.Name, err.Error())
		return
	}

	err = wpr.RClient.Publish(ctx, channelName, wpr.Name+SEPARATOR+value).Err()
	if err != nil {
		sl.L.Debug("[%s] Error Publish:%s", wpr.Name, err.Error())
		if wpr.TimeDelay[channelName] == 0 {
			wpr.TimeDelay[channelName] = 1
		}
		wpr.TimeDelay[channelName] = wpr.TimeDelay[channelName] * 2
		//sl.L.Debug("[%s] TimeDelay: %v; NextTry: %v", wpr.Name, wpr.TimeDelay[channelName], wpr.NextTry[channelName])
		wpr.NextTry[channelName] = ciutils.TimeToInt64(ciutils.Now().Add(time.Duration(wpr.TimeDelay[channelName]) * time.Second))
		return
	}
	wpr.TimeDelay[channelName] = 1
	return
}

func (wpr *Wrapper) RadioKatListner(signal chan os.Signal) {
	var err error
	defer func() {
		if err != nil {
			sl.L.Alert("[%s] Shutdown RadioKat with err: %s", wpr.Name, err.Error())
		}
		wpr.SendToService(MASTER, STOPPED)
	}()

	for {
		select {
		case <-signal:
			sl.L.Alert("[%s] Got cooperative shutdown signal (SIGUSR1)", Wpr.Name)
			wpr.RegularStop()
			close(Wpr.StopChan)
			time.Sleep(5 * time.Second)
			os.Exit(0)
		case <-wpr.StopChan:
			sl.L.Debug("[%s] RadioKat stopped from StopChannel", wpr.Name)
			wpr.RegularStop()
			return
		default:
			var sender, value string
			_, sender, value, err = wpr.ReadGroup()
			if err != nil {
				if err.Error() != JUST_WAIT {
					sl.L.Warning(err.Error())
				}
				time.Sleep(1 * time.Second)
				continue
			}

			sl.L.Debug("[%s] sender: %s value: %s", wpr.Name, sender, value)
			if RadioKat != nil {
				RadioKat(sender, value)
			}
		}
	}
}
