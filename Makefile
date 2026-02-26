BINARY := mine
VERSION := 0.1.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/rnwolfe/mine/internal/version.Version=$(VERSION) \
	-X github.com/rnwolfe/mine/internal/version.Commit=$(COMMIT) \
	-X github.com/rnwolfe/mine/internal/version.Date=$(DATE)

.PHONY: build install clean test lint filelen run

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

filelen:
	@EXIT=0; \
	for f in $$(find cmd/ internal/ -name "*.go" -not -name "*_test.go" | sort); do \
	  lines=$$(wc -l < "$$f"); \
	  if [ "$$lines" -gt 500 ]; then \
	    if ! grep -qxF "$$f" .github/filelen-exceptions.txt 2>/dev/null; then \
	      echo "::error::$$f: $$lines lines (limit: 500)"; \
	      EXIT=1; \
	    fi; \
	  fi; \
	done; \
	if [ $$EXIT -eq 0 ]; then \
	  echo "filelen: all files within 500-line limit"; \
	else \
	  echo "Add to .github/filelen-exceptions.txt to acknowledge existing violations (with a tracking issue)."; \
	  exit 1; \
	fi

run: build
	./bin/$(BINARY)

# Quick dev cycle
dev:
	go run -ldflags "$(LDFLAGS)" . $(ARGS)
