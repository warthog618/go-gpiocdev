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

// This example drives GPIO 4, which is pin J8-7 on a Raspberry Pi.
// The pin is toggled high and low at 1Hz with a 50% duty cycle.
// Do not run this on a device which has this pin externally driven.
func main() {
	c, err := gpiod.NewChip("gpiochip0")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	v := 0
	l, err := c.RequestLine(rpi.GPIO4, gpiod.AsOutput(v))
	if err != nil {
		panic(err)
	}
	defer func() {
		l.Reconfigure(gpiod.AsInput)
		l.Close()
	}()

	// capture exit signals to ensure pin is reverted to input on exit.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	for {
		select {
		case <-time.After(500 * time.Millisecond):
			v ^= 1
			l.SetValue(v)
			fmt.Println("Toggled", v)
		case <-quit:
			return
		}
	}
}
