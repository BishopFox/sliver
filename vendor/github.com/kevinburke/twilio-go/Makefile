.PHONY: test vet release

# would be great to make the bash location portable but not sure how
SHELL = /bin/bash -o pipefail

test: lint
	go test ./...

lint: fmt
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

race-test: lint
	go test -race ./...

race-test-short: lint
	go test -short -race ./...

fmt:
	go fmt ./...

ci: | $(DIFFER)
	# would love to run differ make authors here, but Github doesn't check out
	# the full history
	go run -v github.com/kevinburke/differ@latest $(MAKE) fmt
	$(MAKE) lint race-test-short

release: race-test-short
	go run -v github.com/kevinburke/differ@latest $(MAKE) authors
	go run -v github.com/kevinburke/differ@latest $(MAKE) fmt
	go run -v github.com/kevinburke/bump_version@latest --tag-prefix=v minor http.go

force: ;

AUTHORS.txt: .mailmap force
	go run -v github.com/kevinburke/write_mailmap@latest > AUTHORS.txt

authors: AUTHORS.txt
