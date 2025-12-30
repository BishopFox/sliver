SHELL := /bin/bash

DIST_DIR ?= dist
GO_BIN_NAME := go-keystone
WASM_BIN_NAME := keystone
GO_BIN ?= $(shell if [ -x /opt/homebrew/bin/go ]; then echo /opt/homebrew/bin/go; elif command -v go >/dev/null 2>&1; then command -v go; else echo go; fi)

KEYSTONE_WASM_VERSION ?= v0.0.1
KEYSTONE_WASM_URL := https://github.com/moloch--/keystone/releases/download/$(KEYSTONE_WASM_VERSION)/keystone.wasm

WASM_PUBLIC_DIR := wasm
WASM_PUBLIC_MODULE := $(WASM_PUBLIC_DIR)/$(WASM_BIN_NAME).wasm
WASM_EXPORT_DIR := $(DIST_DIR)/wasm
WASM_MODULE := $(WASM_EXPORT_DIR)/$(WASM_BIN_NAME).wasm

GO_SOURCES := $(shell find cli -type f -name '*.go') $(shell find . -maxdepth 1 -type f -name '*.go')

# Cross-compile matrix used by `make all` (overridable by setting GO_PLATFORMS).
GO_PLATFORMS ?= darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 linux/amd64 linux/arm64

HOST_GOOS := $(shell $(GO_BIN) env GOOS)
HOST_GOARCH := $(shell $(GO_BIN) env GOARCH)
HOST_PLATFORM := $(HOST_GOOS)/$(HOST_GOARCH)

define GO_OUTPUT_FILENAME
$(DIST_DIR)/$(GO_BIN_NAME)_$(subst /,-,$(1))$(if $(filter windows/%,$(1)),.exe,)
endef

define GO_BUILD_RULE
$(call GO_OUTPUT_FILENAME,$(1)): $(GO_SOURCES) go.mod go.sum $(WASM_PUBLIC_MODULE) | $(DIST_DIR)
	GOOS=$(word 1,$(subst /, ,$(1))) GOARCH=$(word 2,$(subst /, ,$(1))) CGO_ENABLED=0 $(GO_BIN) build -v -trimpath -ldflags "-s -w" -o $$@ ./cli
endef

ALL_PLATFORMS := $(sort $(GO_PLATFORMS) $(HOST_PLATFORM))
$(foreach platform,$(ALL_PLATFORMS),$(eval $(call GO_BUILD_RULE,$(platform))))

HOST_OUTPUT := $(call GO_OUTPUT_FILENAME,$(HOST_PLATFORM))
GO_OUTPUTS := $(foreach platform,$(GO_PLATFORMS),$(call GO_OUTPUT_FILENAME,$(platform)))


.PHONY: all go wasm clean
.DEFAULT_GOAL := go

go: $(HOST_OUTPUT)

all: $(GO_OUTPUTS)

$(DIST_DIR):
	mkdir -p $@

$(WASM_EXPORT_DIR):
	mkdir -p $@

$(WASM_PUBLIC_MODULE):
	@set -euo pipefail; \
	tmp="$$(mktemp)"; \
	mkdir -p "$(dir $@)"; \
	if curl --fail --location --silent --show-error "$(KEYSTONE_WASM_URL)" -o "$$tmp"; then \
		mv "$$tmp" "$@"; \
	else \
		rm -f "$$tmp"; \
		exit 1; \
	fi

$(WASM_MODULE): $(WASM_PUBLIC_MODULE) | $(WASM_EXPORT_DIR)
	cp -f $< $@

wasm: $(WASM_PUBLIC_MODULE) $(WASM_MODULE)

clean:
	rm -rf $(DIST_DIR)

clean-wasm:
	rm -rf $(WASM_EXPORT_DIR)
