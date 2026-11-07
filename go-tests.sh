#!/bin/bash

# Sliver Implant Framework
# Copyright (C) 2019  Bishop Fox

# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

set -u
set -o pipefail

HOST_GO="$(command -v go 2>/dev/null || true)"
if [ -z "$HOST_GO" ] || [ ! -x "$HOST_GO" ]; then
	echo "Host Go toolchain not found" >&2
	exit 1
fi

SKIP_GENERATE=0
UNIT_ONLY=0
GENERATE_ONLY=0
GENERATE_RUN=""

usage() {
	cat <<'EOF'
Usage: ./go-tests.sh [--skip-generate | --unit-only | --generate-only] [--run REGEX]

  --skip-generate  Skip implant compilation tests.
  --unit-only      Skip implant compilation and C2 end-to-end tests.
  --generate-only  Run only the server/generate package tests.
  --run REGEX      Select generation tests by Go test name regex; requires --generate-only.
  --help          Show this help.
EOF
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	--skip-generate)
		SKIP_GENERATE=1
		;;
	--unit-only)
		UNIT_ONLY=1
		SKIP_GENERATE=1
		;;
	--generate-only)
		GENERATE_ONLY=1
		;;
	--run)
		if [ "$#" -lt 2 ] || [ -z "$2" ] || [[ "$2" == --* ]]; then
			echo "--run requires a non-empty regex" >&2
			exit 2
		fi
		GENERATE_RUN="$2"
		shift
		;;
	--run=*)
		GENERATE_RUN="${1#--run=}"
		if [ -z "$GENERATE_RUN" ]; then
			echo "--run requires a non-empty regex" >&2
			exit 2
		fi
		;;
	--help | -h)
		usage
		exit 0
		;;
	*)
		echo "Unknown argument: $1" >&2
		usage >&2
		exit 2
		;;
	esac
	shift
done

if [ "$GENERATE_ONLY" -eq 1 ] && [ "$SKIP_GENERATE" -eq 1 ]; then
	echo "--generate-only cannot be combined with --skip-generate or --unit-only" >&2
	exit 2
fi
if [ -n "$GENERATE_RUN" ] && [ "$GENERATE_ONLY" -eq 0 ]; then
	echo "--run requires --generate-only" >&2
	exit 2
fi

echo "----------------------------------------------------------------"
echo "WARNING: Running unit tests on slow systems can take a LONG time"
echo "         Recommended to only run on 16+ CPU cores and 32Gb+ RAM"
echo "----------------------------------------------------------------"
TAGS=osusergo,netgo,go_sqlite

TEST_TMP_ROOT="$(mktemp -d)"
if [ ! -d "$TEST_TMP_ROOT" ]; then
	echo "Failed to create temporary test root directory" >&2
	exit 1
fi

cleanup() {
	local status=$?
	trap - EXIT INT TERM

	# Some long-running tests (notably in server/generate) may still have child
	# build processes touching files under the temp root when cleanup starts.
	if command -v pkill >/dev/null 2>&1; then
		pkill -TERM -P $$ 2>/dev/null || true
		pkill -TERM -f "$TEST_TMP_ROOT" 2>/dev/null || true
		sleep 1
		pkill -KILL -P $$ 2>/dev/null || true
		pkill -KILL -f "$TEST_TMP_ROOT" 2>/dev/null || true
	fi
	wait 2>/dev/null || true

	for _ in 1 2 3 4 5; do
		rm -rf "$TEST_TMP_ROOT" 2>/dev/null || true
		if [ ! -e "$TEST_TMP_ROOT" ]; then
			break
		fi
		sleep 1
	done

	if [ -e "$TEST_TMP_ROOT" ]; then
		echo "WARNING: Failed to fully clean temp dir: $TEST_TMP_ROOT" >&2
	fi

	exit "$status"
}
trap cleanup EXIT INT TERM

export SLIVER_ROOT_DIR="$TEST_TMP_ROOT/sliver"
export SLIVER_CLIENT_DIR="$TEST_TMP_ROOT/sliver-client"
export SLIVER_CLIENT_ROOT_DIR="$SLIVER_CLIENT_DIR" # legacy compatibility
export HOME="$TEST_TMP_ROOT/home"
export XDG_CONFIG_HOME="$TEST_TMP_ROOT/xdg-config"
export XDG_CACHE_HOME="$TEST_TMP_ROOT/xdg-cache"
export XDG_DATA_HOME="$TEST_TMP_ROOT/xdg-data"
export TMPDIR="$TEST_TMP_ROOT/tmp"
export TMP="$TMPDIR"
export TEMP="$TMPDIR"
export GOCACHE="$TEST_TMP_ROOT/go-cache"
export GOTMPDIR="$TEST_TMP_ROOT/go-tmp"
export GOFLAGS="${GOFLAGS:-} -mod=vendor"

