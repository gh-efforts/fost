SHELL=/usr/bin/env bash

GOFLAGS+=-ldflags=-X="fost/build.CurrentCommit"="+git$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))"
FFI_PATH=extern/filecoin-ffi/
.PHONY: default, build
default:  build;

build:
	git submodule update --init --recursive
	make -C $(FFI_PATH)
	go build $(GOFLAGS)
