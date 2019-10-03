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
	a, err := adc0832.New(
		tclk,
		tset,
		cfg.MustGet("clk").Int(),
		cfg.MustGet("csz").Int(),
		cfg.MustGet("di").Int(),
		cfg.MustGet("do").Int())
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
		"tclk": "2500ns",
		"tset": "2500ns", // should be at least tclk - enforced in main
		"csz":  J8p29,
		"clk":  J8p31,
		"do":   J8p33,
		"di":   J8p35,
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

// Convenience mapping from Raspberry Pi J8 pinouts to BCM pinouts.
const (
	J8p27 = iota
	J8p28
	J8p3
	J8p5
	J8p7
	J8p29
	J8p31
	J8p26
	J8p24
	J8p21
	J8p19
	J8p23
	J8p32
	J8p33
	J8p8
	J8p10
	J8p36
	J8p11
	J8p12
	J8p35
	J8p38
	J8p40
	J8p15
	J8p16
	J8p18
	J8p22
	J8p37
	J8p13
)
