GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

VERSION ?= $(shell git describe --tags --always --dirty 2> /dev/null )
LDFLAGS=-ldflags "-X=main.version=$(VERSION)"

spis=$(patsubst %.go, %, $(wildcard example/spi/*/*.go))
examples=$(patsubst %.go, %, $(wildcard example/*/*.go))
bins= $(spis) $(examples)

cmds=$(patsubst %.go, %, $(wildcard cmd/gpio*/gpio*.go))

all: tools $(bins)

$(cmds) : % : %.go
	cd $(@D); \
	$(GOBUILD) $(LDFLAGS)

$(bins) : % : %.go
	cd $(@D); \
	$(GOBUILD)

clean: 
	$(GOCLEAN) ./...

tools: $(cmds)

