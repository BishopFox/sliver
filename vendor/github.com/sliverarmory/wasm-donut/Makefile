GO ?= go
GOBIN ?= $(CURDIR)/bin

.PHONY: all wasm install build clean

all: install

wasm:
	./donut/wasm/build.sh

install: wasm
	mkdir -p "$(GOBIN)"
	GOBIN="$(GOBIN)" $(GO) install ./cmd/wasm-donut

build: install

clean:
	rm -f "$(GOBIN)/wasm-donut"
