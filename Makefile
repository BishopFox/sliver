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

MIN_SUPPORTED_GO_MAJOR_VERSION = 1
MIN_SUPPORTED_GO_MINOR_VERSION = 25
GO_VERSION_VALIDATION_ERR_MSG = Golang version is not supported, please update to at least $(MIN_SUPPORTED_GO_MAJOR_VERSION).$(MIN_SUPPORTED_GO_MINOR_VERSION)

SLIVER_PUBLIC_KEY ?= RWTZPg959v3b7tLG7VzKHRB1/QT+d3c71Uzetfa44qAoX5rH7mGoQTTR
ARMORY_PUBLIC_KEY ?= RWSBpxpRWDrD7Fe+VvRE3c2VEDC2NK80rlNCj+BX0gz44Xw07r6KQD9L
ARMORY_REPO_URL ?= https://api.github.com/repos/sliverarmory/armory/releases
CLIENT_ASSETS_PKG = github.com/bishopfox/sliver/client/assets
SLIVER_UPDATE_PKG = github.com/bishopfox/sliver/client/command/update
PB_COMPILERS = protoc protoc-gen-go protoc-gen-go-grpc

ifneq ($(OS),Windows_NT)

#
# Prerequisites
#
# https://stackoverflow.com/questions/5618615/check-if-a-program-exists-from-a-makefile
EXECUTABLES = uname sed git date cut $(GO)
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

#
# Build Information
#
GO_MAJOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_MINOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)

LDFLAGS = -ldflags "-s -w \
	-X $(SLIVER_UPDATE_PKG).SliverPublicKey=$(SLIVER_PUBLIC_KEY) \
	-X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) \
	-X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"

# Debug builds shouldn't be stripped (-s -w flags)
LDFLAGS_DEBUG = -ldflags "-X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) \
	-X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"

SED_INPLACE := sed -i
STATIC_TARGET := linux

UNAME_S := $(shell uname -s)
UNAME_P := $(shell uname -p)

# Programs required for generating protobuf/grpc files
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
		ifeq ($(origin GOARCH), undefined)
			ENV += GOARCH=arm64
		endif
	endif
endif

ifeq ($(MAKECMDGOALS), linux)
	# Redefine LDFLAGS to add the static part
	LDFLAGS = -ldflags "-s -w \
		-extldflags '-static' \
		-X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) \
		-X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"
endif

#
# Targets
#
.PHONY: default
default: clean validate-go-version
	env -u GOOS -u GOARCH $(MAKE) GOOS= GOARCH= .downloaded_assets
	$(ENV) $(if $(GOOS),GOOS=$(GOOS)) $(if $(GOARCH),GOARCH=$(GOARCH)) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server$(ARTIFACT_SUFFIX) ./server
	$(ENV) $(if $(GOOS),GOOS=$(GOOS)) $(if $(GOARCH),GOARCH=$(GOARCH)) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),client $(LDFLAGS) -o sliver-client$(ARTIFACT_SUFFIX) ./client

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
servers: clean .downloaded_assets validate-go-version
	GOOS=windows GOARCH=amd64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server_windows-amd64$(ARTIFACT_SUFFIX).exe ./server
	GOOS=windows GOARCH=arm64 $(ENV) CGO_ENABLED=0 $(GO) build -mod=vendor -trimpath $(TAGS),server $(LDFLAGS) -o sliver-server_windows-arm64$(ARTIFACT_SUFFIX).exe ./server
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
	$(ENV) $(GO) run -mod=vendor ./util/cmd/assets
	touch ./.downloaded_assets


#
# >>> WINDOWS <<<
#
else

SHELL := cmd.exe
.SHELLFLAGS := /C

GO_VERSION := $(patsubst go%,%,$(strip $(shell $(GO) env GOVERSION)))
GO_MAJOR_VERSION := $(word 1,$(subst ., ,$(GO_VERSION)))
GO_MINOR_VERSION := $(word 2,$(subst ., ,$(GO_VERSION)))

