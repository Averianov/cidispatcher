package dispatcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/Averianov/cidispatcher/wrapper"
	sl "github.com/Averianov/cisystemlog"
)

const SYS_MEMFD_CREATE = 319 // Only for Linux x86_64
const KILLING_ATTEMPT int = 3

type Task struct {
	sync.Mutex
	Ctx          context.Context
	Cancel       context.CancelFunc
	Name         string
	ElfPayload   []byte
	StMustStart  bool
	StInProgress bool
	StLaunched   bool
	Required     []string
	Cmd          *exec.Cmd
	Reminder     int
	Wpr          *wrapper.Wrapper
	Env          []string
}

func (task *Task) LaunchInMemory(args []string) (err error) {

	namePtr, err := syscall.BytePtrFromString(task.Name)
	if err != nil {
		sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		return
	}

	fd, _, errno := syscall.Syscall(SYS_MEMFD_CREATE, uintptr(unsafe.Pointer(namePtr)), 1, 0)
	if err != nil {
		sl.L.Warning("[task] err: memfd %s create failed: %w", task.Name, errno)
		return
	}

	file := os.NewFile(fd, task.Name)
	defer file.Close()

	if len(task.ElfPayload) < 4 {
		sl.L.Warning("[task] broken elf file: len=%d", len(task.ElfPayload))
		return
	}

	_, err = file.Write(task.ElfPayload)
	if err != nil {
		sl.L.Warning("[task] err: failed to write %s to memfd: %s", task.Name, err.Error())
		return
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		return
	}

	_, err = task.Check()
	if err == nil {
		sl.L.Warning("[task] %s exist; skip launch", task.Name)
		return
	}

	task.Ctx, task.Cancel = context.WithCancel(context.Background())
	path := fmt.Sprintf("/proc/self/fd/%d", fd)
	sl.L.Info("[task] Up %s by address %s %s", task.Name, path, args)
	task.Cmd = exec.CommandContext(task.Ctx, path, args...)
	//task.Cmd := exec.Command(path, args...)
	//task.Cmd.ExtraFiles = []*os.File{file}
	task.Cmd.Stdout = os.Stdout
	task.Cmd.Stderr = os.Stderr
	task.Cmd.Stdin = os.Stdin
	task.Cmd.Env = append(task.Cmd.Env, task.Env...)

	if len(task.ElfPayload) < 4 || string(task.ElfPayload[:4]) != "\x7fELF" {
		sl.L.Warning("[task] payload is not a valid ELF (magic bytes missing)")
	}
	err = task.Cmd.Start()
	if err != nil {
		sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		err = syscall.Close(int(fd))
		if err != nil {
			sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		}
		return
	}

	go func() {
		err := task.Cmd.Wait() // Auto "get" process, when die
		if err != nil {
			sl.L.Warning("[task] %s process finished with error: %s", task.Name, err)
		} else {
			sl.L.Info("[task] %s process finished successfully", task.Name)
		}
		task.Cmd = nil
		task.Stopped()
	}()

	task.Lock()
	task.StInProgress = true
	task.Unlock()

	sl.L.Debug("[task] %s got pid %d", task.Name, task.Cmd.Process.Pid)
	return
}

// Check task as runned
func (task *Task) Check() (launched *os.Process, err error) {
	if task.Cmd == nil {
		err = fmt.Errorf("[task] %s not launched", task.Name)
		sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		task.Stopped()
		return
	}

	launched, err = os.FindProcess(task.Cmd.Process.Pid)
	if err != nil {
		sl.L.Warning("[task] process by pid %d not found; err: %s", task.Cmd.Process.Pid, err.Error())
		task.Stopped()
		return
	}

	err = task.Wpr.SendToService(task.Name, "get status")
	if err != nil {
		sl.L.Warning("[master] %s err: %s ", task.Name, err.Error())
		return
	}

	sl.L.Info("[task] %s exist by pid %d", task.Name, task.Cmd.Process.Pid)
	return
}

// Stop signal to task to default stopping proccess
func (task *Task) Stop() (err error) {
	var process *os.Process
	process, err = task.Check()
	if process == nil && err != nil {
		sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		return
	}

	switch task.Reminder {
	case 0:
		sl.L.Info("[task] try stop %s by pid %d; reminder No%d", task.Name, task.Cmd.Process.Pid, task.Reminder)
		task.Cancel()
		err = process.Signal(syscall.SIGTERM)
		if err != nil {
			sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		}
	case KILLING_ATTEMPT:
		sl.L.Info("[task] try kill %s by pid %d; reminder No%d", task.Name, task.Cmd.Process.Pid, task.Reminder)
		task.Kill(process)
		if err != nil {
			sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		}
	}

	task.Reminder++
	sl.L.Info("[task] %s reminder No%d before killed", task.Name, task.Reminder)
	return
}

// Kill task by pid
func (task *Task) Kill(process *os.Process) (err error) {
	err = process.Signal(syscall.SIGKILL)
	if err != nil {
		sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
		if strings.Contains(err.Error(), "os: process already finished") {
			sl.L.Debug("[task] %s start cmd.Wait for pid %d", task.Name, task.Cmd.Process.Pid)
			//go task.Cmd.Wait()
		} else {
			parts := strings.Split(task.Cmd.Path, "/")
			fdStr := parts[len(parts)-1]
			fd, err := strconv.Atoi(fdStr)
			err = syscall.Close(int(fd))
			if err != nil {
				sl.L.Warning("[task] %s err: %s ", task.Name, err.Error())
			}
		}
		return
	}

	sl.L.Info("[task] %s try killed by pid %d", task.Name, task.Cmd.Process.Pid)
	task.Stopped()
	return
}

// Enable mark task to staring proccess
func (task *Task) Enable() {
	if task.StMustStart {
		if task.StLaunched {
			sl.L.Debug("[task] %s already runned", task.Name)
		} else {
			sl.L.Info("[task] starting %s in progress...", task.Name)
		}
		return
	}

	task.Lock()
	task.StMustStart = true
	task.Unlock()

	sl.L.Info("[task] %s - enabled", task.Name)
}

// Disable mark task to stopping proccess
func (task *Task) Disable() (err error) {
	if !task.StMustStart {
		if !task.StLaunched {
			sl.L.Debug("[task] %s already stopped", task.Name)
		} else {
			sl.L.Info("[task] stopping %s in progress...", task.Name)
		}
		return
	}

	task.Lock()
	task.StMustStart = false
	task.StInProgress = true
	task.Unlock()
	sl.L.Info("[task] %s - disabled", task.Name)
	return
}

// Started mark task as started
func (task *Task) Started() {
	if task.StLaunched {
		sl.L.Debug("[task] %s already started", task.Name)
		return
	}

	task.Lock()
	task.StInProgress = false
	task.StLaunched = true
	task.Reminder = 0
	task.Unlock()
	sl.L.Info("[task] %s started", task.Name)
}

// Stopped mark task as stopped
func (task *Task) Stopped() {
	if !task.StLaunched {
		sl.L.Debug("[task] %s already stopped", task.Name)
		return
	}

	task.Lock()
	task.StInProgress = false
	task.StLaunched = false
	task.Reminder = 0
	task.Unlock()
	sl.L.Info("[task] %s stopped", task.Name)
}

// func runForWindows() {
// 	var childBinary []byte
// 	tempFile := filepath.Join(os.TempDir(), "internal_module.exe")
// 	os.WriteFile(tempFile, childBinary, 0755)

// 	cmd := exec.Command(tempFile)
// 	cmd.Run()

// 	os.Remove(tempFile) // Remove after work
// }
