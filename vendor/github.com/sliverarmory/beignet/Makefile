GO ?= go
ZIG ?= zig

BIN ?= beignet
RUNNER_BIN ?= testdata/runner/runner

GO_CACHE_DIR ?= $(CURDIR)/.go-cache
ZIG_CACHE_DIR ?= $(CURDIR)/.zig-cache

export GOCACHE := $(GO_CACHE_DIR)
export ZIG_GLOBAL_CACHE_DIR := $(ZIG_CACHE_DIR)
export ZIG_LOCAL_CACHE_DIR := $(ZIG_CACHE_DIR)

.PHONY: all runner clean

all: $(BIN)

$(GO_CACHE_DIR):
	@mkdir -p $@

$(ZIG_CACHE_DIR):
	@mkdir -p $@

$(BIN): $(GO_CACHE_DIR)
	$(GO) build -o $@ ./cli

runner: $(RUNNER_BIN)

$(RUNNER_BIN): testdata/runner/runner.c $(ZIG_CACHE_DIR)
	$(ZIG) cc -target aarch64-macos -o $@ $<

clean:
	rm -f $(BIN) $(RUNNER_BIN)
	rm -f runner payload.bin
	rm -rf $(GO_CACHE_DIR) $(ZIG_CACHE_DIR)

