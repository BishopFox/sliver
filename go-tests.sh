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

set -euo pipefail

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

BASE_TAGS="osusergo,netgo,go_sqlite"
ALL_TAGS="server,client,${BASE_TAGS}"

SLIVER_ROOT_DIR="$(mktemp -d)"
SLIVER_CLIENT_ROOT_DIR="$(mktemp -d)"
export SLIVER_ROOT_DIR
export SLIVER_CLIENT_ROOT_DIR

cleanup() {
    rm -rf "${SLIVER_ROOT_DIR:-}" "${SLIVER_CLIENT_ROOT_DIR:-}" "${SLIVER_ROOT_DIR_E2E:-}" "${SLIVER_CLIENT_ROOT_DIR_E2E:-}"
}
trap cleanup EXIT

TEST_PKGS=()
while IFS= read -r pkg; do
    TEST_PKGS+=("$pkg")
done < <(
    go list -e -tags="$ALL_TAGS" -f '{{if not .Error}}{{.ImportPath}}{{end}}' ./... \
        | rg -v '^github.com/bishopfox/sliver/implant/sliver$' \
        | rg -v '^github.com/bishopfox/sliver/implant/sliver/runner$' \
        | rg -v '^github.com/bishopfox/sliver/implant/sliver/proxy$' \
        | rg -v '^github.com/bishopfox/sliver/implant/sliver/transports/httpclient/drivers/win/wininet$' \
        | rg -v '^github.com/bishopfox/sliver/client/command/websites$' \
        | rg -v '^github.com/bishopfox/sliver/server/generate$'
)

if [ "${#TEST_PKGS[@]}" -eq 0 ]; then
    echo "No test packages selected"
    exit 1
fi

go test -vet=off -count=1 -timeout 45m -tags="$ALL_TAGS" "${TEST_PKGS[@]}"

if [ "$SKIP_GENERATE" -eq 0 ]; then
    GOPROXY=off go test -vet=off -count=1 -timeout 6h -tags="server,${BASE_TAGS}" ./server/generate
else
    echo "Skipping ./server/generate tests (--skip-generate)"
fi

SLIVER_ROOT_DIR_E2E="$(mktemp -d)"
SLIVER_CLIENT_ROOT_DIR_E2E="$(mktemp -d)"

if SLIVER_ROOT_DIR="$SLIVER_ROOT_DIR_E2E" SLIVER_CLIENT_ROOT_DIR="$SLIVER_CLIENT_ROOT_DIR_E2E" go test -vet=off -tags="server,${BASE_TAGS},sliver_e2e" ./server/c2 -run 'Test(MTLS|WG)Yamux_' -count=1 -timeout 30m ; then
    :
else
    cat "$SLIVER_ROOT_DIR_E2E/logs/sliver.log" 2>/dev/null || true
    exit 1
fi

if SLIVER_ROOT_DIR="$SLIVER_ROOT_DIR_E2E" SLIVER_CLIENT_ROOT_DIR="$SLIVER_CLIENT_ROOT_DIR_E2E" go test -vet=off -tags="server,${BASE_TAGS},sliver_e2e" ./server/c2 -run 'TestDNS_' -count=1 -timeout 30m ; then
    :
else
    cat "$SLIVER_ROOT_DIR_E2E/logs/sliver.log" 2>/dev/null || true
    exit 1
fi
