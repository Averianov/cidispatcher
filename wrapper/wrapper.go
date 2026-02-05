package wrapper

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"time"

	sl "github.com/Averianov/cisystemlog"
	"github.com/redis/go-redis/v9"
)

const (
	MASTER             			  string = "master"
	DEFAULT_JSON_CONFIG_FILE_PATH string = "./config.json"
	PORT_FILE_PATH				  string = "./port"
	DEFAULT_TRYING_COUNT 		  int    = 3
)

var (
	Wpr *Wrapper
)

type Wrapper struct {
	Name      string
	RClient   *redis.Client
	PubSub    *redis.PubSub
	StopChan  chan struct{}
}

func CreateWrapper(name string) (wpr *Wrapper, err error) {
	if name == "" || name == " "{
		err = fmt.Errorf("%s", "missing service name")
		return
	}

	// check nameings task in process and in dispatcher
	if name != MASTER {
		val, exists := os.LookupEnv("NAME")
		if !exists {
			err = fmt.Errorf("service with name \"%s\" not equal naming with started process \"%s\"", name, val)
			return
		}
	}

	sl.CreateLogs(name, "./log/", 4, 2)

	var raw []byte
	raw, err = os.ReadFile(PORT_FILE_PATH)
	if err != nil {
		sl.L.Warning("[%s] %s", name, err.Error())
		return
	}

	rport := string(raw)
	sl.L.Info("[%s] Service started at %s", name, string(rport))
	Wpr = &Wrapper{
		Name:      name,
		StopChan:  make(chan struct{}),
	}

	sl.L.Debug("[%s] connect to Redis on: %s", name, rport)
	Wpr.RClient = redis.NewClient(&redis.Options{
		Addr:             "localhost:" + rport,
		ReadTimeout:      -1, // Отключает сетевой таймаут на чтение
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
		time.Sleep(5*time.Second)
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
