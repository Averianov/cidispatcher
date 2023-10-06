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
	Service  func(context.Context, ...interface{}) error
	Error    chan error // if crash service			-> try return to status RUN
	Must     Status     // for check differents status
	Current  Status     // for check differents status
	Required []*Task
	Val      []interface{}
}

func CreateTask(name Daemon, must Status, errChanel chan error, service func(context.Context, ...interface{}) error) (t *Task) {
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
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			L.Alert(task.ServiceTemplate, "Critical error in service %s: %v", task.Name, recoverErr)
			task.Error <- fmt.Errorf("%v", recoverErr)
		}
		//L.Info(task.Start, "defer in %s", task.Name)
		task.Locker.Lock()
		task.Current = STOP
		task.Locker.Unlock()
	}()
	task.Locker.Lock()
	task.Current = RUN
	task.Locker.Unlock()

	//task.Error <- task.Service(task.Ctx)
	err := task.Service(task.Ctx, task.Val)
	if err != nil {
		task.Error <- err
	}
	//L.Info(task.Start, "end in %s", task.Name)
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
