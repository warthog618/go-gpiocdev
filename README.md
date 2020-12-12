# gpiod

[![Build Status](https://travis-ci.org/warthog618/gpiod.svg)](https://travis-ci.org/warthog618/gpiod)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/warthog618/gpiod)](https://pkg.go.dev/github.com/warthog618/gpiod)
[![Go Report Card](https://goreportcard.com/badge/github.com/warthog618/gpiod)](https://goreportcard.com/report/github.com/warthog618/gpiod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/gpiod/blob/master/LICENSE)

A native Go library for Linux GPIO.

**gpiod** is a library for accessing GPIO pins/lines on Linux platforms using
the GPIO character device.

The goal of this library is to provide the Go equivalent of the C
**[libgpiod](https://git.kernel.org/pub/scm/libs/libgpiod/libgpiod.git/)**
library. The intent is not to mirror the **libgpiod** API but to provide the
equivalent functionality.

:warning: v0.6.0 introduces a few API breaking changes.  Refer to
the [release notes](#release-notes) if updating from an older version.

## Features

Supports the following functionality per line and for collections of lines:

- direction (input/output)<sup>**1**</sup>
- write (active/inactive)
- read (active/inactive)
- active high/low (defaults to high)
- output mode (push-pull/open-drain/open-source)
- pull up/down<sup>**2**</sup>
- watches and edge detection (rising/falling/both)
- chip and line labels
- debouncing input lines<sup>**3**</sup>
- different configurations for lines within a collection<sup>**3**</sup>

<sup>**1**</sup> Dynamically changing line direction without releasing the line
requires Linux v5.5 or later.

<sup>**2**</sup> Requires Linux v5.5 or later.

<sup>**3**</sup> Requires Linux v5.10 or later.

All library functions are safe to call from different goroutines.

## Quick Start

A simple piece of wire example that reads the value of an input line (pin 2) and
writes its value to an output line (pin 3):

```go
import "github.com/warthog618/gpiod"

...

c, _ := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("softwire"))
in, _ := c.RequestLine(2, gpiod.AsInput)
val, _ := in.Value()
out, _ := c.RequestLine(3, gpiod.AsOutput(val))

...
```

Error handling and releasing of resources omitted for brevity.

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

The list of currently available GPIO chips is returned by the *Chips* function:

```go
cc := gpiod.Chips()
```

Default attributes for Lines requested from the Chip can be set via
[configuration options](#configuration-options) to
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
from the chip to access the value.

Once requested, the line info can also be read from the line:

```go
inf, _ := l.Info()
infs, _ := ll.Info()
```

#### Info Watches

Changes to the line info can be monitored by adding an info watch for the line:

```go
func infoChangeHandler( evt gpiod.LineInfoChangeEvent) {
    // handle change in line info
}

inf, _ := c.WatchLineInfo(4, infoChangeHandler)
```

Note that the info watch does not monitor the line value (active or inactive)
only its configuration.  Refer to [Edge Watches](#edge-watches) for monitoring
line value.

An info watch can be cancelled by unwatching:

```go
c.UnwatchLineInfo(4)
```

or by closing the chip.

### Line Requests

To read or alter the value of a
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

The initial configuration of the line can be set by providing line
[configuration options](#configuration-options), as shown in this *AsOutput*
example:

```go
l, _ := c.RequestLine(4, gpiod.AsOutput(1))  // as an output line
```

Multiple lines from the same chip may be requested as a collection of
[lines](https://pkg.go.dev/github.com/warthog618/gpiod#Lines) using
[*Chip.RequestLines*](https://pkg.go.dev/github.com/warthog618/gpiod#Chip.RequestLines):

```go
ll, _ := c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput(0, 0, 1, 1))
```

When no longer required, the line(s) should be closed to release resources:

```go
l.Close()
ll.Close()
```

### Line Values

Lines must be requsted using [*Chip.RequestLines*](#line-requests) before their
values can be accessed.

#### Read Input

The current line value can be read with the
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

#### Write Output

The current line value can be set with the
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

#### Edge Watches

The value of an input line can be watched and trigger calls to handler
functions.

The watch can be on rising or falling edges, or both.

The events are passed to a handler function provided using the
*WithEventHandler(eh)* option.  The handler function is passed a
[*LineEvent*](https://pkg.go.dev/github.com/warthog618/gpiod#LineEvent), which
contains details of the edge event including the offset of the triggering line,
the time the edge was detected and the type of edge detected:

```go
func handler(evt gpiod.LineEvent) {
  // handle edge event
}

l, _ = c.RequestLine(rpi.J8p7, gpiod.WithEventHandler(handler), gpiod.WithBothEdges)
```

An edge watch can be removed by closing the line:

```go
l.Close()
```

or by reconfiguring the requested lines to disable edge detection:

```go
l.Reconfigure(gpiod.WithoutEdges)
```

Also see the [watcher](example/watcher/watcher.go) example.

### Line Configuration

Line configuration is set via [options](#configuration-options) to
*Chip.RequestLine(s)* and *Line.Reconfigure*.  These override any default which
may be set in *NewChip*.

Note that configuration options applied to a collection of lines apply to all
lines in the collection, unless they are applied to a subset of the requested
lines using the *WithLines* option.

#### Reconfiguration

Requested lines may be reconfigured using the Reconfigure method:

```go
l.Reconfigure(gpiod.AsInput)            // set direction to Input
ll.Reconfigure(gpiod.AsOutput(1, 0))    // set direction to Output (and values to active and inactive)
```

The *Line.Reconfigure* method accepts differential changes to the configuration
for the lines, so option categories not specified or overridden by the specified
changes will remain unchanged.

The *Line.Reconfigure* method requires Linux v5.5 or later.

#### Complex Configurations

It is sometimes necessary for the configuration of lines within a request to
have slightly different configurations.  Line options may be applied to a subset
of requested lines using the *WithLines(offsets, options)* option.

The following example requests a set of output lines and sets some of the lines
in the request to active low:

```go
ll, _ = c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput(0, 0, 1, 1),
    gpiod.WithLines([]int{0, 3}, gpiod.AsActiveLow),
    gpiod.AsOpenDrain)
```

The configuration of the subset of lines inherits the configuration of the
request at the point the *WithLines* is invoked.  Subsequent changes to the
request configuration do not alter the configuration of the subset - in the
example above, lines 0 and 3 will not be configured as open-drain.

Once a line's configuration has branched from the request configuration it can
only be altered with *WithLines* options:

```go
ll.Reconfigure(gpiod.WithLines([]int{0}, gpiod.AsActiveHigh))
```

or reset to the request configuration using the *Defaulted* option:

```go
ll.Reconfigure(gpiod.WithLines([]int{3}, gpiod.Defaulted))
```

Complex configurations require Linux v5.10 or later.

#### Categories

Most line configuration options belong to one of the following categories:

- Active Level
- Direction
- Bias
- Drive
- Debounce
- Edge Detection
- Event Clock

Only one option from each category may be applied.  If multiple options from a
category are applied then all but the last are ignored.

##### Active Level

The values used throughout the API for line values are the logical value, which
is 0 for inactive and 1 for active. The physical value considered active can be
controlled using the *AsActiveHigh* and *AsActiveLow* options:

```go
l, _ := c.RequestLine(4,gpiod.AsActiveLow) // during request
l.Reconfigure(gpiod.AsActiveHigh)          // once requested
```

Lines are typically active high by default.

##### Direction

The line direction can be controlled using the *AsInput* and *AsOutput* options:

```go
l, _ := c.RequestLine(4,gpiod.AsInput) // during request
l.Reconfigure(gpiod.AsInput)           // set direction to Input
l.Reconfigure(gpiod.AsOutput(0))       // set direction to Output (and value to inactive)
```

##### Bias

The bias options control the pull up/down state of the line:

```go
l,_ := c.RequestLine(4,gpiod.WithPullUp)  // during request
l.Reconfigure(gpiod.WithBiasDisabled)      // once requested
```

The bias options require Linux v5.5 or later.

##### Drive

The drive options control how an output line is driven when active and inactive:

```go
l,_ := c.RequestLine(4,gpiod.AsOpenDrain) // during request
l.Reconfigure(gpiod.AsOpenSource)         // once requested
```

The default drive for output lines is push-pull, which actively drives the line
in both directions.

##### Debounce

Input lines may be debounced using the *WithDebounce* option.  The debouncing will
be performed by the underlying hardware, if supported, else by the Linux
kernel.

```go
period := 10 * time.Millisecond
l, _ = c.RequestLine(4, gpiod.WithDebounce(period))// during request
l.Reconfigure(gpiod.WithDebounce(period))         // once requested
```

The WithDebounce option requires Linux v5.10 or later.

##### Edge Detection

The edge options control which edges on input lines will generate edge events.
Edge events are passed to the event handler specified in the *WithEventHandler(eh)*
option.

By default edge detection is not enabled on requested lines.

Refer to [Edge Watches](#edge-watches) for examples of the edge detection options.

##### Event Clock

The event clock options control the source clock used to timestamp edge events.
This is only useful for Linux kernels v5.11 and later - prior to that the clock
source is fixed.

The event clock source used by the kernel has changed over time as follows:

Kernel Version | Clock source
--- | ---
pre-v5.7 | CLOCK_REALTIME
v5.7 - v5.10 | CLOCK_MONOTONIC
v5.11 and later | configurable

Determining which clock the edge event timestamps contain is currently left as
an exercise for the user.

#### Configuration Options

The available configuration options are:

Option | Category | Description
---|---|---
*WithConsumer*<sup>**1**</sup> | Info | Set the consumer label for the lines
*AsActiveLow* | Level | Treat a low physical line value as active
*AsActiveHigh* | Level | Treat a high physical line value as active (**default**)
*AsInput* | Direction | Request lines as input
*AsIs*<sup>**2**</sup> | Direction | Request lines in their current input/output state (**default**)
*AsOutput(\<values\>...)*<sup>**3**</sup> | Direction | Request lines as output with the provided values
*AsPushPull* | Drive | Request output lines drive both high and low (**default**)
*AsOpenDrain* | Drive | Request lines as open drain outputs
*AsOpenSource* | Drive | Request lines as open source outputs
*WithEventHandler(eh)<sup>**1**</sup>* |  | Send edge events detected on requested lines to the provided handler
*WithEventBufferSize(num)<sup>**1**,**5**</sup>* |  | Suggest the minimum number of events that can be stored in the kernel event buffer for the requested lines
*WithFallingEdge* | Edge Detection<sup>**3**</sup> | Request lines with falling edge detection
*WithRisingEdge* | Edge Detection<sup>**3**</sup> | Request lines with rising edge detection
*WithBothEdges* | Edge Detection<sup>**3**</sup> | Request lines with rising and falling edge detection
*WithoutEdges*<sup>**5**</sup> | Edge Detection<sup>**3**</sup> | Request lines with edge detection disabled (**default**)
*WithBiasAsIs* | Bias<sup>**4**</sup> | Request the lines have their bias setting left unaltered (**default**)
*WithBiasDisabled* | Bias<sup>**4**</sup> | Request the lines have internal bias disabled
*WithPullDown* | Bias<sup>**4**</sup> | Request the lines have internal pull-down enabled
*WithPullUp* | Bias<sup>**4**</sup> | Request the lines have internal pull-up enabled
*WithDebounce(period)*<sup>**5**</sup> | Debounce | Request the lines be debounced with the provided period
*WithMonotonicEventClock* | Event Clock | Request the timestamp in edge events use the monotonic clock (**default**)
*WithRealtimeEventClock*<sup>**6**</sup> | Event Clock | Request the timestamp in edge events use the realtime clock
*WithLines(offsets, options...)*<sup>3,5</sup> |  | Specify configuration options for a subset of lines in a request
*Defaulted*<sup>**5**</sup> |  | Reset the configuration for a request to the default configuration, or the configuration of a particular line in a request to the default for that request

The options described as **default** are generally not required, except to override other options earlier in a chain of configuration options.

<sup>**1**</sup> Can be applied to either *NewChip* or *Chip.RequestLine*, but
cannot be used with *Line.Reconfigure*.

<sup>**2**</sup> Can be applied to *Chip.RequestLine*, but cannot be used
with *NewChip* or *Line.Reconfigure*.

<sup>**3**</sup> Can be applied to either *Chip.RequestLine* or
*Line.Reconfigure*, but cannot be used with *NewChip*.

<sup>**4**</sup> Requires Linux v5.5 or later.

<sup>**5**</sup> Requires Linux v5.10 or later.

<sup>**6**</sup> Requires Linux v5.11 or later.

## Tools

A command line utility, **gpiodctl**, can be found in the cmd directory and is
provided to allow manual or scripted manipulation of GPIO lines.  This utility
combines the Go equivalent of all the **libgpiod** command line tools into a
single tool.

```
gpiodctl is a utility to control GPIO lines on Linux GPIO character devices

Usage:
  gpiodctl [flags]
  gpiodctl [command]

Available Commands:
  detect      Detect available GPIO chips
  find        Find a GPIO line by name
  get         Get the state of a line or lines
  help        Help about any command
  info        Info about chip lines
  mon         Monitor the state of a line or lines
  set         Set the state of a line or lines
  version     Display the version
  watch       Watch lines for changes to the line info

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

- gpio-mockup (**default**)
- Raspberry Pi

#### gpio-mockup

The gpio-mockup platform is any Linux platform with a recent kernel that supports
the **gpio-mockup** loadable module. **gpio-mockup** must be built as a module
and the test user must have rights to load and unload the module.

The **gpio-mockup** is the default platform for tests and benchmarks as it does
not interact with physical hardware and so is always safe to run.

#### Raspberry Pi

On Raspberry Pi, the tests are intended to be run on a board with J8 pins 11 and
12 floating and with pins 15 and 16 tied together, possibly using a jumper
across the header.

:warning: The tests set J8 pins 11, 12 and 16 to outputs so **DO NOT**
run them on hardware where any of those pins is being externally driven.

The Raspberry Pi platform is selected by specifying the platform parameter on
the test command line:

```
go test -platform=rpi
```

Tests have been run successfully on Raspberry Pi Zero W and Pi 4B.  The library
should also work on other Raspberry Pi variants, I just haven't gotten around to
testing them yet.

The tests can be cross-compiled from other platforms using:

```
GOOS=linux GOARCH=arm GOARM=6 go test -c
```

Later Pis can also use ARM7 (GOARM=7).

### Benchmarks

The tests include benchmarks on reads, writes, bulk reads and writes,  and
interrupt latency.

These are the results from a Raspberry Pi Zero W running Linux v5.10 and built
with go1.15.6:

```
$ ./gpiod.test -platform=rpi -test.bench=.*
goos: linux
goarch: arm
pkg: github.com/warthog618/gpiod
BenchmarkChipNewClose              265       3949958 ns/op
BenchmarkLineInfo                28420         40192 ns/op
BenchmarkLineReconfigure         26079         46121 ns/op
BenchmarkLineValue              114961         10176 ns/op
BenchmarkLinesValues             66969         17367 ns/op
BenchmarkLineSetValue            92529         12531 ns/op
BenchmarkLinesSetValues          65965         17309 ns/op
BenchmarkInterruptLatency         1827        638202 ns/op
PASS
```

## Prerequisites

The library targets Linux with support for the GPIO character device API.  That
generally means that **/dev/gpiochip0** exists.

The caller must have access to the character device - typically
**/dev/gpiochip0**.  That is generally root unless you have changed the
permissions of that device.

The Bias line options and the Line.Reconfigure method both require Linux v5.5 or
later.

Debounce and other uAPI v2 features require Linux v5.10 or later.

The requirements for each [configuration option](#configuration-options) are
noted in that section.

## Release Notes

### v0.6.0

*gpiod* now supports both the old GPIO uAPI (v1) and the newer (v2) introduced
in Linux v5.10. The library automatically detects the available uAPI versions
and makes use of the latest.

Applications written for uAPI v1 will continue to work with uAPI v2.

Applications that make use of v2 specific features will return errors when run
on Linux kernels prior to v5.10.

Breaking API changes:

1. The event handler parameter has been moved from edge options into the
   *WithEventHandler(eh)* option to allow for reconfiguration of edge detection
   which is supported in Linux v5.10.

   Old edge options should be replaced with the *WithEventHandler* option and
   the now parameterless edge option, e.g.:

   ```sed
   s/gpiod\.WithBothEdges(/gpiod.WithBothEdges, gpiod.WithEventHandler(/g
   ```

2. *WithBiasDisable* is renamed *WithBiasDisabled*.  This option is probably
   rarely used and the renaming is trivial, so no backward compatibility is
   provided.

3. *FindLine* has been dropped as line names are not guaranteed to be unique.
   Iterating over the available chips and lines to search for line by name can
   be easily done - the *Chips* function provides the list of available chips as
   a starting point.

   Refer to the *find* command in **gpiodctl** for example code.
