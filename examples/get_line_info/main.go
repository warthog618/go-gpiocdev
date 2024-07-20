// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

// A simple example that read the info for line 22 on gpiochip0.
package main

import (
	"fmt"
	"os"

	"github.com/warthog618/go-gpiocdev"
)

// Reads the info for line 22 on gpiochip0.
func main() {
	c, err := gpiocdev.NewChip("gpiochip0")
	if err != nil {
		fmt.Printf("Opening chip returned error: %s\n", err)
		os.Exit(1)
	}
	defer c.Close()

	info, err := c.LineInfo(22)
	if err != nil {
		fmt.Printf("Reading line info returned error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("%v\n", info)
}
