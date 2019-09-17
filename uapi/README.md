# uapi

[![GoDoc](https://godoc.org/github.com/warthog618/gpiod?status.svg)](https://godoc.org/github.com/warthog618/gpiod/uapi)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/warthog618/gpiod/blob/master/LICENSE)

GPIOD UAPI is a thin layer over the system ioctl calls that comprise the Linux GPIO UAPI.

This library is used by **[gpiod](https://github.com/warthog618/gpiod)** to interact with the Linux kernel.

The library is exposed to allow for testing of the UAPI with the minimal amount of Go in the way.

**gpiod** provides a higher level of abstraction, so for general use you probably want to be using that.

## API

The GPIO UAPI comprises six ioctls:

IOCTL | Scope | Description
---|--- | ---
[GetChipInfo](https://godoc.org/github.com/warthog618/gpiod/uapi#GetChipInfo) | chip | Returns information about the chip itself.
[GetLineInfo](https://godoc.org/github.com/warthog618/gpiod/uapi#GetLineInfo) | chip | Returns information about a particular line on the chip.
[GetLineHandle](https://godoc.org/github.com/warthog618/gpiod/uapi#GetLineHandle) | chip | Requests a set of lines, and returns a file handle for ioctl commands.  The set may be any subset of the lines supported by the chip, including a single line.  This may be used for both input and output lines.  The lines remain reserved by the caller until the returned fd is closed.
[GetLineEvent](https://godoc.org/github.com/warthog618/gpiod/uapi#GetLineEvent) | chip | Requests an individual input line with edge detection enabled, and returns a file handle for ioctl commands and to return edge events.  Events can only be requested on input lines.  The line remains reserved by the caller until the returned fd is closed.
[GetLineValues](https://godoc.org/github.com/warthog618/gpiod/uapi#GetLineValues) | line | Returns the current value of a set of lines.
[SetLineValues](https://godoc.org/github.com/warthog618/gpiod/uapi#SetLineValues) | line | Sets the current value of a set of lines.

## Usage

The following is a brief example of the usage of each of the major functions:

```go
    f, _ := os.OpenFile("/dev/gpiochip0", unix.O_CLOEXEC, unix.O_RDONLY)

    // get chip info
    ci, _ := uapi.GetChipInfo(f.Fd())

    // get line info
    li, _ := uapi.GetLineInfo(f.Fd(), offset)

    // request a line
    hr := uapi.HandleRequest{
        Lines: uint32(len(offsets)),
        Flags: handleFlags,
        // initialise Offsets, DefaultValues and Consumer...
    }
    err := uapi.GetLineHandle(f.Fd(), &hr)

    // request a line with events
    er := uapi.EventRequest{
        Offset:      offset,
        HandleFlags: handleFlags,
        EventFlags:  eventFlags,
        // initialise Consumer...
    }
    err := uapi.GetLineEvent(f.Fd(), &er)
    if err != nil {
        // wait on er.fd for events...

        // read event
        evt, _ := uapi.ReadEvent(er.fd)
    }

    // get values
    var values uapi.HandleData
    _ := uapi.GetLineValues(er.fd, &values)

    // set values
    values[0] = uint8(value)
    _ := uapi.SetLineValues(hr.fd, values)

```

Error handling and other tedious bits, such as initialising the arrays in the requests, omitted for brevity.

Refer to **[gpiod](https://github.com/warthog618/gpiod)** for a concrete example of uapi usage.