mkdir -p \
	"$SLIVER_ROOT_DIR" \
	"$SLIVER_CLIENT_DIR" \
	"$HOME" \
	"$XDG_CONFIG_HOME" \
	"$XDG_CACHE_HOME" \
	"$XDG_DATA_HOME" \
	"$TMPDIR" \
	"$GOCACHE" \
	"$GOTMPDIR"
export PATH="$SLIVER_ROOT_DIR/go/bin:$PATH"

echo "Using isolated temp directories:"
echo "  SLIVER_ROOT_DIR=$SLIVER_ROOT_DIR"
echo "  SLIVER_CLIENT_DIR=$SLIVER_CLIENT_DIR"

print_failure_logs() {
	cat "$SLIVER_ROOT_DIR/logs/sliver.log" 2>/dev/null || true
}

run_test_cmd() {
	local name="$1"
	shift

	echo
	echo "==> $name"
	if "$@"; then
		return 0
	fi

	print_failure_logs
	return 1
}

should_skip_package() {
	local pkg="$1"
	local tags="${2:-}"
	local go_list_error
	local test_file_counts
	if [ -n "$tags" ]; then
		go_list_error="$("$HOST_GO" list -e -f '{{if .Error}}{{.Error}}{{end}}' "-tags=$tags" "$pkg" 2>/dev/null || true)"
		test_file_counts="$("$HOST_GO" list -e -f '{{len .TestGoFiles}} {{len .XTestGoFiles}}' "-tags=$tags" "$pkg" 2>/dev/null || true)"
	else
		go_list_error="$("$HOST_GO" list -e -f '{{if .Error}}{{.Error}}{{end}}' "$pkg" 2>/dev/null || true)"
		test_file_counts="$("$HOST_GO" list -e -f '{{len .TestGoFiles}} {{len .XTestGoFiles}}' "$pkg" 2>/dev/null || true)"
	fi
	if [[ "$go_list_error" == *"build constraints exclude all Go files"* ]]; then
		echo
		echo "==> Skipping $pkg (unsupported on current platform)"
		return 0
	fi

	# File discovery intentionally sees tests for every OS. Avoid forcing a
	# package build when all of its tests are excluded by the current target's
	# build constraints; those tests remain discoverable on their target OS.
	if [ "$test_file_counts" = "0 0" ]; then
		echo
		echo "==> Skipping $pkg (no tests for current platform)"
		return 0
	fi
	return 1
}

collect_test_dirs() {
	if command -v rg >/dev/null 2>&1; then
		rg --files -g '*_test.go'
	else
		find client docs implant protobuf server test util -type f -name '*_test.go' -print
	fi
}

unpack_server_assets() {
	if [ -x "./sliver-server" ]; then
		if ./sliver-server unpack --force; then
			return 0
		fi
		echo "sliver-server unpack failed, falling back to go run ./server unpack --force"
	fi
	"$HOST_GO" run -tags=server,go_sqlite ./server unpack --force
}

run_generate_tests() {
	# Reclaim the host build cache before generation so long CI runs do not
	# exhaust the runner's disk while retaining the implant compiler cache.
	run_test_cmd "clean Go build cache" "$HOST_GO" clean -cache || return 1
	export GOPROXY=off
	# Generate tests are memory intensive (garble/c-shared cross-builds). Keep
	# parallelism conservative by default to avoid OOM kills on smaller systems.
	local generate_gomaxprocs="${SLIVER_GENERATE_GOMAXPROCS:-2}"
	local generate_go_p="${SLIVER_GENERATE_GO_P:-1}"
	local generate_test_parallel="${SLIVER_GENERATE_TEST_PARALLEL:-1}"
	local generate_test_cmd=(
		"$HOST_GO" test -timeout 6h -p "$generate_go_p" -parallel "$generate_test_parallel" -tags="server,$TAGS,sliver_generate"
	)
	if [ -n "$GENERATE_RUN" ]; then
		# Keep test selection out of GOFLAGS, which implant compiler children inherit.
		generate_test_cmd+=(-run "$GENERATE_RUN" -count=1 -v)
	fi
	generate_test_cmd+=(./server/generate)
	run_test_cmd "./server/generate" \
		env GOMAXPROCS="$generate_gomaxprocs" "${generate_test_cmd[@]}"
}

run_test_cmd "unpack server assets" unpack_server_assets || exit 1

