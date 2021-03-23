#
# Makefile for Sliver
#

GO ?= go
ENV = CGO_ENABLED=1
TAGS = -tags osusergo,netgo,sqlite_omit_load_extension

#
# Version Information
#
GO_VERSION = $(shell $(GO) version)
VERSION = $(shell git describe --abbrev=0)
COMPILED_AT = $(shell date +%s)
RELEASES_URL = https://api.github.com/repos/BishopFox/sliver/releases
PKG = github.com/bishopfox/sliver/client/version
GIT_DIRTY = $(shell git diff --quiet|| echo 'Dirty')
GIT_COMMIT = $(shell git rev-parse HEAD)
LDFLAGS = -ldflags "-s -w \
	-X $(PKG).Version=$(VERSION) \
	-X \"$(PKG).GoVersion=$(GO_VERSION)\" \
	-X $(PKG).CompiledAt=$(COMPILED_AT) \
	-X $(PKG).GithubReleasesURL=$(RELEASES_URL) \
	-X $(PKG).GitCommit=$(GIT_COMMIT) \
	-X $(PKG).GitDirty=$(GIT_DIRTY)"


#
# Prerequisites 
#
# https://stackoverflow.com/questions/5618615/check-if-a-program-exists-from-a-makefile
EXECUTABLES = protoc protoc-gen-go uname sed git zip go date
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

SED_INPLACE := sed -i
STATIC_TARGET := linux

UNAME_S := $(shell uname -s)
UNAME_P := $(shell uname -p)

# If the target is Windows from Linux/Darwin, check for mingw
CROSS_COMPILERS = x86_64-w64-mingw32-gcc x86_64-w64-mingw32-g++

# *** Start Darwin ***
ifeq ($(UNAME_S),Darwin)
	SED_INPLACE := sed -i ''
	STATIC_TARGET := macos

ifeq ($(UNAME_P),arm)
	ENV += GOARCH=arm64
endif

ifeq ($(MAKECMDGOALS), windows)
	K := $(foreach exec,$(CROSS_COMPILERS),\
			$(if $(shell which $(exec)),some string,$(error "Missing cross-compiler $(exec) in PATH")))
	ENV += CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++
endif

endif
# *** End Darwin ***

# *** Start Linux ***
ifeq ($(UNAME_S),Linux)
ifeq ($(MAKECMDGOALS), windows)
	K := $(foreach exec,$(CROSS_COMPILERS),\
			$(if $(shell which $(exec)),some string,$(error "Missing cross-compiler $(exec) in PATH")))
	ENV += CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++
endif
endif

ifeq ($(MAKECMDGOALS), linux)
	# Redefine LDFLAGS to add the static part
	LDFLAGS = -ldflags "-s -w \
		-extldflags '-static' \
		-X $(PKG).Version=$(VERSION) \
		-X \"$(PKG).GoVersion=$(GO_VERSION)\" \
		-X $(PKG).CompiledAt=$(COMPILED_AT) \
		-X $(PKG).GithubReleasesURL=$(RELEASES_URL) \
		-X $(PKG).GitCommit=$(GIT_COMMIT) \
		-X $(PKG).GitDirty=$(GIT_DIRTY)"
endif
# *** End Linux ***

#
# Targets
#
.PHONY: default
default: clean pb
	$(ENV) $(GO) build -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server ./server
	$(ENV) $(GO) build -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client ./client

.PHONY: macos
macos: clean pb
	GOOS=darwin GOARCH=amd64 $(ENV) $(GO) build -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server ./server
	GOOS=darwin GOARCH=amd64 $(ENV) $(GO) build -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client ./client

.PHONY: macos-arm64
macos-arm64: clean pb
	GOOS=darwin GOARCH=arm64 $(ENV) $(GO) build -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server_arm64 ./server
	GOOS=darwin GOARCH=arm64 $(ENV) $(GO) build -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_arm64 ./client

.PHONY: linux
linux: clean pb
	GOOS=linux $(ENV) $(GO) build -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server ./server
	GOOS=linux $(ENV) $(GO) build -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client ./client

.PHONY: windows
windows: clean pb
	GOOS=windows $(ENV) $(GO) build -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server.exe ./server
	GOOS=windows $(ENV) $(GO) build -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client.exe ./client

.PHONY: pb
pb:
	protoc -I protobuf/ protobuf/commonpb/common.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/sliverpb/sliver.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/clientpb/client.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/rpcpb/services.proto --go_out=plugins=grpc,paths=source_relative:protobuf/

.PHONY: clean-all
clean-all: clean
	rm -rf ./server/assets/fs/darwin/amd64
	rm -rf ./server/assets/fs/darwin/arm64
	rm -rf ./server/assets/fs/windows/amd64
	rm -rf ./server/assets/fs/linux/amd64
	rm -f ./server/assets/fs/*.zip

.PHONY: clean
clean:
	rm -f ./protobuf/client/*.pb.go
	rm -f ./protobuf/sliver/*.pb.go
	rm -f sliver-client_arm64 sliver-server_arm64
	rm -f sliver-client sliver-server *.exe

