// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"fmt"

	"github.com/warthog618/gpiod"
)

func main() {
	// get a chip and set the default label to apply to requested lines
	c, _ := gpiod.NewChip("gpiochip0", gpiod.WithConsumer("gpiodetect"))

	// get info about a line
	inf, _ := c.LineInfo(2)
	fmt.Println("name", inf.Name)

	// request a line as is, so not altering direction settings.
	l, _ := c.RequestLine(2)

	// request a line as input, so altering direction settings.
	l, _ = c.RequestLine(2, gpiod.AsInput)

	// request a line as output - initially set active
	l, _ = c.RequestLine(3, gpiod.AsOutput(1))
	//
	l.SetValue(0) // then set inactive

	// Get a line value
	v, _ := l.Value()
	fmt.Println("value:", v)

	// request a line with edge detection
	handler := func(gpiod.LineEvent) {
		// handle the edge event here
	}
	l, _ = c.RequestLine(4, gpiod.WithRisingEdge(handler))

	// request a bunch of lines
	ll, _ := c.RequestLines([]int{0, 1, 2, 3}, gpiod.AsOutput(0, 0, 1, 1))
	// ...
	ll.SetValues([]int{0, 1, 1, 0})
}
