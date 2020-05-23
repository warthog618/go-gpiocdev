// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

package main

import (
	"fmt"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
)

// Watches GPIO 4 (Raspberry Pi J8-7) and reports when it changes state.
func main() {
	c, err := gpiod.NewChip("gpiochip0")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	offset := rpi.J8p7
	l, err := c.RequestLine(offset,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			t := time.Now()
			edge := "rising"
			if evt.Type == gpiod.LineEventFallingEdge {
				edge = "falling"
			}
			fmt.Printf("event:%3d %-7s %s (%s)\n",
				evt.Offset,
				edge,
				t.Format(time.RFC3339Nano),
				evt.Timestamp)
		}))
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// In a real application the main thread would do something useful.
	// But we'll just run for a minute then exit.
	fmt.Printf("Watching Pin %d...\n", offset)
	time.Sleep(time.Minute)
	fmt.Println("exiting...")
}
