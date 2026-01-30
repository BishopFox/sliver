# SOURCEDIR=.
# SOURCES = $(shell find $(SOURCEDIR) -name '*.go')
# VERSION=$(git describe --always --tags)
# BINARY=bin/pd

# bin: $(BINARY)

# $(BINARY): $(SOURCES)
# 	go build -o $(BINARY) command/*

.PHONY: build
build: build-deps
	go build -mod=vendor -o pd ./command

.PHONY: build-deps
build-deps:
	go get
	go mod verify
	go mod vendor

.PHONY: install
install: build
	cp pd $(GOROOT)/bin

.PHONY: test
test:
	go test -v ./...

.PHONY: deploy
deploy:
	- curl -sL https://git.io/goreleaser | bash
