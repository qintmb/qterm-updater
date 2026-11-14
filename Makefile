BINARY  := qterm-updater
PREFIX  ?= $(HOME)/.local
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build install test clean

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) ./cmd/qterm-updater

install: build
	install -d $(PREFIX)/bin
	install -m 0755 $(BINARY) $(PREFIX)/bin/$(BINARY)
	@echo "installed $(PREFIX)/bin/$(BINARY) ($(VERSION))"

test:
	go build ./...
	go vet ./...
	go test ./...

clean:
	rm -f $(BINARY)
