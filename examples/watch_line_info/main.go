// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

// A simple example that watches an input pin and reports edge events.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/warthog618/go-gpiocdev"
)

func eventHandler(evt gpiocdev.LineInfoChangeEvent) {
	t := time.Now()
	fmt.Printf("%s event: %#v\n", t.Format(time.RFC3339Nano), evt)
}

// Watches lines 21-23 on gpiochip0 and and reports when they change state.
func main() {
	offsets := []int{21, 22, 23}
	chip := "gpiochip0"
	c, err := gpiocdev.NewChip(chip)
	if err != nil {
		fmt.Printf("Opening chip returned error: %s\n", err)
		os.Exit(1)
	}
	defer c.Close()

	for _, o := range offsets {
		info, err := c.WatchLineInfo(o, eventHandler)
		if err != nil {
			fmt.Printf("Watching line %d returned error: %s\n", o, err)
			os.Exit(1)
		}
		fmt.Printf("Watching Pin %s:%d: %#v\n", chip, o, info)
	}
	// In a real application the main thread would do something useful.
	// But we'll just run for a minute then exit.
	time.Sleep(time.Minute)
	fmt.Println("watch_line_info exiting...")
}
