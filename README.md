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

    // request a line as output - initially set active
    li, _ = c.RequestLine(3, gpiod.AsOutput(1))
    v, _ = li.Value()
    li.SetValue(0) // then set inactive

    // request a line with edge detection
    handler := func(gpiod.LineEvent) {
        // handle the edge event here
    }
    li, _ = c.RequestLine(4, gpiod.WithRisingEdge(handler))
    v, _ = li.Value()

    // request a bunch of lines
    ll, _ := c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput(0, 0, 1, 1))
    vv, _ := li.Values()
    ll.SetValues(0, 1, 1, 0)

```

Error handling omitted for brevity.

## Tests

The library tests are still a work in progress...

### Platforms

The tests can be run on either of two platforms:

- Raspberry Pi
- gpio-mockup

#### Raspberry Pi

On Raspberry Pi, the tests are intended to be run on a board with J8 pins 11 and 12 floating and with pins 15 and 16 tied together, possibly using a jumper across the header.  The tests set J8 pins 11, 12 and 16 to outputs so **DO NOT** run them on hardware where that pin is being externally driven.

The test user must have access to the **/dev/gpiochip0** character device.

Tests have been run successfully on Raspberry Pi Zero W.  The library should also work on other Raspberry Pi variants, I just haven't gotten around to testing them yet.

The tests can be cross-compiled from other platforms using

```sh
GOOS=linux GOARCH=arm GOARM=6 go test -c
```

Later Pis can also use ARM7 (GOARM=7).

#### gpio-mockup

Other than the Raspberry Pi, the tests can be run on any Linux platform with a recent kernel that supports the gpio-mockup loadable module.  gpio-mockup must be built as a module and the test user must have rights to load and unload the module.

The tests require a kernel release 5.1.0 or later to run.  For all the tests to pass a kernel 5.3.0 or later is required.

### Benchmarks

The tests include benchmarks on reads, writes, bulk reads and writes,  and interrupt latency.

These are the results from a Raspberry Pi Zero W built with Go 1.13:

```sh
$ ./gpiod.test -test.bench=.*
goos: linux
goarch: arm
pkg: gpiod
BenchmarkLineValue              248344          5659 ns/op
BenchmarkLinesValues            142282          8377 ns/op
BenchmarkLineSetValue           178704          6632 ns/op
BenchmarkLinesSetValues         135058          7875 ns/op
BenchmarkInterruptLatency         2040        530332 ns/op
PASS
```

## Prerequisites

The library targets Linux with support for the GPIO character device API.  That generally means that **/dev/gpiochip0** exists.

The caller must have access to the character device - **/dev/gpiochip0**.  That is generally root unless you have changed the permissions of that device.
