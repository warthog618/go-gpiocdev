// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

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

}

// Watches GPIO 4 (Raspberry Pi J8-7) and reports when it changes state.
func main() {
	c, err := gpiod.NewChip("gpiochip0")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	offset := rpi.GPIO19
	ch := make(chan gpiod.LineEvent)
	l, err := c.RequestLine(offset,
		gpiod.WithPullUp,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(ch))
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

	t := time.NewTicker(time.Minute)
Loop:
	for {
		select {
		case evt := <-ch:
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
		case <-t.C:
			break Loop
		}

	}
	fmt.Println("exiting...")
}
