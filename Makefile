export GO111MODULE = on

ifeq ($(shell uname),Darwin)
	BINDIR = binaries/darwin
else ifeq ($(shell uname),Linux)
	BINDIR = binaries/linux_x86_64
endif

PATH := $(shell pwd)/$(BINDIR):$(PATH)

LDFLAGS = -ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"

.PHONY: all
all: build test

.PHONY: build
build:
	if [ ! -d bin ]; then mkdir bin; fi
	if [ ! -d logs ]; then mkdir logs; fi
	go build -o bin/go-chunked-streaming-server $(LDFLAGS) main.go

.PHONY: fmt
fmt:
	find . -not -path "./vendor/*" -name '*.go' -type f | sed 's#\(.*\)/.*#\1#' | sort -u | xargs -n1 -I {} bash -c "cd {} && goimports -w *.go && gofmt -w -s -l *.go"

.PHONY: test
test:
ifndef BINDIR
	$(error Unable to set PATH based on current platform.)
endif
	#TODO go test $(V) ./handlers

.PHONY: clean
clean:
	go clean
	rm -f bin/go-chunked-streaming-server
	rm -rf content/*

.PHONY: build-example-chunked-streaming
build-example-chunked-streaming:
	if [ ! -d bin ]; then mkdir bin; fi
	go build -o bin/examples-chunked-streaming-post-client examples/chunked-streaming/post_client/main.go
	go build -o bin/examples-chunked-streaming-input-stream examples/chunked-streaming/input_stream/main.go
