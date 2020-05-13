#
# Makefile for Sliver
#

GO ?= go
ENV = CGO_ENABLED=0
TAGS = -tags netgo


#
# Prerequisites 
#
# https://stackoverflow.com/questions/5618615/check-if-a-program-exists-from-a-makefile
EXECUTABLES = protoc protoc-gen-go packr uname sed git zip go date
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

SED_INPLACE := sed -i

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	SED_INPLACE := sed -i ''
endif


#
# Version Information
#
VERSION = 1.0.0
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
.PHONY: macos
macos: clean pb
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: linux
linux: clean pb
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: windows
windows: clean pb
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-client.exe ./client


#
# Static Targets
#
.PHONY: static-macos
static-macos: clean pb packr
	packr
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: static-windows
static-windows: clean pb packr
	packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server

.PHONY: static-linux
static-linux: clean pb packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

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
	rm -f ./client/version/version.go
	rm -f sliver-client sliver-server *.exe

