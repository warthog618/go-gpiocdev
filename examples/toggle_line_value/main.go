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

	"github.com/warthog618/go-gpiocdev"
)

// This example drives line 22 on gpiochip0.
// The pin is toggled high and low at 1Hz with a 50% duty cycle.
// DO NOT run this on a device which has this pin externally driven.
func main() {
	offset := 22
	chip := "gpiochip0"
	v := 0
	l, err := gpiocdev.RequestLine(chip, offset, gpiocdev.AsOutput(v))
	if err != nil {
		panic(err)
	}
	// revert line to input on the way out.
	defer func() {
		l.Reconfigure(gpiocdev.AsInput)
		fmt.Printf("Input pin %s:%d\n", chip, offset)
		l.Close()
	}()
	values := map[int]string{0: "inactive", 1: "active"}
	fmt.Printf("Set pin %s:%d %s\n", chip, offset, values[v])

	// capture exit signals to ensure pin is reverted to input on exit.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	for {
		select {
		case <-time.After(500 * time.Millisecond):
			v ^= 1
			l.SetValue(v)
			fmt.Printf("Set pin %s:%d %s\n", chip, offset, values[v])
		case <-quit:
			return
		}
	}
}
