GO ?= go

# The CLI does not use cgo; forcing this makes cross-compiles more reliable.
CGO_ENABLED ?= 0

EXE :=
ifeq ($(GOOS),windows)
EXE := .exe
endif

OUT ?= malasada$(EXE)

STAGE0_BINS := internal/stage0/stage0_linux_amd64.bin internal/stage0/stage0_linux_arm64.bin

.PHONY: all build test clean stage0 check-stage0

all: build

# Build the CLI for the current GOOS/GOARCH (or overridden values).
build: check-stage0
	 GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		$(GO) build -trimpath -o $(OUT) ./cli

test: check-stage0
	 $(GO) test ./...

clean:
	rm -f $(OUT)

# Force regeneration of the embedded stage0 blobs.
stage0:
	$(GO) generate ./...

check-stage0:
	@for f in $(STAGE0_BINS); do \
		if [ ! -f "$$f" ]; then \
			echo "Missing $$f"; \
			echo "Run: make stage0"; \
			exit 1; \
		fi; \
	done
