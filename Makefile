#
# Makefile for Sliver
#

GO ?= go
ARTIFACT_SUFFIX ?= 
ENV =
TAGS ?= -tags go_sqlite
CGO_ENABLED = 0

ifneq (,$(findstring cgo_sqlite,$(TAGS)))
	CGO_ENABLED = 1
endif

#
# Prerequisites 
#
# https://stackoverflow.com/questions/5618615/check-if-a-program-exists-from-a-makefile
EXECUTABLES = uname sed git zip date cut $(GO)
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

#
# Version Information
#
GO_VERSION = $(shell $(GO) version)
GO_MAJOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_MINOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
MIN_SUPPORTED_GO_MAJOR_VERSION = 1
MIN_SUPPORTED_GO_MINOR_VERSION = 21
GO_VERSION_VALIDATION_ERR_MSG = Golang version is not supported, please update to at least $(MIN_SUPPORTED_GO_MAJOR_VERSION).$(MIN_SUPPORTED_GO_MINOR_VERSION)

VERSION ?= $(shell git describe --abbrev=0)
COMPILED_AT = $(shell date +%s)
RELEASES_URL ?= https://api.github.com/repos/BishopFox/sliver/releases
ARMORY_PUBLIC_KEY ?= RWSBpxpRWDrD7Fe+VvRE3c2VEDC2NK80rlNCj+BX0gz44Xw07r6KQD9L
ARMORY_REPO_URL ?= https://api.github.com/repos/sliverarmory/armory/releases
VERSION_PKG = github.com/bishopfox/sliver/client/version
CLIENT_ASSETS_PKG = github.com/bishopfox/sliver/client/assets

GIT_DIRTY = $(shell git diff --quiet|| echo 'Dirty')
GIT_COMMIT = $(shell git rev-parse HEAD)

LDFLAGS = -ldflags "-s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X \"$(VERSION_PKG).GoVersion=$(GO_VERSION)\" \
	-X $(VERSION_PKG).CompiledAt=$(COMPILED_AT) \
	-X $(VERSION_PKG).GithubReleasesURL=$(RELEASES_URL) \
	-X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
	-X $(VERSION_PKG).GitDirty=$(GIT_DIRTY) \
	-X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) \
	-X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"

# Debug builds shouldn't be stripped (-s -w flags)
LDFLAGS_DEBUG = -ldflags "-X $(VERSION_PKG).Version=$(VERSION) \
	-X \"$(VERSION_PKG).GoVersion=$(GO_VERSION)\" \
	-X $(VERSION_PKG).CompiledAt=$(COMPILED_AT) \
	-X $(VERSION_PKG).GithubReleasesURL=$(RELEASES_URL) \
	-X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
	-X $(VERSION_PKG).GitDirty=$(GIT_DIRTY) \
	-X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) \
	-X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"

SED_INPLACE := sed -i
STATIC_TARGET := linux

UNAME_S := $(shell uname -s)
UNAME_P := $(shell uname -p)

# Programs required for generating protobuf/grpc files
PB_COMPILERS = protoc protoc-gen-go protoc-gen-go-grpc
ifeq ($(MAKECMDGOALS), pb)
	K := $(foreach exec,$(PB_COMPILERS),\
			$(if $(shell which $(exec)),some string,$(error "Missing protobuf util $(exec) in PATH")))
endif

# *** Darwin ***
ifeq ($(UNAME_S),Darwin)
	SED_INPLACE := sed -i ''
	STATIC_TARGET := macos
endif

# If no target is specified, determine GOARCH
ifeq ($(UNAME_P),arm)
	ifeq ($(MAKECMDGOALS), )
		ENV += GOARCH=arm64
	endif
endif

ifeq ($(MAKECMDGOALS), linux)
	# Redefine LDFLAGS to add the static part
	LDFLAGS = -ldflags "-s -w \
		-extldflags '-static' \
		-X $(VERSION_PKG).Version=$(VERSION) \
		-X \"$(VERSION_PKG).GoVersion=$(GO_VERSION)\" \
		-X $(VERSION_PKG).CompiledAt=$(COMPILED_AT) \
		-X $(VERSION_PKG).GithubReleasesURL=$(RELEASES_URL) \
		-X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) \
		-X $(VERSION_PKG).GitDirty=$(GIT_DIRTY) \
		-X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) \
		-X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"
endif

