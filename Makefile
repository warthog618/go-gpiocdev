# SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
#
# SPDX-License-Identifier: MIT

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

VERSION ?= $(shell git describe --tags --always --dirty 2> /dev/null )
LDFLAGS=-ldflags "-X=main.version=$(VERSION)"

spis=$(patsubst %.go, %, $(wildcard example/spi/*/*.go))
examples=$(patsubst %.go, %, $(wildcard example/*/*.go))
bins= $(spis) $(examples)

cli=$(patsubst %.go, %, $(wildcard cli/gpio*.go))

all: tools $(bins)

$(cli) : % : %.go
	cd $(@D); \
	$(GOBUILD) -o gpiocdev $(LDFLAGS)

$(bins) : % : %.go
	cd $(@D); \
	$(GOBUILD)

clean: 
	$(GOCLEAN) ./...
	rm cli/gpiocdev

tools: $(cli)

