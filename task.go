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
	StopFunc     func()            // user stop function. used when context not available in task
	Service      func(*Task) error // task function
	StMustStart  bool              // for check differents status
	StInProgress bool              // for check when daemon starting
	StLaunched   bool              // for check differents status
	Required     []*Task
}

// CreateTask retun new task
func CreateTask(name Daemon, must bool, service func(*Task) error) (t *Task) {
	t = &Task{
		Name:         name,
		Service:      service,
		StMustStart:  must,
		StInProgress: false,
		StLaunched:   false,
	}
	L.Debug("created Task '%s'", name)
	return
}

// ServiceTemplate is wrapper to task function.\n
// Execut start task function and mark task when they stopped.
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
	err = task.Service(task)
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

// Started mark task as started.\n
// Used from task function! User check when it must doing.
func (task *Task) Started() {
	task.Locker.Lock()
	task.StInProgress = false
	task.StLaunched = true
	task.Locker.Unlock()
	L.Info("task %s started", task.Name)
}

// Stop mark task to start stopping proccess
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

// Stopped mark task as stopped
func (task *Task) Stopped() {
	task.Locker.Lock()
	task.StInProgress = false
	task.StLaunched = false
	task.Locker.Unlock()
	L.Info("task %s stopped", task.Name)
}
