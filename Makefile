BINARY := mine
VERSION := 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/rnwolfe/mine/internal/version.Version=$(VERSION) \
	-X github.com/rnwolfe/mine/internal/version.Commit=$(COMMIT) \
	-X github.com/rnwolfe/mine/internal/version.Date=$(DATE)

.PHONY: build install clean test lint run

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

install: build
	cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY) 2>/dev/null || \
	cp bin/$(BINARY) $(HOME)/.local/bin/$(BINARY) 2>/dev/null || \
	sudo cp bin/$(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -rf bin/

test:
	go test ./... -v -count=1 -race

cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

lint:
	go vet ./...

run: build
	./bin/$(BINARY)

# Quick dev cycle
dev:
	go run -ldflags "$(LDFLAGS)" . $(ARGS)
