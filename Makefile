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
	@echo '    clean                    Clear ./build/executable/ directory.'
	@echo '    workers                  Build executable workers to ./build/executable/ directory.'
	@echo '    prepare                  Preparing executable files to Go in memory.'
	@echo '    run                      Start test project without compile.'
	@echo ''
	@echo 'Targets run by default are: fmt deps vet lint build test-unit.'
	@echo ''

.PHONY: all workers clean $(WORKERS)

all: workers prepare run
### rebuild workers #############################################

RAW_DIR := ./build/raw
EXE_DIR := ./build/executable

SOURCES := $(wildcard $(RAW_DIR)/*/main.go)
WORKERS := $(patsubst $(RAW_DIR)/%/main.go, %, $(SOURCES))

workers: clean $(WORKERS)

$(WORKERS): %:
	@mkdir -p $(EXE_DIR)
	go build -o $(EXE_DIR)/$* $(RAW_DIR)/$*/main.go

clean:
	rm -rf $(EXE_DIR)
	go clean -cache

##################################################################
prepare:
	go get github.com/Averianov/ftgc
	echo 'package main; import ftgc "github.com/Averianov/ftgc"; func main() {ftgc.ConvertDirectory("./build/executable", "./build/memfd", "")}' > temp.go && go run temp.go && rm temp.go

run: 
	go mod tidy
	go clean -cache
	go run ./cmd/main.go
