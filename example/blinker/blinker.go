// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A simple example that toggles an output pin.
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

// This example drives GPIO 22, which is pin J8-15 on a Raspberry Pi.
// The pin is toggled high and low at 1Hz with a 50% duty cycle.
// Do not run this on a device which has this pin externally driven.
func main() {
	offset := rpi.J8p15
	v := 0
	l, err := gpiod.RequestLine("gpiochip0", offset, gpiod.AsOutput(v))
	if err != nil {
		panic(err)
	}
	// revert line to input on the way out.
	defer func() {
		l.Reconfigure(gpiod.AsInput)
		l.Close()
	}()
	values := map[int]string{0: "inactive", 1: "active"}
	fmt.Printf("Set pin %d %s\n", offset, values[v])

	// capture exit signals to ensure pin is reverted to input on exit.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	for {
		select {
		case <-time.After(2 * time.Second):
			v ^= 1
			l.SetValue(v)
			fmt.Printf("Set pin %d %s\n", offset, values[v])
		case <-quit:
			return
		}
	}
}
