# gpiod

[![GoDoc](https://godoc.org/github.com/warthog618/gpiod?status.svg)](https://godoc.org/github.com/warthog618/gpiod)
[![Go Report Card](https://goreportcard.com/badge/github.com/warthog618/gpiod)](https://goreportcard.com/report/github.com/warthog618/gpiod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/gpiod/blob/master/LICENSE)

A native Go GPIO library for Linux.

The goal of this library is to provide the Go equivalent of **[libgpiod](https://git.kernel.org/pub/scm/libs/libgpiod/libgpiod.git/)**.

The intent is not to mirror the **libgpiod** API but to provide the equivalent functionality.

## Features

Supports the following functionality per line and for groups of lines:

- direction (input / output )
- write (active / inactive)
- read (active / inactive)
- active high/low (defaults to high)
- output mode (normal / open drain / open source)
- edge detection (rising / falling / both)
- chip and line labels

Note that setting pull up/down is not supported.  This a limitation of the current GPIO UAPI.

## Usage

```go
import "github.com/warthog618/gpiod"
```

The following provides a quick overview of common use cases using **gpiod**:

Get a chip and set the default label to apply to requested lines:

```go
    c, _ := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("gpiodetect"))
```

Get info about a line:

```go
    inf, _ := c.LineInfo(2)
```

Note that the line info does not include the value.  The line must be requested to access the value.

Request a line as is, so not altering direction settings:

```go
    l, _ := c.RequestLine(2)
```

Request a line as input, so altering direction settings:

```go
    l, _ = c.RequestLine(2, gpiod.AsInput())
```

Request a line as output - initially set active, then set inactive:

```go
    l, _ = c.RequestLine(3, gpiod.AsOutput(1))
    // ...
    l.SetValue(0)
```

Note that sets can only be applied to lines explicitly requested as outputs, as is the case here.

Request a line as an open-drain output:

```go
    l, _ = c.RequestLine(3, gpiod.AsOpenDrain())
```

Get the value of a line (which must be requested first):

```go
    v,_ := l.Value()
```

Request a line with edge detection:

```go
    handler := func(evt gpiod.LineEvent) {
        // handle the edge event here
    }
    l, _ = c.RequestLine(4, gpiod.WithRisingEdge(handler))
```

Request a bunch of lines as output and set initial values, then change later:

```go
    ll, _ := c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput(0, 0, 1, 1))
    // ...
    ll.SetValues([]int{0, 1, 1, 0})
```

Get the values of a set of lines (which must be requested first):

```go
    vv,_ := ll.GetValues()
```

Error handling omitted for brevity.

### API

The API consists of three major objects - the Chip and the Line/Lines.

The [Chip](https://godoc.org/github.com/warthog618/gpiod#Chip) represents the GPIO chip itself.  The Chip is the source of chip and line info and the constructor of Line/Lines.

Lines must be requested from the Chip before you can do anything useful with them, such as setting or getting values, or detecting edges.  Also note that, as per the underlying UAPI, the majority of line attributes, including input/output, active low, and open drain/source, can only be set at request time, and are immutable while the request is held.

The [Line](https://godoc.org/github.com/warthog618/gpiod#Line) and [Lines](https://godoc.org/github.com/warthog618/gpiod#Lines) are essentially the same.  They both represent a requested set of lines, but in the case of the Line that set is one.  The Line API is slightly simplified as it only has to deal with the one line, not a larger set.

Both Chip and Line/Lines are safe to call from different goroutines.

## Tools

The Go equivalent of the **libgpiod** command line tools can be found in the cmd directory.

Those tools are:

Tool | Description
--- | ---
gpiodetect | Report all the gpiochips available on the system.
gpioinfo | Report the details of all the lines available on gpiochips.
gpiofind | Find the gpiochip and offset of a line by name.
gpioget | Get the value of a line or a set of lines on one gpiochip.
gpioset | Set of value of a line or a set of lines on one gpiochip.
gpiomon | Report edges detected on a line or set of lines on one gpiochip.

## Tests

The library is fully tested, other than some error cases and sanity checks that are difficult to trigger.

### Platforms

The tests can be run on either of two platforms:

- Raspberry Pi
- gpio-mockup

#### Raspberry Pi

On Raspberry Pi, the tests are intended to be run on a board with J8 pins 11 and 12 floating and with pins 15 and 16 tied together, possibly using a jumper across the header.  The tests set J8 pins 11, 12 and 16 to outputs so **DO NOT** run them on hardware where any of those pins is being externally driven.

The test user must have access to the **/dev/gpiochip0** character device.

Tests have been run successfully on Raspberry Pi Zero W.  The library should also work on other Raspberry Pi variants, I just haven't gotten around to testing them yet.

The tests can be cross-compiled from other platforms using:

```sh
GOOS=linux GOARCH=arm GOARM=6 go test -c
```

Later Pis can also use ARM7 (GOARM=7).

#### gpio-mockup

Other than the Raspberry Pi, the tests can be run on any Linux platform with a recent kernel that supports the **gpio-mockup** loadable module.  **gpio-mockup** must be built as a module and the test user must have rights to load and unload the module.

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
