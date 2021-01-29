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


# Creates the static go asset archives

GO_VER="1.16beta1"
GO_ARCH_1="amd64"
GO_ARCH_2="arm64"
BLOAT_FILES="AUTHORS CONTRIBUTORS PATENTS VERSION favicon.ico robots.txt CONTRIBUTING.md LICENSE README.md ./doc ./test ./api ./misc"

PROTOBUF_COMMIT=347cf4a86c1cb8d262994d8ef5924d4576c5b331
GOLANG_SYS_COMMIT=669c56c373c468cbe0f0c12b7939832b26088d33

if ! [ -x "$(command -v curl)" ]; then
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

# --- Linux --- 
curl --output go$GO_VER.linux-amd64.tar.gz https://dl.google.com/go/go$GO_VER.linux-$GO_ARCH_1.tar.gz
tar xvf go$GO_VER.linux-amd64.tar.gz
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
echo " 3rd Party Assets"
echo "-----------------------------------------------------------------"
cd $WORK_DIR

curl -L --output $PROTOBUF_COMMIT.zip https://github.com/golang/protobuf/archive/$PROTOBUF_COMMIT.zip
unzip $PROTOBUF_COMMIT.zip
rm -f $PROTOBUF_COMMIT.zip
mv protobuf-$PROTOBUF_COMMIT protobuf
zip -r protobuf.zip ./protobuf
cp -vv protobuf.zip $OUTPUT_DIR/protobuf.zip

curl -L --output $GOLANG_SYS_COMMIT.tar.gz https://github.com/golang/sys/archive/$GOLANG_SYS_COMMIT.tar.gz
tar xfv $GOLANG_SYS_COMMIT.tar.gz
rm -f $GOLANG_SYS_COMMIT.tar.gz
mv sys-$GOLANG_SYS_COMMIT sys
zip -r $OUTPUT_DIR/golang_x_sys.zip sys

curl -L --output $GOLANG_CRYPTO_SSH_COMMIT.tar.gz https://github.com/golang/crypto/archive/$GOLANG_CRYPTO_SSH_COMMIT.tar.gz
tar xfv $GOLANG_CRYPTO_SSH_COMMIT.tar.gz
rm -f $GOLANG_CRYPTO_SSH_COMMIT.tar.gz
mv crypto-$GOLANG_CRYPTO_SSH_COMMIT crypto
zip -r $OUTPUT_DIR/assets/golang_x_crypto_ssh.zip crypto

# end
echo -e "clean up: $WORK_DIR"
rm -rf $WORK_DIR
echo -e "\n[*] All done\n"
