package dispatcher

import (
	"context"
	"fmt"
	"time"

	sl "github.com/Averianov/cisystemlog"
)

var L *sl.Logs

type Daemon string
type Status rune

const (
	STOP           Status        = 's'
	RUN            Status        = 'r'
	CHECK_DURATION time.Duration = time.Second * 3
)

type Dispatcher struct {
	Tasks          map[Daemon]*Task
	Variables      map[Daemon]chan interface{}
	Error          chan error
	CheckDureation time.Duration
}

// CreateDispatcher make dispatcher object where cd is duration for check tasks
func CreateDispatcher(l *sl.Logs, cd time.Duration) (d *Dispatcher) {
	if l == nil {
		L = sl.CreateLogs(false, 5)
	} else {
		L = l
	}

	d = &Dispatcher{
		Tasks: map[Daemon]*Task{},
	}
	if cd == 0 {
		d.CheckDureation = CHECK_DURATION
	}
	L.Info(CreateDispatcher, "CreateDispatcher")
	return
}

func (d *Dispatcher) Start() (err error) {
	defer func() {
		err = fmt.Errorf("Dispatcher was down")
	}()
	L.Info(d.Start, "start Dispatcher")
	timeToCheck := true
	tick := time.NewTicker(d.CheckDureation)

	for {
		select {
		case <-tick.C: // wait time to one check
			timeToCheck = true

		case err := <-d.Error: // if end or crash service
			if err != nil {
				L.Alert(d.Start, "%v", err)
			}
		default:
			if timeToCheck {
				for _, task := range d.Tasks {
					if task.Current != task.Must {
						L.Info(d.Start, "task %s need action; Must: %s; Current: %s", task.Name, string(task.Must), string(task.Current))

						if task.Must == RUN {
							L.Info(d.Start, "task %s; Up service", task.Name)
							readyToStart := true
							if task.Current == STOP {
								for _, req := range task.Required { // try start required services
									if req.Must == STOP {
										req.Start()
									}
									if req.Current == STOP { // wait when started each required services
										readyToStart = false
									}
								}

								if readyToStart {
									if task.Service != nil {
										task.Locker.Lock()
										task.Ctx, task.Cancel = context.WithCancel(context.Background())
										task.Locker.Unlock()
										L.Info(d.Start, "launched task %s", task.Name)
										go task.ServiceTemplate()
									} else {
										L.Info(d.Start, "service %s not available", task.Name)
									}
								}
							}
						}

						if task.Must == STOP {
							L.Info(d.Start, "task %s; Down service", task.Name)
							task.Cancel()
						}

					}
				}
				timeToCheck = false // for exclude many check trying
			} else {
				time.Sleep(time.Second)
			}
		}
	}
}

func (d *Dispatcher) AddTask(name Daemon, must Status, service func(context.Context, ...interface{}) error, val ...interface{}) (t *Task) {
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
