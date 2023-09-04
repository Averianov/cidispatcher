package dispatcher

import (
	"sync"
	"time"
)

type Task struct {
	sync.Mutex
	Name     string
	Service  func(chan struct{}, chan error, ...interface{})
	Launched func(chan struct{}, chan error, ...interface{})
	Error    chan error    // if crash service			-> try return to status RUN
	Down     chan struct{} // for prity down service 	-> make must status to STOP
	Up       chan struct{} // for up service 			-> make must status to RUN
	Must     Status        // for check differents status
	Current  Status        // for check differents status
}

func CreateTask(name string, must Status, service func(chan struct{}, chan error, ...interface{})) (t *Task) {
	t = &Task{
		Name:     name,
		Service:  service,
		Launched: nil,
		Error:    make(chan error),
		Down:     make(chan struct{}),
		Up:       make(chan struct{}),
		Must:     must,
		Current:  STOP,
	}
	L.Info(CreateTask, "CreateTask '%s'", t.Name)
	return
}

func (task *Task) Tracker() {

	L.Info(task.Tracker, "start Tracker by task %s", task.Name)

	tick := time.NewTicker(CHECK_TASK)

	for {
		select {
		case <-tick.C:
			if task.Current != task.Must {
				L.Info(task.Tracker, "task %s need action; Must: %s; Current: %s", task.Name, string(task.Must), string(task.Current))
				if task.Must == RUN {
					if task.Launched == nil {
						L.Info(task.Tracker, "task %s launched: %v; Start", task.Name, task.Launched != nil)
						go task.Start()
					} else {
						L.Info(task.Tracker, "task %s launched: %v; Up", task.Name, task.Launched == nil)
						go func() { task.Up <- struct{}{} }()
					}
				}
				if task.Must == STOP {
					go task.Stop()
				}
			}

		case <-task.Up:
			L.Info(task.Tracker, "launched task %s", task.Name)
			go task.Launched(task.Down, task.Error)
			task.Lock()
			task.Current = RUN
			task.Unlock()

		case err := <-task.Error: // if end or crash service
			L.Alert(task.Tracker, "task down with error: %v", err)
			task.Lock()
			task.Current = STOP
			task.Unlock()
		}
	}
}

func (task *Task) Start() {
	if task.Current == RUN && task.Launched != nil {
		L.Info(task.Start, "Already runned")
		return
	}

	task.Lock()
	task.Must = RUN
	task.Launched = task.Service
	task.Unlock()

	task.Up <- struct{}{}
	L.Info(task.Start, "start task %s", task.Name)
}

func (task *Task) Stop() {

	L.Info(task.Start, "stop1 task %s", task.Name)
	task.Lock()
	task.Must = STOP
	task.Down <- struct{}{}
	task.Launched = nil
	task.Current = STOP
	task.Unlock()

	L.Info(task.Start, "stop task %s", task.Name)
	return
}

func (task *Task) UpdateService(service func(chan struct{}, chan error, ...interface{})) {
	task.Lock()
	task.Service = service
	task.Unlock()

	L.Info(task.Start, "update service in task %v", task.Name)
}
