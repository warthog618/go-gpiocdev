<!--
SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>

SPDX-License-Identifier: MIT
-->

# uapi

[![PkgGoDev](https://pkg.go.dev/badge/github.com/warthog618/go-gpiocdev/uapi)](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/go-gpiocdev/blob/master/LICENSE)

**gpiocdev uapi** is a thin layer over the system ioctl calls that comprise the Linux GPIO uAPI.

This library is used by **[gpiocdev](https://github.com/warthog618/go-gpiocdev)** to interact with the Linux kernel.

The library is exposed to allow for testing of the uAPI with the minimal amount of Go in the way.

**gpiocdev** provides a higher level of abstraction, so for general use you probably want to be using that.

## API

The library targets the latest version of the GPIO uAPI, v2, supported by Linux 5.10 or later,
and exposes functions that provide a thin wrapper around the uAPI IOCTL calls.

Older versions of the **uapi** module, up to v0.9.x, supported both versions of the uAPI and can
still be used if you are stuck on a kernel older than Linux 5.10.

The GPIO uAPI v2 comprises eight ioctls:

IOCTL | Scope | Function | Description
---|---|---|---
GetChipInfo| chip | [GetChipInfo](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#GetChipInfo) | Return information about the chip itself.
GetLineInfoV2| chip | [GetLineInfo](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#GetLineInfo) | Return information about a particular line on the chip.
GetLine| chip | [GetLine](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#GetLine) | Request a set of lines, and returns a file handle for ioctl commands.  The set may be any subset of the lines supported by the chip, including a single line.  This may be used for both input and output lines.  The lines remain reserved by the caller until the returned fd is closed.
GetLineValuesV2| line | [GetLineValues](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#GetLineValues) | Return the current value of a set of lines in an existing line request.
SetLineValuesV2| line | [SetLineValues](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#SetLineValues) | Set the current value of a set of lines in an existing line request.
SetLineConfigV2| line | [SetLineConfig](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#SetLineConfig) | Update the configuration of the lines in an existing line request.
WatchLineInfoV2| chip| [WatchLineInfo](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#WatchLineInfo) | Add a watch for changes to the info of a particular line on the chip.
UnwatchLineInfo| chip| [UnwatchLineInfo](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#UnwatchLineInfo) | Remove a watch for changes to the info of a particular line on the chip.

Additionally, the [ReadLineEvent](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#ReadLineEvent)
function reads [LineEvent](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#LineEvent)s
from the fd returned by GetLine, and the
[ReadLineInfoChanged](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#ReadLineInfoChanged)
function reads [LineInfoChanged](https://pkg.go.dev/github.com/warthog618/go-gpiocdev/uapi#LineInfoChanged)
events from the chip fd for watches set by WatchLineInfo.

## Usage

The following is a brief example of the usage of the major functions:

```go
    f, _ := os.OpenFile("/dev/gpiochip0", unix.O_CLOEXEC, unix.O_RDONLY)

    // get chip info
    ci, _ := uapi.GetChipInfo(f.Fd())
    fmt.Print(ci)

    // get line info
    li, _ := uapi.GetLineInfo(f.Fd(), offset)
    fmt.Print(li)

    // request a line
    lr := uapi.LineRequest{
        Lines: uint32(len(offsets)),
        Config: uapi.LineConfig{
            Flags: uapi.LineFlagOutput,
        },
        // initialise Offsets, OutputValues and Consumer...
    }
    err := uapi.GetLine(f.Fd(), &lr)

    // request a line with events
    lr = uapi.LineRequest{
        Lines: uint32(len(offsets)),
        Config: uapi.LineConfig{
            Flags: uapi.LineFlagInput | uapi.LineFlagActiveLow | uapi.LineFlagEdgeBoth,
        },
        // initialise Offsets and Consumer...
    }
    err = uapi.GetLine(f.Fd(), &lr)
    if err != nil {
        // wait on lr.fd for events...

        // read event
        evt, _ := uapi.ReadLineEvent(uintptr(lr.Fd))
        fmt.Print(evt)
    }

    // get values
    var values uapi.LineValues
    err = uapi.GetLineValues(uintptr(lr.Fd), &values)

    // set values
    err = uapi.SetLineValues(uintptr(lr.Fd), values)

    // update line config - change to outputs
    err = uapi.SetLineConfig(uintptr(lr.Fd), &uapi.LineConfig{
        Flags: uapi.LineFlagOutput,
        NumAttrs: 1,
        // initialise OutputValues...
    })

```

Error handling and other tedious bits, such as initialising the arrays in the requests, omitted for brevity.

Refer to **[gpiocdev](https://github.com/warthog618/go-gpiocdev)** for a concrete example of uapi usage.

## Tests

The library is fully tested, other than some error cases and sanity checks that
are difficult to trigger.

The tests require a kernel release 5.19 or later to run, built with
**CONFIG_GPIO_SIM** set or as a module.

The tests must be run as root, to allow contruction of **gpio-sims**.
They can still be built as an unprivileged user, e.g.

```shell
$ go test -c
```

but must be run as root.

The tests can also be cross-compiled for other platforms.
e.g. build tests for a Raspberry Pi using:

```shell
$ GOOS=linux GOARCH=arm GOARM=6 go test -c
```

Later Pis can also use ARM7 (GOARM=7).

### Benchmarks

The tests include benchmarks on reads, writes, bulk reads and writes,  and
interrupt latency.

These are the results from a Raspberry Pi Zero W running Linux 7.2 and built
with go1.27.1:

```shell
$ ./uapi.test -test.bench=.* -test.run=^$
goos: linux
goarch: arm
pkg: github.com/warthog618/go-gpiocdev/uapi
cpu: ARMv6-compatible processor rev 7 (v6l)
BenchmarkChipOpenClose             	    2044	    710681 ns/op
BenchmarkLineInfo                  	   35424	     42201 ns/op
BenchmarkGetLine                   	    6002	    173513 ns/op
BenchmarkGetLineWithEdges          	     460	   2792954 ns/op
BenchmarkGetLineWithEdgesDebounced 	    5095	    250814 ns/op
BenchmarkGetLineValuesOne          	   69099	     15488 ns/op
BenchmarkGetLineValuesTen          	   47217	     22792 ns/op
BenchmarkSetLineValuesOne          	   85342	     15057 ns/op
BenchmarkSetLineValuesTen          	   52056	     22442 ns/op
BenchmarkSetLineValuesSparse       	   49161	     21117 ns/op
BenchmarkSetLineConfig             	   65604	     16552 ns/op
BenchmarkWatchLineInfo             	   18624	     68620 ns/op
PASS
```

The latency benchmark is no longer representative as the measurement now depends
on how quickly **gpio-sim** can toggle lines, and that is considerably slower
than how quickly **gpiocdev** responds.  For comparison, the same test using
looped Raspberry Pi lines produced a result of ~640μsec on the same platform.

And on a Raspberry Pi 4 running Linux 7.2 and built with go1.27.1:

```shell
$ ./uapi.test -test.bench=.* -test.run=^$
goos: linux
goarch: arm64
pkg: github.com/warthog618/go-gpiocdev/uapi
BenchmarkChipOpenClose-4               	   44066	     26387 ns/op
BenchmarkLineInfo-4                    	  549898	      2158 ns/op
BenchmarkGetLine-4                     	   92293	     12060 ns/op
BenchmarkGetLineWithEdges-4            	    9670	    122422 ns/op
BenchmarkGetLineWithEdgesDebounced-4   	   62787	     18034 ns/op
BenchmarkGetLineValuesOne-4            	 1129938	      1071 ns/op
BenchmarkGetLineValuesTen-4            	  523214	      2336 ns/op
BenchmarkSetLineValuesOne-4            	 1024548	      1166 ns/op
BenchmarkSetLineValuesTen-4            	  527864	      2274 ns/op
BenchmarkSetLineValuesSparse-4         	  779002	      1553 ns/op
BenchmarkSetLineConfig-4               	 1157347	      1038 ns/op
BenchmarkWatchLineInfo-4               	  272320	      3818 ns/op
PASS
```

