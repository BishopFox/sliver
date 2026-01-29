.PHONY: install test
BUMP_VERSION := $(GOPATH)/bin/bump_version
GODOCDOC := $(GOPATH)/bin/godocdoc
STATICCHECK := $(GOPATH)/bin/staticcheck

test: lint
	go test ./...

install:
	go get ./...
	go install ./...

lint:
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...
	go vet ./...

race-test: lint
	go test -race ./...

$(BUMP_VERSION):
	go get github.com/kevinburke/bump_version

release: test | $(BUMP_VERSION)
	$(BUMP_VERSION) minor types.go

docs: | $(GODOCDOC)
	$(GODOCDOC)
