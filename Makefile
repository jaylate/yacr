GOSRC       := $(wildcard *.go)
CSRCDIR     := csrc
CSRC        := $(wildcard $(CSRCDIR)/*.c)
BINDIR      := ./bin
CINITOBJ    := $(BINDIR)/init
YACR        := $(BINDIR)/yacr

.PHONY: all fmt build clean run buildinit
all: build

$(BINDIR):
	mkdir -p $@

buildinit: $(CSRC) | $(BINDIR)
	$(CC) $(CSRC) -o $(CINITOBJ)

fmt: $(GOSRC)
	go fmt ./...

build: fmt buildinit
	go build -o $(BINDIR) ./...

run: build
	$(YACR) run /bin/sh

clean:
	go clean
	rm -rf $(BINDIR)

