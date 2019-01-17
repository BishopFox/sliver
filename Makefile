#
# Makefile for Sliver
#

GO = go
ENV = CGO_ENABLED=0
TAGS = -tags netgo
LDFLAGS = -ldflags '-s -w'


macos: clean pb
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

linux: clean pb
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

windows: clean pb
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server


#
# Static builds were we bundle everything together
# TODO: I think the `sed` command syntax is only valid on MacOS
#
static-macos: clean pb
	packr
	sed -i '' '/$*.windows\/*./d' ./server/a_main-packr.go
	sed -i '' '/$*.linux\/*./d' ./server/a_main-packr.go
	GOOS=darwin $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

static-windows: clean pb
	packr
	sed -i '' '/$*.darwin\/*./d' ./server/a_main-packr.go
	sed -i '' '/$*.linux\/*./d' ./server/a_main-packr.go
	GOOS=windows $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server.exe ./server

static-linux: clean pb
	packr
	sed -i '' '/$*.darwin\/*./d' ./server/a_main-packr.go
	sed -i '' '/$*.windows\/*./d' ./server/a_main-packr.go
	GOOS=linux $(ENV) $(GO) build $(TAGS) $(LDFLAGS) -o sliver-server ./server

pb:
	protoc -I protobuf/ protobuf/sliver.proto --go_out=protobuf/

clean:
	packr clean
	rm -f ./protobuf/*.pb.go
	rm -f sliver-server *.exe
