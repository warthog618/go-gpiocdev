// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A clone of libgpiod gpiodetect.
package main

import (
	"fmt"
	"os"

	"github.com/warthog618/config"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/gpiod"
)

var version = "undefined"

func main() {
	loadConfig()
	rc := 0
	cc := gpiod.Chips()
	for _, path := range cc {
		c, err := gpiod.NewChip(path)
		if err != nil {
			logErr(err)
			rc = 1
			continue
		}
		fmt.Printf("%s [%s] (%d lines)\n", c.Name, c.Label, c.Lines())
		c.Close()
	}
	os.Exit(rc)
}

func loadConfig() {
	ff := []pflag.Flag{
		{Short: 'h', Name: "help", Options: pflag.IsBool},
		{Short: 'v', Name: "version", Options: pflag.IsBool},
	}
	cfg := config.New(pflag.New(pflag.WithFlags(ff)))
	if v, _ := cfg.Get("help"); v.Bool() {
		printHelp()
		os.Exit(0)
	}
	if v, _ := cfg.Get("version"); v.Bool() {
		printVersion()
		os.Exit(0)
	}
}

func logErr(err error) {
	fmt.Fprintln(os.Stderr, "gpiodetect:", err)
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS]\n", os.Args[0])
	fmt.Println("List all GPIO chips, print their labels and number of GPIO lines.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help:\t\tdisplay this message and exit")
	fmt.Println("  -v, --version:\tdisplay the version and exit")
}

func printVersion() {
	fmt.Printf("%s (gpiod) %s\n", os.Args[0], version)
}
