<!--
SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>

SPDX-License-Identifier: MIT
-->

# uapi

[![PkgGoDev](https://pkg.go.dev/badge/github.com/warthog618/gpiod/uapi)](https://pkg.go.dev/github.com/warthog618/gpiod/uapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/gpiod/blob/master/LICENSE)

GPIOD UAPI is a thin layer over the system ioctl calls that comprise the Linux GPIO UAPI.

This library is used by **[gpiod](https://github.com/warthog618/gpiod)** to interact with the Linux kernel.

The library is exposed to allow for testing of the UAPI with the minimal amount of Go in the way.

**gpiod** provides a higher level of abstraction, so for general use you probably want to be using that.

## API

Both versions of the GPIO UAPI are supported; the current v2 and the deprecated v1.

### V2

The GPIO UAPI v2 comprises eight ioctls (two of which are unchanged from v1):

IOCTL | Scope | Description
---|--- | ---
[GetChipInfo](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetChipInfo) | chip | Return information about the chip itself.
[GetLineInfoV2](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetLineInfoV2) | chip | Return information about a particular line on the chip.
[GetLine](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetLine) | chip | Request a set of lines, and returns a file handle for ioctl commands.  The set may be any subset of the lines supported by the chip, including a single line.  This may be used for both input and output lines.  The lines remain reserved by the caller until the returned fd is closed.
[GetLineValuesV2](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetLineValuesV2) | line | Return the current value of a set of lines in an existing line request.
[SetLineValuesV2](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#SetLineValuesV2) | line | Set the current value of a set of lines in an existing line request.
[SetLineConfigV2](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#SetLineConfigV2) | line | Update the configuration of the lines in an existing line request.
[WatchLineInfoV2](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#WatchLineInfoV2) | chip | Add a watch for changes to the info of a particular line on the chip.
[UnwatchLineInfo](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#UnwatchLineInfo) | chip | Remove a watch for changes to the info of a particular line on the chip.

### V1

The GPIO UAPI v1 comprises nine ioctls:

IOCTL | Scope | Description
---|--- | ---
[GetChipInfo](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetChipInfo) | chip | Return information about the chip itself.
[GetLineInfo](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetLineInfo) | chip | Return information about a particular line on the chip.
[GetLineHandle](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetLineHandle) | chip | Request a set of lines, and returns a file handle for ioctl commands.  The set may be any subset of the lines supported by the chip, including a single line.  This may be used for both input and output lines.  The lines remain reserved by the caller until the returned fd is closed.
[GetLineEvent](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetLineEvent) | chip | Request an individual input line with edge detection enabled, and returns a file handle for ioctl commands and to return edge events.  Events can only be requested on input lines.  The line remains reserved by the caller until the returned fd is closed.
[GetLineValues](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#GetLineValues) | line | Return the current value of a set of lines in an existing handle or event request.
[SetLineValues](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#SetLineValues) | line | Set the current value of a set of lines in an existing handle request.
[SetLineConfig](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#SetLineConfig) | line | Update the configuration of the lines in an existing handle request.
[WatchLineInfo](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#WatchLineInfo) | chip | Add a watch for changes to the info of a particular line on the chip.
[UnwatchLineInfo](https://pkg.go.dev/github.com/warthog618/gpiod/uapi#UnwatchLineInfo) | chip | Remove a watch for changes to the info of a particular line on the chip.

## Usage

The following is a brief example of the usage of the major functions using v2 of the UAPI:

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
            Flags: uapi.LineFlagV2Output,
        },
        // initialise Offsets, OutputValues and Consumer...
    }
    err := uapi.GetLine(f.Fd(), &lr)

    // request a line with events
    lr = uapi.LineRequest{
        Lines: uint32(len(offsets)),
        Config: uapi.LineConfig{
            Flags: uapi.LineFlagV2Input | uapi.LineFlagV2ActiveLow | uapi.LineFlagV2EdgeBoth,
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
    err = uapi.GetLineValuesV2(uintptr(lr.Fd), &values)

    // set values
    err = uapi.SetLineValuesV2(uintptr(lr.Fd), values)

    // update line config - change to outputs
    err = uapi.SetLineConfigV2(uintptr(lr.Fd), &uapi.LineConfig{
        Flags: uapi.LineFlagV2Output,
        NumAttrs: 1,
        // initialise OutputValues...
    })

```

Error handling and other tedious bits, such as initialising the arrays in the requests, omitted for brevity.

Refer to **[gpiod](https://github.com/warthog618/gpiod)** for a concrete example of uapi usage.

This is essentially the same example using v1 of the UAPI:

```go
    f, _ := os.OpenFile("/dev/gpiochip0", unix.O_CLOEXEC, unix.O_RDONLY)

    // get chip info
    ci, _ := uapi.GetChipInfo(f.Fd())
    fmt.Print(ci)

    // get line info
    li, _ := uapi.GetLineInfo(f.Fd(), offset)
    fmt.Print(li)

    // request a line
    hr := uapi.HandleRequest{
    Lines: uint32(len(offsets)),
    Flags: uapi.HandleRequestOutput,
    // initialise Offsets, DefaultValues and Consumer...
    }
    err := uapi.GetLineHandle(f.Fd(), &hr)

    // request a line with events
    er := uapi.EventRequest{
    Offset:      uint32(offset),
    HandleFlags: uapi.HandleRequestActiveLow,
    EventFlags:  uapi.EventRequestBothEdges,
    // initialise Consumer...
    }
    err = uapi.GetLineEvent(f.Fd(), &er)
    if err != nil {
    // wait on er.fd for events...

    // read event
    evt, _ := uapi.ReadEvent(uintptr(er.Fd))
    fmt.Print(evt)
    }

    // get values
    var values uapi.HandleData
    err = uapi.GetLineValues(uintptr(er.Fd), &values)

    // set values
    values[0] = uint8(value)
    err = uapi.SetLineValues(uintptr(hr.Fd), values)

    // update line config - change to active low
    err = uapi.SetLineConfig(uintptr(hr.Fd), &uapi.HandleConfig{
        Flags: uapi.HandleRequestInput,
    })

```
