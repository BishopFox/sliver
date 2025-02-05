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

GO_VER="1.23.5"
GARBLE_VER="1.23.5"
ZIG_VER="0.13.0"
SGN_VER="0.0.3"

BLOAT_FILES="AUTHORS CONTRIBUTORS PATENTS VERSION favicon.ico robots.txt SECURITY.md CONTRIBUTING.md LICENSE README.md ./doc ./test ./api ./misc"

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

REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OUTPUT_DIR=$REPO_DIR/server/assets/fs
mkdir -p $OUTPUT_DIR
WORK_DIR=`mktemp -d`

echo "-----------------------------------------------------------------"
echo "$WORK_DIR (Output: $OUTPUT_DIR)"
echo "-----------------------------------------------------------------"
cd $WORK_DIR

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
echo "curl -L --fail --output $OUTPUT_DIR/darwin/amd64/zig https://ziglang.org/download/$ZIG_VER/zig-macos-x86_64-$ZIG_VER.tar.xz"
curl -L --fail --output $OUTPUT_DIR/darwin/amd64/zig.tar.xz https://ziglang.org/download/$ZIG_VER/zig-macos-x86_64-$ZIG_VER.tar.xz                                                
echo "curl -L --fail --output $OUTPUT_DIR/darwin/arm64/zig https://ziglang.org/download/$ZIG_VER/zig-macos-aarch64-$ZIG_VER.tar.xz"
curl -L --fail --output $OUTPUT_DIR/darwin/arm64/zig.tar.xz https://ziglang.org/download/$ZIG_VER/zig-macos-aarch64-$ZIG_VER.tar.xz
echo "curl -L --fail --output $OUTPUT_DIR/linux/amd64/zig https://ziglang.org/download/$ZIG_VER/zig-linux-x86_64-$ZIG_VER.tar.xz"
curl -L --fail --output $OUTPUT_DIR/linux/amd64/zig.tar.xz https://ziglang.org/download/$ZIG_VER/zig-linux-x86_64-$ZIG_VER.tar.xz
echo "curl -L --fail --output $OUTPUT_DIR/linux/arm64/zig https://ziglang.org/download/$ZIG_VER/zig-linux-aarch64-$ZIG_VER.tar.xz"
curl -L --fail --output $OUTPUT_DIR/linux/arm64/zig.tar.xz https://ziglang.org/download/$ZIG_VER/zig-linux-aarch64-$ZIG_VER.tar.xz
# Of course Windows has to be different, because it's awful (zip file instead of a tarball)
echo "curl -L --fail --output $OUTPUT_DIR/windows/amd64/zig.zip https://ziglang.org/download/$ZIG_VER/zig-windows-x86_64-$ZIG_VER.zip"
curl -L --fail --output $OUTPUT_DIR/windows/amd64/zig.zip https://ziglang.org/download/$ZIG_VER/zig-windows-x86_64-$ZIG_VER.zip


echo "-----------------------------------------------------------------"
echo " Garble"
echo "-----------------------------------------------------------------"
echo "curl -L --fail --output $OUTPUT_DIR/linux/amd64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux"
curl -L --fail --output $OUTPUT_DIR/linux/amd64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux
echo "curl -L --fail --output $OUTPUT_DIR/linux/arm64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux-arm64"
curl -L --fail --output $OUTPUT_DIR/linux/arm64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux-arm64
echo "curl -L --fail --output $OUTPUT_DIR/windows/amd64/garble.exe https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_windows.exe"
curl -L --fail --output $OUTPUT_DIR/windows/amd64/garble.exe https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_windows.exe
echo "curl -L --fail --output $OUTPUT_DIR/darwin/amd64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_macos-amd64"
curl -L --fail --output $OUTPUT_DIR/darwin/amd64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_macos-amd64
echo "curl -L --fail --output $OUTPUT_DIR/darwin/arm64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_macos-arm64"
curl -L --fail --output $OUTPUT_DIR/darwin/arm64/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_macos-arm64


echo "-----------------------------------------------------------------"
echo " Shikata ga nai (ノ ゜Д゜)ノ ︵ 仕方がない"
echo "-----------------------------------------------------------------"
# Linux (amd64)
echo "curl -L --fail --output $OUTPUT_DIR/linux/amd64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_linux-amd64.zip"
curl -L --fail --output $OUTPUT_DIR/linux/amd64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_linux-amd64.zip
# Linux (arm64)
echo "curl -L --fail --output $OUTPUT_DIR/linux/arm64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_linux-arm64.zip"
curl -L --fail --output $OUTPUT_DIR/linux/arm64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_linux-arm64.zip
# Windows (amd64)
echo "curl -L --fail --output $OUTPUT_DIR/windows/amd64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_windows-amd64.zip"
curl -L --fail --output $OUTPUT_DIR/windows/amd64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_windows-amd64.zip
# MacOS (amd64)
echo "curl -L --fail --output $OUTPUT_DIR/darwin/amd64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_macos-amd64.zip"
curl -L --fail --output $OUTPUT_DIR/darwin/amd64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_macos-amd64.zip
# MacOS (arm64)
echo "curl -L --fail --output $OUTPUT_DIR/darwin/arm64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_macos-arm64.zip"
curl -L --fail --output $OUTPUT_DIR/darwin/arm64/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_macos-arm64.zip

# --- Cleanup ---
echo -e "clean up: $WORK_DIR"
rm -rf $WORK_DIR
echo -e "\n[*] All done\n"
