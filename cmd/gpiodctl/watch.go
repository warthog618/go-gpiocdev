// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// +build linux

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	watchCmd.Flags().UintVarP(&watchOpts.NumEvents, "num-events", "n", 0, "exit after n events")
	watchCmd.Flags().BoolVarP(&watchOpts.Verbose, "verbose", "v", false, "display complete line info")
	rootCmd.AddCommand(watchCmd)
}

var (
	watchCmd = &cobra.Command{
		Use:                   "watch [flags] <chip> [offset1]...",
		Short:                 "Watch lines for changes to the line info",
		Long:                  `Wait for changes to info on GPIO lines and print them to standard output.`,
		Args:                  cobra.MinimumNArgs(1),
		RunE:                  watch,
		DisableFlagsInUseLine: true,
	}
	watchOpts = struct {
		Verbose   bool
		NumEvents uint
	}{}
)

func watch(cmd *cobra.Command, args []string) error {
	name := args[0]
	oo, err := parseOffsets(args[1:])
	if err != nil {
		return err
	}
	c, err := gpiod.NewChip(name)
	if err != nil {
		return err
	}
	defer c.Close()
	evtchan := make(chan gpiod.LineInfoChangeEvent)
	eh := func(evt gpiod.LineInfoChangeEvent) {
		evtchan <- evt
	}
	for _, o := range oo {
		info, err := c.WatchLineInfo(o, eh)
		if err != nil {
			return fmt.Errorf("error requesting watch on line %d: %s", o, err)
		}
		if watchOpts.Verbose {
			printLineInfo(info)
		}
	}
	if len(oo) == 0 {
		for o := 0; o < c.Lines(); o++ {
			info, err := c.WatchLineInfo(o, eh)
			if err != nil {
				return fmt.Errorf("error requesting watch on line %d: %s", o, err)
			}
			if watchOpts.Verbose {
				printLineInfo(info)
			}
		}
	}
	watchWait(evtchan)
	return nil
}

func watchWait(evtchan <-chan gpiod.LineInfoChangeEvent) {
	sigdone := make(chan os.Signal, 1)
	signal.Notify(sigdone, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigdone)
	count := uint(0)
	etypes := map[gpiod.LineInfoChangeType]string{
		gpiod.LineRequested:    "requested",
		gpiod.LineReleased:     "released",
		gpiod.LineReconfigured: "reconfigured",
	}
	for {
		select {
		case evt := <-evtchan:
			t := time.Now()
			fmt.Printf("event:%3d %-12s %s (%s)\n",
				evt.Info.Offset,
				etypes[evt.Type],
				t.Format(time.RFC3339Nano),
				evt.Timestamp)
			if watchOpts.Verbose {
				printLineInfo(evt.Info)
			}
			count++
			if watchOpts.NumEvents > 0 && count >= watchOpts.NumEvents {
				return
			}
		case <-sigdone:
			return
		}
	}
}
