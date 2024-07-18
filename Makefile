# SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
#
# SPDX-License-Identifier: MIT

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

VERSION ?= $(shell git describe --tags --always --dirty 2> /dev/null )
LDFLAGS=-ldflags "-X=main.version=$(VERSION)"

examples=$(patsubst %.go, %, $(wildcard examples/*/main.go))
bins= $(examples)

all: $(bins)

$(bins) : % : %.go
	cd $(@D); \
	$(GOBUILD)

clean: 
	$(GOCLEAN) ./...

