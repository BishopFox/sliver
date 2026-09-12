#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
INSTALL_SCRIPT="$REPO_ROOT/docs/sliver-docs/public/install"
ALPINE_IMAGE="${ALPINE_IMAGE:-alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b}"
TEST_ROOT="$(mktemp -d /tmp/sliver-install-test.XXXXXX)"
OPENRC_CONTAINER=""
OPENRC_IMAGE=""

cleanup() {
    case "$OPENRC_CONTAINER" in
        sliver-install-openrc-test-*)
            docker rm -f "$OPENRC_CONTAINER" > /dev/null 2>&1 || true
            ;;
    esac
    case "$OPENRC_IMAGE" in
        sha256:*)
            docker image rm -f "$OPENRC_IMAGE" > /dev/null 2>&1 || true
            ;;
    esac
    case "$TEST_ROOT" in
        /tmp/sliver-install-test.*)
            rm -rf -- "$TEST_ROOT"
            ;;
    esac
}
trap cleanup EXIT

mkdir -p \
    "$TEST_ROOT/common" \
    "$TEST_ROOT/alpine" \
    "$TEST_ROOT/openrc" \
    "$TEST_ROOT/openrc-real" \
    "$TEST_ROOT/mixed"

cat > "$TEST_ROOT/common/uname" <<'EOF'
#!/bin/sh
if [ "${1:-}" = "-m" ]; then
    echo x86_64
else
    exec /bin/uname "$@"
fi
EOF

cat > "$TEST_ROOT/common/minisign" <<'EOF'
#!/bin/sh
exit 0
EOF

cat > "$TEST_ROOT/common/curl" <<'EOF'
#!/bin/sh
set -eu

output=""
url=""
while [ "$#" -gt 0 ]; do
    case "$1" in
        --output|-o)
            output="$2"
            shift 2
            ;;
        -*)
            shift
            ;;
        *)
            url="$1"
            shift
            ;;
    esac
done

printf '%s\n' "$url" >> /tmp/curl-requests.log

case "$url" in
    *api.github.com*)
        cat <<'JSON'
{"browser_download_url":"https://example.invalid/sliver-server_linux-amd64"}
{"browser_download_url":"https://example.invalid/sliver-server_linux-amd64.minisig"}
{"browser_download_url":"https://example.invalid/sliver-client_linux-amd64"}
{"browser_download_url":"https://example.invalid/sliver-client_linux-amd64.minisig"}
JSON
        ;;
    */sliver-server_linux-amd64)
        cat > "$output" <<'SERVER'
#!/bin/sh
case "${1:-}" in
    unpack)
        exit 0
        ;;
    operator)
        shift
        save_dir=""
        while [ "$#" -gt 0 ]; do
            case "$1" in
                --save)
                    save_dir="$2"
                    shift 2
                    ;;
                *)
                    shift
                    ;;
            esac
        done
        mkdir -p "$save_dir"
        : > "$save_dir/operator.cfg"
        ;;
    daemon)
        echo $$ > /tmp/sliver-daemon.pid
        trap 'rm -f /tmp/sliver-daemon.pid; exit 0' TERM INT
        while :; do
            sleep 1
        done
        ;;
esac
SERVER
        chmod 755 "$output"
        ;;
    */sliver-client_linux-amd64)
        cat > "$output" <<'CLIENT'
#!/bin/sh
exit 0
CLIENT
        chmod 755 "$output"
        ;;
    *)
        : > "$output"
        ;;
esac
EOF

cat > "$TEST_ROOT/openrc/supervise-daemon" <<'EOF'
#!/bin/sh
exit 0
EOF

cat > "$TEST_ROOT/alpine/apk" <<'EOF'
#!/bin/sh
printf 'apk %s\n' "$*" >> /tmp/installer-test.log
EOF

cat > "$TEST_ROOT/openrc/apk" <<'EOF'
#!/bin/sh
printf 'apk %s\n' "$*" >> /tmp/installer-test.log
EOF

cat > "$TEST_ROOT/openrc/openrc-run" <<'EOF'
#!/bin/sh
exit 0
EOF

