//go:build linux

package main

import (
	_ "github.com/Averianov/cidispatcher/build/memfd" // for upload Payloads (path from go.mod naming module + /build/memfd)

	dspr "github.com/Averianov/cidispatcher"
)

const (
	LOGGER  string = "logger"
	WORKER1 string = "worker1"
	WORKER2 string = "worker2"
	WORKER3 string = "worker3"
)

// Example use dispatcher
func main() {
	dspr.ProcessConfigs[LOGGER] = dspr.ProcessConfig{
		Name: LOGGER, 
		MustStart: false, 
		Required: []string{}, 
		Env: map[string]string{"testname":"testvalue",
	}}
	dspr.ProcessConfigs[WORKER1] = dspr.ProcessConfig{WORKER1, true, []string{LOGGER}, map[string]string{}}
	dspr.ProcessConfigs[WORKER2] = dspr.ProcessConfig{WORKER2, false, []string{LOGGER}, map[string]string{}}
	dspr.ProcessConfigs[WORKER3] = dspr.ProcessConfig{WORKER3, false, []string{LOGGER}, map[string]string{}}

	dspr.CreateDispatcher(0)
	dspr.D.Launch()
}
