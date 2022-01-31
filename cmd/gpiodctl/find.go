// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	findCmd.Flags().IntVar(&findOpts.AbiV, "abiv", 0, "use specified ABI version.")
	findCmd.Flags().MarkHidden("abiv")
	rootCmd.AddCommand(findCmd)
}

var (
	findCmd = &cobra.Command{
		Use:                   "find [flags] <line>...",
		Short:                 "Find a GPIO line by name",
		Long:                  `Find a GPIO line by name.  The output of this command can be used as input for get/set/watch.`,
		Args:                  cobra.MinimumNArgs(1),
		Run:                   find,
		DisableFlagsInUseLine: true,
	}
	findOpts = struct {
		AbiV int
	}{}
)

func find(cmd *cobra.Command, args []string) {
	copts := []gpiod.ChipOption{}
	if findOpts.AbiV != 0 {
		copts = append(copts, gpiod.WithABIVersion(findOpts.AbiV))
	}
	for _, linename := range args {
		for _, cname := range gpiod.Chips() {
			c, err := gpiod.NewChip(cname, copts...)
			if err != nil {
				continue
			}
			for o := 0; o < c.Lines(); o++ {
				inf, err := c.LineInfo(o)
				if err != nil {
					continue
				}
				if inf.Name == linename {
					fmt.Printf("%s %d\n", cname, o)
				}
			}
			c.Close()
		}
	}
}