cat > "$TEST_ROOT/openrc/rc-update" <<'EOF'
#!/bin/sh
printf 'rc-update %s\n' "$*" >> /tmp/installer-test.log
EOF

cat > "$TEST_ROOT/openrc/rc-service" <<'EOF'
#!/bin/sh
printf 'rc-service %s\n' "$*" >> /tmp/installer-test.log
EOF

cat > "$TEST_ROOT/openrc-real/apk" <<'EOF'
#!/bin/sh
printf 'apk %s\n' "$*" >> /tmp/installer-test.log
EOF

cat > "$TEST_ROOT/mixed/apk" <<'EOF'
#!/bin/sh
printf 'apk %s\n' "$*" >> /tmp/installer-test.log
EOF

cat > "$TEST_ROOT/mixed/apt-get" <<'EOF'
#!/bin/sh
printf 'apt-get %s\n' "$*" >> /tmp/installer-test.log
EOF

cat > "$TEST_ROOT/mixed/systemctl" <<'EOF'
#!/bin/sh
printf 'systemctl %s\n' "$*" >> /tmp/installer-test.log
exit 0
EOF


for command in openrc-run rc-update rc-service supervise-daemon; do
    cat > "$TEST_ROOT/mixed/$command" <<'EOF'
#!/bin/sh
printf '%s %s\n' "$(basename "$0")" "$*" >> /tmp/installer-test.log
EOF
done

cat > "$TEST_ROOT/runner" <<'EOF'
#!/bin/sh
set -eu

scenario="$1"
manager_dir="$scenario"
if [ "$scenario" = "unmanaged" ]; then
    manager_dir="alpine"
fi
export PATH="/test/$manager_dir:/test/common:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
: > /tmp/installer-test.log
: > /tmp/curl-requests.log

run_installer() {
    if ! /bin/sh /install > /tmp/installer-output.log 2>&1; then
        cat /tmp/installer-output.log >&2
        return 1
    fi
}

assert_common_install() {
    test -x /root/sliver-server
    test -x /usr/local/bin/sliver-client
    test -L /usr/local/bin/sliver
    test -f /root/.sliver-client/configs/operator.cfg
}

case "$scenario" in
    alpine)
        rm -rf /run/openrc /run/systemd
        run_installer
        assert_common_install
        test ! -e /etc/init.d/sliver
        test ! -e /etc/systemd/system/sliver.service
        grep -q '^apk add --no-cache curl git make build-base$' /tmp/installer-test.log
        grep -q 'No running systemd or OpenRC instance detected' /tmp/installer-output.log
        grep -q 'exec /root/sliver-server daemon' /tmp/installer-output.log
        ;;
    openrc)
        mkdir -p /etc/init.d /run/openrc
        : > /run/openrc/softlevel
        run_installer
        assert_common_install
        test -x /etc/init.d/sliver
        test "$(stat -c '%U:%G:%a' /etc/init.d/sliver)" = 'root:root:755'
        grep -q '^supervisor=supervise-daemon$' /etc/init.d/sliver
        grep -q '^respawn_delay=3$' /etc/init.d/sliver
        grep -q '^respawn_max=0$' /etc/init.d/sliver
        grep -q '^rc-update add sliver default$' /tmp/installer-test.log
        grep -q '^rc-service sliver start$' /tmp/installer-test.log
        run_installer
        grep -q '^rc-service sliver stop$' /tmp/installer-test.log
        ;;
    mixed)
        : > /etc/alpine-release
        printf 'ID=debian\n' > /etc/os-release
        mkdir -p /run/openrc /run/systemd/system /etc/systemd/system
        : > /run/openrc/softlevel
        run_installer
        assert_common_install
        test -f /etc/systemd/system/sliver.service
        test ! -e /etc/init.d/sliver
        grep -q '^apt-get install -yqq curl build-essential git$' /tmp/installer-test.log
        if grep -q '^apk ' /tmp/installer-test.log; then
            echo 'apk was selected on a non-Alpine system' >&2
            exit 1
        fi
        grep -q '^systemctl daemon-reload$' /tmp/installer-test.log
        grep -q '^systemctl start sliver$' /tmp/installer-test.log
        ;;
    unmanaged)
        rm -rf /run/openrc /run/systemd
        run_installer
        /root/sliver-server daemon &
        daemon_pid=$!
        for _ in 1 2 3 4 5; do
            test -f /tmp/sliver-daemon.pid && break
            sleep 1
        done
        test -f /tmp/sliver-daemon.pid
        : > /tmp/curl-requests.log
        if /bin/sh /install > /tmp/installer-upgrade-output.log 2>&1; then
            echo 'unmanaged upgrade unexpectedly succeeded' >&2
            exit 1
        fi
        grep -q 'Stop it with the container or process supervisor' /tmp/installer-upgrade-output.log
        test ! -s /tmp/curl-requests.log
        kill -0 "$daemon_pid"
        kill "$daemon_pid"
        wait "$daemon_pid"
        ;;
