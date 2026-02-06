//go:build linux

package dispatcher

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
	"github.com/Averianov/ciutils"
	ftgc "github.com/Averianov/ftgc"
	"github.com/alicebob/miniredis/v2"
)

const (
	DEFAULT_CHECK_DURATION time.Duration = 10 // seconds
)

var (
	D              *Dispatcher
	ProcessConfigs map[string]ProcessConfig = make(map[string]ProcessConfig)
)

type ProcessConfig struct {
	Name      string            `json:"name"`
	MustStart bool              `json:"must_start"`
	Required  []string          `json:"required"`
	Env       map[string]string `json:"env"`
}

type Dispatcher struct {
	CheckDureation time.Duration
	Wpr            *wrapper.Wrapper
	Tasks          map[string]*Task
}

func CreateDispatcher(cd time.Duration, logLevel int32, sizeLogFile int64) (d *Dispatcher) {
	var err error

	if cd == 0 {
		cd = time.Second * DEFAULT_CHECK_DURATION
	} else {
		cd = time.Second * cd
	}

	D = new(Dispatcher)
	D.CheckDureation = cd
	D.Tasks = map[string]*Task{}

	var mr *miniredis.Miniredis
	mr, err = miniredis.Run()
	if err != nil {
		panic(fmt.Sprintf("[master] %s", err.Error()))
	}

	var f *os.File
	f, err = os.OpenFile(wrapper.PORT_FILE_PATH, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(fmt.Sprintf("[master] %s", err.Error()))
	}
	f.WriteString(mr.Port())
	f.Close()

	sl.L.Info("[master] Radis server up on %s", mr.Port())

	D.Wpr, err = wrapper.CreateWrapper(wrapper.MASTER, logLevel, sizeLogFile)
	if err != nil {
		panic(fmt.Sprintf("[master] %s", err.Error()))
	}

	// upload Payload Data
	// sl.L.Debug("[master] ToGo: %v\n", ftgc.ToGo) // static map with byte data from FileToGoConverter
	for _, pc := range ProcessConfigs {
		pc.Name = strings.ToUpper(pc.Name)
		if raw, ok := ftgc.ToGo[pc.Name]; ok { // name in map FileToGoConverter in uppercase; name in uppercase
			sl.L.Info("[master] add task %s", pc.Name)
			D.Tasks[pc.Name] = &Task{
				Name:        pc.Name,
				ElfPayload:  raw,
				StMustStart: pc.MustStart,
				Required:    []string{},
				Wpr:         D.Wpr,
			}
			for _, required := range pc.Required {
				D.Tasks[pc.Name].Required = append(D.Tasks[pc.Name].Required, strings.ToUpper(required))
			} 
			pc.Env[wrapper.NAME] = pc.Name
			pc.Env[wrapper.LOG_LEVEL] = ciutils.IntToStr(int(logLevel))
			pc.Env[wrapper.SIZE_LOG_FILE] = ciutils.Int64ToStr(sizeLogFile)
			for name, val := range pc.Env {
				D.Tasks[pc.Name].Env = append(D.Tasks[pc.Name].Env, fmt.Sprintf("%s=%s", strings.ToUpper(name), strings.ToUpper(val)))
				sl.L.Debug("[master] task %s - env %v", pc.Name, D.Tasks[pc.Name].Env)
			}
		} else {
			panic(fmt.Sprintf("[master] not found %s raw data\n", pc.Name))
		}
	}
	return D
}

func (d *Dispatcher) Launch() {
	defer D.Wpr.RegularStop()
	defer os.Remove(wrapper.PORT_FILE_PATH)

	go d.RadioKat()
	d.StatusChecker()
}

func (d *Dispatcher) RadioKat() {
	for {
		_, value, err := d.Wpr.ReadGroup(1)
		if err != nil {
			sl.L.Warning(err.Error())
			continue
		}
		sl.L.Debug("[master] GOT: %s", value)
		raw := strings.Split(value, " ")
		if len(raw) == 2 {
			if task, ok := d.Tasks[strings.ToUpper(raw[1])]; ok {
				switch raw[0] {
				case "launched":
					task.Started()
				case "stopped":
					task.Stopped()
				case "start":
					task.Enable()
				case "stop":
					sl.L.Alert("[master] start recurcive stopping tasks from %s", task.Name)
					d.RecurciveStop(task)
				default:
					sl.L.Debug("[master] get some: %s", value)
				}
			}
		}
	}
}

func (d *Dispatcher) RecurciveStop(task *Task) {
	task.Disable()
	task.Stop()
	for _, childrenTask := range D.Tasks { // potencial children task
		for _, mainTaskName := range childrenTask.Required {
			if task.Name == mainTaskName {
				sl.L.Info("[master] task %s - looping stop children task %s", task.Name, childrenTask.Name)
				d.RecurciveStop(childrenTask)
			}
		}
	}
}

func (d *Dispatcher) RecurciveEnable(task *Task) {
	task.Enable()
	for _, mainTaskName := range task.Required {
		if mainTask, ok := D.Tasks[mainTaskName]; ok && !mainTask.StMustStart {
			sl.L.Info("[master] task %s - looping enable main task %s", task.Name, mainTask.Name)
			d.RecurciveEnable(mainTask)
		}
	}
}

