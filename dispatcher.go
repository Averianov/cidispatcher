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
var D *Dispatcher

type Daemon string

const (
	CHECK_DURATION time.Duration = 3 // in seconds
	STDIN_TASK     string        = "StdInTask"
)

type Dispatcher struct {
	Locker         sync.Mutex
	CheckDureation time.Duration
	Tasks          map[Daemon]*Task
}

// CreateDispatcher make dispatcher object where cd is duration for check tasks in seconds.
// If mother project not use cisystemlog, then dispatcher create self this log
// or mother project can present pointer to the cisystemlog
func CreateDispatcher(l *sl.Logs, cd time.Duration) (d *Dispatcher) {
	if l == nil {
		L = sl.CreateLogs(4, 5) // debug level; 5Mb in log file
		L.RemoveLogFile(3)
	} else {
		L = l
	}

	d = &Dispatcher{
		Tasks: map[Daemon]*Task{},
	}
	if cd == 0 {
		cd = CHECK_DURATION
	} else {
		d.CheckDureation = time.Second * cd
	}
	D = d
	L.Info("CreateDispatcher")
	return
}

// Checking execute checking tasks.
// main process
func (d *Dispatcher) Checking() (err error) {
	defer func() {
		if err == nil {
			err = fmt.Errorf("Dispatcher was down")
		}
	}()
	L.Info("start Dispatcher")

	//go d.StdIn()
	d.AddTask("STDIN_TASK", true, StdIn, []*Task{})

	var mustExit, timeToCheck, readyToStart bool = true, true, true
	tick := time.NewTicker(d.CheckDureation)
	for {
		select {
		case <-tick.C:
			timeToCheck = true

		default:
			if timeToCheck {
				//### check Gracefull shutdown application ##########
				mustExit = true
				for _, task := range d.Tasks {
					if task.StMustStart || task.StMustStart != task.StLaunched { // if not ready to shutdown - disable marker
						mustExit = false
						break
					}
				}
				if mustExit {
					L.Warning("Gracefull shutdown application")
					time.Sleep(time.Second * 3)
					os.Exit(0)
				}

				//### check Tasks #####################################
				for _, task := range d.Tasks {
					readyToStart = true
					for _, req := range task.Required {
						if task.StLaunched { // if required task down or removed, then stop this task
							existRequiredTask := false
							for _, t := range d.Tasks {
								if req.Name == t.Name {
									existRequiredTask = true
								}
							}

							if !existRequiredTask {
								L.Alert("required task %s is not exist. Stop task %s", req.Name, task.Name)
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
						L.Info("task %s need action; Must: %v; InProgress %v; Current: %v",
							task.Name, task.StMustStart, task.StInProgress, task.StLaunched)
						if task.StMustStart { // if must start
							L.Info("task %s; Try Up service", task.Name)
							if task.StInProgress {
								L.Info("task %s; Starting in progress", task.Name)
								continue
							}

							if readyToStart {
								if task.Service != nil {
									task.Locker.Lock()
									task.StInProgress = true
									task.Ctx, task.Cancel = context.WithCancel(context.Background())
									task.Locker.Unlock()
									L.Info("launch task %s", task.Name)
									go task.ServiceTemplate()
								} else {
									L.Info("service in task %s is not available", task.Name)
								}
							}
						}

						if !task.StMustStart { // if must stop
							L.Info("task %s; Try Down service", task.Name)
							for _, t := range d.Tasks {
								for _, rt := range t.Required {
									if rt.Name == task.Name {
										L.Info("stop child task %s who required main task %s", rt.Name, task.Name)
										t.Stop()
									}
								}
							}

							if task.StInProgress {
								L.Info("task %s; Stopping in progress", task.Name)
								continue
							}
							if task.StLaunched {
								task.Locker.Lock()
								task.StInProgress = true
								task.Locker.Unlock()
								L.Info("down task %s", task.Name)
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

// AddTask make new task and add it to dispatcher map.
// required: name, start status, task function and required tasks
func (d *Dispatcher) AddTask(name Daemon, mustStart bool, service func(*Task) error, required []*Task) (t *Task) {
	for _, rt := range required {
		if _, ok := d.Tasks[rt.Name]; !ok {
			L.Warning("required task is not available")
			return
		}
	}
	var ok bool
	if t, ok = d.Tasks[name]; ok {
		L.Warning("task %s is available", name)
		t.Locker.Lock()
		t.Service = service
		t.StMustStart = mustStart
		t.Required = required
		t.Locker.Unlock()
		return
	} else {
		t = CreateTask(name, mustStart, service)
		t.Required = required
		d.Locker.Lock()
		d.Tasks[t.Name] = t
		d.Locker.Unlock()
	}
	return
}

// RemoveTask removed single task if it free from dependecies
func (d *Dispatcher) RemoveTask(t *Task) (ok bool) {
	if t == nil {
		return false
	}
	L.Info("task %s; try remove", t.Name)

	for _, task := range d.Tasks {
		for _, req := range task.Required {
			if t == req {
				L.Warning("cannot remove task %s. It have depends", t.Name)
				return false
			}
		}
	}

	for {
		if !t.StLaunched {
			d.Locker.Lock()
			delete(d.Tasks, t.Name)
			d.Locker.Unlock()
			L.Info("task %s; deleted", t.Name)
			return true
		} else if t.StMustStart {
			t.Stop()
		}
		time.Sleep(time.Second)
	}
}

// RemoveTaskAndRequired removed current task and other tasks who required this task
func (d *Dispatcher) RemoveTaskAndRequired(t *Task) (ok bool) {
	if t == nil {
		return false
	}
	L.Info("task %s; try remove", t.Name)

	for _, task := range d.Tasks {
		for _, req := range task.Required {
			if t == req {
				if !d.RemoveTaskAndRequired(task) {
					return false
				}
			}
		}
	}

	for {
		if !t.StLaunched {
			d.Locker.Lock()
			delete(d.Tasks, t.Name)
			d.Locker.Unlock()
			L.Info("task %s; deleted", t.Name)
			return true
		} else if t.StMustStart {
			t.Stop()
		}
		time.Sleep(time.Second)
	}
}

// Stop execute gracefull shutdown application.
// Stopped all tasks
func (d *Dispatcher) Stop() {
	for _, task := range d.Tasks {
		task.Stop()
	}
}

// StdIn is default task who listen cmd input and execute user command.
// More command will be added
func StdIn(t *Task) (err error) {
	if D == nil {
		err = fmt.Errorf("Dispatcher is not available")
		return
	}
	t.Started()
	for {
		in := bufio.NewReader(os.Stdin)
		inpuText, _ := in.ReadString('\n')
		switch inpuText {
		case "exit\n":
			L.Warning("got request for exit")
			D.Stop()
			return
		case "tasks\n":
			L.Info("got request for tasks status")
			for _, task := range D.Tasks {
				var status string
				if task.StLaunched {
					if task.StMustStart != task.StLaunched {
						status = "Must Stopped"
					} else {
						status = "Launched"
					}
				} else {
					if task.StMustStart != task.StLaunched {
						status = "Must launched"
					} else {
						status = "Stopped"
					}
				}
				if task.StInProgress {
					status = status + "; In progress..."
				}
				L.Info("task %s - %s ", task.Name, status)
			}
		}
	}
}
