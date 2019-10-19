// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	getCmd.Flags().BoolVarP(&getOpts.ActiveLow, "active-low", "l", false, "treat the line state as active low")
	getCmd.Flags().BoolVarP(&getOpts.AsIs, "as-is", "a", false, "request the line as-is rather than as an input")
	getCmd.Flags().BoolVarP(&getOpts.PullUp, "pull-up", "u", false, "enable internal pull-up")
	getCmd.Flags().BoolVarP(&getOpts.PullDown, "pull-down", "d", false, "enable internal pull-down")
	getCmd.Flags().BoolVar(&getOpts.BiasDisable, "bias-disable", false, "disable internal bias")
	rootCmd.AddCommand(getCmd)
}

var (
	getCmd = &cobra.Command{
		Use:   "get <chip> <offset1>...",
		Short: "Get the state of a line or lines",
		Long:  `Read the state of a line or lines from a GPIO chip.`,
		Args:  cobra.MinimumNArgs(2),
		RunE:  get,
	}
	getOpts = struct {
		ActiveLow   bool
		AsIs        bool
		PullUp      bool
		PullDown    bool
		BiasDisable bool
	}{}
)

func get(cmd *cobra.Command, args []string) error {
	if getOpts.PullUp && getOpts.PullDown {
		return errors.New("can't pull-up and pull-down at the same time")
	}
	name := args[0]
	oo, err := parseOffsets(args[1:])
	if err != nil {
		return err
	}
	c, err := gpiod.NewChip(name, gpiod.WithConsumer("gpiodctl-get"))
	if err != nil {
		return err
	}
	defer c.Close()
	opts := makeGetOpts()
	l, err := c.RequestLines(oo, opts...)
	if err != nil {
		return fmt.Errorf("error requesting GPIO line: %s", err)
	}
	defer l.Close()
	vv := make([]int, len(l.Offsets()))
	err = l.Values(vv)
	if err != nil {
		return fmt.Errorf("error reading GPIO state: %s", err)
	}
	vstr := fmt.Sprintf("%d", vv[0])
	for _, v := range vv[1:] {
		vstr += fmt.Sprintf(" %d", v)
	}
	fmt.Println(vstr)
	return nil
}

func makeGetOpts() []gpiod.LineOption {
	opts := []gpiod.LineOption{}
	if getOpts.ActiveLow {
		opts = append(opts, gpiod.AsActiveLow)
	}
	if !getOpts.AsIs {
		opts = append(opts, gpiod.AsInput)
	}
	switch {
	case getOpts.BiasDisable:
		opts = append(opts, gpiod.WithBiasDisable)
	case getOpts.PullUp:
		opts = append(opts, gpiod.WithPullUp)
	case getOpts.PullDown:
		opts = append(opts, gpiod.WithPullDown)
	}
	return opts
}

func parseOffsets(args []string) ([]int, error) {
	oo := []int(nil)
	for _, arg := range args {
		o, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse offset '%s'", arg)
		}
		oo = append(oo, int(o))
	}
	return oo, nil
}
