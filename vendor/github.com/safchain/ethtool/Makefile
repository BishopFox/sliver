all: build

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build
