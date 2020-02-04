// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
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

	l, err := c.RequestLine(rpi.J8p7,
		gpiod.WithPullUp,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			fmt.Printf("Pin 4 event: %v\n", evt)
		}))
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// capture exit signals to ensure resources are released on exit.
	// !!! not required for gpiod???
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	// In a real application the main thread would do something useful.
	// But we'll just run for a minute then exit.
	fmt.Println("Watching Pin 4...")
	select {
	case <-time.After(time.Minute):
	case <-quit:
	}
}
