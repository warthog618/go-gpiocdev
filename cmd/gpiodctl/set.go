// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	setCmd.Flags().BoolVarP(&setOpts.ActiveLow, "active-low", "l", false, "treat the line state as active low")
	setCmd.Flags().BoolVarP(&setOpts.OpenDrain, "open-drain", "d", false, "set the output to open-drain")
	setCmd.Flags().BoolVarP(&setOpts.OpenSource, "open-source", "s", false, "set the output to open-source")
	setCmd.Flags().BoolVar(&setOpts.PullUp, "pull-up", false, "enable internal pull-up")
	setCmd.Flags().BoolVar(&setOpts.PullDown, "pull-down", false, "enable internal pull-down")
	setCmd.Flags().BoolVar(&setOpts.BiasDisable, "bias-disable", false, "disable internal bias")
	setCmd.Flags().BoolVarP(&setOpts.User, "user", "u", false, "wait for the user to press Enter then exit")
	setCmd.Flags().BoolVarP(&setOpts.Wait, "wait", "w", false, "wait for a SIGINT or SIGTERM to exit")
	setCmd.Flags().StringVarP(&setOpts.Time, "time", "t", "", "wait for a period of time then exit.")
	setCmd.SetHelpTemplate(setCmd.HelpTemplate() + extendedSetHelp)
	rootCmd.AddCommand(setCmd)
}

var extendedSetHelp = `
Times:
  A time is a sequence of decimal numbers, each with optional fraction
  and a mandatory unit suffix, such as "300ms", "1.5h" or "2h45m".

  Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

Note:
  On exit the line reverts to its default state.
`

var (
	setCmd = &cobra.Command{
		Use:     "set <chip> <offset1>=<state1>...",
		Short:   "Set the state of a line or lines",
		Long:    `Set the state of lines on a GPIO chip and maintain the state until exit.`,
		Args:    cobra.MinimumNArgs(2),
		PreRunE: preset,
		RunE:    set,
	}
	setOpts = struct {
		ActiveLow   bool
		OpenDrain   bool
		OpenSource  bool
		Wait        bool
		User        bool
		Time        string
		PullUp      bool
		PullDown    bool
		BiasDisable bool
	}{}
)

func preset(cmd *cobra.Command, args []string) error {
	if len(setOpts.Time) != 0 {
		d, err := time.ParseDuration(setOpts.Time)
		if err != nil {
			return err
		}
		if d < 0 {
			return fmt.Errorf("time (%s) must be positive", setOpts.Time)
		}
	}
	if setOpts.OpenDrain && setOpts.OpenSource {
		return errors.New("can't select both open-drain and open-source")
	}
	return nil
}

func set(cmd *cobra.Command, args []string) error {
	name := args[0]
	ll := []int(nil)
	vv := []int(nil)
	for _, arg := range args[1:] {
		o, v, err := parseLineValue(arg)
		if err != nil {
			return err
		}
		ll = append(ll, o)
		vv = append(vv, v)
	}
	c, err := gpiod.NewChip(name, gpiod.WithConsumer("gpiodctl-set"))
	if err != nil {
		return err
	}
	defer c.Close()
	opts := makeSetOpts(vv)
	l, err := c.RequestLines(ll, opts...)
	if err != nil {
		return fmt.Errorf("error requesting GPIO line: %s", err)
	}
	defer l.Close()
	setWait()
	return nil
}

func setWait() {
	done := make(chan int)
	if len(setOpts.Time) > 0 {
		duration, _ := time.ParseDuration(setOpts.Time)
		fmt.Printf("waiting for %s...\n", duration)
		go func() {
			time.Sleep(duration)
			done <- 1
		}()
	}
	if setOpts.Wait {
		sigdone := make(chan os.Signal, 1)
		signal.Notify(sigdone, os.Interrupt, os.Kill)
		defer signal.Stop(sigdone)
		fmt.Println("waiting for signal...")
		go func() {
			<-sigdone
			done <- 2
		}()
	}
	if setOpts.User {
		fmt.Println("Press enter to exit...")
		go func() {
			reader := bufio.NewReader(os.Stdin)
			reader.ReadLine()
			done <- 3
		}()
	}
	<-done
}

func makeSetOpts(vv []int) []gpiod.LineOption {
	opts := []gpiod.LineOption{gpiod.AsOutput(vv...)}
	if setOpts.ActiveLow {
		opts = append(opts, gpiod.AsActiveLow)
	}
	if setOpts.OpenDrain {
		opts = append(opts, gpiod.AsOpenDrain)
	}
	if setOpts.OpenSource {
		opts = append(opts, gpiod.AsOpenSource)
	}
	switch {
	case setOpts.BiasDisable:
		opts = append(opts, gpiod.WithBiasDisable)
	case setOpts.PullUp:
		opts = append(opts, gpiod.WithPullUp)
	case setOpts.PullDown:
		opts = append(opts, gpiod.WithPullDown)
	}
	return opts
}

func parseLineValue(arg string) (int, int, error) {
	aa := strings.Split(arg, "=")
	if len(aa) != 2 {
		return 0, 0, fmt.Errorf("invalid offset<->state mapping: %s", arg)
	}
	o, err := strconv.ParseUint(aa[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("can't parse offset '%s'", arg)
	}
	v, err := strconv.ParseInt(aa[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("can't parse state '%s'", arg)
	}
	return int(o), int(v), nil
}
