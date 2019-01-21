#
# Makefile for Sliver
#

GO = go
ENV = CGO_ENABLED=0
TAGS = -tags netgo
LDFLAGS = -ldflags '-s -w'

SED_INPLACE := sed -i
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	SED_INPLACE := sed -i ''
endif


macos: clean pb
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

linux: clean pb
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

windows: clean pb
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server


#
# Static builds were we bundle everything together
#
static-macos: clean pb
	packr
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/a_main-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/a_main-packr.go
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

static-windows: clean pb
	packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/a_main-packr.go
	$(SED_INPLACE) '/$*.linux\/go\.zip/d' ./server/a_main-packr.go
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server

static-linux: clean pb
	packr
	$(SED_INPLACE) '/$*.darwin\/go\.zip/d' ./server/a_main-packr.go
	$(SED_INPLACE) '/$*.windows\/go\.zip/d' ./server/a_main-packr.go
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

pb:
	protoc -I protobuf/ protobuf/sliver.proto --go_out=protobuf/

clean:
	packr clean
	rm -f ./protobuf/*.pb.go
	rm -f sliver-server *.exe
