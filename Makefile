.PHONY: build test lint vet fmt run clean

BINARY := abalone

## build: compile the binary
build:
	go build -o $(BINARY) .

## test: run all tests
test:
	go test ./... -count=1

## test-v: run all tests (verbose)
test-v:
	go test ./... -v -count=1

## lint: run fmt check + vet
lint: fmt-check vet

## vet: run go vet
vet:
	go vet ./...

## fmt: format all Go files
fmt:
	go fmt ./...

## fmt-check: verify formatting (fails if files would change)
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

## run: build and start the server
run: build
	./$(BINARY)

## clean: remove build artifacts
clean:
	rm -f $(BINARY)

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
