default: test build
all: default

test: test/lint test/unit test/cover test/performance

test/lint:
	golangci-lint run

test/lint/fix:
	wsl --fix ./... || true
	golangci-lint run --fix

test/unit:
	go test -race ./...

test/cover: module=./...
test/cover:
	go test -coverprofile=coverage.out $(module)
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test/performance:
	go test -run=^$$ -bench=. -cpu 4 ./...

include ./build/package/Makefile
include ./build/ci/Makefile
include ./doc/build/Makefile
