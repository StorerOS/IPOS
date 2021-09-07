PACKAGES=$(shell go list ./...)
OUTPUT?=build/ipos

export GO111MODULE = on

BUILD_TAGS?='ipos'
LD_FLAGS = -X github.com/storeros/ipos/version.GitCommit=`git rev-parse --short=8 HEAD` -s -w
BUILD_FLAGS = -ldflags "$(LD_FLAGS)"

all: build

########################################
### Build IPOS

build:
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -tags $(BUILD_TAGS) -o $(OUTPUT) ./cmd/ipos/

########################################
### Formatting, linting, and testing

fmt:
	@go fmt ./...

lint:
	@echo "--> Running linter"
	@golangci-lint run

test:
	@echo "--> Running go test"
	@go test -p 1 $(PACKAGES)

.PHONY: build