#
# You will need TinyGo and Rust installed to compile the WASM modules.
#
# After installing Rust `rustup target add wasm32-unknown-unknown`
#

CARGO ?= cargo
EXECUTABLES = $(CARGO)
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

all: hex.wasm

hex.wasm:
	cd hex && $(CARGO) build --release --target wasm32-unknown-unknown && cp target/wasm32-unknown-unknown/release/hex.wasm ../hex.wasm

clean:
	rm -f hex.wasm
