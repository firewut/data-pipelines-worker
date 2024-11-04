.PHONY: build-darwin build-linux build-windows test clean start build-rpm

BINARY_NAME=data-pipelines-worker
TEST_UNIT_PATH=./test/unit/...
TEST_FUNCTIONAL_PATH=./test/functional/...
CONFIG_FILE=${PWD}/config/config.yaml
TEST_TIMEOUT=20s
RPM_SPEC_FILE=data-pipelines-worker.spec

export CONFIG_FILE

build-darwin:
	GOARCH=amd64 GOOS=darwin CGO_ENABLED=0  go build -ldflags="-s -w" -o bin/${BINARY_NAME}-darwin cmd/data-pipelines/worker.go
	
build-linux:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/${BINARY_NAME}-linux cmd/data-pipelines/worker.go
	
build-windows:
	GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/${BINARY_NAME}-windows cmd/data-pipelines/worker.go

test: test-unit test-functional

test-unit:
	go clean -testcache
	go test -race -timeout ${TEST_TIMEOUT} -v ${TEST_UNIT_PATH}

test-functional:
	go clean -testcache
	go test -race -timeout ${TEST_TIMEOUT} -v ${TEST_FUNCTIONAL_PATH}

clean:
	go clean
	rm bin/${BINARY_NAME}-darwin
	rm bin/${BINARY_NAME}-linux
	rm bin/${BINARY_NAME}-windows

start:
	go run -race cmd/data-pipelines/worker.go
	# go run -race cmd/data-pipelines/worker.go --http-api-port=8080

build-rpm: build-linux
	mkdir -p ${HOME}/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
	cp bin/${BINARY_NAME}-linux ${HOME}/rpmbuild/SOURCES/
	cp -r config ${HOME}/rpmbuild/SOURCES/
	cp ./data-pipelines-worker.spec ${HOME}/rpmbuild/SPECS/

	rpmbuild -ba $(RPM_SPEC_FILE)
