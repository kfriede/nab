.PHONY: build test lint clean install release-dry-run

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

all: lint test build
