// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// +build linux

// A collection of code snippets contained in the READMEs.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

func main() {
	// Chip Initialisation
	c, _ := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("myapp"))

	// Quick Start
	in, _ := c.RequestLine(2, gpiod.AsInput)
	val, _ := in.Value()
	out, _ := c.RequestLine(3, gpiod.AsOutput(val))
	in.Close()
	out.Close()

	// Line Requests
	l, _ := c.RequestLine(4)
	l.Close()
	l, _ = c.RequestLine(rpi.J8p7) // Using Raspberry Pi J8 mapping.
	l.Close()
	l, _ = c.RequestLine(4, gpiod.AsOutput(1))

	ll, _ := c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput(0, 0, 1, 1))

	// Line Info
	inf, _ := c.LineInfo(2)
	inf, _ = c.LineInfo(rpi.J8p7)
	inf, _ = l.Info()
	infs, _ := ll.Info()
	fmt.Printf("name: %s\n", inf.Name)
	fmt.Printf("name: %s\n", infs[0].Name)

	// Info Watch
	inf, _ = c.WatchLineInfo(4, infoChangeHandler)
	c.UnwatchLineInfo(4)

	// Active Level
	l, _ = c.RequestLine(4, gpiod.AsActiveLow) // during request
	l.Reconfigure(gpiod.AsActiveHigh)          // once requested

	// Direction
	l.Reconfigure(gpiod.AsInput)        // Set direction to Input
	l.Reconfigure(gpiod.AsOutput(1, 0)) // Set direction to Output (and values to active and inactive)

	// Input
	r, _ := l.Value() // Read state from line (active/inactive)
	fmt.Printf("value: %d\n", r)

	rr := []int{0, 0, 0, 0}
	ll.Values(rr) // Read state from a group of lines

	// Output
	l.SetValue(1) // Set line active
	l.SetValue(0) // Set line inactive

	ll.SetValues([]int{0, 1, 0, 1}) // Set a group of lines

	// Bias
	l, _ = c.RequestLine(4, gpiod.WithPullUp) // during request
	l.Reconfigure(gpiod.WithBiasDisabled)     // once requested

	// Debounce
	period := 10 * time.Millisecond
	l, _ = c.RequestLine(4, gpiod.WithDebounce(period)) // during request
	l.Reconfigure(gpiod.WithDebounce(period))           // once requested

	// Edge Watches
	l, _ = c.RequestLine(rpi.J8p7, gpiod.WithEventHandler(handler), gpiod.WithBothEdges)
	l.Reconfigure(gpiod.WithoutEdges)

	// Options
	ll, _ = c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput(0, 0, 1, 1),
		gpiod.WithLines([]int{0, 3}, gpiod.AsActiveLow),
		gpiod.AsOpenDrain)
	ll.Reconfigure(gpiod.WithLines([]int{0}, gpiod.AsActiveHigh))
	ll.Reconfigure(gpiod.WithLines([]int{3}, gpiod.Defaulted))

	// Line Requests (2)
	l.Close()
	ll.Close()

	// Chip Initialisation (2)
	c.Close()
}

func handler(evt gpiod.LineEvent) {
	// handle change in line state
}

func infoChangeHandler(evt gpiod.LineInfoChangeEvent) {
	// handle change in line info
}

func uapiV2() error {
	offset := 32
	offsets := []int{1, 2, 3}

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
		// initialise Offsets, DefaultValues and Consumer...
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
		Flags:    uapi.LineFlagV2Output,
		NumAttrs: 1,
		// initialise OutputValues...
	})

	return err
}

func uapiV1() error {
	offset := 32
	offsets := []int{1, 2, 3}
	value := 1

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

	return err
}
