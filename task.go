package dispatcher

import (
	"context"
	"fmt"
	"sync"
)

type Task struct {
	Locker   sync.Mutex
	Ctx      context.Context
	Cancel   context.CancelFunc
	Name     Daemon
	Service  func(*Task) error
	Error    chan error // if crash service			-> try return to status RUN
	Must     Status     // for check differents status
	Current  Status     // for check differents status
	Required []*Task
	Val      []interface{}
}

func CreateTask(name Daemon, must Status, errChanel chan error, service func(*Task) error) (t *Task) {
	t = &Task{
		Name:    name,
		Service: service,
		Error:   errChanel,
		Must:    must,
		Current: STOP,
	}
	L.Info(CreateTask, "CreateTask '%s'", t.Name)
	return
}

func (task *Task) ServiceTemplate() {
	var err error
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			L.Alert(task.ServiceTemplate, "Critical error in service %s: %v", task.Name, recoverErr)
			err = fmt.Errorf("%v", recoverErr)
		}
		L.Info(task.Start, "defer in %s with err %v", task.Name, err)
		task.Stopped()
	}()

	//task.Error <- task.Service(task.Ctx, task.Val)
	err = task.Service(task)
	//L.Info(task.Start, "end in %s with err %v", task.Name, err)
}

func (task *Task) Start() {
	if task.Must == RUN {
		if task.Current == RUN {
			L.Info(task.Start, "service %s already runned", task.Name)
		} else {
			L.Info(task.Start, "starting service %s in progress...", task.Name)
		}
		return
	}

	task.Locker.Lock()
	task.Must = RUN
	task.Locker.Unlock()

	L.Info(task.Start, "start task %s", task.Name)
}

func (task *Task) Started() {
	task.Locker.Lock()
	task.Current = RUN
	task.Locker.Unlock()
	L.Info(task.Start, "task %s started", task.Name)
}

func (task *Task) Stop() {
	if task.Must == STOP {
		if task.Current == STOP {
			L.Info(task.Stop, "service %s already stopped", task.Name)
		} else {
			L.Info(task.Stop, "stopping service %s in progress...", task.Name)
		}
		return
	}

	task.Locker.Lock()
	task.Must = STOP
	task.Locker.Unlock()

	L.Info(task.Stop, "stop task %s", task.Name)
	return
}

func (task *Task) Stopped() {
	task.Locker.Lock()
	task.Current = STOP
	task.Locker.Unlock()
	L.Info(task.Start, "task %s stopped", task.Name)
}
