# CI Dispatcher

**CI_Dispatcher** is a robust task orchestration engine designed specifically for monolithic Go architectures. It is a template for creating monolithic projects. Uses miniredis. 

Built in alignment with the <a href="https://avelactica.by/" >AVELACTICA</a> **Collective Intelligence (CI)** development standards, it ensures that high-concurrency systems remain stable and predictable.

## Goal

The primary mission of **CI_Dispatcher** is to provide a safety net for monolithic processes. In complex environments, uncontrolled goroutines can lead to memory leaks or deadlocks. 

This dispatcher ensures:
* **Controlled Lifecycle:** Every process is tracked; no goroutine loses control or becomes "orphaned."
* **Stuck-Process Prevention:** Built-in mechanisms to identify and resolve hanging tasks.
* **Graceful Management:** Seamlessly handles shutdowns and task distribution within the monolith.

## The template for projects
* `./build/raw/*` – Task definitions and raw process logic.
* `./build/memfd/*` – Adaptation of executable files into byte code for accretion.
* `./build/executable/*` – Directory for intermediate storage of executable files. Please do not use space in naming.
* `./Makefile` – Pre-configured automation for the dispatcher lifecycle.
* `./cmd/core/main.go` – The example central entry point (add dispatcher and tasks to project).

Add to projects main.go the import path like "_ **yourprojectname**/build/memfd" to adding Payload to project. 


### Prerequisites
* Go 1.20+ (recommended)
* GNU Make

### Quick Start
To use the example project based on cidispatcher:

```bash
# Clone the repository
git clone git@github.com:Averianov/cidispatcher.git
cd cidispatcher

# Build the elfs from ./build/raw/* to ./build/executable/*
make workers

# Convertation elf files to []byte in go - from ./build/executable/* to ./build/memfd
make prepare

# Run example project
make run

# Or just
make all
