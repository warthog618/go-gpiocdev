// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// +build linux

// A clone of libgpiod gpiofind.
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
	flags := loadConfig()
	linename := flags.Args()[0]
	ec := 1
	for _, cname := range gpiod.Chips() {
		c, err := gpiod.NewChip(cname)
		if err != nil {
			continue
		}
		for o := 0; o < c.Lines(); o++ {
			inf, err := c.LineInfo(o)
			if err != nil {
				continue
			}
			if inf.Name == linename {
				fmt.Printf("%s %d\n", cname, o)
				ec = 0
			}
		}
		c.Close()
	}
	os.Exit(ec)
}

func loadConfig() *pflag.Getter {
	ff := []pflag.Flag{
		{Short: 'h', Name: "help", Options: pflag.IsBool},
		{Short: 'v', Name: "version", Options: pflag.IsBool},
	}
	flags := pflag.New(pflag.WithFlags(ff))
	cfg := config.New(flags)
	if v, _ := cfg.Get("help"); v.Bool() {
		printHelp()
		os.Exit(0)
	}
	if v, _ := cfg.Get("version"); v.Bool() {
		printVersion()
		os.Exit(0)
	}
	if flags.NArg() != 1 {
		die("exactly one GPIO line name must be specified")
	}
	return flags
}

func die(reason string) {
	fmt.Fprintln(os.Stderr, "gpiofind: "+reason)
	os.Exit(1)
}

func logErr(err error) {
	fmt.Fprintln(os.Stderr, "gpiofind:", err)
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS] <name>\n", os.Args[0])
	fmt.Println("Find a GPIO line by name. The output of this command can be used as input for gpioget/set.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help:\t\tdisplay this message and exit")
	fmt.Println("  -v, --version:\tdisplay the version and exit")
}

func printVersion() {
	fmt.Printf("%s (gpiod) %s\n", os.Args[0], version)
}
