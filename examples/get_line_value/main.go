// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A simple example that reads an input pin.
package main

import (
	"fmt"

	"github.com/warthog618/go-gpiocdev"
)

// This example reads line 22 on gpiochip0.
func main() {
	offset := 22
	chip := "gpiochip0"
	l, err := gpiocdev.RequestLine(chip, offset, gpiocdev.AsInput)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	values := map[int]string{0: "inactive", 1: "active"}
	v, err := l.Value()
	fmt.Printf("%s:%d %s\n", chip, offset, values[v])
}
