// SPDX-FileCopyrightText: 2024 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A simple example that finds a line by name.
package main

import (
	"fmt"
	"os"

	"github.com/warthog618/go-gpiocdev"
)

// Finds the chip and offset of a named line.
func main() {
	name := "GPIO22"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	chip, offset, err := gpiocdev.FindLine(name)
	if err != nil {
		fmt.Printf("Finding line %s returned error: %s\n", name, err)
		os.Exit(1)
	}
	fmt.Printf("%s:%d %s\n", chip, offset, name)
}
