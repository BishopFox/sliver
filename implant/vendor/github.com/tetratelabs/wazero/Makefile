
# Make functions strip spaces and use commas to separate parameters. The below variables escape these characters.
comma := ,
space :=
space +=

gofumpt := mvdan.cc/gofumpt@v0.4.0
gosimports := github.com/rinchsan/gosimports/cmd/gosimports@v0.3.4
golangci_lint := github.com/golangci/golangci-lint/cmd/golangci-lint@v1.50.1
# sync this with netlify.toml!
hugo          := github.com/gohugoio/hugo@v0.105.0

# Make 3.81 doesn't support '**' globbing: Set explicitly instead of recursion.
all_sources   := $(wildcard *.go */*.go */*/*.go */*/*/*.go */*/*/*.go */*/*/*/*.go)
all_testdata  := $(wildcard testdata/* */testdata/* */*/testdata/* */*/testdata/*/* */*/*/testdata/*)
all_testing   := $(wildcard internal/testing/* internal/testing/*/* internal/testing/*/*/*)
all_examples  := $(wildcard examples/* examples/*/* examples/*/*/* */*/example/* */*/example/*/* */*/example/*/*/*)
all_it        := $(wildcard internal/integration_test/* internal/integration_test/*/* internal/integration_test/*/*/*)
# main_sources exclude any test or example related code
main_sources  := $(wildcard $(filter-out %_test.go $(all_testdata) $(all_testing) $(all_examples) $(all_it), $(all_sources)))
# main_packages collect the unique main source directories (sort will dedupe).
# Paths need to all start with ./, so we do that manually vs foreach which strips it.
main_packages := $(sort $(foreach f,$(dir $(main_sources)),$(if $(findstring ./,$(f)),./,./$(f))))

ensureCompilerFastest := -ldflags '-X github.com/tetratelabs/wazero/internal/integration_test/vs.ensureCompilerFastest=true'
.PHONY: bench
bench:
	@go test -run=NONE -benchmem -bench=. ./internal/integration_test/bench/...
	@go test -benchmem -bench=. ./internal/integration_test/vs/... $(ensureCompilerFastest)

.PHONY: bench.check
bench.check:
	@go build ./internal/integration_test/bench/...
	@# Don't use -test.benchmem as it isn't accurate when comparing against CGO libs
	@for d in vs/time vs/wasmedge vs/wasmer vs/wasmtime ; do \
		cd ./internal/integration_test/$$d ; \
		go test -bench=. . -tags='wasmedge' $(ensureCompilerFastest) ; \
		cd - ;\
	done

bench_testdata_dir := internal/integration_test/bench/testdata
.PHONY: build.bench
build.bench:
	@tinygo build -o $(bench_testdata_dir)/case.wasm -scheduler=none --no-debug -target=wasi $(bench_testdata_dir)/case.go

.PHONY: test.examples
test.examples:
	@go test ./examples/... ./imports/assemblyscript/example/... ./imports/emscripten/... ./imports/go/example/... ./imports/wasi_snapshot_preview1/example/...

.PHONY: build.examples.as
build.examples.as:
	@cd ./imports/assemblyscript/example/testdata && npm install && npm run build

%.wasm: %.zig
	@(cd $(@D); zig build -Drelease-small=true)
	@mv $(@D)/zig-out/*/$(@F) $(@D)

.PHONY: build.examples.zig
build.examples.zig: examples/allocation/zig/testdata/greet.wasm imports/wasi_snapshot_preview1/example/testdata/zig/cat.wasm imports/wasi_snapshot_preview1/testdata/zig/wasi.wasm
	@cd internal/testing/dwarftestdata/testdata/zig; zig build; mv zig-out/*/main.wasm ./ # Need DWARF custom sections.

tinygo_sources := examples/basic/testdata/add.go examples/allocation/tinygo/testdata/greet.go examples/cli/testdata/cli.go imports/wasi_snapshot_preview1/example/testdata/tinygo/cat.go
.PHONY: build.examples.tinygo
build.examples.tinygo: $(tinygo_sources)
	@for f in $^; do \
	    tinygo build -o $$(echo $$f | sed -e 's/\.go/\.wasm/') -scheduler=none --no-debug --target=wasi $$f; \
	done
	# Need DWARF sections.
	tinygo build -o internal/testing/dwarftestdata/testdata/tinygo/main.wasm -scheduler=none --target=wasi internal/testing/dwarftestdata/testdata/tinygo/main.go

# We use zig to build C as it is easy to install and embeds a copy of zig-cc.
c_sources := imports/wasi_snapshot_preview1/example/testdata/zig-cc/cat.c imports/wasi_snapshot_preview1/testdata/zig-cc/wasi.c
.PHONY: build.examples.zig-cc
build.examples.zig-cc: $(c_sources)
	@for f in $^; do \
	    zig cc --target=wasm32-wasi -Oz -o $$(echo $$f | sed -e 's/\.c/\.wasm/') $$f; \
	done

