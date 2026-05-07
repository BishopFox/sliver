GOTEST ?= go test


install:
	@dep ensure

build-example:
	go build -o example-auth ./_example/auth
	go build -o example-notify ./_example/notify

test:
	${GOTEST} -v -race ./...
