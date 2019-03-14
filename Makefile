#
# Makefile for Sliver
#

GO ?= go
ENV = CGO_ENABLED=0
TAGS = -tags netgo
LDFLAGS = -ldflags '-s -w'
GIT_VERSION = $(shell git rev-parse HEAD)

SED_INPLACE := sed -i
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	SED_INPLACE := sed -i ''
endif


.PHONY: macos
macos: clean version pb
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: linux
linux: clean version pb
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-client ./client

.PHONY: windows
windows: clean version pb
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-client.exe ./client


#
# Static builds were we bundle everything together
#
.PHONY: static-macos
static-macos: clean version pb packr
	packr
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: static-windows
static-windows: clean version pb packr
	packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server

.PHONY: static-linux
static-linux: clean version pb packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/assets/a_assets-packr.go
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/assets/a_assets-packr.go
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: pb
pb:
	go install ./vendor/github.com/golang/protobuf/protoc-gen-go
	protoc -I protobuf/ protobuf/sliver/sliver.proto --go_out=protobuf/
	protoc -I protobuf/ protobuf/client/client.proto --go_out=protobuf/

.PHONY: version
version:
	echo 'package assets\n\nconst GitVersion = "$(GIT_VERSION)"\n' > ./server/assets/version.go 

.PHONY: packr
packr:
	cd ./server/
	packr
	cd ..

.PHONY: clean-version
clean-version:
	echo 'package assets\n\nconst GitVersion = ""\n' > ./server/assets/version.go 

.PHONY: clean-all
clean-all: clean clean-version
	rm -f ./assets/darwin/go.zip
	rm -f ./assets/windows/go.zip
	rm -f ./assets/linux/go.zip
	rm -f ./assets/*.zip

.PHONY: clean
clean: clean-version
	packr clean
	rm -f ./protobuf/client/*.pb.go
	rm -f ./protobuf/sliver/*.pb.go
	rm -f sliver-client sliver-server *.exe