LDFLAGS = -ldflags "-s -w -X $(SLIVER_UPDATE_PKG).SliverPublicKey=$(SLIVER_PUBLIC_KEY) -X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) -X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"

LDFLAGS_DEBUG = -ldflags "-X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) -X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"

COMMA := ,

ifeq ($(MAKECMDGOALS), linux)
	LDFLAGS = -ldflags "-s -w -extldflags '-static' -X $(CLIENT_ASSETS_PKG).DefaultArmoryPublicKey=$(ARMORY_PUBLIC_KEY) -X $(CLIENT_ASSETS_PKG).DefaultArmoryRepoURL=$(ARMORY_REPO_URL)"
endif

define windows_exec
$(strip $(foreach envvar,$(1),set "$(envvar)" && ))$(2)
endef

define windows_go_build
$(call windows_exec,$(ENV) GOOS=$(1) GOARCH=$(2) CGO_ENABLED=$(3),"$(GO)" build $(4) $(TAGS)$(COMMA)$(5) $(6) -o $(7) ./$(5))
endef

.PHONY: default
default: clean validate-go-version
	$(call windows_exec,$(ENV) GOOS= GOARCH=,"$(MAKE)" GOOS= GOARCH= .downloaded_assets)
	$(call windows_go_build,$(GOOS),$(GOARCH),$(CGO_ENABLED),-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server$(ARTIFACT_SUFFIX))
	$(call windows_go_build,$(GOOS),$(GOARCH),0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client$(ARTIFACT_SUFFIX))

.PHONY: client
client: clean .downloaded_assets validate-go-version
	$(call windows_go_build,$(GOOS),$(GOARCH),0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client)

.PHONY: macos-amd64
macos-amd64: clean .downloaded_assets validate-go-version
	$(call windows_go_build,darwin,amd64,$(CGO_ENABLED),-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server$(ARTIFACT_SUFFIX))
	$(call windows_go_build,darwin,amd64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client$(ARTIFACT_SUFFIX))

.PHONY: macos-arm64
macos-arm64: clean .downloaded_assets validate-go-version
	$(call windows_go_build,darwin,arm64,$(CGO_ENABLED),-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server$(ARTIFACT_SUFFIX))
	$(call windows_go_build,darwin,arm64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client$(ARTIFACT_SUFFIX))

.PHONY: linux-amd64
linux-amd64: clean .downloaded_assets validate-go-version
	$(call windows_go_build,linux,amd64,$(CGO_ENABLED),-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server$(ARTIFACT_SUFFIX))
	$(call windows_go_build,linux,amd64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client$(ARTIFACT_SUFFIX))

.PHONY: linux-arm64
linux-arm64: clean .downloaded_assets validate-go-version
	$(call windows_go_build,linux,arm64,$(CGO_ENABLED),-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server$(ARTIFACT_SUFFIX))
	$(call windows_go_build,linux,arm64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client$(ARTIFACT_SUFFIX))

.PHONY: windows-amd64
windows-amd64: clean .downloaded_assets validate-go-version
	$(call windows_go_build,windows,amd64,$(CGO_ENABLED),-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server$(ARTIFACT_SUFFIX).exe)
	$(call windows_go_build,windows,amd64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client$(ARTIFACT_SUFFIX).exe)

.PHONY: clients
clients: clean .downloaded_assets validate-go-version
	$(call windows_go_build,darwin,amd64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_macos-amd64$(ARTIFACT_SUFFIX))
	$(call windows_go_build,darwin,arm64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_macos-arm64$(ARTIFACT_SUFFIX))
	$(call windows_go_build,linux,386,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_linux-386$(ARTIFACT_SUFFIX))
	$(call windows_go_build,linux,amd64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_linux-amd64$(ARTIFACT_SUFFIX))
	$(call windows_go_build,linux,arm64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_linux-arm64$(ARTIFACT_SUFFIX))
	$(call windows_go_build,windows,386,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_windows-386$(ARTIFACT_SUFFIX).exe)
	$(call windows_go_build,windows,amd64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_windows-amd64$(ARTIFACT_SUFFIX).exe)
	$(call windows_go_build,windows,arm64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_windows-arm64$(ARTIFACT_SUFFIX).exe)
	$(call windows_go_build,freebsd,amd64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_freebsd-amd64$(ARTIFACT_SUFFIX))
	$(call windows_go_build,freebsd,arm64,0,-mod=vendor -trimpath,client,$(LDFLAGS),sliver-client_freebsd-arm64$(ARTIFACT_SUFFIX))