esac
EOF

chmod 755 \
    "$TEST_ROOT/common/uname" \
    "$TEST_ROOT/common/minisign" \
    "$TEST_ROOT/common/curl" \
    "$TEST_ROOT/alpine/apk" \
    "$TEST_ROOT/openrc/apk" \
    "$TEST_ROOT/openrc/openrc-run" \
    "$TEST_ROOT/openrc/supervise-daemon" \
    "$TEST_ROOT/openrc/rc-update" \
    "$TEST_ROOT/openrc/rc-service" \
    "$TEST_ROOT/openrc-real/apk" \
    "$TEST_ROOT/mixed/apk" \
    "$TEST_ROOT/mixed/apt-get" \
    "$TEST_ROOT/mixed/systemctl" \
    "$TEST_ROOT/mixed/openrc-run" \
    "$TEST_ROOT/mixed/rc-update" \
    "$TEST_ROOT/mixed/rc-service" \
    "$TEST_ROOT/mixed/supervise-daemon" \
    "$TEST_ROOT/runner"

docker run --rm --network none \
    -v "$INSTALL_SCRIPT:/install:ro" \
    "$ALPINE_IMAGE" sh -n /install

echo "Testing installer scenario: non-root rejection"
if docker run --rm --network none --user 65534 \
    -v "$INSTALL_SCRIPT:/install:ro" \
    "$ALPINE_IMAGE" /bin/sh /install > /dev/null 2>&1; then
    echo "non-root installer invocation unexpectedly succeeded" >&2
    exit 1
fi

echo "Testing installer scenario: stock Alpine wget bootstrap and real apk"
docker run --rm \
    -v "$INSTALL_SCRIPT:/site/install:ro" \
    -v "$TEST_ROOT:/test:ro" \
    "$ALPINE_IMAGE" sh -euxc '
        content_length="$(wc -c < /site/install)"
        {
            printf "HTTP/1.1 200 OK\r\nContent-Length: %s\r\nConnection: close\r\n\r\n" "$content_length"
            cat /site/install
        } | nc -l -s 127.0.0.1 -p 8080 &
        server_pid=$!
        trap "kill $server_pid 2> /dev/null || true" EXIT
        sleep 1
        wget -qO- http://127.0.0.1:8080/install |
            env PATH=/test/common:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
            sh > /tmp/installer-output.log
        test -x /root/sliver-server
        test -x /usr/local/bin/sliver-client
        test ! -e /etc/init.d/sliver
        test ! -e /etc/systemd/system/sliver.service
        apk info -e curl git make build-base
        apk add --no-cache minisign
        command -v minisign
        grep -q "exec /root/sliver-server daemon" /tmp/installer-output.log
    '

for scenario in alpine openrc mixed unmanaged; do
    echo "Testing installer scenario: $scenario"
    docker run --rm --network none \
        -v "$INSTALL_SCRIPT:/install:ro" \
        -v "$TEST_ROOT:/test:ro" \
        "$ALPINE_IMAGE" "/test/runner" "$scenario"
done