# Here are the emcc args we use:
#
# * `-Oz` - most optimization for code size.
# * `--profiling` - adds the name section.
# * `-s STANDALONE_WASM` - ensures wasm is built for a non-js runtime.
# * `-s EXPORTED_FUNCTIONS=_malloc,_free` - export allocation functions so that
#   they can be used externally as "malloc" and "free".
# * `-s WARN_ON_UNDEFINED_SYMBOLS=0` - imports not defined in JavaScript error
#   otherwise. See https://github.com/emscripten-core/emscripten/issues/13641
# * `-s TOTAL_STACK=8KB -s TOTAL_MEMORY=64KB` - reduce memory default from 16MB
#   to one page (64KB). To do this, we have to reduce the stack size.
# * `-s ALLOW_MEMORY_GROWTH` - allows "memory.grow" instructions to succeed, but
#   requires a function import "emscripten_notify_memory_growth".
emscripten_sources := $(wildcard imports/emscripten/testdata/*.cc)
.PHONY: build.examples.emscripten
build.examples.emscripten: $(emscripten_sources)
	@for f in $^; do \
		em++ -Oz --profiling \
		-s STANDALONE_WASM \
		-s EXPORTED_FUNCTIONS=_malloc,_free \
		-s WARN_ON_UNDEFINED_SYMBOLS=0 \
		-s TOTAL_STACK=8KB -s TOTAL_MEMORY=64KB \
		-s ALLOW_MEMORY_GROWTH \
		--std=c++17 -o $$(echo $$f | sed -e 's/\.cc/\.wasm/') $$f; \
	done

%/greet.wasm : cargo_target := wasm32-unknown-unknown
%/cat.wasm : cargo_target := wasm32-wasi
%/wasi.wasm : cargo_target := wasm32-wasi

.PHONY: build.examples.rust
build.examples.rust: examples/allocation/rust/testdata/greet.wasm imports/wasi_snapshot_preview1/example/testdata/cargo-wasi/cat.wasm imports/wasi_snapshot_preview1/testdata/cargo-wasi/wasi.wasm internal/testing/dwarftestdata/testdata/rust/main.wasm.xz

# Normally, we build release because it is smaller. Testing dwarf requires the debug build.
internal/testing/dwarftestdata/testdata/rust/main.wasm.xz:
	cd $(@D) && cargo wasi build
	mv $(@D)/target/wasm32-wasi/debug/main.wasm $(@D)
	cd $(@D) && xz -k -f ./main.wasm # Rust's DWARF section is huge, so compress it.

# Builds rust using cargo normally, or cargo-wasi.
%.wasm: %.rs
	@(cd $(@D); cargo $(if $(findstring wasi,$(cargo_target)),wasi build,build --target $(cargo_target)) --release)
	@mv $(@D)/target/$(cargo_target)/release/$(@F) $(@D)

spectest_base_dir := internal/integration_test/spectest
spectest_v1_dir := $(spectest_base_dir)/v1
spectest_v1_testdata_dir := $(spectest_v1_dir)/testdata
spec_version_v1 := wg-1.0
spectest_v2_dir := $(spectest_base_dir)/v2
spectest_v2_testdata_dir := $(spectest_v2_dir)/testdata
# Latest draft state as of Nov 9, 2022.
spec_version_v2 := f9b461a312426a60f2f81dcb19b39b66b90a5447

.PHONY: build.spectest
build.spectest:
	@$(MAKE) build.spectest.v1
	@$(MAKE) build.spectest.v2

.PHONY: build.spectest.v1
build.spectest.v1: # Note: wabt by default uses >1.0 features, so wast2json flags might drift as they include more. See WebAssembly/wabt#1878
	@rm -rf $(spectest_v1_testdata_dir)
	@mkdir -p $(spectest_v1_testdata_dir)
	@cd $(spectest_v1_testdata_dir) \
		&& curl -sSL 'https://api.github.com/repos/WebAssembly/spec/contents/test/core?ref=$(spec_version_v1)' | jq -r '.[]| .download_url' | grep -E ".wast" | xargs -Iurl curl -sJL url -O
	@cd $(spectest_v1_testdata_dir) && for f in `find . -name '*.wast'`; do \
		perl -pi -e 's/\(assert_return_canonical_nan\s(\(invoke\s"f32.demote_f64"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\))\)/\(assert_return $$1 \(f32.const nan:canonical\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_arithmetic_nan\s(\(invoke\s"f32.demote_f64"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\))\)/\(assert_return $$1 \(f32.const nan:arithmetic\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_canonical_nan\s(\(invoke\s"f64\.promote_f32"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\))\)/\(assert_return $$1 \(f64.const nan:canonical\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_arithmetic_nan\s(\(invoke\s"f64\.promote_f32"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\))\)/\(assert_return $$1 \(f64.const nan:arithmetic\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_canonical_nan\s(\(invoke\s"[a-z._0-9]+"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\))\)/\(assert_return $$1 \($$2.const nan:canonical\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_arithmetic_nan\s(\(invoke\s"[a-z._0-9]+"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\))\)/\(assert_return $$1 \($$2.const nan:arithmetic\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_canonical_nan\s(\(invoke\s"[a-z._0-9]+"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\s\([a-z0-9.\s+-:]+\)\))\)/\(assert_return $$1 \($$2.const nan:canonical\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_arithmetic_nan\s(\(invoke\s"[a-z._0-9]+"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\s\([a-z0-9.\s+-:]+\)\))\)/\(assert_return $$1 \($$2.const nan:arithmetic\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_canonical_nan\s(\(invoke\s"[a-z._0-9]+"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\))\)/\(assert_return $$1 \($$2.const nan:canonical\)\)/g' $$f; \
		perl -pi -e 's/\(assert_return_arithmetic_nan\s(\(invoke\s"[a-z._0-9]+"\s\((f[0-9]{2})\.const\s[a-z0-9.+:-]+\)\))\)/\(assert_return $$1 \($$2.const nan:arithmetic\)\)/g' $$f; \
		wast2json \
			--disable-saturating-float-to-int \
			--disable-sign-extension \
			--disable-simd \
			--disable-multi-value \
			--disable-bulk-memory \
			--disable-reference-types \
			--debug-names $$f; \
	done

.PHONY: build.spectest.v2
build.spectest.v2: # Note: SIMD cases are placed in the "simd" subdirectory.
	@mkdir -p $(spectest_v2_testdata_dir)
	@cd $(spectest_v2_testdata_dir) \
		&& curl -sSL 'https://api.github.com/repos/WebAssembly/spec/contents/test/core?ref=$(spec_version_v2)' | jq -r '.[]| .download_url' | grep -E ".wast" | xargs -Iurl curl -sJL url -O
	@cd $(spectest_v2_testdata_dir) \
		&& curl -sSL 'https://api.github.com/repos/WebAssembly/spec/contents/test/core/simd?ref=$(spec_version_v2)' | jq -r '.[]| .download_url' | grep -E ".wast" | xargs -Iurl curl -sJL url -O
	@cd $(spectest_v2_testdata_dir) && for f in `find . -name '*.wast'`; do \
		wast2json --debug-names $$f; \
	done

.PHONY: test
test:
	@go test $$(go list ./... | grep -vE '$(spectest_v1_dir)|$(spectest_v2_dir)') -timeout 120s
	@cd internal/version/testdata && go test ./... -timeout 120s

.PHONY: coverage
coverpkg = $(subst $(space),$(comma),$(main_packages))
coverage: ## Generate test coverage
	@go test -coverprofile=coverage.txt -covermode=atomic --coverpkg=$(coverpkg) $(main_packages)
	@go tool cover -func coverage.txt

.PHONY: spectest
spectest:
	@$(MAKE) spectest.v1
	@$(MAKE) spectest.v2

spectest.v1:
	@go test $$(go list ./... | grep $(spectest_v1_dir)) -timeout 120s

spectest.v2:
	@go test $$(go list ./... | grep $(spectest_v2_dir)) -timeout 120s

golangci_lint_path := $(shell go env GOPATH)/bin/golangci-lint

$(golangci_lint_path):
	@go install $(golangci_lint)

golangci_lint_goarch ?= $(shell go env GOARCH)

.PHONY: lint
lint: $(golangci_lint_path)
	@GOARCH=$(golangci_lint_goarch) CGO_ENABLED=0 $(golangci_lint_path) run --timeout 5m

.PHONY: format
format:
	@go run $(gofumpt) -l -w .
	@go run $(gosimports) -local github.com/tetratelabs/ -w $(shell find . -name '*.go' -type f)

.PHONY: check
check:
	@GOARCH=amd64 GOOS=dragonfly go build ./... # Check if the internal/platform can be built on compiler-unsupported platforms
	@GOARCH=386 GOOS=linux go build ./... # Check if the internal/platform can be built on compiler-unsupported platforms
	@$(MAKE) lint golangci_lint_goarch=arm64
	@$(MAKE) lint golangci_lint_goarch=amd64
	@$(MAKE) format
	@go mod tidy
	@if [ ! -z "`git status -s`" ]; then \
		echo "The following differences will fail CI until committed:"; \
		git diff --exit-code; \
	fi

.PHONY: site
site: ## Serve website content
	@git submodule update --init
	@cd site && go run $(hugo) server --minify --disableFastRender --baseURL localhost:1313 --cleanDestinationDir -D

.PHONY: clean
clean: ## Ensure a clean build
	@rm -rf dist build coverage.txt
	@go clean -testcache

fuzz_timeout_seconds ?= 10
.PHONY: fuzz
fuzz:
	@cd internal/integration_test/fuzz && cargo fuzz run basic -- -max_total_time=$(fuzz_timeout_seconds)
	@cd internal/integration_test/fuzz && cargo fuzz run validation -- -max_total_time=$(fuzz_timeout_seconds)
