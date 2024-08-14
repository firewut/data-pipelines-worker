.PHONY: build test clean start

BINARY-NAME=data-pipelines-worker
TEST_UNIT_PATH=./test/unit/...
TEST_FUNCTIONAL_PATH=./test/functional/...
CONFIG_FILE=${PWD}/config/config.yaml

export CONFIG_FILE

build:
	GOARCH=amd64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin cmd/data-pipelines/worker.go
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux cmd/data-pipelines/worker.go
	GOARCH=amd64 GOOS=windows go build -o bin/${BINARY_NAME}-windows cmd/data-pipelines/worker.go

test: test-unit test-functional

test-unit:
	go clean -testcache
	go test -race -timeout 10s -v ${TEST_UNIT_PATH}

test-functional:
	go clean -testcache
	go test -race -timeout 10s -v ${TEST_FUNCTIONAL_PATH}

clean:
	go clean
	rm bin/${BINARY_NAME}-darwin
	rm bin/${BINARY_NAME}-linux
	rm bin/${BINARY_NAME}-windows

start:
	go run -race cmd/data-pipelines/worker.go