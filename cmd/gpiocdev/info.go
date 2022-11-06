// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
    "github.com/warthog618/go-gpiocdev"
)

func init() {
	infoCmd.Flags().IntVar(&infoOpts.AbiV, "abiv", 0, "use specified ABI version.")
	infoCmd.Flags().MarkHidden("abiv")
	rootCmd.AddCommand(infoCmd)
}

var (
	infoCmd = &cobra.Command{
		Use:                   "info [flags] [chip]...",
		Short:                 "Info about chip lines",
		Long:                  `Print information about all lines of the specified GPIO chip(s) (or all gpiochips if none are specified).`,
		Run:                   info,
		DisableFlagsInUseLine: true,
	}
	infoOpts = struct {
		AbiV int
	}{}
)

func info(cmd *cobra.Command, args []string) {
	rc := 0
	cc := []string(nil)
	cc = append(cc, args...)
	if len(cc) == 0 {
		cc = gpiocdev.Chips()
	}
	copts := []gpiocdev.ChipOption{}
	if infoOpts.AbiV != 0 {
		copts = append(copts, gpiocdev.WithABIVersion(infoOpts.AbiV))
	}
	for _, path := range cc {
		c, err := gpiocdev.NewChip(path, copts...)
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

func printLineInfo(li gpiocdev.LineInfo) {
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
	attrs := []string(nil)
	if li.Config.Direction == gpiocdev.LineDirectionOutput {
		attrs = append(attrs, "output")
	} else {
		attrs = append(attrs, "input")
	}
	if li.Config.ActiveLow {
		attrs = append(attrs, "active-low")
	}
	if li.Used {
		attrs = append(attrs, "used")
	}
	switch li.Config.Drive {
	case gpiocdev.LineDriveOpenDrain:
		attrs = append(attrs, "open-drain")
	case gpiocdev.LineDriveOpenSource:
		attrs = append(attrs, "open-source")
	}
	switch li.Config.Bias {
	case gpiocdev.LineBiasPullUp:
		attrs = append(attrs, "pull-up")
	case gpiocdev.LineBiasPullDown:
		attrs = append(attrs, "pull-down")
	case gpiocdev.LineBiasDisabled:
		attrs = append(attrs, "bias-disabled")
	}
	if li.Config.DebouncePeriod != 0 {
		attrs = append(attrs,
			fmt.Sprintf("debounce-period=%s", li.Config.DebouncePeriod))
	}
	attrstr := ""
	if len(attrs) > 0 {
		attrstr = "[" + strings.Join(attrs, ",") + "]"
	}
	fmt.Printf("\tline %3d:%12s %16s %s\n",
		li.Offset, li.Name, li.Consumer, attrstr)
}
