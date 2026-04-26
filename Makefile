BINARY := mip
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/vlln/mip/internal/version.Version=$(VERSION) \
	-X github.com/vlln/mip/internal/version.Commit=$(COMMIT) \
	-X github.com/vlln/mip/internal/version.Date=$(DATE)

.PHONY: all build test clean release version

all: test build

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/mip

test:
	go test ./...

clean:
	rm -rf bin dist

release:
	VERSION="$(VERSION)" COMMIT="$(COMMIT)" DATE="$(DATE)" ./scripts/build-release.sh

version: build
	./bin/$(BINARY) version

