#
# Prerequisites 
#
# https://stackoverflow.com/questions/5618615/check-if-a-program-exists-from-a-makefile
GO ?= tinygo
EXECUTABLES = $(GO)
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))


base64.wasm:
	cd base64 && $(GO) build -opt=2 -o ../base64.wasm -scheduler=none -target=wasi ./base64.go

clean:
	rm -f base64.wasm
