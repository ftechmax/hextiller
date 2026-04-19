BINARY := hextiller
DIST := dist
PKG := ./cmd/hextiller

PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

GOFLAGS ?=
LDFLAGS ?= -s -w

.PHONY: all build build-linux build-windows test install uninstall clean

all: build-linux build-windows

build: build-linux

build-linux:
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY)-linux-amd64 $(PKG)

build-windows:
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY)-windows-amd64.exe $(PKG)

test:
	go test ./...

install: build-linux
	install -Dm755 $(DIST)/$(BINARY)-linux-amd64 $(BINDIR)/$(BINARY)

uninstall:
	rm -f $(BINDIR)/$(BINARY)

clean:
	rm -rf $(DIST)
