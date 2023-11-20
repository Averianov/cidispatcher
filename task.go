package dispatcher

import (
	"context"
	"sync"
)

type Task struct {
	Locker       sync.Mutex
	Name         Daemon
	Ctx          context.Context
	Cancel       context.CancelFunc
	StopFunc     func()
	Service      func(*Task) error
	Error        chan error // if crash service			-> try return to status true
	StMustStart  bool       // for check differents status
	StInProgress bool       // for check when daemon starting
	StLaunched   bool       // for check differents status
	Required     []*Task
	Val          []interface{}
}

func CreateTask(name Daemon, must bool, service func(*Task) error) (t *Task) {
	t = &Task{
		Name:         name,
		Service:      service,
		StMustStart:  must,
		StInProgress: false,
		StLaunched:   false,
	}
	L.Info("CreateTask '%s'", t.Name)
	return
}

func (task *Task) ServiceTemplate() {
	var err error
	defer func() {
		// if recoverErr := recover(); recoverErr != nil {
		// 	L.Alert("Critical error in %s: %v", task.Name, recoverErr)
		// 	err = fmt.Errorf("%v", recoverErr)
		// } else if err != nil {
		// 	L.Warning("defer in %s with err: %v", task.Name, err)
		// }
		L.Warning("defer in %s with err: %v", task.Name, err)
		task.Stopped()
	}()

	//task.Error <- task.Service(task.Ctx, task.Val)
	err = task.Service(task)
	//L.Info("end in %s with err %v", task.Name, err)
}

func (task *Task) Start() {
	if task.StMustStart == true {
		if task.StLaunched == true {
			L.Info("service %s already runned", task.Name)
		} else {
			L.Info("starting service %s in progress...", task.Name)
		}
		return
	}

	task.Locker.Lock()
	task.StMustStart = true
	task.Locker.Unlock()

	L.Info("start task %s", task.Name)
}

func (task *Task) Started() {
	task.Locker.Lock()
	task.StInProgress = false
	task.StLaunched = true
	task.Locker.Unlock()
	L.Info("task %s started", task.Name)
}

func (task *Task) Stop() {
	if task.StMustStart == false {
		if task.StLaunched == false {
			L.Info("service %s already stopped", task.Name)
		} else {
			L.Info("stopping service %s in progress...", task.Name)
		}
		return
	}

	task.Locker.Lock()
	task.StMustStart = false
	task.Locker.Unlock()

	L.Info("stop task %s", task.Name)
	return
}

func (task *Task) Stopped() {
	task.Locker.Lock()
	task.StInProgress = false
	task.StLaunched = false
	task.Locker.Unlock()
	L.Info("task %s stopped", task.Name)
}
