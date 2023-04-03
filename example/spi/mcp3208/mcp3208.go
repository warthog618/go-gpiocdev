// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// An example of reading values from a MCP3208 using the bit-bashed SPI driver.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
	"github.com/warthog618/gpiod/spi/mcp3w0c"
)

// This example reads both channels from an MCP3208 connected to the RPI by four
// data lines - CSZ, CLK, DI, and DO. The pin assignments are defined in cfg.
// All pins other than DO are outputs so do not run this example on a board
// where those pins serve other purposes.
func main() {
	cfg := struct {
		chip string
		clk  int
		csz  int
		do   int
		di   int
		tclk time.Duration
		tset time.Duration
	}{
		chip: "gpiochip0",
		csz:  rpi.J8p37,
		clk:  rpi.J8p36,
		do:   rpi.J8p40,
		di:   rpi.J8p38,
		tclk: time.Nanosecond * 500,
		tset: time.Nanosecond * 250,
	}
	c, err := gpiod.NewChip(cfg.chip, gpiod.WithConsumer("mcp3208"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp3208: %s\n", err)
		os.Exit(1)
	}
	adc, err := mcp3w0c.NewMCP3208(
		c,
		cfg.clk,
		cfg.csz,
		cfg.di,
		cfg.do,
		mcp3w0c.WithTclk(cfg.tclk),
		mcp3w0c.WithTset(cfg.tset),
	)
	c.Close()
	if err != nil {
		fmt.Printf("mcp3208: %s\n", err)
		os.Exit(1)
	}
	defer adc.Close()
	for ch := 0; ch < 8; ch++ {
		d, err := adc.Read(ch)
		if err != nil {
			fmt.Printf("error reading ch%d: %s\n", ch, err)
			continue
		}
		fmt.Printf("ch%d=0x%04x\n", ch, d)
	}
}
