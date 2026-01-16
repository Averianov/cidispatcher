#!/usr/bin/make

GOCMD=$(shell which go)
GOMOD=$(shell which go) mod
GOLINT=$(shell which golint)
GODOC=$(shell which doc)
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOLIST=$(GOCMD) list
GOVET=$(GOCMD) vet
GORUN=$(GOCMD) run

help:
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@echo '    build                    Preparing www content && Build executable file.'
	@echo '    runwrap                      Start test wrapper.'	
	@echo '    runtest                      Start project without compile.'	
	@echo '    runplace                      Start project without compile.'	
	@echo '    runtime                      Start project without compile.'	
	@echo '    test				        Run integration tests.'
	@echo ''
	@echo 'Targets run by default are: fmt deps vet lint build test-unit.'
	@echo ''


build: 
	#g++ -std=c++11 -shared -fPIC -o ./cmd/cgo/libthread_wrapper.so ./cmd/cgo/thread_wrapper.cpp -lpthread
	#go get -u ./...
	go mod tidy
	#go build -o ./testdispatcher ./cmd
	CGO_ENABLED=1 CGO_LDFLAGS="-L. -lthread_wrapper" go build -o wrapper ./cmd/cgo/main.go

run:
	#LD_LIBRARY_PATH=./cmd/cgo/
	CGO_ENABLED=1 go run ./cmd/cgo/main.go

runtest:
	go run cmd/test/main.go

runplace:
	go build -gcflags=-S cmd/simple/main.go

runtime:
	GOTRACEBACK=system go run cmd/simple/main.go

test:
	$(GOTEST) ./...
