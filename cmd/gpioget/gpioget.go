// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// A clone of libgpiod gpioget.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/warthog618/config"
	"github.com/warthog618/config/dict"
	"github.com/warthog618/config/keys"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/gpiod"
)

var version = "undefined"

func main() {
	cfg, flags := loadConfig()
	name := flags.Args()[0]
	c, err := gpiod.NewChip(name, gpiod.WithConsumer("gpioget"))
	if err != nil {
		die(err.Error())
	}
	defer c.Close()
	oo := parseOffsets(flags.Args()[1:])
	opts := makeOpts(cfg)
	l, err := c.RequestLines(oo, opts...)
	if err != nil {
		die("error requesting GPIO lines:" + err.Error())
	}
	defer l.Close()
	vv := make([]int, len(l.Offsets()))
	err = l.Values(vv)
	if err != nil {
		die("error reading GPIO values:" + err.Error())
	}
	vstr := fmt.Sprintf("%d", vv[0])
	for _, v := range vv[1:] {
		vstr += fmt.Sprintf(" %d", v)
	}
	fmt.Println(vstr)
}

func makeOpts(cfg *config.Config) []gpiod.LineOption {
	opts := []gpiod.LineOption{}
	if cfg.MustGet("active-low").Bool() {
		opts = append(opts, gpiod.AsActiveLow)
	}
	if !cfg.MustGet("as-is").Bool() {
		opts = append(opts, gpiod.AsInput)
	}
	bias := strings.ToLower(cfg.MustGet("bias").String())
	switch bias {
	case "pull-up":
		opts = append(opts, gpiod.WithPullUp)
	case "pull-down":
		opts = append(opts, gpiod.WithPullDown)
	case "disable":
		opts = append(opts, gpiod.WithBiasDisabled)
	case "as-is":
		fallthrough
	default:
	}
	return opts
}

func parseOffsets(args []string) []int {
	oo := []int(nil)
	for _, arg := range args {
		o := parseLineOffset(arg)
		oo = append(oo, o)
	}
	return oo
}

func parseLineOffset(arg string) int {
	o, err := strconv.ParseUint(arg, 10, 64)
	if err != nil {
		die(fmt.Sprintf("can't parse offset '%s'", arg))
	}
	return int(o)
}

func loadConfig() (*config.Config, *pflag.Getter) {
	ff := []pflag.Flag{
		{Short: 'h', Name: "help", Options: pflag.IsBool},
		{Short: 'v', Name: "version", Options: pflag.IsBool},
		{Short: 'l', Name: "active-low", Options: pflag.IsBool},
		{Short: 'a', Name: "as-is", Options: pflag.IsBool},
		{Short: 'b', Name: "bias"},
	}
	defaults := dict.New(dict.WithMap(
		map[string]interface{}{
			"help":       false,
			"version":    false,
			"active-low": false,
			"as-is":      false,
			"bias":       "as-is",
		}))
	flags := pflag.New(pflag.WithFlags(ff),
		pflag.WithKeyReplacer(keys.NullReplacer()),
	)
	cfg := config.New(flags, config.WithDefault(defaults))
	if cfg.MustGet("help").Bool() {
		printHelp()
		os.Exit(0)
	}
	if cfg.MustGet("version").Bool() {
		printVersion()
		os.Exit(0)
	}
	switch flags.NArg() {
	case 0:
		die("gpiochip must be specified")
	case 1:
		die("at least one GPIO line offset must be specified")
	}
	return cfg, flags
}

func die(reason string) {
	fmt.Fprintln(os.Stderr, "gpioget: "+reason)
	os.Exit(1)
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS] <gpiochip> <offset 1> <offset 2> ...\n", os.Args[0])
	fmt.Println("Read line value(s) from a GPIO chip.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help:\t\tdisplay this message and exit")
	fmt.Println("  -v, --version:\tdisplay the version and exit")
	fmt.Println("  -l, --active-low:\tset the line active state to low")
	fmt.Println("  -a, --as-is:\t\trequest the line as-is rather than as an input")
	fmt.Println("  -b, --bias=STRING:\tset the line bias")
	fmt.Println()
	fmt.Println("Biases:")
	fmt.Println("  as-is:\tleave bias unchanged (default)")
	fmt.Println("  disable:\tdisable bias")
	fmt.Println("  pull-up:\tenable pull-up")
	fmt.Println("  pull-down:\tenable pull-down")
}

func printVersion() {
	fmt.Printf("%s (gpiod) %s\n", os.Args[0], version)
}
