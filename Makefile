GOSRC       := $(wildcard *.go)
BINDIR      := ./bin
YACR        := $(BINDIR)/yacr

.PHONY: all fmt vet build clean run test
all: build

$(BINDIR):
	mkdir -p $@

fmt: $(GOSRC)
	go fmt ./...

vet: $(GOSRC)
	go vet ./...

build: fmt vet | $(BINDIR)
	go build -o $(YACR) .

run: build
	$(YACR) run /bin/sh

test:
	go test ./...

clean:
	go clean
	rm -rf $(BINDIR)
