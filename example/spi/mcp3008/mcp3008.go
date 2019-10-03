// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

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
	"github.com/warthog618/gpiod/spi/mcp3w0c"
)

// This example reads both channels from an MCP3008 connected to the RPI by four
// data lines - CSZ, CLK, DI, and DO. The default pin assignments are defined in
// loadConfig, but can be altered via configuration (env, flag or config file).
// All pins other than DO are outputs so do not run this example on a board
// where those pins serve other purposes.
func main() {
	cfg := loadConfig()
	tclk := cfg.MustGet("tclk").Duration()
	adc, err := mcp3w0c.NewMCP3008(
		tclk,
		cfg.MustGet("clk").Int(),
		cfg.MustGet("csz").Int(),
		cfg.MustGet("di").Int(),
		cfg.MustGet("do").Int())
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp3008: %s\n", err)
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

func loadConfig() *config.Config {
	defaultConfig := map[string]interface{}{
		"tclk": "500ns",
		"clk":  J8p36,
		"csz":  J8p37,
		"di":   J8p38,
		"do":   J8p40,
	}
	def := dict.New(dict.WithMap(defaultConfig))
	flags := []pflag.Flag{
		{Short: 'c', Name: "config-file"},
	}
	// highest priority sources first - flags override environment
	cfg := config.New(
		pflag.New(pflag.WithFlags(flags)),
		env.New(env.WithEnvPrefix("MCP3008_")),
		config.WithDefault(def))
	cfg.Append(
		blob.NewConfigFile(cfg, "config.file", "mcp3008.json", json.NewDecoder()))
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

// GPIO aliases to J8 pins
const (
	GPIO2  = J8p3
	GPIO3  = J8p5
	GPIO4  = J8p7
	GPIO5  = J8p29
	GPIO6  = J8p31
	GPIO7  = J8p26
	GPIO8  = J8p24
	GPIO9  = J8p21
	GPIO10 = J8p19
	GPIO11 = J8p23
	GPIO12 = J8p32
	GPIO13 = J8p33
	GPIO14 = J8p8
	GPIO15 = J8p10
	GPIO16 = J8p36
	GPIO17 = J8p11
	GPIO18 = J8p12
	GPIO19 = J8p35
	GPIO20 = J8p38
	GPIO21 = J8p40
	GPIO22 = J8p15
	GPIO23 = J8p16
	GPIO24 = J8p18
	GPIO25 = J8p22
	GPIO26 = J8p37
	GPIO27 = J8p13
)
