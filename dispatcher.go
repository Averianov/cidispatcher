package dispatcher

import (
	"context"
	"fmt"
	"sync"
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
	Locker         sync.Mutex
	Tasks          map[Daemon]*Task
	Variables      map[Daemon]chan interface{}
	Error          chan error
	CheckDureation time.Duration
}

// CreateDispatcher make dispatcher object where cd is duration for check tasks in seconds
func CreateDispatcher(l *sl.Logs, cd time.Duration) (d *Dispatcher) {
	if l == nil {
		L = sl.CreateLogs(4, 5)
	} else {
		L = l
	}

	d = &Dispatcher{
		Tasks: map[Daemon]*Task{},
	}
	if cd == 0 {
		d.CheckDureation = CHECK_DURATION * time.Second
	} else {
		d.CheckDureation = cd * time.Second
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
		case <-tick.C:
			L.Debug(d.Start, "wait time to one check")
			timeToCheck = true

		case err := <-d.Error: // if end or crash service
			if err != nil {
				L.Alert(d.Start, "%v", err)
			}
		default:
			if timeToCheck {
				L.Debug(d.Start, "check tasks")
				for _, task := range d.Tasks {
					readyToStart := true
					for _, req := range task.Required {
						if task.Current == RUN { // if required task down or removed, then stop this task
							exist := false
							for _, t := range d.Tasks {
								if req.Name == t.Name {
									exist = true
								}
							}

							if !exist {
								L.Alert(d.Start, "required task %s is not exist. Stop task %s", req.Name, task.Name)
								task.Cancel()
							}
						}
						if task.Must == RUN {
							if req.Must == STOP {
								req.Start() // try start required services
							}
							if req.Current == STOP {
								readyToStart = false
								if task.Current == RUN {
									task.Cancel() // wait when started each required services
								}
							}
						}
					}

					if task.Current != task.Must {
						L.Info(d.Start, "task %s need action; Must: %s; Current: %s", task.Name, string(task.Must), string(task.Current))

						if task.Must == RUN {
							L.Info(d.Start, "task %s; try Up service", task.Name)
							if readyToStart && task.Current == STOP {
								if task.Service != nil {
									task.Locker.Lock()
									task.Current = RUN
									task.Ctx, task.Cancel = context.WithCancel(context.Background())
									task.Locker.Unlock()
									L.Info(d.Start, "launched task %s", task.Name)
									go task.ServiceTemplate()
								} else {
									L.Info(d.Start, "service %s not available", task.Name)
								}
							}

						}

						if task.Must == STOP && task.Current == RUN {
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

func (d *Dispatcher) AddTask(name Daemon, must Status, service func(context.Context, ...interface{}) error, required []*Task, val ...interface{}) (t *Task) {
	if _, ok := d.Tasks[name]; ok {
		L.Alert(d.AddTask, "Task with current name is available")
		return
	}
	t = CreateTask(name, must, d.Error, service)
	t.Val = val
	t.Required = required
	d.Locker.Lock()
	d.Tasks[t.Name] = t
	d.Locker.Unlock()
	return
}

func (d *Dispatcher) RemoveTask(t *Task) {
	L.Info(d.Start, "task %s; try remove", t.Name)

	for _, task := range d.Tasks {
		for _, req := range task.Required {
			if t == req {
				L.Warning(d.Start, "Cannot delete task %s because it required for other task", t.Name)
				return
			}
		}
	}

	for {
		if t.Current == STOP {
			d.Locker.Lock()
			delete(d.Tasks, t.Name)
			d.Locker.Unlock()
			L.Info(d.Start, "task %s; deleted", t.Name)
			return
		} else if t.Must == RUN {
			t.Stop()
		}
		time.Sleep(time.Second)
	}
}
