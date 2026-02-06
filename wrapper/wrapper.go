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
	NAME                 string = "NAME"
	MASTER               string = "MASTER"
	LOG_LEVEL            string = "LOGLEVEL"
	SIZE_LOG_FILE        string = "SIZE_LOG_FILE"
	PORT_FILE_PATH       string = "./port"
	DEFAULT_TRYING_COUNT int    = 3
)

var (
	Wpr *Wrapper
)

type Wrapper struct {
	Name     string
	RClient  *redis.Client
	PubSub   *redis.PubSub
	StopChan chan struct{}
	Env      map[string]string
}

func CreateWrapper(name string, logLevel int32, sizeLogFile int64) (wpr *Wrapper, err error) {
	if name == "" {
		err = fmt.Errorf("%s", "missing service name")
		return
	}
	name = strings.ToUpper(name)
	//fmt.Printf("wpr name: '%s'\n", name)

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
		val, ok := os.LookupEnv(LOG_LEVEL)
		if !ok {
			sizeLogFile = 0
		} else {
			sizeLogFile = ciutils.StrToInt64(val)
		}
	}

	sl.CreateLogs(name, "./log/", logLevel, sizeLogFile)

	Wpr = &Wrapper{
		StopChan: make(chan struct{}),
		Env:      make(map[string]string),
	}

	// Recheck nameing task in process as in dispatcher
	if name != MASTER && len(os.Environ()) > 0 {
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

	if Wpr.Name != MASTER {
		err = Wpr.sendToMaster("launched "+Wpr.Name, DEFAULT_TRYING_COUNT)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	signal.Notify(sig, syscall.SIGUSR1) // for cooperative shutdown
	go func() {
		<-sig
		sl.L.Alert("[%s] Cooperative shutdown (SIGUSR1)", Wpr.Name)
		close(Wpr.StopChan)
		time.Sleep(5 * time.Second)
		os.Exit(0)
		//panic(fmt.Sprintf("[%s] Cooperative shutdown (SIGUSR1)", Wpr.Name)) // temporary ??
	}()
	return Wpr, err
}

func (wpr *Wrapper) RegularStop() {
	Wpr.sendToMaster("stopped "+Wpr.Name, DEFAULT_TRYING_COUNT)
	Wpr.PubSub.Close()
}

func (wpr *Wrapper) StartService(serviceName string) (err error) {
	err = wpr.sendToMaster("start "+serviceName, DEFAULT_TRYING_COUNT)
	if err != nil {
		sl.L.Warning("[%s] %s", Wpr.Name, err.Error())
	}
	return
}

func (wpr *Wrapper) StopService(serviceName string) (err error) {
	err = wpr.sendToMaster("stop "+serviceName, DEFAULT_TRYING_COUNT)
	if err != nil {
		sl.L.Warning("[%s] %s", Wpr.Name, err.Error())
	}
	return
}

func (wpr *Wrapper) ReadGroup(tryCount int) (channal, msg string, err error) {
	var rmsg *redis.Message
	rmsg, err = wpr.PubSub.ReceiveMessage(context.Background())
	if err != nil {
		if tryCount > 0 {
			tryCount--
			time.Sleep(500 * time.Millisecond) // 0.5 sec
			return wpr.ReadGroup(tryCount)
		}
		sl.L.Warning("[%s] %s", wpr.Name, err.Error())
		return
	}

	//sl.L.Debug("[%s] GOT RAW %v", wpr.Name, rmsg)
	if raw := strings.Split(rmsg.Payload, " "); len(raw) > 1 {
		switch raw[0] {
		case "get":
			switch raw[1] {
			case "status":
				sl.L.Debug("[%s] Got: %s", wpr.Name, raw)
				err = Wpr.sendToMaster("launched "+Wpr.Name, DEFAULT_TRYING_COUNT)
				if err != nil {
					sl.L.Warning("[%s] %s", wpr.Name, err.Error())
					return
				}
				return
			}
		}
	}

	channal = rmsg.Channel
	msg = rmsg.Payload
	return
}

func (wpr *Wrapper) SendToService(serviceName, value string) (err error) {
	ctx := context.Background()
	serviceName = strings.ToUpper(serviceName)
	sl.L.Debug("[%s] Send to service %s: %s", wpr.Name, serviceName, value)
	err = wpr.RClient.Publish(ctx, serviceName, value).Err()
	if err != nil {
		sl.L.Warning("[%s] Error Publish:%s", wpr.Name, err.Error())
		time.Sleep(1 * time.Second)
	}
	return
}

func (wpr *Wrapper) sendToMaster(value string, tryCount int) (err error) {
	sl.L.Debug("[%s] Send to Master %s, try count: %d", wpr.Name, value, tryCount)
	err = wpr.SendToService(MASTER, value)
	if err != nil && tryCount > 0 {
		tryCount--
		time.Sleep(500 * time.Millisecond) // 0.5 sec
		err = wpr.sendToMaster(value, tryCount)
	}
	return
}
