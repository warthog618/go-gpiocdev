// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// A clone of libgpiod gpioinfo.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/warthog618/config"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/uapi"
)

var version = "undefined"

func main() {
	flags := loadConfig()
	rc := 0
	cc := append([]string(nil), flags.Args()...)
	if len(cc) == 0 {
		cc = gpiod.Chips()
	}
	for _, path := range cc {
		c, err := gpiod.NewChip(path)
		if err != nil {
			logErr(err)
			rc = 1
			continue
		}
		fmt.Printf("%s - %d lines:\n", c.Name, c.Lines())
		for o := 0; o < c.Lines(); o++ {
			li, err := c.LineInfo(o)
			if err != nil {
				logErr(err)
				rc = 1
				continue
			}
			printLineInfo(li)
		}
		c.Close()
	}
	os.Exit(rc)
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
	return flags
}

func logErr(err error) {
	fmt.Fprintln(os.Stderr, "gpioinfo:", err)
}

func printLineInfo(li gpiod.LineInfo) {
	if len(li.Name) == 0 {
		li.Name = "unnamed"
	}
	if li.Config.Flags.IsUsed() {
		if len(li.Consumer) == 0 {
			li.Consumer = "kernel"
		}
		if strings.Contains(li.Consumer, " ") {
			li.Consumer = "\"" + li.Consumer + "\""
		}
	} else {
		li.Consumer = "unused"
	}
	dirn := "input"
	if li.Config.Direction == uapi.LineDirectionOutput {
		dirn = "output"
	}
	active := "active-high"
	if li.Config.Flags.IsActiveLow() {
		active = "active-low"
	}
	flags := []string(nil)
	if li.Config.Flags.IsUsed() {
		flags = append(flags, "used")
	}
	if li.Config.Drive == uapi.LineDriveOpenDrain {
		flags = append(flags, "open-drain")
	}
	if li.Config.Drive == uapi.LineDriveOpenSource {
		flags = append(flags, "open-source")
	}
	if li.Config.Bias == uapi.LineBiasPullUp {
		flags = append(flags, "pull-up")
	}
	if li.Config.Bias == uapi.LineBiasPullDown {
		flags = append(flags, "pull-down")
	}
	if li.Config.Flags.HasBias() &&
		li.Config.Bias == uapi.LineBiasDisabled {
		flags = append(flags, "bias-disabled")
	}
	flstr := ""
	if len(flags) > 0 {
		flstr = "[" + strings.Join(flags, " ") + "]"
	}
	fmt.Printf("\tline %3d:%12s%12s%8s%13s%s\n",
		li.Offset, li.Name, li.Consumer, dirn, active, flstr)
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS] <gpiochip1> ...\n", os.Args[0])
	fmt.Println("Print information about all lines of the specified GPIO chip(s) (or all gpiochips if none are specified).")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help:\t\tdisplay this message and exit")
	fmt.Println("  -v, --version:\tdisplay the version and exit")
}

func printVersion() {
	fmt.Printf("%s (gpiod) %s\n", os.Args[0], version)
}
