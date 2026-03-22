.PHONY: build test lint clean install release-dry-run build-cowork

build:
	go build -o nab .

install:
	go install .

test:
	go test -race ./...

lint:
	golangci-lint run

clean:
	rm -f nab
	rm -rf dist/

release-dry-run:
	goreleaser release --snapshot --clean

# Cross-compile static binaries for Claude Cowork's Linux VM
build-cowork:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/nab-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/nab-linux-arm64 .
	@echo "Cowork binaries built in dist/"

all: lint test build
