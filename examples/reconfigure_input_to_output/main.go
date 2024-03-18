// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A simple example that requests a line as an input and subsequently switches it to an output.
// DO NOT run this on a platform where that line is externally driven.
package main

import (
	"fmt"
	"time"

	"github.com/warthog618/go-gpiocdev"
)

// This example requests line 23 on gpiochip0 as an input then switches it to an output.
// DO NOT run this on a platform where that line is externally driven.
func main() {
	offset := 23
	chip := "gpiochip0"
	l, err := gpiocdev.RequestLine(chip, offset, gpiocdev.AsInput)
	if err != nil {
		panic(err)
	}
	// revert line to input on the way out.
	defer func() {
		l.Reconfigure(gpiocdev.AsInput)
		fmt.Printf("Input pin: %s:%d\n", chip, offset)
		l.Close()
	}()

	values := map[int]string{0: "inactive", 1: "active"}
	v, err := l.Value()
	fmt.Printf("Read pin: %s:%d %s\n", chip, offset, values[v])

	l.Reconfigure(gpiocdev.AsOutput(v))
	fmt.Printf("Set pin %s:%d %s\n", chip, offset, values[v])
	time.Sleep(500 * time.Millisecond)
	v ^= 1
	l.SetValue(v)
	fmt.Printf("Set pin %s:%d %s\n", chip, offset, values[v])
}
