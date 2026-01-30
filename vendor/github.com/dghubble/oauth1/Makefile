.PHONY: all
all: test vet fmt

.PHONY: test
test:
	@go test $$(go list ./... | grep -v examples) -cover

.PHONY: vet
vet:
	@go vet -all $$(go list ./... | grep -v examples)

.PHONY: fmt
fmt:
	@test -z $$(go fmt ./...)

