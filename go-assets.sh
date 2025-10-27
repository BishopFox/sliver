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

set -e

# Creates the static go asset archives

GO_VER="1.25.3"
GARBLE_VER="1.25.3"
ZIG_VER="0.15.1"

# Zig significantly throttles downloads from the main site, so we use
# community mirrors. We fetch the list of mirrors at runtime, but
# fall back to a set of default mirrors if the fetch fails.
# We also use the official public key for verifying Zig downloads from mirrors.
ZIG_MINISIGN_PUBKEY="RWSGOq2NVecA2UPNdBUZykf1CCb147pkmdtYxgb3Ti+JO/wCYvhbAb/U"
DEFAULT_ZIG_MIRRORS=(
  "https://pkg.machengine.org/zig"
  "https://zigmirror.hryx.net/zig"
  "https://zig.linus.dev/zig"
  "https://zig.squirl.dev"
  "https://zig.florent.dev"
  "https://zig.mirror.mschae23.de/zig"
  "https://zigmirror.meox.dev"
)
ZIG_MIRRORS=()
MINISIGN_HELPER=""
MINISIGN_HELPER_DIR=""

BLOAT_FILES="AUTHORS CONTRIBUTORS PATENTS VERSION favicon.ico robots.txt SECURITY.md CONTRIBUTING.md README.md ./doc ./test ./api ./misc"

if ! [ -x "$(command -v curl)" ]; then
  echo 'Error: curl is not installed.' >&2
  exit 1
fi

if ! [ -x "$(command -v zip)" ]; then
  echo 'Error: zip is not installed.' >&2
  exit 1
fi

if ! [ -x "$(command -v unzip)" ]; then
  echo 'Error: unzip is not installed.' >&2
  exit 1
fi

if ! [ -x "$(command -v tar)" ]; then
  echo 'Error: tar is not installed.' >&2
  exit 1
fi

if ! [ -x "$(command -v go)" ]; then
  echo 'Error: go is not installed.' >&2
  exit 1
fi

REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OUTPUT_DIR=$REPO_DIR/server/assets/fs
mkdir -p $OUTPUT_DIR
WORK_DIR=`mktemp -d`

echo "-----------------------------------------------------------------"
echo "$WORK_DIR (Output: $OUTPUT_DIR)"
echo "-----------------------------------------------------------------"
cd $WORK_DIR

