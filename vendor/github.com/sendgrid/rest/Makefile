.PHONY: test install

install:
	go get -t -v ./...

test: install
	go test -race -cover -v ./...
