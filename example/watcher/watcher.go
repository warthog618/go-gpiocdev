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
	edges := map[gpiod.LineEventType]string{1: "rising", 2: "falling"}
	l, err := c.RequestLine(offset,
		gpiod.WithPullUp,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			t := time.Time{}.Add(evt.Timestamp)
			fmt.Printf("Pin %d event: %s %s\n",
				evt.Offset,
				t.Format(time.StampMicro),
				edges[evt.Type])
		}))
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// In a real application the main thread would do something useful.
	// But we'll just run for a minute then exit.
	fmt.Printf("Watching Pin %d...\n", offset)
	time.Sleep(time.Minute)
	fmt.Println("exitting...")
}