.PHONY: servers
servers: clean .downloaded_assets validate-go-version
	$(call windows_go_build,windows,amd64,0,-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server_windows-amd64$(ARTIFACT_SUFFIX).exe)
	$(call windows_go_build,windows,arm64,0,-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server_windows-arm64$(ARTIFACT_SUFFIX).exe)
	$(call windows_go_build,linux,amd64,0,-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server_linux-amd64$(ARTIFACT_SUFFIX))
	$(call windows_go_build,linux,arm64,0,-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server_linux-arm64$(ARTIFACT_SUFFIX))
	$(call windows_go_build,darwin,arm64,0,-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server_darwin-arm64$(ARTIFACT_SUFFIX))
	$(call windows_go_build,darwin,amd64,0,-mod=vendor -trimpath,server,$(LDFLAGS),sliver-server_darwin-amd64$(ARTIFACT_SUFFIX))

.PHONY: pb
pb: validate-pb-compilers
	protoc -I protobuf/ protobuf/commonpb/common.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/sliverpb/sliver.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/clientpb/client.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/dnspb/dns.proto --go_out=paths=source_relative:protobuf/
	protoc -I protobuf/ protobuf/rpcpb/services.proto --go_out=paths=source_relative:protobuf/ --go-grpc_out=protobuf/ --go-grpc_opt=paths=source_relative

.PHONY: debug
debug: clean
	$(call windows_go_build,$(GOOS),$(GOARCH),$(CGO_ENABLED),-mod=vendor,server,$(LDFLAGS_DEBUG),sliver-server$(ARTIFACT_SUFFIX))
	$(call windows_go_build,$(GOOS),$(GOARCH),0,-mod=vendor,client,$(LDFLAGS_DEBUG),sliver-client$(ARTIFACT_SUFFIX))

.PHONY: validate-pb-compilers
validate-pb-compilers:
	@for %%P in ($(PB_COMPILERS)) do @where.exe %%P >NUL || (echo Missing protobuf util %%P in PATH & exit /b 1)

validate-go-version:
	@if $(GO_MAJOR_VERSION) GTR $(MIN_SUPPORTED_GO_MAJOR_VERSION) (exit /b 0) else if $(GO_MAJOR_VERSION) LSS $(MIN_SUPPORTED_GO_MAJOR_VERSION) (echo $(GO_VERSION_VALIDATION_ERR_MSG) & exit /b 1) else if $(GO_MINOR_VERSION) LSS $(MIN_SUPPORTED_GO_MINOR_VERSION) (echo $(GO_VERSION_VALIDATION_ERR_MSG) & exit /b 1)

.PHONY: clean-all
clean-all: clean
	-rmdir /S /Q server\assets\fs\darwin\amd64
	-rmdir /S /Q server\assets\fs\darwin\arm64
	-rmdir /S /Q server\assets\fs\windows\amd64
	-rmdir /S /Q server\assets\fs\linux\amd64
	-del /Q /F server\assets\fs\*.zip 2>NUL
	-del /Q /F .downloaded_assets 2>NUL

.PHONY: clean
clean:
	-del /Q /F sliver-client sliver-client_* sliver-server sliver-server_* sliver-*.exe 2>NUL

.downloaded_assets:
	$(call windows_exec,$(ENV),"$(GO)" run -mod=vendor ./util/cmd/assets)
	@type NUL > .downloaded_assets

endif
