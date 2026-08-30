GO ?= go
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

BINARY := opfor$(if $(filter windows,$(GOOS)),.exe)

.PHONY: all test-sleep-java bench-sleep bench-sleep-compare
all:
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -o ./$(BINARY) ./cmd/opfor

# Run every official Sleep differential in strict mode. OPFOR_SLEEP_JAR must
# name the authentic Sleep 2.1 JAR; tests verify its pinned SHA-256 digest.
test-sleep-java:
	@test -n "$(OPFOR_SLEEP_JAR)" || { echo "OPFOR_SLEEP_JAR is required" >&2; exit 1; }
	OPFOR_REQUIRE_SLEEP_JAR=1 TZ=UTC $(GO) test ./internal/opfor -json -count=1 -timeout=900s

# Smoke every benchmark once by default. Override BENCHTIME for measurements,
# for example: make bench-sleep BENCHTIME=3s.
BENCHTIME ?= 1x
bench-sleep:
	$(GO) test ./internal/opfor -run='^$$' -bench='^BenchmarkSleep' -benchtime=$(BENCHTIME) -count=1 -timeout=600s

# Compare equivalent workloads in-process in OPFOR and the pinned official
# Sleep 2.1 Java interpreter. The JAR is verified before measurement.
SLEEP_COMPARE_FLAGS ?=
bench-sleep-compare:
	@test -n "$(OPFOR_SLEEP_JAR)" || { echo "OPFOR_SLEEP_JAR is required" >&2; exit 1; }
	$(GO) run ./internal/cmd/sleepcompare -jar "$(OPFOR_SLEEP_JAR)" $(SLEEP_COMPARE_FLAGS)
