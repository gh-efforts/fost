SHELL=/usr/bin/env bash

GOFLAGS+=-ldflags=-X="fost/build.CurrentCommit"="+git$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))"

.PHONY: default
default:  binary;

binary:
	go build $(GOFLAGS)
