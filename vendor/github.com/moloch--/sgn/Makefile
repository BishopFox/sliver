CURRENT_DIR=$(shell pwd)
BINARY=./build/sgn
BUILD=CGO_ENABLED=0 go build
OUT_DIR=${CURRENT_DIR}/build
BUILD_FLAGS=-trimpath -ldflags="-s -w -X github.com/moloch--/sgn/config.Version=`git describe --tags --always --dirty`" 

# Builds the project
default:
	${BUILD} ${BUILD_FLAGS} -o ${OUT_DIR}/
# Builds the project with full static linking
static:
	${BUILD} -o ${OUT_DIR}/

386:
	GOARCH=386 ${BUILD} ${BUILD_FLAGS} -o ${OUT_DIR}/
linux_amd64:
	GOOS=linux  GOARCH=amd64 ${BUILD} ${BUILD_FLAGS} -o ${OUT_DIR}/
linux_386:
	GOOS=linux  GOARCH=386 ${BUILD} ${BUILD_FLAGS} -o ${OUT_DIR}/
windows_amd64:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o ${OUT_DIR}/sgn.exe
windows_386:
	GOOS=windows GOARCH=386 go build -ldflags="-s -w" -trimpath -o ${OUT_DIR}/sgn32.exe
darwin_amd64:
	GOOS=darwin GOARCH=amd64 ${BUILD} ${BUILD_FLAGS} -o ${OUT_DIR}/ 

# Cleans our project: deletes binaries
clean:
	rm -rf ./build

.PHONY: clean install
