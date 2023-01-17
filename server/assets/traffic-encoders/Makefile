#
# You will need TinyGo and Rust installed to compile the WASM modules.
#
# After installing Rust `rustup target add wasm32-unknown-unknown`
#

GO ?= tinygo
CARGO ?= cargo
EXECUTABLES = $(GO) $(CARGO)
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

all: base64.wasm hex.wasm

base64.wasm:
	cd base64 && $(GO) build -opt=2 -o ../base64.wasm -scheduler=none -target=wasi ./base64.go

hex.wasm:
	cd hex && $(CARGO) build --release --target wasm32-unknown-unknown && cp target/wasm32-unknown-unknown/release/hex.wasm ../hex.wasm

clean:
	rm -f base64.wasm
