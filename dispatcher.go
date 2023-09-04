package dispatcher

import (
	"time"

	sl "github.com/Averianov/cisystemlog"
)

var L *sl.Logs

type Status rune

const (
	STOP       Status        = 's'
	RUN        Status        = 'r'
	CHECK_TASK time.Duration = time.Second * 3
)

type Dispatcher struct {
	Tasks     map[string]*Task
	Variables map[string]chan interface{}
}

func CreateDispatcher() (d *Dispatcher) {
	L = sl.CreateLogs(false, 10)
	go L.StartLoggerAgent(make(chan string))

	d = &Dispatcher{
		Tasks: map[string]*Task{},
	}
	L.Info(CreateDispatcher, "CreateDispatcher")
	return
}

func (d *Dispatcher) AddTask(name string, must Status, service func(chan struct{}, chan error, ...interface{})) (t *Task) {
	t = CreateTask(name, must, service)
	d.Tasks[t.Name] = t
	return
}

func (d *Dispatcher) RemoveTask(t *Task) (err error) {
	d.Tasks[t.Name].Stop()
	delete(d.Tasks, t.Name)
	t = nil
	return
}
