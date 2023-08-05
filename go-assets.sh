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

GO_VER="1.20.7"
GARBLE_VER="1.20.7"
SGN_VER="0.0.3"

GO_ARCH_1="amd64"
GO_ARCH_2="arm64"
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
curl --output go$GO_VER.darwin-$GO_ARCH_1.tar.gz https://dl.google.com/go/go$GO_VER.darwin-$GO_ARCH_1.tar.gz
tar xvf go$GO_VER.darwin-$GO_ARCH_1.tar.gz

cd go
rm -rf $BLOAT_FILES
zip -r ../src.zip ./src  # Zip up /src we only need to do this once
rm -rf ./src
rm -f ./pkg/tool/darwin_$GO_ARCH_1/doc
rm -f ./pkg/tool/darwin_$GO_ARCH_1/tour
rm -f ./pkg/tool/darwin_$GO_ARCH_1/test2json
cd ..
cp -vv src.zip $OUTPUT_DIR/src.zip
rm -f src.zip

zip -r darwin-go.zip ./go
mkdir -p $OUTPUT_DIR/darwin/$GO_ARCH_1
cp -vv darwin-go.zip $OUTPUT_DIR/darwin/$GO_ARCH_1/go.zip

rm -rf ./go
rm -f darwin-go.zip go$GO_VER.darwin-$GO_ARCH_1.tar.gz

# --- Darwin (arm64) --- 
curl --output go$GO_VER.darwin-$GO_ARCH_2.tar.gz https://dl.google.com/go/go$GO_VER.darwin-$GO_ARCH_2.tar.gz
tar xvf go$GO_VER.darwin-$GO_ARCH_2.tar.gz

cd go
rm -rf $BLOAT_FILES
zip -r ../src.zip ./src  # Zip up /src we only need to do this once
rm -rf ./src
rm -f ./pkg/tool/darwin_$GO_ARCH_2/doc
rm -f ./pkg/tool/darwin_$GO_ARCH_2/tour
rm -f ./pkg/tool/darwin_$GO_ARCH_2/test2json
cd ..
cp -vv src.zip $OUTPUT_DIR/src.zip
rm -f src.zip

zip -r darwin-go.zip ./go
mkdir -p $OUTPUT_DIR/darwin/$GO_ARCH_2
cp -vv darwin-go.zip $OUTPUT_DIR/darwin/$GO_ARCH_2/go.zip

rm -rf ./go
rm -f darwin-go.zip go$GO_VER.darwin-$GO_ARCH_2.tar.gz

# --- Linux (amd64) --- 
curl --output go$GO_VER.linux-$GO_ARCH_1.tar.gz https://dl.google.com/go/go$GO_VER.linux-$GO_ARCH_1.tar.gz
tar xvf go$GO_VER.linux-$GO_ARCH_1.tar.gz
cd go
rm -rf $BLOAT_FILES
rm -rf ./src
rm -f ./pkg/tool/linux_$GO_ARCH_1/doc
rm -f ./pkg/tool/linux_$GO_ARCH_1/tour
rm -f ./pkg/tool/linux_$GO_ARCH_1/test2json
cd ..
zip -r linux-go.zip ./go
mkdir -p $OUTPUT_DIR/linux/$GO_ARCH_1
cp -vv linux-go.zip $OUTPUT_DIR/linux/$GO_ARCH_1/go.zip
rm -rf ./go
rm -f linux-go.zip go$GO_VER.linux-$GO_ARCH_1.tar.gz

# --- Linux (arm64) --- 
curl --output go$GO_VER.linux-$GO_ARCH_2.tar.gz https://dl.google.com/go/go$GO_VER.linux-$GO_ARCH_2.tar.gz
tar xvf go$GO_VER.linux-$GO_ARCH_2.tar.gz
cd go
rm -rf $BLOAT_FILES
rm -rf ./src
rm -f ./pkg/tool/linux_$GO_ARCH_2/doc
rm -f ./pkg/tool/linux_$GO_ARCH_2/tour
rm -f ./pkg/tool/linux_$GO_ARCH_2/test2json
cd ..
zip -r linux-go.zip ./go
mkdir -p $OUTPUT_DIR/linux/$GO_ARCH_2
cp -vv linux-go.zip $OUTPUT_DIR/linux/$GO_ARCH_2/go.zip
rm -rf ./go
rm -f linux-go.zip go$GO_VER.linux-$GO_ARCH_2.tar.gz

