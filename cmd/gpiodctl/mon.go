// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	monCmd.Flags().BoolVarP(&monOpts.ActiveLow, "active-low", "l", false, "treat the line state as active low")
	monCmd.Flags().StringVarP(&monOpts.Bias, "bias", "b", "as-is", "set the line bias.")
	monCmd.Flags().StringVarP(&monOpts.Edge, "edge", "e", "both", "select the edge detection.")
	monCmd.Flags().UintVarP(&monOpts.NumEvents, "num-events", "n", 0, "exit after n edges")
	monCmd.Flags().BoolVarP(&monOpts.Quiet, "quiet", "q", false, "don't display event details")
	monCmd.SetHelpTemplate(monCmd.HelpTemplate() + extendedMonHelp)
	rootCmd.AddCommand(monCmd)
}

var extendedMonHelp = `
Edges:
  both:         both rising and falling edge events are detected
                and reported
  rising:       only rising edge events are detected and reported
  falling:      only falling edge events are detected and reported

Biases:
  as-is:        leave bias unchanged
  disable:      disable bias
  pull-up:      enable pull-up
  pull-down:    enable pull-down
`

var (
	monCmd = &cobra.Command{
		Use:                   "mon [flags] <chip> <offset1>...",
		Short:                 "Monitor the state of a line or lines",
		Long:                  `Wait for events on GPIO lines and print them to standard output.`,
		Args:                  cobra.MinimumNArgs(2),
		RunE:                  mon,
		DisableFlagsInUseLine: true,
	}
	monOpts = struct {
		ActiveLow bool
		Bias      string
		Edge      string
		Quiet     bool
		NumEvents uint
	}{}
)

func mon(cmd *cobra.Command, args []string) error {
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
	signal.Notify(sigdone, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigdone)
	count := uint(0)
	for {
		select {
		case evt := <-evtchan:
			if !monOpts.Quiet {
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
			if monOpts.NumEvents > 0 && count >= monOpts.NumEvents {
				return
			}
		case <-sigdone:
			return
		}
	}
}

func makeMonOpts(eh gpiod.EventHandler) []gpiod.LineReqOption {
	opts := []gpiod.LineReqOption{}
	if monOpts.ActiveLow {
		opts = append(opts, gpiod.AsActiveLow)
	}
	edge := strings.ToLower(monOpts.Edge)
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
	bias := strings.ToLower(monOpts.Bias)
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
