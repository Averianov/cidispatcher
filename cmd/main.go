//go:build linux

package main

import (
	dspr "github.com/Averianov/cidispatcher"
)

const(
	LOGGER string = "logger"
	WORKER1 string = "worker1"
	WORKER2 string = "worker2"
	WORKER3 string = "worker3"
)

// Example use dispatcher
func main() {
	dspr.ProcessConfigs[LOGGER] = dspr.ProcessConfig{LOGGER, false, []string{}}
	dspr.ProcessConfigs[WORKER1] = dspr.ProcessConfig{WORKER1, true, []string{LOGGER}}
	dspr.ProcessConfigs[WORKER2] = dspr.ProcessConfig{WORKER2, false, []string{LOGGER}}
	dspr.ProcessConfigs[WORKER3] = dspr.ProcessConfig{WORKER3, false, []string{LOGGER}}

	dspr.CreateDispatcher(0)
	dspr.D.Launch()
}
