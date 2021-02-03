// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// A simple example that watches an input pin and reports edge events.
package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
)

func eventHandler(evt gpiod.LineEvent) {
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
}

// Watches GPIO 4 (Raspberry Pi J8-7) and reports when it changes state.
func main() {
	c, err := gpiod.NewChip("gpiochip0")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	offset := rpi.J8p7
	l, err := c.RequestLine(offset,
		gpiod.WithPullUp,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(eventHandler))
	if err != nil {
		fmt.Printf("RequestLine returned error: %s\n", err)
		if err == syscall.Errno(22) {
			fmt.Println("Note that the WithPullUp option requires kernel V5.5 or later - check your kernel version.")
		}
		os.Exit(1)
	}
	defer l.Close()

	// In a real application the main thread would do something useful.
	// But we'll just run for a minute then exit.
	fmt.Printf("Watching Pin %d...\n", offset)
	time.Sleep(time.Minute)
	fmt.Println("exiting...")
}
