# gpiod

[![Build Status](https://travis-ci.org/warthog618/gpiod.svg)](https://travis-ci.org/warthog618/gpiod)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/warthog618/gpiod)
[![Go Report Card](https://goreportcard.com/badge/github.com/warthog618/gpiod)](https://goreportcard.com/report/github.com/warthog618/gpiod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/gpiod/blob/master/LICENSE)

A native Go library for Linux GPIO.

**gpiod** is a library for accessing GPIO pins/lines on Linux platforms using
the GPIO character device.

The goal of this library is to provide the Go equivalent of the C
**[libgpiod](https://git.kernel.org/pub/scm/libs/libgpiod/libgpiod.git/)**
library. The intent is not to mirror the **libgpiod** API but to provide the
equivalent functionality.

## Features

Supports the following functionality per line and for collections of lines:

- direction (input/output)<sup>1</sup>
- write (active/inactive)
- read (active/inactive)
- active high/low (defaults to high)
- output mode (normal/open-drain/open-source)
- pull up/down<sup>2</sup>
- watches and edge detection (rising/falling/both)
- chip and line labels

<sup>1</sup>Dynamically changing line direction without releasing the line
requires Linux v5.5 or later.

<sup>2</sup>Pull up/down support requires Linux v5.5 or later.

All library functions are safe to call from different goroutines.

## Usage

```go
import "github.com/warthog618/gpiod"
```

Error handling is omitted from the following examples for brevity.

### Chip Initialization

The Chip object is used to request lines from a GPIO chip.

A Chip object is constructed using the
[*NewChip*](https://pkg.go.dev/github.com/warthog618/gpiod#NewChip) function.

```go
c, _ := gpiod.NewChip("gpiochip0")
```

The parameter is the chip name, which corresponds to the name of the device in
the **/dev** directory, so in this example **/dev/gpiochip0**.

Default attributes for Lines requested from the Chip can be set via
[options](#line-options) to
[*NewChip*](https://pkg.go.dev/github.com/warthog618/gpiod#NewChip).

```go
c, _ := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("myapp"))
```

In this example the consumer label is defaulted to "myapp".

When no longer required, the chip should be closed to release resources:

```go
c.Close()
```

Closing a chip does not close or otherwise alter the state of any lines
requested from the chip.

### Line Requests

To alter the state of a
[line](https://pkg.go.dev/github.com/warthog618/gpiod#Line) it must first be
requested from the Chip, using
[*Chip.RequestLine*](https://pkg.go.dev/github.com/warthog618/gpiod#Chip.RequestLine):

```go
l, _ := c.RequestLine(4)                    // in its existing state
```

The offset parameter identifies the line on the chip, and is specific to the
GPIO chip.  To improve readability, convenience mappings can be provided for
specific devices, such as the Raspberry Pi:

```go
l, _ := c.RequestLine(rpi.J8p7)             // using Raspberry Pi J8 mapping
```

The initial state of the line can be set by providing [line
options](#line-options), as shown in this *AsOutput* example:

```go
l, _ := c.RequestLine(4,gpiod.AsOutput(1))  // as an output line
```

Multiple lines from the same chip may be requested as a collection of
[lines](https://pkg.go.dev/github.com/warthog618/gpiod#Lines) using
[*Chip.RequestLines*](https://pkg.go.dev/github.com/warthog618/gpiod#Chip.RequestLines):

```go
ll, _ := c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput(0, 0, 1, 1))
```

Note that [line options](#line-options) applied to a collection of lines apply
to all lines in the collection.

When no longer required, the line(s) should be closed to release resources:

```go
l.Close()
ll.Close()
```

### Line Info

[Info](https://pkg.go.dev/github.com/warthog618/gpiod#LineInfo) about a line can
be read at any time from the chip using the
[*LineInfo*](https://pkg.go.dev/github.com/warthog618/gpiod#Chip.LineInfo)
method:

```go
inf, _ := c.LineInfo(4)
inf, _ := c.LineInfo(rpi.J8p7) // Using Raspberry Pi J8 mapping
```

Note that the line info does not include the value.  The line must be requested
from the chip to access the value.  Once requested, the line info can also be
read from the line:

```go
inf, _ := l.Info()
infs, _ := ll.Info()
```

### Direction

The line direction can be controlled using the *AsInput* and *AsOutput* [line
options](#line-options):

```go
l, _ := c.RequestLine(4,gpiod.AsInput) // during request
l.Reconfigure(gpiod.AsInput)           // set direction to Input
l.Reconfigure(gpiod.AsOutput(0))       // set direction to Output (and value to inactive)
```

### Read Input

The current line level can be read with the
[*Value*](https://pkg.go.dev/github.com/warthog618/gpiod#Line.Value)
method:

```go
r, _ := l.Value()  // Read state from line (active / inactive)
```

For collections of lines, the level of all lines is read simultaneously using
the [*Values*](https://pkg.go.dev/github.com/warthog618/gpiod#Lines.SetValues)
method:

```go
rr := []int{0, 0, 0, 0} // buffer to read into...
ll.Values(rr)           // Read the state of a collection of lines
```

### Write Output

The current line level can be set with the
[*SetValue*](https://pkg.go.dev/github.com/warthog618/gpiod#Line.SetValue)
method:

```go
l.SetValue(1)     // Set line active
l.SetValue(0)     // Set line inactive
```

Also refer to the [blinker](example/blinker/blinker.go) example.

For collections of lines, all lines are set simultaneously using the
[*SetValues*](https://pkg.go.dev/github.com/warthog618/gpiod#Lines.SetValues)
method:

```go
ll.SetValues([]int{0, 1, 0, 1}) // Set a collection of lines
```

### Watches

The state of an input line can be watched and trigger calls to handler
functions.

The watch can be on rising or falling edges, or both.

The handler function is passed a
[*LineEvent*](https://pkg.go.dev/github.com/warthog618/gpiod#LineEvent), which
contains the offset of the triggering line, the time the edge was detected and
the type of edge detected:

```go
func handler(evt gpiod.LineEvent) {
  // handle change in line state
}

l, _ := c.RequestLine(rpi.J8p7, gpiod.WithBothEdges(handler)))
```

A watch can be removed by closing the line:

```go
l.Close()
```

Also see the [watcher](example/watcher/watcher.go) example.

### Find

Lines can be found by the GPIO label name as returned in line info and set by
device-tree using the
[*FindLine*](https://pkg.go.dev/github.com/warthog618/gpiod#FindLine) function:

```go
chipname, offset, _ := gpiod.FindLine("LED A")
c, _ := gpiod.NewChip(chipname, gpiod.WithConsumer("myapp"))
l, _ := c.RequestLine(offset)
```

### Active Level

The values used throughout the API for line values are the logical value, which
is 0 for inactive and 1 for active. The physical level considered active can be
controlled using the *AsActiveHigh* and *AsActiveLow* [line
options](#line-options):

```go
l, _ := c.RequestLine(4,gpiod.AsActiveLow) // during request
l.Reconfigure(gpiod.AsActiveHigh)          // once requested
```

Lines are typically active high by default.

### Bias

The bias [line options](#line-options) control the pull up/down state of the
line:

```go
l,_ := c.RequestLine(4,gpiod.WithPullUp)  // during request
l.Reconfigure(gpiod.WithBiasDisable)      // once requested
```

Note that bias options require Linux v5.5 or later.

### Drive

The drive options control how an output line is driven when active and inactive:

```go
l,_ := c.RequestLine(4,gpiod.AsOpenDrain) // during request
l.Reconfigure(gpiod.AsOpenSource)         // once requested
```

The default drive for output lines is push-pull, which drives the line in both
directions.

### Line Options

Line attributes are set via options to *Chip.RequestLine(s)* and
*Line.Reconfigure*.  These override any default which may be set in *NewChip*.
Only one option from each category may be applied.  If multiple options from a
category are applied then all but the last are ignored.

The line options are:

Option | Category | Description
---|---|---
*WithConsumer*<sup>1</sup>|Info|Set the consumer label for the lines
*AsActiveLow*|Level|Treat a low physical line level as active
*AsActiveHigh*|Level|Treat a high physical line level as active (default)
*AsInput*|Direction|Request lines as input
*AsIs*<sup>2</sup>|Direction|Request lines in their current input/output state (default)
*AsOutput(\<values\>...)*<sup>3</sup>|Direction|Request lines as output with initial values
*AsPushPull*|Drive<sup>3</sup>|Request output lines drive both high and low (default)
*AsOpenDrain*|Drive<sup>3</sup>|Request lines as open drain outputs
*AsOpenSource*|Drive<sup>3</sup>|Request lines as open source outputs
*WithFallingEdge(eh)*|Edge<sup>4</sup>|Request lines with falling edge detection, with events passed to the provided event handler
*WithRisingEdge(eh)*|Edge<sup>4</sup>|Request lines with rising edge detection, with events passed to the provided event handler
*WithBothEdges(eh)*|Edge<sup>4</sup>|Request lines with rising and falling edge detection, with events passed to the provided event handler
*WithBiasDisable*|Bias<sup>5</sup>|Request the lines have internal bias disabled
*WithPullDown*|Bias<sup>5</sup>|Request the lines have internal pull-down enabled
*WithPullUp*|Bias<sup>5</sup>|Request the lines have internal pull-up enabled

<sup>1</sup> WithConsumer can be provided to either *NewChip* or
*Chip.RequestLine(s)*, and cannot be used with *Line.Reconfigure*.

<sup>2</sup> The AsIs option can only be provided to *Chip.RequestLine(s)*, and
cannot be used with *NewChip* or *Line.Reconfigure*.

<sup>3</sup> The AsOutput and Drive options can only be provided to either
*Chip.RequestLine(s)* or *Line.Reconfigure*, and cannot be used with *NewChip*.

<sup>4</sup> Edge options can only be provided to *Chip.RequestLine(s)*, and
cannot be used with *NewChip* or *Line.Reconfigure*.

<sup>5</sup> Bias options require Linux v5.5 or later.

## Tools

A command line utility, **gpiodctl**, can be found in the cmd directory and is
provided to allow manual or scripted manipulation of GPIO lines.  This utility
combines the Go equivalent of all the **libgpiod** command line tools into a
single tool.

```sh
gpiodctl is a utility to control GPIO lines on Linux GPIO character devices

Usage:
  gpiodctl [flags]
  gpiodctl [command]

Available Commands:
  detect      Detect available GPIO chips
  find        Find a GPIO line by name
  get         Get the state of a line
  help        Help about any command
  info        Info about chip lines
  mon         Monitor the state of a line
  set         Set the state of a line
  version     Display the version

Flags:
  -h, --help   help for gpiodctl

Use "gpiodctl [command] --help" for more information about a command.

```

The Go equivalent of each of the **libgpiod** command line tools can also be
found in the cmd directory.

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

The library is fully tested, other than some error cases and sanity checks that
are difficult to trigger.

The tests require a kernel release 5.1.0 or later to run.  For all the tests to
pass a kernel 5.5.0 or later is required.

The test user must have access to the **/dev/gpiochip0** character device.

### Platforms

The tests can be run on either of two platforms:

- gpio-mockup (default)
- Raspberry Pi

#### gpio-mockup

The gpio-mockup platform is any Linux platform with a recent kernel that supports
the **gpio-mockup** loadable module. **gpio-mockup** must be built as a module
and the test user must have rights to load and unload the module.

The **gpio-mockup** is the default platform for tests and benchmarks as it does not interact with physical hardware and so is always safe to run.

#### Raspberry Pi

On Raspberry Pi, the tests are intended to be run on a board with J8 pins 11 and
12 floating and with pins 15 and 16 tied together, possibly using a jumper
across the header.  The tests set J8 pins 11, 12 and 16 to outputs so **DO NOT**
run them on hardware where any of those pins is being externally driven.

The Raspberry Pi platform is selected by specifying the platform parameter on the test command line:

```sh
go test -platform=rpi
```

Tests have been run successfully on Raspberry Pi Zero W and Pi 4B.  The library
should also work on other Raspberry Pi variants, I just haven't gotten around to
testing them yet.

The tests can be cross-compiled from other platforms using:

```sh
GOOS=linux GOARCH=arm GOARM=6 go test -c
```

Later Pis can also use ARM7 (GOARM=7).

### Benchmarks

The tests include benchmarks on reads, writes, bulk reads and writes,  and
interrupt latency.

These are the results from a Raspberry Pi Zero W built with Go 1.13:

```sh
$ ./gpiod.test -platform=rpi -test.bench=.*
goos: linux
goarch: arm
pkg: gpiod
BenchmarkLineValue             157851          7160 ns/op
BenchmarkLinesValues           152865          7599 ns/op
BenchmarkLineSetValue          171585          6782 ns/op
BenchmarkLinesSetValues        155041          7995 ns/op
BenchmarkInterruptLatency        2041        581938 ns/op
PASS
```

## Prerequisites

The library targets Linux with support for the GPIO character device API.  That
generally means that **/dev/gpiochip0** exists.

The Bias line options and the Line.Reconfigure method both require Linux v5.5 or
later.

The caller must have access to the character device - typically
**/dev/gpiochip0**.  That is generally root unless you have changed the
permissions of that device.
