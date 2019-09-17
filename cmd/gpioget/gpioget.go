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

	"github.com/warthog618/config"
	"github.com/warthog618/config/dict"
	"github.com/warthog618/config/keys"
	"github.com/warthog618/config/pflag"
	"github.com/warthog618/gpiod"
)

var version = "undefined"

func main() {
	shortFlags := map[byte]string{
		'h': "help",
		'v': "version",
		'l': "active-low",
		'a': "as-is",
	}
	defaults := dict.New(dict.WithMap(
		map[string]interface{}{
			"active-low": false,
			"as-is":      false,
		}))
	flags := pflag.New(pflag.WithShortFlags(shortFlags),
		pflag.WithKeyReplacer(keys.NullReplacer()))
	cfg := config.New(flags, config.WithDefault(defaults))
	if v, err := cfg.Get("help"); err == nil && v.Bool() {
		printHelp()
		os.Exit(0)
	}
	if v, err := cfg.Get("version"); err == nil && v.Bool() {
		printVersion()
		os.Exit(0)
	}
	switch flags.NArg() {
	case 0:
		die("gpiochip must be specified")
	case 1:
		die("at least one GPIO line offset must be specified")
	}

	path := flags.Args()[0]
	c, err := gpiod.NewChip(path, gpiod.WithConsumer("gpioget"))
	if err != nil {
		die(err.Error())
	}
	defer c.Close()
	ll := []int(nil)
	for _, o := range flags.Args()[1:] {
		v, err := strconv.ParseUint(o, 10, 64)
		if err != nil {
			die(fmt.Sprintf("can't parse offset '%s'", o))
		}
		ll = append(ll, int(v))
	}
	opts := []gpiod.LineOption{}
	if cfg.MustGet("active-low").Bool() {
		opts = append(opts, gpiod.AsActiveLow())
	}
	if !cfg.MustGet("as-is").Bool() {
		opts = append(opts, gpiod.AsInput())
	}
	l, err := c.RequestLines(ll, opts...)
	if err != nil {
		die("error requesting GPIO lines:" + err.Error())
	}
	defer l.Close()
	vv, err := l.Values()
	if err != nil {
		die("error reading GPIO values:" + err.Error())
	}
	vstr := fmt.Sprintf("%d", vv[0])
	for _, v := range vv[1:] {
		vstr += fmt.Sprintf(" %d", v)
	}
	fmt.Println(vstr)
}

func die(reason string) {
	fmt.Fprintln(os.Stderr, "gpioget: "+reason)
	os.Exit(1)
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS] <gpiochip> <offset 1> <offset 2> ...\n", os.Args[0])
	fmt.Println("Read line value(s) from a GPIO chip.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help:\t\tdisplay this message and exit")
	fmt.Println("  -v, --version:\tdisplay the version and exit")
	fmt.Println("  -l, --active-low:\tset the line active state to low")
	fmt.Println("  -a, --as-is:\t\trequest the line as-is rather than as an input")
}

func printVersion() {
	fmt.Printf("%s (gpiod) %s\n", os.Args[0], version)
}
