GOSRC       := $(wildcard *.go)
CSRCDIR     := csrc
CSRC        := $(wildcard $(CSRCDIR)/*.c)
BINDIR      := ./bin
CINITOBJ    := $(BINDIR)/init
YACR        := $(BINDIR)/yacr

CC          ?= gcc
CFLAGS      ?= -Wall -Wextra

.PHONY: all fmt build clean run buildinit test
all: build

$(BINDIR):
	mkdir -p $@

buildinit: $(CSRC) | $(BINDIR)
	$(CC) $(CFLAGS) $(CSRC) -o $(CINITOBJ)

fmt: $(GOSRC)
	go fmt ./...

vet: $(GOSRC)
	go vet ./...

build: fmt vet buildinit
	go build -o $(BINDIR) ./...

run: build
	$(YACR) run /bin/sh

test:
	go test -v ./...

clean:
	go clean
	rm -rf $(BINDIR)
