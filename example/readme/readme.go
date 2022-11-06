// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A collection of code snippets contained in the READMEs.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/device/rpi"
	"github.com/warthog618/go-gpiocdev/uapi"
	"golang.org/x/sys/unix"
)

func main() {
	// Chip Initialisation
	c, _ := gpiocdev.NewChip("gpiochip0", gpiocdev.WithConsumer("myapp"))

	// Quick Start
	in, _ := gpiocdev.RequestLine("gpiochip0", 2, gpiocdev.AsInput)
	val, _ := in.Value()
	out, _ := gpiocdev.RequestLine("gpiochip0", 3, gpiocdev.AsOutput(val))
	in.Close()
	out.Close()

	// Line Requests
	l, _ := gpiocdev.RequestLine("gpiochip0", 4)
	l.Close()
	l, _ = c.RequestLine(4)
	l.Close()
	l, _ = c.RequestLine(rpi.J8p7) // Using Raspberry Pi J8 mapping.
	l.Close()
	l, _ = c.RequestLine(4, gpiocdev.AsOutput(1))

	ll, _ := gpiocdev.RequestLines("gpiochip0", []int{0, 1, 2, 3}, gpiocdev.AsOutput(0, 0, 1, 1))
	ll.Close()

	ll, _ = c.RequestLines([]int{0, 1, 2, 3}, gpiocdev.AsOutput(0, 0, 1, 1))

	// Line Info
	inf, _ := c.LineInfo(2)
	fmt.Printf("name: %s\n", inf.Name) // ineffassign bypass
	inf, _ = c.LineInfo(rpi.J8p7)
	fmt.Printf("name: %s\n", inf.Name) // ineffassign bypass
	inf, _ = l.Info()
	infs, _ := ll.Info()
	fmt.Printf("name: %s\n", inf.Name)
	fmt.Printf("name: %s\n", infs[0].Name)

	// Info Watch
	inf, _ = c.WatchLineInfo(4, infoChangeHandler)
	c.UnwatchLineInfo(4)

	// Active Level
	l, _ = c.RequestLine(4, gpiocdev.AsActiveLow) // during request
	l.Reconfigure(gpiocdev.AsActiveHigh)          // once requested

	// Direction
	l.Reconfigure(gpiocdev.AsInput)        // Set direction to Input
	l.Reconfigure(gpiocdev.AsOutput(1, 0)) // Set direction to Output (and values to active and inactive)

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
	l, _ = c.RequestLine(4, gpiocdev.WithPullUp) // during request
	l.Reconfigure(gpiocdev.WithBiasDisabled)     // once requested

	// Debounce
	period := 10 * time.Millisecond
	l, _ = c.RequestLine(4, gpiocdev.WithDebounce(period)) // during request
	l.Reconfigure(gpiocdev.WithDebounce(period))           // once requested

	// Edge Watches
	l, _ = c.RequestLine(rpi.J8p7, gpiocdev.WithEventHandler(handler), gpiocdev.WithBothEdges)
	l.Reconfigure(gpiocdev.WithoutEdges)

	// Options
	ll, _ = c.RequestLines([]int{0, 1, 2, 3}, gpiocdev.AsOutput(0, 0, 1, 1),
		gpiocdev.WithLines([]int{0, 3}, gpiocdev.AsActiveLow),
		gpiocdev.AsOpenDrain)
	ll.Reconfigure(gpiocdev.WithLines([]int{0}, gpiocdev.AsActiveHigh))
	ll.Reconfigure(gpiocdev.WithLines([]int{3}, gpiocdev.Defaulted))

	// Line Requests (2)
	l.Close()
	ll.Close()

	// Chip Initialisation (2)
	c.Close()
}

func handler(evt gpiocdev.LineEvent) {
	// handle change in line state
}

func infoChangeHandler(evt gpiocdev.LineInfoChangeEvent) {
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
	if err != nil {
		return err
	}

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
	if err != nil {
		return err
	}

	// set values
	err = uapi.SetLineValuesV2(uintptr(lr.Fd), values)
	if err != nil {
		return err
	}

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
	if err != nil {
		return err
	}

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
	if err != nil {
		return err
	}

	// set values
	values[0] = uint8(value)
	err = uapi.SetLineValues(uintptr(hr.Fd), values)
	if err != nil {
		return err
	}

	// update line config - change to active low
	err = uapi.SetLineConfig(uintptr(hr.Fd), &uapi.HandleConfig{
		Flags: uapi.HandleRequestInput,
	})

	return err
}
