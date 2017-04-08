NAME := geocoding
VERBOSE := $(if $(VERBOSE),-v)
VERSION := $(shell git describe --tags --abbrev=0)
REVISION := $(shell git rev-parse --short HEAD)
OSARCH="darwin/amd64 linux/386 linux/amd64 windows/386 windows/amd64"

SRCS := $(shell find . -type f -name '*.go')
LDFLAGS := -ldflags="-X 'main.version=$(VERSION)' -X 'main.Revision=$(REVISION)'"

.DEFAULT_GOAL=build

## Download dependencies
deps:
	go get -d $(VERBOSE)

## Setup
setup: deps
	go get github.com/golang/lint/golint
	go get github.com/Songmu/make2help/cmd/make2help
	go get github.com/mitchellh/gox
	go get github.com/tcnksm/ghr

## Lint
lint: setup
	go vet .
	golint -set_exit_status . || exit $$?

## Build binary
build: setup lint
	go build -a $(LDFLAGS)

## Cross-build
cross-build: setup lint
	rm -rf ./out
	gox -osarch $(OSARCH) $(BUILD_FLAGS) -output "./out/${NAME}_${VERSION}_{{.OS}}_{{.Arch}}"

## install binaries
install: setup lint
	go install $(VERBOSE) $(BUILD_FLAGS)

package: cross-build
	rm -rf pkg \
		&& mkdir pkg \
		&& pushd out \
		&& cp -p * ../pkg/ \
		&& popd

## Release
release: package
	ghr $(VERSION) pkg

## Show help
help:
	@make2help $(MAKEFILE_LIST)

clean:
	@rm -f .#* \#* geocoding
	@rm -rf bin pkg dist

.PHONY: install clean
