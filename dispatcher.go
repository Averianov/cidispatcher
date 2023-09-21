package dispatcher

import (
	"context"
	"time"

	sl "github.com/Averianov/cisystemlog"
)

var L *sl.Logs

type Status rune

const (
	STOP           Status        = 's'
	RUN            Status        = 'r'
	CHECK_DURATION time.Duration = time.Second * 3
)

type Dispatcher struct {
	Tasks          map[string]*Task
	Variables      map[string]chan interface{}
	Error          chan error
	CheckDureation time.Duration
}

func CreateDispatcher(cd time.Duration) (d *Dispatcher) {
	L = sl.CreateLogs(false, 10)
	go L.StartLoggerAgent(make(chan string))

	d = &Dispatcher{
		Tasks: map[string]*Task{},
	}
	if cd == 0 {
		d.CheckDureation = CHECK_DURATION
	}
	L.Info(CreateDispatcher, "CreateDispatcher")
	return
}

func (d *Dispatcher) Tracker() {
	L.Info(d.Tracker, "start Dispatcher")
	timeToCheck := true
	tick := time.NewTicker(d.CheckDureation)

	for {
		select {
		case <-tick.C: // wait time to one check
			timeToCheck = true

		case err := <-d.Error: // if end or crash service
			if err != nil {
				L.Alert(d.Tracker, "%v", err)
			}
		default:
			if timeToCheck {
				for _, task := range d.Tasks {
					if task.Current != task.Must {
						L.Info(d.Tracker, "task %s need action; Must: %s; Current: %s", task.Name, string(task.Must), string(task.Current))

						if task.Must == RUN {
							L.Info(d.Tracker, "task %s; Up service", task.Name)
							if task.Current == STOP {
								if task.Service != nil {
									task.Locker.Lock()
									task.Ctx, task.Cancel = context.WithCancel(context.Background())
									task.Locker.Unlock()
									L.Info(d.Tracker, "launched task %s", task.Name)
									go task.ServiceTemplate()
								} else {
									L.Info(d.Tracker, "service %s not available", task.Name)
								}
							}
						}

						if task.Must == STOP {
							L.Info(d.Tracker, "task %s; Down service", task.Name)
							task.Cancel()
						}

					}
				}
				timeToCheck = false
			} else {
				time.Sleep(time.Second)
			}
		}
	}
}

func (d *Dispatcher) AddTask(name string, must Status, service func(context.Context, ...interface{}) error) (t *Task) {
	t = CreateTask(name, must, d.Error, service)
	d.Tasks[t.Name] = t
	return
}

func (d *Dispatcher) RemoveTask(t *Task) {
	for {
		if t.Current == STOP {
			delete(d.Tasks, t.Name)
			return
		} else if t.Must == RUN {
			t.Stop()
		}
		time.Sleep(time.Second)
	}
}
