// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// A clone of libgpiod gpioset.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

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
	c, err := gpiod.NewChip(name, gpiod.WithConsumer("gpioset"))
	if err != nil {
		die(err.Error())
	}
	defer c.Close()
	ll := []int(nil)
	vv := []int(nil)
	for _, arg := range flags.Args()[1:] {
		o, v := parseLineValue(arg)
		ll = append(ll, o)
		vv = append(vv, v)
	}
	opts := makeOpts(cfg)
	l, err := c.RequestLines(ll, opts...)
	if err != nil {
		die("error requesting GPIO lines:" + err.Error())
	}
	defer l.Close()
	err = l.SetValues(vv)
	if err != nil {
		die("error setting GPIO values:" + err.Error())
	}
	wait(cfg)
	os.Exit(0)
}

func wait(cfg *config.Config) {
	mode := cfg.MustGet("mode").String()
	switch mode {
	case "wait":
		fmt.Println("Press enter to exit...")
		reader := bufio.NewReader(os.Stdin)
		reader.ReadLine()
	case "time":
		sec := cfg.MustGet("sec").Int()
		usec := cfg.MustGet("usec").Int()
		duration := time.Duration(sec)*time.Second + time.Duration(usec)*time.Microsecond
		fmt.Printf("Waiting for %s...\n", duration)
		time.Sleep(duration)
	case "signal":
		sigdone := make(chan os.Signal, 1)
		signal.Notify(sigdone, os.Interrupt, os.Kill)
		defer signal.Stop(sigdone)
		fmt.Println("Waiting for signal...")
		<-sigdone
	}
}

func makeOpts(cfg *config.Config) []gpiod.LineOption {
	opts := []gpiod.LineOption{gpiod.AsOutput()}
	if cfg.MustGet("active-low").Bool() {
		opts = append(opts, gpiod.AsActiveLow())
	}
	if cfg.MustGet("open-drain").Bool() {
		opts = append(opts, gpiod.AsOpenDrain())
	}
	if cfg.MustGet("open-source").Bool() {
		opts = append(opts, gpiod.AsOpenSource())
	}
	return opts
}

func parseLineValue(arg string) (int, int) {
	aa := strings.Split(arg, "=")
	if len(aa) != 2 {
		die(fmt.Sprintf("invalid offset<->value mapping: %s", arg))
	}
	o, err := strconv.ParseUint(aa[0], 10, 64)
	if err != nil {
		die(fmt.Sprintf("can't parse offset '%s'", arg))
	}
	v, err := strconv.ParseInt(aa[1], 10, 64)
	if err != nil {
		die(fmt.Sprintf("can't parse value '%s'", arg))
	}
	return int(o), int(v)
}

func loadConfig() (*config.Config, *pflag.Getter) {
	ff := []pflag.Flag{
		{Short: 'h', Name: "help", Options: pflag.IsBool},
		{Short: 'v', Name: "version", Options: pflag.IsBool},
		{Short: 'l', Name: "active-low", Options: pflag.IsBool},
		{Short: 'D', Name: "open-drain", Options: pflag.IsBool},
		{Short: 'S', Name: "open-source", Options: pflag.IsBool},
		{Short: 'm', Name: "mode"},
		{Short: 's', Name: "sec"},
		{Short: 'u', Name: "usec"},
	}
	defaults := dict.New(dict.WithMap(
		map[string]interface{}{
			"help":        false,
			"version":     false,
			"active-low":  false,
			"open-drain":  false,
			"open-source": false,
			"mode":        "exit",
			"sec":         0,
			"usec":        0,
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
	mode := cfg.MustGet("mode").String()
	switch mode {
	case "wait":
	case "time":
	case "signal":
	case "exit":
	default:
		die(fmt.Sprintf("invalid mode: %s", mode))
	}
	if cfg.MustGet("open-drain").Bool() && cfg.MustGet("open-source").Bool() {
		die("can't select both open-drain and open-source")
	}
	switch flags.NArg() {
	case 0:
		die("gpiochip must be specified")
	case 1:
		die("at least one GPIO line offset to value mapping must be specified")
	}
	return cfg, flags
}

func die(reason string) {
	fmt.Fprintln(os.Stderr, "gpioset: "+reason)
	os.Exit(1)
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS] <gpiochip> <offset 1>=<value1> <offset 2>=<value2> ...\n", os.Args[0])
	fmt.Println("Set GPIO line values of a GPIO chip and maintain the state until the process exits.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help:\t\tdisplay this message and exit")
	fmt.Println("  -v, --version:\tdisplay the version and exit")
	fmt.Println("  -l, --active-low:\tset the line active state to low")
	fmt.Println("  -D, --open-drain:\tset the line as open drain")
	fmt.Println("  -S, --open-source:\tset the line as open source")
	fmt.Println("  -m, --mode=[exit|wait|time|signal] (defaults to 'exit'):")
	fmt.Println("		\ttell the program what to do after setting values")
	fmt.Println("  -s, --sec=SEC:\tspecify the number of seconds to wait (only valid for --mode=time)")
	fmt.Println("  -u, --usec=USEC:\tspecify the number of microseconds to wait (only valid for --mode=time)")
	// don't support -b, --background to daemonise - there are other ways to
	// achieve that.
	fmt.Println("")
	fmt.Println("Modes:")
	fmt.Println("  exit:\t\tset values and exit immediately")
	fmt.Println("  wait:\t\tset values and wait for user to press ENTER")
	fmt.Println("  time:\t\tset values and waits for a specified amount of time")
	fmt.Println("  signal:\tset values and wait for SIGINT or SIGTERM")
	fmt.Println("")
	fmt.Println("Note: the state of a GPIO line controlled over the character device reverts to default")
	fmt.Println("when the last process referencing the file descriptor representing the device file exits.")
	fmt.Println("This means that it's wrong to run gpioset, have it exit and expect the line to continue")
	fmt.Println("being driven high or low. It may happen if given line is floating but it must be interpreted")
	fmt.Println("as undefined behavior.")
}

func printVersion() {
	fmt.Printf("%s (gpiod) %s\n", os.Args[0], version)
}
