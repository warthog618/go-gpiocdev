// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// +build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	setCmd.Flags().BoolVarP(&setOpts.ActiveLow, "active-low", "l", false, "treat the line state as active low")
	setCmd.Flags().StringVarP(&setOpts.Bias, "bias", "b", "as-is", "set the line bias.")
	setCmd.Flags().StringVarP(&setOpts.Drive, "drive", "d", "push-pull", "set the line drive.")
	setCmd.Flags().BoolVarP(&setOpts.User, "user", "u", false, "wait for the user to press Enter then exit")
	setCmd.Flags().BoolVarP(&setOpts.Wait, "wait", "w", false, "wait for a SIGINT or SIGTERM to exit")
	setCmd.Flags().StringVarP(&setOpts.Time, "time", "t", "", "wait for a period of time then exit.")
	setCmd.SetHelpTemplate(setCmd.HelpTemplate() + extendedSetHelp)
	rootCmd.AddCommand(setCmd)
}

var extendedSetHelp = `
Biases:
  as-is:        leave bias unchanged
  disable:      disable bias
  pull-up:      enable pull-up
  pull-down:    enable pull-down

Drives:
  push-pull:    drive the line both high and low
  open-drain:   drive the line low or go high impedance
  open-source:  drive the line high or go high impedance

Times:
  A time is a sequence of decimal numbers, each with optional fraction
  and a mandatory unit suffix, such as "300ms", "1.5h" or "2h45m".

  Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

Note:
  On exit the line reverts to its default state.
`

var (
	setCmd = &cobra.Command{
		Use:                   "set [flags] <chip> <offset1>=<state1>...",
		Short:                 "Set the state of a line or lines",
		Long:                  `Set the state of lines on a GPIO chip and maintain the state until exit.`,
		Args:                  cobra.MinimumNArgs(2),
		PreRunE:               preset,
		RunE:                  set,
		DisableFlagsInUseLine: true,
	}
	setOpts = struct {
		ActiveLow bool
		Bias      string
		Drive     string
		Wait      bool
		User      bool
		Time      string
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
		signal.Notify(sigdone, os.Interrupt, syscall.SIGTERM)
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

func makeSetOpts(vv []int) []gpiod.LineReqOption {
	opts := []gpiod.LineReqOption{gpiod.AsOutput(vv...)}
	if setOpts.ActiveLow {
		opts = append(opts, gpiod.AsActiveLow)
	}
	bias := strings.ToLower(setOpts.Bias)
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
	drive := strings.ToLower(setOpts.Drive)
	switch drive {
	case "open-drain":
		opts = append(opts, gpiod.AsOpenDrain)
	case "open-source":
		opts = append(opts, gpiod.AsOpenSource)
	case "push-pull":
		fallthrough
	default:
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
