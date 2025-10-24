## shm
[![Status](https://github.com/gen2brain/shm/actions/workflows/test.yml/badge.svg)](https://github.com/gen2brain/shm/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gen2brain/shm.svg)](https://pkg.go.dev/github.com/gen2brain/shm)
[![Go Report Card](https://goreportcard.com/badge/github.com/gen2brain/shm?branch=master)](https://goreportcard.com/report/github.com/gen2brain/shm) 

`shm` implements System V shared memory functions (`shmctl`, `shmget`, `shmat`, `shmdt`) in pure Go (except `solaris`, see [#24357](https://github.com/golang/go/issues/24357)).

### Installation

    go get -u github.com/gen2brain/shm

### More

For System V Message Queue IPC (i.e. `msgctl`, `msgget`, `msgrcv`, `msgsnd`) see [ipc](https://github.com/siadat/ipc).
