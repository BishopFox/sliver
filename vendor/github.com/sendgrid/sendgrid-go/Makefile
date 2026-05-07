.PHONY: test install test-integ test-docker

install:
	go get -t -v ./...

test:
	./go.coverage.sh
	bash -c 'diff -u <(echo -n) <(gofmt -d -s .)'

test-integ: test

version ?= latest
test-docker:
	curl -s https://raw.githubusercontent.com/sendgrid/sendgrid-oai/HEAD/prism/prism.sh -o prism.sh
	version=$(version) bash ./prism.sh
