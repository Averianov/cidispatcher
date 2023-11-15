package dispatcher

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	sl "github.com/Averianov/cisystemlog"
)

var L *sl.Logs

type Daemon string

//type Status bool

const (
	// STOP           bool          = false
	// RUN            bool          = true
	CHECK_DURATION time.Duration = 3
)

type Dispatcher struct {
	Locker         sync.Mutex
	Tasks          map[Daemon]*Task
	Variables      map[Daemon]chan interface{}
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
	d.CheckDureation = time.Second * cd

	L.Info(CreateDispatcher, "CreateDispatcher")
	return
}

func (d *Dispatcher) Checking() (err error) {
	defer func() {
		if err == nil {
			err = fmt.Errorf("Dispatcher was down")
		}
	}()
	L.Info(d.Checking, "start Dispatcher")
	timeToCheck := true
	tick := time.NewTicker(d.CheckDureation)

	var stdIn chan string
	var line string
	go func() {
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			stdIn <- s.Text()
		}
		stdIn <- s.Err().Error()
	}()
	for {
		select {
		case line = <-stdIn:
			fmt.Printf("read line: %s-\n", line)
			if line == "exit" {
				err = fmt.Errorf("Gracefull shutdown application")
				return
			}

		case <-tick.C:
			// L.Debug(d.Checking, "wait time to one check - %v", d.CheckDureation)
			timeToCheck = true
			break

		// case err := <-d.Error: // if end or crash service
		// 	if err != nil {
		// 		L.Alert(d.Checking, "%v", err)
		// 	}
		default:
			if timeToCheck {
				for _, task := range d.Tasks {
					// L.Debug(d.Checking, "check task %s", task.Name)
					readyToStart := true
					for _, req := range task.Required {
						if task.StLaunched { // if required task down or removed, then stop this task
							existRequiredTask := false
							for _, t := range d.Tasks {
								if req.Name == t.Name {
									existRequiredTask = true
								}
							}

							if !existRequiredTask {
								L.Alert(d.Checking, "required task %s is not exist. Stop task %s", req.Name, task.Name)
								task.Cancel()
							}
						}
						if task.StMustStart {
							if !req.StMustStart {
								req.Start() // try start required services
							}
							if !req.StLaunched {
								readyToStart = false
								if task.StLaunched {
									task.Cancel() // wait when started each required services
								}
							}
						}
					}

					if task.StLaunched != task.StMustStart { // if needs any actions
						L.Info(d.Checking, "task %s need action; Must: %v; InProgress %v; Current: %v",
							task.Name, task.StMustStart, task.StInProgress, task.StLaunched)
						if task.StMustStart { // if must start
							L.Info(d.Checking, "task %s; Try Up service", task.Name)
							if task.StInProgress {
								L.Info(d.Checking, "task %s; Starting in progress", task.Name)
								continue
							}

							if readyToStart {
								if task.Service != nil {
									task.Locker.Lock()
									task.StInProgress = true
									task.Ctx, task.Cancel = context.WithCancel(context.Background())
									task.Locker.Unlock()
									L.Info(d.Checking, "launch task %s", task.Name)
									go task.ServiceTemplate()
								} else {
									L.Info(d.Checking, "service in task %s is not available", task.Name)
								}
							}
						}

						if !task.StMustStart { // if must stop
							L.Info(d.Checking, "task %s; Try Down service", task.Name)
							if task.StInProgress {
								L.Info(d.Checking, "task %s; Stopping in progress", task.Name)
								continue
							}
							if task.StLaunched {
								task.Locker.Lock()
								task.StInProgress = true
								task.Locker.Unlock()
								L.Info(d.Checking, "down task %s", task.Name)
								if task.StopFunc != nil {
									task.StopFunc()
								}
								task.Cancel()
							}
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

func (d *Dispatcher) AddTask(name Daemon, mustStart bool, service func(*Task) error, required []*Task, val ...interface{}) (t *Task) {
	if _, ok := d.Tasks[name]; ok {
		L.Alert(d.AddTask, "Task with current name is available")
		return
	}
	t = CreateTask(name, mustStart, service)
	t.Val = val
	t.Required = required
	d.Locker.Lock()
	d.Tasks[t.Name] = t
	d.Locker.Unlock()
	return
}

func (d *Dispatcher) RemoveTask(t *Task) {
	L.Info(d.Checking, "task %s; try remove", t.Name)

	for _, task := range d.Tasks {
		for _, req := range task.Required {
			if t == req {
				L.Warning(d.Checking, "Cannot delete task %s because it required for other task", t.Name)
				return
			}
		}
	}

	for {
		if t.StLaunched == false {
			d.Locker.Lock()
			delete(d.Tasks, t.Name)
			d.Locker.Unlock()
			L.Info(d.Checking, "task %s; deleted", t.Name)
			return
		} else if t.StMustStart == true {
			t.Stop()
		}
		time.Sleep(time.Second)
	}
}
