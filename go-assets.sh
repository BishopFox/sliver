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
# You'll need wget, tar, and unzip commands

GO_VER="1.12.5"
BLOAT_FILES="AUTHORS CONTRIBUTORS PATENTS VERSION favicon.ico robots.txt CONTRIBUTING.md LICENSE README.md ./doc ./test"


REPO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
WORK_DIR=`mktemp -d`

echo "-----------------------------------------------------------------"
echo $WORK_DIR
echo "-----------------------------------------------------------------"
cd $WORK_DIR
mkdir -p $REPO_DIR/assets/

# --- Darwin --- 
wget https://dl.google.com/go/go$GO_VER.darwin-amd64.tar.gz
tar xvf go$GO_VER.darwin-amd64.tar.gz

cd go
rm -rf $BLOAT_FILES
zip -r ../src.zip ./src  # Zip up /src we only need to do this once
rm -rf ./src
rm -f ./pkg/tool/darwin_amd64/doc
rm -f ./pkg/tool/darwin_amd64/tour
rm -f ./pkg/tool/darwin_amd64/test2json
cd ..
cp -vv src.zip $REPO_DIR/assets/src.zip
rm -f src.zip

zip -r darwin-go.zip ./go
mkdir -p $REPO_DIR/assets/darwin/
cp -vv darwin-go.zip $REPO_DIR/assets/darwin/go.zip

rm -rf ./go
rm -f darwin-go.zip go$GO_VER.darwin-amd64.tar.gz


# --- Linux --- 
wget https://dl.google.com/go/go$GO_VER.linux-amd64.tar.gz
tar xvf go$GO_VER.linux-amd64.tar.gz
cd go
rm -rf $BLOAT_FILES
rm -rf ./src
rm -f ./pkg/tool/linux_amd64/doc
rm -f ./pkg/tool/linux_amd64/tour
rm -f ./pkg/tool/linux_amd64/test2json
cd ..
zip -r linux-go.zip ./go
mkdir -p $REPO_DIR/assets/linux/
cp -vv linux-go.zip $REPO_DIR/assets/linux/go.zip
rm -rf ./go
rm -f linux-go.zip go$GO_VER.linux-amd64.tar.gz

# --- Windows --- 
wget https://dl.google.com/go/go$GO_VER.windows-amd64.zip
unzip go$GO_VER.windows-amd64.zip
cd go
rm -rf $BLOAT_FILES
rm -rf ./src
rm -f ./pkg/tool/windows_amd64/doc.exe
rm -f ./pkg/tool/windows_amd64/tour.exe
rm -f ./pkg/tool/windows_amd64/test2json.exe
cd ..
zip -r windows-go.zip ./go
mkdir -p $REPO_DIR/assets/windows/
cp -vv windows-go.zip $REPO_DIR/assets/windows/go.zip
rm -rf ./go
rm -f windows-go.zip go$GO_VER.windows-amd64.zip


echo "-----------------------------------------------------------------"
echo " 3rd Party Assets"
echo "-----------------------------------------------------------------"
cd $WORK_DIR

PROTOBUF_COMMIT=347cf4a86c1cb8d262994d8ef5924d4576c5b331
wget https://github.com/golang/protobuf/archive/$PROTOBUF_COMMIT.zip
unzip $PROTOBUF_COMMIT.zip
rm -f $PROTOBUF_COMMIT.zip
mv protobuf-$PROTOBUF_COMMIT protobuf
zip -r protobuf.zip ./protobuf
cp -vv protobuf.zip $REPO_DIR/assets/protobuf.zip

wget https://go.googlesource.com/sys/+archive/master.tar.gz
mkdir $WORK_DIR/sys
cd $WORK_DIR/sys
tar xfv ../master.tar.gz
rm -rf ../master.tar.gz
cd ..
zip -r $REPO_DIR/assets/golang_x_sys.zip sys

# end
echo -e "clean up: $WORK_DIR"
rm -rf $WORK_DIR
echo -e "\n[*] All done\n"