# Load official Zig mirror list with a fallback to bundled defaults.
load_zig_mirrors() {
  local mirrors_text
  local -a mirrors=()

  if mirrors_text=$(curl -sS -L --fail https://ziglang.org/download/community-mirrors.txt 2>/dev/null); then
    while IFS= read -r line; do
      line="${line%%#*}"
      line="$(printf '%s' "$line" | sed -e 's/[[:space:]]*$//' -e 's/^[[:space:]]*//')"
      if [[ -n "$line" ]]; then
        mirrors+=("$line")
      fi
    done <<< "$mirrors_text"
  fi

  if [ "${#mirrors[@]}" -eq 0 ]; then
    mirrors=("${DEFAULT_ZIG_MIRRORS[@]}")
  fi

  ZIG_MIRRORS=("${mirrors[@]}")
}

# Shuffle mirror list for each download attempt to spread load.
randomized_mirrors() {
  if [ "${#ZIG_MIRRORS[@]}" -eq 0 ]; then
    load_zig_mirrors
  fi
  local -a shuffled=("${ZIG_MIRRORS[@]}")
  local i j temp
  for ((i=${#shuffled[@]}-1; i>0; i--)); do
    j=$((RANDOM % (i + 1)))
    temp=${shuffled[i]}
    shuffled[i]=${shuffled[j]}
    shuffled[j]=$temp
  done
  printf '%s\n' "${shuffled[@]}"
}

# Build a temporary helper that verifies signatures via util/minisign.
ensure_minisign_helper() {
  if [ -n "$MINISIGN_HELPER" ] && [ -x "$MINISIGN_HELPER" ]; then
    return
  fi

  local helper_dir
  helper_dir=$(mktemp -d "$REPO_DIR/.zig-verify.XXXXXX")

  if ! (cd "$REPO_DIR" && GO111MODULE=on GOFLAGS=-mod=vendor go build -o "$helper_dir/zig-verify" ./util/cmd/zigverify); then
    echo 'Error: unable to build minisign verification helper.' >&2
    exit 1
  fi

  MINISIGN_HELPER="$helper_dir/zig-verify"
  MINISIGN_HELPER_DIR="$helper_dir"
  trap 'rm -rf "$MINISIGN_HELPER_DIR"' EXIT
}

verify_minisig() {
  ensure_minisign_helper
  ZIG_PUBLIC_KEY="$ZIG_MINISIGN_PUBKEY" "$MINISIGN_HELPER" "$1" "$2"
}

download_zig() {
  local platform="$1"
  local arch="$2"
  local remote_name="$3"
  local local_name="$4"
  local dest_dir="$OUTPUT_DIR/$platform/$arch"
  local dest_path="$dest_dir/$local_name"
  local minisig_url="https://ziglang.org/download/$ZIG_VER/$remote_name.minisig"
  local mirror_url
  local success=0

  mkdir -p "$dest_dir"
  rm -f "$dest_path"

  while read -r mirror; do
    mirror_url="${mirror%/}/$ZIG_VER/$remote_name"
    echo "Attempting Zig download from $mirror_url"
    local tmp_tar
    local tmp_sig
    tmp_tar=$(mktemp)
    tmp_sig=$(mktemp)
    local verification_failed=0
    if curl -L --fail --output "$tmp_tar" "$mirror_url"; then
      if curl -L --fail --output "$tmp_sig" "$minisig_url"; then
        if verify_minisig "$tmp_tar" "$tmp_sig"; then
          mv "$tmp_tar" "$dest_path"
          rm -f "$tmp_sig"
          echo "Downloaded and verified Zig package -> $dest_path"
          success=1
          break
        else
          echo "[!] Signature verification failed for $mirror_url" >&2
          verification_failed=1
        fi
      fi
    fi
    if [ "$verification_failed" -eq 1 ]; then
      echo "[!] Deleting corrupted download $tmp_tar" >&2
    fi
    rm -f "$tmp_tar" "$tmp_sig"
    echo "[!] Failed to download or verify Zig from $mirror_url" >&2
  done < <(randomized_mirrors)

  if [ "$success" -ne 1 ]; then
    echo "[!] Error: unable to download and verify Zig package $remote_name" >&2
    exit 1
  fi
}

# --- Darwin (amd64) --- 
curl --output go$GO_VER.darwin-amd64.tar.gz https://dl.google.com/go/go$GO_VER.darwin-amd64.tar.gz
tar xvf go$GO_VER.darwin-amd64.tar.gz

cd go
rm -rf $BLOAT_FILES
zip -r ../src.zip ./src  # Zip up /src we only need to do this once
rm -rf ./src
rm -f ./pkg/tool/darwin_amd64/doc
rm -f ./pkg/tool/darwin_amd64/tour
rm -f ./pkg/tool/darwin_amd64/test2json
cd ..
cp -vv src.zip $OUTPUT_DIR/src.zip
rm -f src.zip

zip -r darwin-go.zip ./go
mkdir -p $OUTPUT_DIR/darwin/amd64
cp -vv darwin-go.zip $OUTPUT_DIR/darwin/amd64/go.zip

rm -rf ./go
rm -f darwin-go.zip go$GO_VER.darwin-amd64.tar.gz

# --- Darwin (arm64) --- 
curl --output go$GO_VER.darwin-arm64.tar.gz https://dl.google.com/go/go$GO_VER.darwin-arm64.tar.gz
tar xvf go$GO_VER.darwin-arm64.tar.gz

cd go
rm -rf $BLOAT_FILES
zip -r ../src.zip ./src  # Zip up /src we only need to do this once
rm -rf ./src
rm -f ./pkg/tool/darwin_arm64/doc
rm -f ./pkg/tool/darwin_arm64/tour
rm -f ./pkg/tool/darwin_arm64/test2json
cd ..
cp -vv src.zip $OUTPUT_DIR/src.zip
rm -f src.zip

zip -r darwin-go.zip ./go
mkdir -p $OUTPUT_DIR/darwin/arm64
cp -vv darwin-go.zip $OUTPUT_DIR/darwin/arm64/go.zip

rm -rf ./go
rm -f darwin-go.zip go$GO_VER.darwin-arm64.tar.gz

# --- Linux (amd64) --- 
curl --output go$GO_VER.linux-amd64.tar.gz https://dl.google.com/go/go$GO_VER.linux-amd64.tar.gz
tar xvf go$GO_VER.linux-amd64.tar.gz
cd go
rm -rf $BLOAT_FILES
rm -rf ./src
rm -f ./pkg/tool/linux_amd64/doc
rm -f ./pkg/tool/linux_amd64/tour
rm -f ./pkg/tool/linux_amd64/test2json
cd ..
zip -r linux-go.zip ./go
mkdir -p $OUTPUT_DIR/linux/amd64
cp -vv linux-go.zip $OUTPUT_DIR/linux/amd64/go.zip
rm -rf ./go
rm -f linux-go.zip go$GO_VER.linux-amd64.tar.gz

# --- Linux (arm64) --- 
curl --output go$GO_VER.linux-arm64.tar.gz https://dl.google.com/go/go$GO_VER.linux-arm64.tar.gz
tar xvf go$GO_VER.linux-arm64.tar.gz
cd go
rm -rf $BLOAT_FILES
rm -rf ./src
rm -f ./pkg/tool/linux_arm64/doc
rm -f ./pkg/tool/linux_arm64/tour
rm -f ./pkg/tool/linux_arm64/test2json
cd ..
zip -r linux-go.zip ./go
mkdir -p $OUTPUT_DIR/linux/arm64
cp -vv linux-go.zip $OUTPUT_DIR/linux/arm64/go.zip
rm -rf ./go
rm -f linux-go.zip go$GO_VER.linux-arm64.tar.gz

# --- Windows --- 
curl --output go$GO_VER.windows-amd64.zip https://dl.google.com/go/go$GO_VER.windows-amd64.zip
unzip go$GO_VER.windows-amd64.zip
cd go
rm -rf $BLOAT_FILES
rm -rf ./src
rm -f ./pkg/tool/windows_amd64/doc.exe
rm -f ./pkg/tool/windows_amd64/tour.exe
rm -f ./pkg/tool/windows_amd64/test2json.exe
cd ..
zip -r windows-go.zip ./go
mkdir -p $OUTPUT_DIR/windows/amd64
cp -vv windows-go.zip $OUTPUT_DIR/windows/amd64/go.zip
rm -rf ./go
rm -f windows-go.zip go$GO_VER.windows-amd64.zip

echo "-----------------------------------------------------------------"
echo " Zig"
echo "-----------------------------------------------------------------"
download_zig "darwin" "amd64" "zig-x86_64-macos-$ZIG_VER.tar.xz" "zig.tar.xz"
download_zig "darwin" "arm64" "zig-aarch64-macos-$ZIG_VER.tar.xz" "zig.tar.xz"
download_zig "linux" "amd64" "zig-x86_64-linux-$ZIG_VER.tar.xz" "zig.tar.xz"
download_zig "linux" "arm64" "zig-aarch64-linux-$ZIG_VER.tar.xz" "zig.tar.xz"
# Windows ships a zip instead of a tarball
download_zig "windows" "amd64" "zig-x86_64-windows-$ZIG_VER.zip" "zig.zip"

echo "-----------------------------------------------------------------"
echo " Garble"
echo "-----------------------------------------------------------------"
echo "curl -L --fail --output $OUTPUT_DIR/linux/amd64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux-amd64"
curl -L --fail --output $OUTPUT_DIR/linux/amd64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux-amd64
echo "curl -L --fail --output $OUTPUT_DIR/linux/arm64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux-arm64"
curl -L --fail --output $OUTPUT_DIR/linux/arm64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux-arm64
echo "curl -L --fail --output $OUTPUT_DIR/windows/amd64/garble.exe https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_windows-amd64.exe"
curl -L --fail --output $OUTPUT_DIR/windows/amd64/garble.exe https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_windows-amd64.exe
echo "curl -L --fail --output $OUTPUT_DIR/darwin/amd64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_darwin-amd64"
curl -L --fail --output $OUTPUT_DIR/darwin/amd64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_darwin-amd64
echo "curl -L --fail --output $OUTPUT_DIR/darwin/arm64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_darwin-arm64"
curl -L --fail --output $OUTPUT_DIR/darwin/arm64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_darwin-arm64

# --- Cleanup ---
echo -e "clean up: $WORK_DIR"
rm -rf $WORK_DIR
echo -e "\n[*] All done\n"
