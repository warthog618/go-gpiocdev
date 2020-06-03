// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// A clone of libgpiod gpiomon.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
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
	c, err := gpiod.NewChip(name, gpiod.WithConsumer("gpiomon"))
	if err != nil {
		die(err.Error())
	}
	defer c.Close()
	oo := parseOffsets(flags.Args()[1:])
	evtchan := make(chan gpiod.LineEvent)
	eh := func(evt gpiod.LineEvent) {
		evtchan <- evt
	}
	opts := makeOpts(cfg, eh)
	l, err := c.RequestLines(oo, opts...)
	if err != nil {
		die("error requesting GPIO lines:" + err.Error())
	}
	defer l.Close()
	wait(cfg, evtchan)
}

func wait(cfg *config.Config, evtchan <-chan gpiod.LineEvent) {
	sigdone := make(chan os.Signal, 1)
	signal.Notify(sigdone, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigdone)
	count := 0
	num := cfg.MustGet("num-events").Int()
	silent := cfg.MustGet("silent").Bool()
	for {
		select {
		case evt := <-evtchan:
			if !silent {
				t := time.Now()
				edge := "rising"
				if evt.Type == gpiod.LineEventFallingEdge {
					edge = "falling"
				}
				fmt.Printf("event:%3d %-7s %s (%s)\n",
					evt.Offset,
					edge,
					t.Format(time.RFC3339Nano),
					evt.Timestamp)
			}
			count++
			if num > 0 && count >= num {
				return
			}
		case <-sigdone:
			return
		}
	}
}

func makeOpts(cfg *config.Config, eh gpiod.EventHandler) []gpiod.LineOption {
	opts := []gpiod.LineOption{}
	if cfg.MustGet("active-low").Bool() {
		opts = append(opts, gpiod.AsActiveLow)
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
	edge := strings.ToLower(cfg.MustGet("edge").String())
	switch edge {
	case "falling":
		opts = append(opts, gpiod.WithFallingEdge(eh))
	case "rising":
		opts = append(opts, gpiod.WithRisingEdge(eh))
	case "both":
		fallthrough
	default:
		opts = append(opts, gpiod.WithBothEdges(eh))
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
		{Short: 'b', Name: "bias"},
		{Short: 'e', Name: "edge"},
		{Short: 's', Name: "silent", Options: pflag.IsBool},
		{Short: 'n', Name: "num-events"},
	}
	defaults := dict.New(dict.WithMap(
		map[string]interface{}{
			"help":       false,
			"version":    false,
			"active-low": false,
			"bias":       "as-is",
			"edge":       "both",
			"num-events": 0,
			"silent":     false,
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
	fmt.Fprintln(os.Stderr, "gpiomon: "+reason)
	os.Exit(1)
}

func printHelp() {
	fmt.Printf("Usage: %s [OPTIONS] <gpiochip> <offset 1> <offset 2>...\n", os.Args[0])
	fmt.Println("Wait for events on GPIO lines and print them to standard output.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help:\t\tdisplay this message and exit")
	fmt.Println("  -v, --version:\tdisplay the version and exit")
	fmt.Println("  -l, --active-low:\tset the line active state to low")
	fmt.Println("  -n, --num-events=NUM:\texit after processing NUM events")
	fmt.Println("  -s, --silent:\t\tdon't print event info")
	fmt.Println("  -b, --bias=STRING:\tset the line bias")
	fmt.Println("  -e, --edge=STRING:\tselect the edge detection")
	fmt.Println()
	fmt.Println("Biases:")
	fmt.Println("  as-is:\tleave bias unchanged (default)")
	fmt.Println("  disable:\tdisable bias")
	fmt.Println("  pull-up:\tenable pull-up")
	fmt.Println("  pull-down:\tenable pull-down")
	fmt.Println()
	fmt.Println("Edges:")
	fmt.Println("  both:\t\tboth rising and falling edge events are detected")
	fmt.Println("  \t\tand reported (default)")
	fmt.Println("  falling:\tonly falling edge events are detected and reported")
	fmt.Println("  rising:\tonly rising edge events are detected and reported")
}

func printVersion() {
	fmt.Printf("%s (gpiod) %s\n", os.Args[0], version)
}
