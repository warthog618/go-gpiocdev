// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	rootCmd.AddCommand(infoCmd)
}

var infoCmd = &cobra.Command{
	Use:                   "info [flags] [chip]...",
	Short:                 "Info about chip lines",
	Long:                  `Print information about all lines of the specified GPIO chip(s) (or all gpiochips if none are specified).`,
	Run:                   info,
	DisableFlagsInUseLine: true,
}

func info(cmd *cobra.Command, args []string) {
	rc := 0
	cc := []string(nil)
	cc = append(cc, args...)
	if len(cc) == 0 {
		cc = gpiod.Chips()
	}
	for _, path := range cc {
		c, err := gpiod.NewChip(path)
		if err != nil {
			logErr(cmd, err)
			rc = 1
			continue
		}
		fmt.Printf("%s - %d lines:\n", c.Name, c.Lines())
		for o := 0; o < c.Lines(); o++ {
			li, err := c.LineInfo(o)
			if err != nil {
				logErr(cmd, err)
				rc = 1
				continue
			}
			printLineInfo(li)
		}
		c.Close()
	}
	os.Exit(rc)
}

func printLineInfo(li gpiod.LineInfo) {
	if len(li.Name) == 0 {
		li.Name = "unnamed"
	}
	if li.Used {
		if len(li.Consumer) == 0 {
			li.Consumer = "kernel"
		}
		if strings.Contains(li.Consumer, " ") {
			li.Consumer = "\"" + li.Consumer + "\""
		}
	} else {
		li.Consumer = "unused"
	}
	dirn := "input"
	if li.Config.Direction == gpiod.LineDirectionOutput {
		dirn = "output"
	}
	active := "active-high"
	if li.Config.ActiveLow {
		active = "active-low"
	}
	attrs := []string(nil)
	if li.Used {
		attrs = append(attrs, "used")
	}
	if li.Config.Drive == gpiod.LineDriveOpenDrain {
		attrs = append(attrs, "open-drain")
	}
	if li.Config.Drive == gpiod.LineDriveOpenSource {
		attrs = append(attrs, "open-source")
	}
	if li.Config.Bias == gpiod.LineBiasPullUp {
		attrs = append(attrs, "pull-up")
	}
	if li.Config.Bias == gpiod.LineBiasPullDown {
		attrs = append(attrs, "pull-down")
	}
	if li.Config.Bias == gpiod.LineBiasDisabled {
		attrs = append(attrs, "bias-disabled")
	}
	if li.Config.DebouncePeriod != 0 {
		attrs = append(attrs,
			fmt.Sprintf("debouncePeriod=%s", li.Config.DebouncePeriod))
	}
	attrstr := ""
	if len(attrs) > 0 {
		attrstr = "[" + strings.Join(attrs, " ") + "]"
	}
	fmt.Printf("\tline %3d:%12s%12s%8s%13s%s\n",
		li.Offset, li.Name, li.Consumer, dirn, active, attrstr)
}
