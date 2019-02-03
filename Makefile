#
# Makefile for Sliver
#

GO ?= go
ENV = CGO_ENABLED=0
TAGS = -tags netgo
LDFLAGS = -ldflags '-s -w'

SED_INPLACE := sed -i
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	SED_INPLACE := sed -i ''
endif


.PHONY: macos
macos: clean pb packr
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: linux
linux: clean pb packr
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: windows
windows: clean pb packr
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server


#
# Static builds were we bundle everything together
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
	go install ./vendor/github.com/golang/protobuf/protoc-gen-go
	protoc -I protobuf/ protobuf/sliver.proto --go_out=protobuf/

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
	rm -f ./protobuf/*.pb.go
	rm -f sliver-server *.exe