#
# Targets
#
.PHONY: default
default: clean .downloaded_assets validate-go-version
	$(ENV) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server$(ARTIFACT_SUFFIX) ./server
	$(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client$(ARTIFACT_SUFFIX) ./client

# Allows you to build a CGO-free client for any target e.g. `GOOS=windows GOARCH=arm64 make client`
# NOTE: WireGuard is not supported on all platforms, but most 64-bit GOOS/GOARCH combinations should work.
.PHONY: client
client: clean .downloaded_assets validate-go-version
	$(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client ./client

.PHONY: macos-amd64
macos-amd64: clean .downloaded_assets validate-go-version
	GOOS=darwin GOARCH=amd64 $(ENV) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server$(ARTIFACT_SUFFIX) ./server
	GOOS=darwin GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client$(ARTIFACT_SUFFIX) ./client

.PHONY: macos-arm64
macos-arm64: clean .downloaded_assets validate-go-version
	GOOS=darwin GOARCH=arm64 $(ENV) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server$(ARTIFACT_SUFFIX) ./server
	GOOS=darwin GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client$(ARTIFACT_SUFFIX) ./client

.PHONY: linux-amd64
linux-amd64: clean .downloaded_assets validate-go-version
	GOOS=linux GOARCH=amd64 $(ENV) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server$(ARTIFACT_SUFFIX) ./server
	GOOS=linux GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client$(ARTIFACT_SUFFIX) ./client

.PHONY: linux-arm64
linux-arm64: clean .downloaded_assets validate-go-version
	GOOS=linux GOARCH=arm64 $(ENV) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server$(ARTIFACT_SUFFIX) ./server
	GOOS=linux GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client$(ARTIFACT_SUFFIX) ./client

.PHONY: windows-amd64
windows-amd64: clean .downloaded_assets validate-go-version
	GOOS=windows GOARCH=amd64 $(ENV) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server$(ARTIFACT_SUFFIX).exe ./server
	GOOS=windows GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client$(ARTIFACT_SUFFIX).exe ./client

.PHONY: clients
clients: clean .downloaded_assets validate-go-version
	GOOS=darwin GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_macos-amd64$(ARTIFACT_SUFFIX) ./client
	GOOS=darwin GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_macos-arm64$(ARTIFACT_SUFFIX) ./client
	GOOS=linux GOARCH=386 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_linux-386$(ARTIFACT_SUFFIX) ./client
	GOOS=linux GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_linux-amd64$(ARTIFACT_SUFFIX) ./client
	GOOS=linux GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_linux-arm64$(ARTIFACT_SUFFIX) ./client
	GOOS=windows GOARCH=386 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_windows-386$(ARTIFACT_SUFFIX).exe ./client
	GOOS=windows GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_windows-amd64$(ARTIFACT_SUFFIX).exe ./client
	GOOS=windows GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_windows-arm64$(ARTIFACT_SUFFIX).exe ./client
	GOOS=freebsd GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_freebsd-amd64$(ARTIFACT_SUFFIX) ./client
	GOOS=freebsd GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client_freebsd-arm64$(ARTIFACT_SUFFIX) ./client

.PHONY: servers
servers: 
	GOOS=windows GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server_windows-amd64$(ARTIFACT_SUFFIX).exe ./server
	GOOS=linux GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server_linux-amd64$(ARTIFACT_SUFFIX) ./server
	GOOS=linux GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server_linux-arm64$(ARTIFACT_SUFFIX) ./server
	GOOS=darwin GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server_darwin-arm64$(ARTIFACT_SUFFIX) ./server
	GOOS=darwin GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server_darwin-amd64$(ARTIFACT_SUFFIX) ./server

.PHONY: pb
pb:
	protoc -I protobuf/ protobuf/commonpb/common.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/sliverpb/sliver.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/clientpb/client.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/dnspb/dns.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/rpcpb/services.proto --go_out=paths=source_relative:protobuf/ --go-grpc_out=protobuf/ --go-grpc_opt=paths=source_relative 

.PHONY: debug
debug: clean
	$(ENV) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -mod=vendor $(TAGS),server $(LDFLAGS_DEBUG) -o sliver-server$(ARTIFACT_SUFFIX) ./server
	$(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor $(TAGS),client $(LDFLAGS_DEBUG) -o sliver-client$(ARTIFACT_SUFFIX) ./client

validate-go-version:
	@if [ $(GO_MAJOR_VERSION) -gt $(MIN_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		exit 0 ;\
	elif [ $(GO_MAJOR_VERSION) -lt $(MIN_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	elif [ $(GO_MINOR_VERSION) -lt $(MIN_SUPPORTED_GO_MINOR_VERSION) ] ; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	fi

.PHONY: clean-all
clean-all: clean
	rm -rf ./server/assets/fs/darwin/amd64
	rm -rf ./server/assets/fs/darwin/arm64
	rm -rf ./server/assets/fs/windows/amd64
	rm -rf ./server/assets/fs/linux/amd64
	rm -f ./server/assets/fs/*.zip
	rm -f ./.downloaded_assets

.PHONY: clean
clean:
	rm -f sliver-client sliver-client_* sliver-server sliver-server_* sliver-*.exe

.downloaded_assets:
	./go-assets.sh
	touch ./.downloaded_assets