# --- Windows --- 
curl --output go$GO_VER.windows-amd64.zip https://dl.google.com/go/go$GO_VER.windows-$GO_ARCH_1.zip
unzip go$GO_VER.windows-amd64.zip
cd go
rm -rf $BLOAT_FILES
rm -rf ./src
rm -f ./pkg/tool/windows_$GO_ARCH_1/doc.exe
rm -f ./pkg/tool/windows_$GO_ARCH_1/tour.exe
rm -f ./pkg/tool/windows_$GO_ARCH_1/test2json.exe
cd ..
zip -r windows-go.zip ./go
mkdir -p $OUTPUT_DIR/windows/$GO_ARCH_1
cp -vv windows-go.zip $OUTPUT_DIR/windows/$GO_ARCH_1/go.zip
rm -rf ./go
rm -f windows-go.zip go$GO_VER.windows-$GO_ARCH_1.zip

echo "-----------------------------------------------------------------"
echo " Garble"
echo "-----------------------------------------------------------------"

echo "curl -L --fail --output $OUTPUT_DIR/linux/$GO_ARCH_1/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux"
curl -L --fail --output $OUTPUT_DIR/linux/$GO_ARCH_1/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux
echo "curl -L --fail --output $OUTPUT_DIR/linux/$GO_ARCH_2/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux-$GO_ARCH_2"
curl -L --fail --output $OUTPUT_DIR/linux/$GO_ARCH_2/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_linux-$GO_ARCH_2
echo "curl -L --fail --output $OUTPUT_DIR/windows/$GO_ARCH_1/garble.exe https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_windows.exe"
curl -L --fail --output $OUTPUT_DIR/windows/$GO_ARCH_1/garble.exe https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_windows.exe
echo "curl -L --fail --output $OUTPUT_DIR/darwin/$GO_ARCH_1/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_macos-$GO_ARCH_1"
curl -L --fail --output $OUTPUT_DIR/darwin/$GO_ARCH_1/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_macos-$GO_ARCH_1
echo "curl -L --fail --output $OUTPUT_DIR/darwin/$GO_ARCH_2/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_macos-$GO_ARCH_2"
curl -L --fail --output $OUTPUT_DIR/darwin/$GO_ARCH_2/garble https://github.com/moloch--/garble/releases/download/v$GARBLE_VER/garble_macos-$GO_ARCH_2


echo "-----------------------------------------------------------------"
echo " Shikata ga nai (ノ ゜Д゜)ノ ︵ 仕方がない"
echo "-----------------------------------------------------------------"

# Linux (amd64)
echo "curl -L --fail --output $OUTPUT_DIR/linux/$GO_ARCH_1/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_linux-$GO_ARCH_1.zip"
curl -L --fail --output $OUTPUT_DIR/linux/$GO_ARCH_1/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_linux-$GO_ARCH_1.zip

# Linux (arm64)
echo "curl -L --fail --output $OUTPUT_DIR/linux/$GO_ARCH_2/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_linux-$GO_ARCH_2.zip"
curl -L --fail --output $OUTPUT_DIR/linux/$GO_ARCH_2/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_linux-$GO_ARCH_2.zip

# Windows (amd64)
echo "curl -L --fail --output $OUTPUT_DIR/windows/$GO_ARCH_1/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_windows-$GO_ARCH_1.zip"
curl -L --fail --output $OUTPUT_DIR/windows/$GO_ARCH_1/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_windows-$GO_ARCH_1.zip

# MacOS (amd64)
echo "curl -L --fail --output $OUTPUT_DIR/darwin/$GO_ARCH_1/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_macos-$GO_ARCH_1.zip"
curl -L --fail --output $OUTPUT_DIR/darwin/$GO_ARCH_1/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_macos-$GO_ARCH_1.zip

# MacOS (arm64)
echo "curl -L --fail --output $OUTPUT_DIR/darwin/$GO_ARCH_2/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_macos-$GO_ARCH_2.zip"
curl -L --fail --output $OUTPUT_DIR/darwin/$GO_ARCH_2/sgn.zip https://github.com/moloch--/sgn/releases/download/v$SGN_VER/sgn_macos-$GO_ARCH_2.zip

# end
echo -e "clean up: $WORK_DIR"
rm -rf $WORK_DIR
echo -e "\n[*] All done\n"
