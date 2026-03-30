VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w

.PHONY: build clean all

build:
	go build -ldflags "$(LDFLAGS)" -o dist/sonarbay .

all: clean
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/sonarbay-win-x64.exe .
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/sonarbay-linux-x64 .
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/sonarbay-linux-arm64 .
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/sonarbay-darwin-arm64 .
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/sonarbay-darwin-x64 .

clean:
	rm -rf dist
	mkdir -p dist
