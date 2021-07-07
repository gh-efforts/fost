SHELL=/usr/bin/env bash

GOFLAGS+=-ldflags=-X="fost/build.CurrentCommit"="+git$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))"

.PHONY: default
default:  linux;
all: linux windows darwin

linux:
	rm -f fost
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o fost $(GOFLAGS)

windows:
	rm -f fost.exe
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GOFLAGS)

darwin:
	rm -f fost
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o fost $(GOFLAGS)