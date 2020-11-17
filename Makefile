#
# Makefile for Sliver
#

GO ?= go
ENV = CGO_ENABLED=1
TAGS = -tags osusergo,netgo,sqlite_omit_load_extension


#
# Prerequisites 
#
# https://stackoverflow.com/questions/5618615/check-if-a-program-exists-from-a-makefile
EXECUTABLES = protoc protoc-gen-go packr uname sed git zip go date
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

SED_INPLACE := sed -i
STATIC_TARGET := static-linux

UNAME_S := $(shell uname -s)

# If the target is Windows from Linux/Darwin, check for mingw
CROSS_COMPILERS = x86_64-w64-mingw32-gcc x86_64-w64-mingw32-g++

# *** Start Darwin ***
ifeq ($(UNAME_S),Darwin)
	SED_INPLACE := sed -i ''
	STATIC_TARGET := static-macos

ifeq ($(MAKECMDGOALS), windows)
	K := $(foreach exec,$(CROSS_COMPILERS),\
			$(if $(shell which $(exec)),some string,$(error "Missing cross-compiler $(exec) in PATH")))
	ENV += CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++
endif
ifeq ($(MAKECMDGOALS), static-windows)
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
ifeq ($(MAKECMDGOALS), static-windows)
	K := $(foreach exec,$(CROSS_COMPILERS),\
			$(if $(shell which $(exec)),some string,$(error "Missing cross-compiler $(exec) in PATH")))
	ENV += CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++
endif

endif 
# *** End Linux ***

#
# Version Information
#
VERSION = 1.1.0
COMPILED_AT = $(shell date +%s)
RELEASES_URL = https://api.github.com/repos/BishopFox/sliver/releases
PKG = github.com/bishopfox/sliver/client/version
GIT_DIRTY = $(shell git diff --quiet|| echo 'Dirty')
GIT_COMMIT = $(shell git rev-parse HEAD)
LDFLAGS = -ldflags "-s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).CompiledAt=$(COMPILED_AT) \
	-X $(PKG).GithubReleasesURL=$(RELEASES_URL) \
	-X $(PKG).GitCommit=$(GIT_COMMIT) \
	-X $(PKG).GitDirty=$(GIT_DIRTY)"


#
# Targets
#
.PHONY: default
default: clean pb
	$(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-server ./server
	$(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: macos
macos: clean pb
	GOOS=darwin $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-server ./server
	GOOS=darwin $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: linux
linux: clean pb
	GOOS=linux $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-server ./server
	GOOS=linux $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: windows
windows: clean pb
	GOOS=windows $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server
	GOOS=windows $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-client.exe ./client


#
# Static Targets
#
.PHONY: static
static: $(STATIC_TARGET)

.PHONY: static-macos
static-macos: clean pb packr
	packr
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=darwin $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-server ./server
	GOOS=darwin $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: static-windows
static-windows: clean pb packr
	packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=windows $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server
	GOOS=windows $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-client.exe ./client

.PHONY: static-linux
static-linux: clean pb packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=linux $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-server ./server
	GOOS=linux $(ENV) $(GO) build -trimpath $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: pb
pb:
	protoc -I protobuf/ protobuf/commonpb/common.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/sliverpb/sliver.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/clientpb/client.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/rpcpb/services.proto --go_out=plugins=grpc,paths=source_relative:protobuf/

.PHONY: packr
packr:
	cd ./server/
	packr
	cd ..

.PHONY: clean-all
clean-all: clean
	rm -f ./assets/darwin/go.zip
	rm -f ./assets/windows/go.zip
	rm -f ./assets/linux/go.zip
	rm -f ./assets/*.zip

.PHONY: clean
clean:
	packr clean
	rm -f ./protobuf/client/*.pb.go
	rm -f ./protobuf/sliver/*.pb.go
	rm -f sliver-client sliver-server *.exe

