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

SKIP_GENERATE=0
for arg in "$@"; do
	if [ "$arg" = "--skip-generate" ]; then
		SKIP_GENERATE=1
	fi
done

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
	local go_list_cmd=(go list -e -f '{{if .Error}}{{.Error}}{{end}}')
	if [ -n "$tags" ]; then
		go_list_cmd+=("-tags=$tags")
	fi
	go_list_cmd+=("$pkg")

	local go_list_error
	go_list_error="$("${go_list_cmd[@]}" 2>/dev/null || true)"
	if [[ "$go_list_error" == *"build constraints exclude all Go files"* ]]; then
		echo
		echo "==> Skipping $pkg (unsupported on current platform)"
		return 0
	fi
	return 1
}

collect_test_dirs() {
	if command -v rg >/dev/null 2>&1; then
		rg --files -g '*_test.go'
	else
		find client implant server util -type f -name '*_test.go' -print
	fi
}

unpack_server_assets() {
	if [ -x "./sliver-server" ]; then
		if ./sliver-server unpack --force; then
			return 0
		fi
		echo "sliver-server unpack failed, falling back to go run ./server unpack --force"
	fi
	go run -tags=server,go_sqlite ./server unpack --force
}

run_test_cmd "unpack server assets" unpack_server_assets || exit 1

TEST_DIRS=()
while IFS= read -r test_dir; do
	TEST_DIRS+=("$test_dir")
done < <(collect_test_dirs | xargs -n1 dirname | sort -u)

CLIENT_TEST_PKGS=()
IMPLANT_TEST_PKGS=()
SERVER_UTIL_TEST_PKGS=()

for test_dir in "${TEST_DIRS[@]}"; do
	pkg="./$test_dir"
	case "$pkg" in
	./server/c2 | ./server/generate)
		# handled separately below
		;;
	./client/*)
		CLIENT_TEST_PKGS+=("$pkg")
		;;
	./implant/*)
		IMPLANT_TEST_PKGS+=("$pkg")
		;;
	./server/* | ./util*)
		SERVER_UTIL_TEST_PKGS+=("$pkg")
		;;
	esac
done

## Client
for pkg in "${CLIENT_TEST_PKGS[@]}"; do
	if should_skip_package "$pkg" "client,$TAGS"; then
		continue
	fi
	run_test_cmd "$pkg" go test -tags="client,$TAGS" "$pkg" || exit 1
done

## Implant
for pkg in "${IMPLANT_TEST_PKGS[@]}"; do
	if should_skip_package "$pkg"; then
		continue
	fi
	run_test_cmd "$pkg" go test "$pkg" || exit 1
done

## Server + Util
for pkg in "${SERVER_UTIL_TEST_PKGS[@]}"; do
	if should_skip_package "$pkg" "server,$TAGS"; then
		continue
	fi
	case "$pkg" in
	./server/assets/traffic-encoders | ./server/encoders)
		run_test_cmd "$pkg" go test -timeout 10m -tags="server,$TAGS" "$pkg" || exit 1
		;;
	./server/rpc)
		run_test_cmd "$pkg" go test -vet=off -timeout 30m -tags="server,$TAGS" "$pkg" || exit 1
		;;
	*)
		run_test_cmd "$pkg" go test -tags="server,$TAGS" "$pkg" || exit 1
		;;
	esac
done

## Server c2 (unit + e2e)
run_test_cmd "./server/c2" go test -tags="server,$TAGS" ./server/c2 || exit 1
run_test_cmd "./server/c2 (e2e yamux)" go test -tags="server,$TAGS,sliver_e2e" ./server/c2 -run 'Test(MTLS|WG)Yamux_' -count=1 || exit 1
run_test_cmd "./server/c2 (e2e dns)" go test -tags="server,$TAGS,sliver_e2e" ./server/c2 -run 'TestDNS_' -count=1 || exit 1

## Server generate
if [ "$SKIP_GENERATE" -eq 0 ]; then
	export GOPROXY=off
	run_test_cmd "./server/generate" go test -timeout 6h -tags="server,$TAGS" ./server/generate || exit 1
else
	echo
	echo "Skipping ./server/generate tests (--skip-generate)"
fi
