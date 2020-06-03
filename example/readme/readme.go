// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// A collection of code snippets contained in the README.
package main

import (
	"fmt"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
)

func main() {
	// Chip Initialisation
	c, _ := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("myapp"))

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

	// Active Level
	l, _ = c.RequestLine(4, gpiod.AsActiveLow) // during request
	l.Reconfigure(gpiod.AsActiveHigh)          // once requested

	// Direction
	l.Reconfigure(gpiod.AsInput)     // Set direction to Input
	l.Reconfigure(gpiod.AsOutput(0)) // Set direction to Output (and value to inactive)

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
	l.Reconfigure(gpiod.WithBiasDisabled)      // once requested

	// Watches
	l, _ = c.RequestLine(rpi.J8p7, gpiod.WithBothEdges(handler))

	// Line Requests (2)
	l.Close()
	ll.Close()

	// Chip Initialisation (2)
	c.Close()
}

func find() *gpiod.Line {
	chipname, offset, _ := gpiod.FindLine("LED A")
	c, _ := gpiod.NewChip(chipname, gpiod.WithConsumer("myapp"))
	l, _ := c.RequestLine(offset)
	return l

}

func handler(evt gpiod.LineEvent) {
	// handle change in line state
}