echo "Testing installer scenario: real OpenRC lifecycle"
OPENRC_IMAGE="$(docker build -q \
    --build-arg "ALPINE_IMAGE=$ALPINE_IMAGE" \
    -f "$REPO_ROOT/docs/sliver-docs/tests/Dockerfile.openrc" \
    "$REPO_ROOT/docs/sliver-docs/tests")"
OPENRC_CONTAINER="sliver-install-openrc-test-$$-$RANDOM"
docker run -d \
    --name "$OPENRC_CONTAINER" \
    --network none \
    --tmpfs /run \
    --tmpfs /run/lock \
    -v "$INSTALL_SCRIPT:/install:ro" \
    -v "$TEST_ROOT:/test:ro" \
    "$OPENRC_IMAGE" > /dev/null

test "$(docker inspect -f '{{.HostConfig.Privileged}}' "$OPENRC_CONTAINER")" = "false"
test -z "$(docker port "$OPENRC_CONTAINER")"

openrc_ready=false
openrc_status=""
for _ in $(seq 1 30); do
    openrc_status="$(docker exec "$OPENRC_CONTAINER" rc-status 2>&1 || true)"
    if grep -q 'Runlevel: default' <<< "$openrc_status"; then
        openrc_ready=true
        break
    fi
    sleep 1
done
if [ "$openrc_ready" != true ]; then
    echo "OpenRC did not reach the default runlevel" >&2
    printf '%s\n' "$openrc_status" >&2
    docker logs "$OPENRC_CONTAINER" >&2 || true
    exit 1
fi

OPENRC_TEST_PATH="/test/openrc-real:/test/common:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
docker exec -e PATH="$OPENRC_TEST_PATH" "$OPENRC_CONTAINER" /bin/sh /install
docker exec "$OPENRC_CONTAINER" rc-service sliver status
docker exec "$OPENRC_CONTAINER" rc-update show default | grep -q sliver

first_pid="$(docker exec "$OPENRC_CONTAINER" cat /tmp/sliver-daemon.pid)"
docker exec "$OPENRC_CONTAINER" kill -9 "$first_pid"
respawned=false
for _ in $(seq 1 10); do
    current_pid="$(docker exec "$OPENRC_CONTAINER" cat /tmp/sliver-daemon.pid 2> /dev/null || true)"
    if [ -n "$current_pid" ] && [ "$current_pid" != "$first_pid" ] && \
        docker exec "$OPENRC_CONTAINER" kill -0 "$current_pid" 2> /dev/null; then
        respawned=true
        break
    fi
    sleep 1
done
test "$respawned" = true
if docker exec "$OPENRC_CONTAINER" kill -0 "$first_pid" 2> /dev/null; then
    echo "OpenRC left the killed Sliver process running" >&2
    exit 1
fi
docker exec "$OPENRC_CONTAINER" rc-service sliver status

pre_upgrade_pid="$current_pid"
docker exec -e PATH="$OPENRC_TEST_PATH" "$OPENRC_CONTAINER" /bin/sh /install
post_upgrade_pid="$(docker exec "$OPENRC_CONTAINER" cat /tmp/sliver-daemon.pid)"
test "$post_upgrade_pid" != "$pre_upgrade_pid"
if docker exec "$OPENRC_CONTAINER" kill -0 "$pre_upgrade_pid" 2> /dev/null; then
    echo "OpenRC left the pre-upgrade Sliver process running" >&2
    exit 1
fi
docker exec "$OPENRC_CONTAINER" rc-service sliver status

docker restart --timeout 10 "$OPENRC_CONTAINER" > /dev/null
openrc_restarted=false
for _ in $(seq 1 30); do
    reboot_pid="$(docker exec "$OPENRC_CONTAINER" cat /tmp/sliver-daemon.pid 2> /dev/null || true)"
    if [ -n "$reboot_pid" ] && [ "$reboot_pid" != "$post_upgrade_pid" ] && \
        docker exec "$OPENRC_CONTAINER" rc-service sliver status > /dev/null 2>&1; then
        openrc_restarted=true
        break
    fi
    sleep 1
done
test "$openrc_restarted" = true

echo "Installer regression tests passed"