if [ "$GENERATE_ONLY" -eq 1 ]; then
	run_generate_tests || exit 1
	exit 0
fi

TEST_DIRS=()
while IFS= read -r test_dir; do
	TEST_DIRS+=("$test_dir")
done < <(collect_test_dirs | xargs -n1 dirname | sort -u)

CLIENT_TEST_PKGS=()
UNTAGGED_TEST_PKGS=()
IMPLANT_TEST_PKGS=()
SERVER_UTIL_TEST_PKGS=()
E2E_SUPPORT_TEST_PKGS=()

for test_dir in "${TEST_DIRS[@]}"; do
	pkg="./$test_dir"
	case "$pkg" in
	./server/c2)
		# handled separately below
		;;
	./client/*)
		CLIENT_TEST_PKGS+=("$pkg")
		;;
	./docs | ./protobuf/*)
		UNTAGGED_TEST_PKGS+=("$pkg")
		;;
	./implant/*)
		IMPLANT_TEST_PKGS+=("$pkg")
		;;
	./server/* | ./util*)
		SERVER_UTIL_TEST_PKGS+=("$pkg")
		;;
	./test/e2e | ./test/e2e/*)
		E2E_SUPPORT_TEST_PKGS+=("$pkg")
		;;
	*)
		echo "No test runner configured for $pkg" >&2
		exit 1
		;;
	esac
done

## Packages without client, implant, or server build tags.
for pkg in "${UNTAGGED_TEST_PKGS[@]}"; do
	if should_skip_package "$pkg"; then
		continue
	fi
	run_test_cmd "$pkg" "$HOST_GO" test "$pkg" || exit 1
done

## Client
for pkg in "${CLIENT_TEST_PKGS[@]}"; do
	if should_skip_package "$pkg" "client,$TAGS"; then
		continue
	fi
	run_test_cmd "$pkg" "$HOST_GO" test -tags="client,$TAGS" "$pkg" || exit 1
done

## Comprehensive E2E support packages. The top-level lifecycle test skips
## unless a server binary is supplied; coverage/report unit tests run here.
for pkg in "${E2E_SUPPORT_TEST_PKGS[@]}"; do
	if should_skip_package "$pkg" "client,$TAGS"; then
		continue
	fi
	run_test_cmd "$pkg" "$HOST_GO" test -tags="client,$TAGS" "$pkg" || exit 1
done

## Implant
for pkg in "${IMPLANT_TEST_PKGS[@]}"; do
	if should_skip_package "$pkg"; then
		continue
	fi
	run_test_cmd "$pkg" "$HOST_GO" test "$pkg" || exit 1
done

## Server + Util, including generation helpers without compilation tests.
for pkg in "${SERVER_UTIL_TEST_PKGS[@]}"; do
	if should_skip_package "$pkg" "server,$TAGS"; then
		continue
	fi
	case "$pkg" in
	./server/assets/traffic-encoders | ./server/encoders)
		run_test_cmd "$pkg" "$HOST_GO" test -timeout 10m -tags="server,$TAGS" "$pkg" || exit 1
		;;
	./server/rpc)
		run_test_cmd "$pkg" "$HOST_GO" test -vet=off -timeout 30m -tags="server,$TAGS" "$pkg" || exit 1
		;;
	*)
		run_test_cmd "$pkg" "$HOST_GO" test -tags="server,$TAGS" "$pkg" || exit 1
		;;
	esac
done

## Server c2 (unit + e2e)
run_test_cmd "./server/c2" "$HOST_GO" test -tags="server,$TAGS" ./server/c2 || exit 1
if [ "$UNIT_ONLY" -eq 1 ]; then
	echo
	echo "Skipping ./server/c2 e2e tests (--unit-only)"
else
	run_test_cmd "./server/c2 (e2e yamux)" "$HOST_GO" test -tags="server,$TAGS,sliver_e2e" ./server/c2 -run 'Test(MTLS|WG)Yamux_' -count=1 || exit 1
	run_test_cmd "./server/c2 (e2e dns)" "$HOST_GO" test -tags="server,$TAGS,sliver_e2e" ./server/c2 -run 'TestDNS_' -count=1 || exit 1
fi

## Implant compilation tests opt in with sliver_generate; helper tests run above.
if [ "$SKIP_GENERATE" -eq 0 ]; then
	run_generate_tests || exit 1
else
	echo
	if [ "$UNIT_ONLY" -eq 1 ]; then
		echo "Skipping ./server/generate compilation tests (--unit-only)"
	else
		echo "Skipping ./server/generate compilation tests (--skip-generate)"
	fi
fi
