# gpiod

[![GoDoc](https://godoc.org/github.com/warthog618/gpiod?status.svg)](https://godoc.org/github.com/warthog618/gpiod)
[![Go Report Card](https://goreportcard.com/badge/github.com/warthog618/gpiod)](https://goreportcard.com/report/github.com/warthog618/gpiod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/gpiod/blob/master/LICENSE)

A native Go GPIO library for Linux.

The goal of this library is to provide the Go equivalent of [libgpiod](https://git.kernel.org/pub/scm/libs/libgpiod/libgpiod.git/).  The intent is not to mirror the libgpiod API but to provide the equivalent functionality.

This is very much a work in progress, so I don't suggest using it for anything serious yet.

## Features

Supports the following functionality:

- Line direction (Input / Output )
- Write (High / Low)
- Read (High / Low)
- Bulk reads and writes
- Line output mode (Normal / Open Drain / OpenSource)
- Edge detection (Rising / Falling / Both)
- Line labels

And when complete will also support:

- Chip and line discovery
- Same tools as libgpiod

Note that pull up and down are not supported.  This a limitation of the current GPIO UAPI.

## Usage

The API is still in flux, but this is a current example:

```go
    // get a chip and set the default label to apply to requested lines
    c, _ := gpiod.NewChip("/dev/gpiochip0", gpiod.WithConsumer("gpiodetect"))

    // get info about a line
    inf, _ := c.LineInfo(2)

    // request a line as is, so not altering direction settings.
    li, _ := c.RequestLine(2)
    v, _ := li.Value()

    // request a line as input, so altering direction settings.
    li, _ = c.RequestLine(2, gpiod.AsInput())
    v, _ = li.Value()

    // request a line as outout
    li, _ = c.RequestLine(3, gpiod.AsOutput())
    v, _ = li.Value()
    li.SetValue(1)

    // request a line with edge detection
    handler := func(gpiod.LineEvent) {
        // handle the edge event here
    }
    li, _ = c.RequestLine(4, gpiod.WithRisingEdge(handler))
    v, _ = li.Value()

    // request a bunch of lines
    ll, _ := c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput())
    vv, _ := li.Values()
    ll.SetValues([]int{0, 1, 1, 0})

```

Error handling ommitted for brevity.

## Prerequisites

The library targets Linux with support for the GPIO character device API.  That generally means that */dev/gpiochip0* exists.

The caller must have access to the character device - */dev/gpiochip0*.  That is generally root unless you have changed the permissions of that device.
