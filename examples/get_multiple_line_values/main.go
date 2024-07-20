// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

// A simple example that reads multiple input pins.
package main

import (
	"fmt"
	"os"

	"github.com/warthog618/go-gpiocdev"
)

// This example reads lines 21 and 22 on gpiochip0.
func main() {
	offsets := []int{21, 22}
	chip := "gpiochip0"
	l, err := gpiocdev.RequestLines(chip, offsets, gpiocdev.AsInput)
	if err != nil {
		fmt.Printf("Requesting lines returned error: %s\n", err)
		os.Exit(1)
	}
	defer l.Close()

	values := map[int]string{0: "inactive", 1: "active"}
	vv := []int{0, 0}
	err = l.Values(vv)
	if err != nil {
		fmt.Printf("Reading values returned error: %s\n", err)
		os.Exit(1)
	}
	for i, o := range offsets {
		fmt.Printf("%s:%d %s\n", chip, o, values[vv[i]])
	}
}
