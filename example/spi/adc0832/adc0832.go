// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// An example of reading values from an ADC0832 using the bit-bashed SPI driver.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
	"github.com/warthog618/gpiod/spi/adc0832"
)

// This example reads both channels from an ADC0832 connected to the RPI by four
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
		csz:  rpi.J8p29,
		clk:  rpi.J8p31,
		do:   rpi.J8p33,
		di:   rpi.J8p35,
		tclk: time.Nanosecond * 2500,
		tset: 0,
	}
	c, err := gpiod.NewChip(cfg.chip, gpiod.WithConsumer("adc0832"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "adc0832: %s\n", err)
		os.Exit(1)
	}
	a, err := adc0832.New(
		c,
		cfg.clk,
		cfg.csz,
		cfg.di,
		cfg.do,
		adc0832.WithTclk(cfg.tclk),
		adc0832.WithTset(cfg.tset),
	)
	c.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "adc0832: %s\n", err)
		os.Exit(1)
	}
	defer a.Close()
	ch0, err := a.Read(0)
	if err != nil {
		fmt.Printf("read error ch0: %s\n", err)
	}
	ch1, err := a.Read(1)
	if err != nil {
		fmt.Printf("read error ch1: %s\n", err)
	}
	fmt.Printf("ch0=0x%02x, ch1=0x%02x\n", ch0, ch1)
}
