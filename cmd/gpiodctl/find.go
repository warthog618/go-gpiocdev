// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	rootCmd.AddCommand(findCmd)
}

var findCmd = &cobra.Command{
	Use:                   "find [flags] <line>...",
	Short:                 "Find a GPIO line by name",
	Long:                  `Find a GPIO line by name.  The output of this command can be used as input for get/set/watch.`,
	Args:                  cobra.MinimumNArgs(1),
	Run:                   find,
	DisableFlagsInUseLine: true,
}

func find(cmd *cobra.Command, args []string) {
	for _, linename := range args {
		for _, cname := range gpiod.Chips() {
			c, err := gpiod.NewChip(cname)
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
