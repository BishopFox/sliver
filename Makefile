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
macos: clean pb
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: linux
linux: clean pb
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: windows
windows: clean pb
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server


#
# Static builds were we bundle everything together
#
.PHONY: static-macos
static-macos: clean pb
	packr
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/a_main-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/a_main-packr.go
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: static-windows
static-windows: clean pb
	packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/a_main-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/a_main-packr.go
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server

.PHONY: static-linux
static-linux: clean pb
	packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/a_main-packr.go
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/a_main-packr.go
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

.PHONY: pb
pb:
	go install ./vendor/github.com/golang/protobuf/protoc-gen-go
	@hash protoc > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/golang/protobuf/protoc-gen-go; \
	fi
	protoc -I protobuf/ protobuf/sliver.proto --go_out=protobuf/

.PHONY: clean-all
clean-all: clean
	rm -f ./server/assets/darwin/go.zip
	rm -f ./server/assets/windows/go.zip
	rm -f ./server/assets/linux/go.zip
	rm -f ./server/assets/*.zip

.PHONY: clean
clean:
	@hash packr > /dev/null 2>&1; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/gobuffalo/packr; \
		$(GO) get -u github.com/gobuffalo/packr/packr; \
	fi
	packr clean
	rm -f ./protobuf/*.pb.go
	rm -f sliver-server *.exe
