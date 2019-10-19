// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	monCmd.Flags().BoolVarP(&monOpts.ActiveLow, "active-low", "l", false, "treat the line state as active low")
	monCmd.Flags().BoolVarP(&monOpts.FallingEdge, "falling-edge", "f", false, "detect only falling edge events")
	monCmd.Flags().BoolVarP(&monOpts.RisingEdge, "rising-edge", "r", false, "detect only rising edge events")
	monCmd.Flags().UintVarP(&monOpts.NumEvents, "num-events", "n", 0, "exit after n edges")
	monCmd.Flags().BoolVarP(&monOpts.Quiet, "quiet", "q", false, "don't display event details")
	monCmd.Flags().BoolVarP(&monOpts.PullUp, "pull-up", "u", false, "enable internal pull-up")
	monCmd.Flags().BoolVarP(&monOpts.PullDown, "pull-down", "d", false, "enable internal pull-down")
	monCmd.Flags().BoolVar(&monOpts.BiasDisable, "bias-disable", false, "disable internal bias")
	monCmd.SetHelpTemplate(monCmd.HelpTemplate() + extendedMonHelp)
	rootCmd.AddCommand(monCmd)
}

var extendedMonHelp = `
By default both rising and falling edge events are detected and reported.
`

var (
	monCmd = &cobra.Command{
		Use:   "mon <chip> <offset1>...",
		Short: "Monitor the state of a line or lines",
		Long:  `Wait for events on GPIO lines and print them to standard output.`,
		Args:  cobra.MinimumNArgs(2),
		RunE:  mon,
	}
	monOpts = struct {
		ActiveLow   bool
		RisingEdge  bool
		FallingEdge bool
		Quiet       bool
		NumEvents   uint
		PullUp      bool
		PullDown    bool
		BiasDisable bool
	}{}
)

func mon(cmd *cobra.Command, args []string) error {
	if monOpts.RisingEdge && monOpts.FallingEdge {
		return errors.New("can't filter both falling-edge and rising-edge events")
	}
	if monOpts.PullUp && monOpts.PullDown {
		return errors.New("can't pull-up and pull-down at the same time")
	}
	name := args[0]
	oo, err := parseOffsets(args[1:])
	if err != nil {
		return err
	}
	c, err := gpiod.NewChip(name, gpiod.WithConsumer("gpiodctl-mon"))
	if err != nil {
		return err
	}
	defer c.Close()
	evtchan := make(chan gpiod.LineEvent)
	eh := func(evt gpiod.LineEvent) {
		evtchan <- evt
	}
	opts := makeMonOpts(eh)
	l, err := c.RequestLines(oo, opts...)
	if err != nil {
		return fmt.Errorf("error requesting GPIO lines: %s", err)
	}
	defer l.Close()
	monWait(evtchan)
	return nil
}

func monWait(evtchan <-chan gpiod.LineEvent) {
	sigdone := make(chan os.Signal, 1)
	signal.Notify(sigdone, os.Interrupt, os.Kill)
	defer signal.Stop(sigdone)
	count := uint(0)
	for {
		select {
		case evt := <-evtchan:
			if !monOpts.Quiet {
				t := time.Unix(0, evt.Timestamp.Nanoseconds())
				edge := "rising"
				if evt.Type == gpiod.LineEventFallingEdge {
					edge = "falling"
				}
				fmt.Printf("event:%3d %-7s %s\n", evt.Offset, edge, t.Format(time.RFC3339Nano))
			}
			count++
			if monOpts.NumEvents > 0 && count >= monOpts.NumEvents {
				return
			}
		case <-sigdone:
			return
		}
	}
}

func makeMonOpts(eh gpiod.EventHandler) []gpiod.LineOption {
	opts := []gpiod.LineOption{}
	if monOpts.ActiveLow {
		opts = append(opts, gpiod.AsActiveLow)
	}
	switch {
	case monOpts.RisingEdge == monOpts.FallingEdge:
		opts = append(opts, gpiod.WithBothEdges(eh))
	case monOpts.RisingEdge:
		opts = append(opts, gpiod.WithRisingEdge(eh))
	case monOpts.FallingEdge:
		opts = append(opts, gpiod.WithFallingEdge(eh))
	}
	switch {
	case monOpts.BiasDisable:
		opts = append(opts, gpiod.WithBiasDisable)
	case monOpts.PullUp:
		opts = append(opts, gpiod.WithPullUp)
	case monOpts.PullDown:
		opts = append(opts, gpiod.WithPullDown)
	}
	return opts
}
