// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package main

import (
	"fmt"
	"os"

	"github.com/warthog618/config"
	"github.com/warthog618/config/blob"
	"github.com/warthog618/config/blob/decoder/json"
	"github.com/warthog618/config/dict"
	"github.com/warthog618/config/env"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
	"github.com/warthog618/gpiod/spi/adc0832"
)

// This example reads both channels from an ADC0832 connected to the RPI by four
// data lines - CSZ, CLK, DI, and DO. The default pin assignments are defined in
// loadConfig, but can be altered via configuration (env, flag or config file).
// All pins other than DO are outputs so do not run this example on a board
// where those pins serve other purposes.
func main() {
	cfg := loadConfig()
	tclk := cfg.MustGet("tclk").Duration()
	tset := cfg.MustGet("tset").Duration()
	if tset < tclk {
		tset = tclk
	}
	chip := cfg.MustGet("gpiochip").String()
	c, err := gpiod.NewChip(chip, gpiod.WithConsumer("adc0832"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "adc0832: %s\n", err)
		os.Exit(1)
	}
	a, err := adc0832.New(
		c,
		tclk,
		tset,
		cfg.MustGet("clk").Int(),
		cfg.MustGet("csz").Int(),
		cfg.MustGet("di").Int(),
		cfg.MustGet("do").Int())
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

func loadConfig() *config.Config {
	defaultConfig := map[string]interface{}{
		"gpiochip": "gpiochip0",
		"tclk":     "2500ns",
		"tset":     "2500ns", // should be at least tclk - enforced in main
		"csz":      rpi.J8p29,
		"clk":      rpi.J8p31,
		"do":       rpi.J8p33,
		"di":       rpi.J8p35,
	}
	def := dict.New(dict.WithMap(defaultConfig))
	flags := []pflag.Flag{
		{Short: 'c', Name: "config-file"},
	}
	cfg := config.New(
		pflag.New(pflag.WithFlags(flags)),
		env.New(env.WithEnvPrefix("ADC0832_")),
		config.WithDefault(def))
	cfg.Append(
		blob.NewConfigFile(cfg, "config.file", "adc0832.json", json.NewDecoder()))
	cfg = cfg.GetConfig("", config.WithMust())
	return cfg
}
