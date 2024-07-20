// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

// A simple example that toggles multiple output pins.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/warthog618/go-gpiocdev"
)

// This example drives lines 21 and 22 on gpiochip0.
// The pins are toggled high and low opposite each other at 1Hz with a 50% duty cycle.
// DO NOT run this on a device which has either of those pins externally driven.
func main() {
	offsets := []int{21, 22}
	chip := "gpiochip0"
	vv := []int{0, 1}
	l, err := gpiocdev.RequestLines(chip, offsets, gpiocdev.AsOutput(vv...))
	if err != nil {
		panic(err)
	}
	// revert lines to input on the way out.
	defer func() {
		l.Reconfigure(gpiocdev.AsInput)
		fmt.Printf("Input pins %s:%v\n", chip, offsets)
		l.Close()
	}()

	values := map[int]string{0: "inactive", 1: "active"}
	for i, o := range offsets {
		fmt.Printf("Set pin %s:%d %s\n", chip, o, values[vv[i]])
	}

	// capture exit signals to ensure pin is reverted to input on exit.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	for {
		select {
		case <-time.After(500 * time.Millisecond):
			for i := range vv {
				vv[i] ^= 1
			}
			l.SetValues(vv)
			for i, o := range offsets {
				fmt.Printf("Set pin %s:%d %s\n", chip, o, values[vv[i]])
			}
		case <-quit:
			return
		}
	}
}