func (d *Dispatcher) ReadyToWork(task *Task) (ready bool) {
	for _, rq := range task.Required { // check available main tasks
		req := d.Tasks[rq]
		if req.StMustStart && req.StLaunched {
			continue
		}
		return false
	}
	sl.L.Debug("[master] task %s - ready to work", task.Name)
	return true
}

func (d *Dispatcher) StatusChecker() (err error) {
	defer func() {
		if err == nil {
			err = fmt.Errorf("[master] Dispatcher was down")
			sl.L.Warning(err.Error())
		}
	}()
	sl.L.Info("[master] start Dispatcher Checker")

	var timeToCheck bool = true
	tick := time.NewTicker(d.CheckDureation)
	for {
		select {
		// case <-d.Wpr.StopChan:
		// 	sl.L.Alert("[master] Get Cooperative shutdown signal. Shutdown all tasks")
		// 	for _, task := range d.Tasks {
		// 		task.StMustStart = false
		// 	}
		case <-tick.C:
			timeToCheck = true

		default:
			if timeToCheck {
				//### Tasks status before changes ####################################
				var msg string
				for _, task := range d.Tasks {
					msg = msg + fmt.Sprintf("				[master]  %s	(Must: %v;	InProgress %v;	Current: %v)\n",
						task.Name, task.StMustStart, task.StInProgress, task.StLaunched)
				}
				sl.L.Debug("[master] \n\n\n################################\n%s", msg)

				//### check Gracefull shutdown application ##########
				readyToExit := true
				for _, task := range d.Tasks {
					if task.StMustStart || task.StMustStart != task.StLaunched { // if not ready to shutdown - disable marker
						sl.L.Debug("[master] some tasks still in work; daemon not ready to exit")
						readyToExit = false
						break
					}

					var prcs *os.Process
					prcs, err = task.Check()
					if err == nil || (prcs != nil && err != nil) { // if process no started or process was frozen
						sl.L.Debug("[master] task %s - has launched process; daemon not ready to exit", task.Name)
						readyToExit = false
						break
					}
				}

				if readyToExit {
					sl.L.Warning("[master] Gracefull shutdown application")
					time.Sleep(time.Second * 3)
					os.Exit(0)
				}

				//### check Tasks #####################################
				for _, task := range d.Tasks {
					switch true {
					case task.StMustStart && task.StInProgress && task.StLaunched: // only check why still in progress
						sl.L.Debug("[master] task %s - still in starting progress; try shutdown zombie process", task.Name)
						err = task.Stop() // send reminders
						if err != nil {
							sl.L.Warning("[master] %s err: %s ", task.Name, err.Error())
						}
					case task.StMustStart && task.StInProgress && !task.StLaunched: // when still not started
						err = task.Stop() // send reminders
						if err != nil {
							sl.L.Warning("[master] %s err: %s ", task.Name, err.Error())
						}
					case task.StMustStart && !task.StInProgress && !task.StLaunched: // must started
						if d.ReadyToWork(task) {
							if task.ElfPayload == nil {
								sl.L.Alert("[master] task %s - service not available", task.Name)
								d.RecurciveStop(task)
								continue
							}
							sl.L.Info("[master] task %s - launch", task.Name)
							err = task.LaunchInMemory([]string{})
							if err != nil {
								sl.L.Warning("[master] %s err: %s ", task.Name, err.Error())
							}
						} else {
							d.RecurciveEnable(task) // enabling all main tasks
						}
					case task.StMustStart && !task.StInProgress && task.StLaunched: // successfull launched
						if !d.ReadyToWork(task) {
							d.RecurciveEnable(task)         // enabling all main tasks
							task.Reminder = KILLING_ATTEMPT // kill process who work without main processes
							err = task.Stop()
							if err != nil {
								sl.L.Warning("[master] %s err: %s ", task.Name, err.Error())
							}
						}
					case !task.StMustStart && !task.StInProgress && !task.StLaunched: // fully stopped
						continue
					case !task.StMustStart && !task.StInProgress && task.StLaunched: // must stopped
						sl.L.Debug("[master] task %s - try shutdown worked process", task.Name)
						err = task.Stop() // send reminders
						if err != nil {
							sl.L.Warning("[master] %s err: %s ", task.Name, err.Error())
						}
					case !task.StMustStart && task.StInProgress && task.StLaunched: // when still not stopped
						sl.L.Debug("[master] task %s - still in stopping progress; try shutdown zombie process", task.Name)
						err = task.Stop() // send reminders
						if err != nil {
							sl.L.Warning("[master] %s err: %s ", task.Name, err.Error())
						}
					case !task.StMustStart && task.StInProgress && !task.StLaunched: // only check why still in progress
						err = task.Stop() // send reminders
						if err != nil {
							sl.L.Warning("[master] %s err: %s ", task.Name, err.Error())
						}
					}
				}

				//### Tasks status after changes ####################################
				msg = ""
				for _, task := range d.Tasks {
					msg = msg + fmt.Sprintf("				[master]  %s	(Must: %v;	InProgress %v;	Current: %v)\n",
						task.Name, task.StMustStart, task.StInProgress, task.StLaunched)
				}
				sl.L.Debug("[master] \n%s\n################################\n\n\n", msg)

				timeToCheck = false // for exclude many check trying
			} else {
				time.Sleep(time.Second)
			}
		}
	}
}
