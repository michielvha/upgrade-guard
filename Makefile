.PHONY: build test lint vuln fmt vet check clean docker-build

BINARY_NAME=upgrade-guard
CMD_PATH=./cmd/upgrade-guard

## Build

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

build-release:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY_NAME) $(CMD_PATH)

## Quality

test:
	go test -race -count=1 ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run

vuln:
	govulncheck ./...

fmt:
	gofumpt -w .
	goimports -w .

vet:
	go vet ./...

## All checks (run before committing)

check: fmt vet lint vuln test

## Docker

docker-build:
	docker build -t $(BINARY_NAME):local .

## Cleanup

clean:
	rm -f $(BINARY_NAME) coverage.out
	rm -rf dist/
