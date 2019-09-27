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
	Use:   "find <line>...",
	Short: "Find a GPIO line by name",
	Long:  `Find a GPIO line by name.  The output of this command can be used as input for gpiod get/set.`,
	Args:  cobra.MinimumNArgs(1),
	Run:   find,
}

func find(cmd *cobra.Command, args []string) {
	for _, linename := range args {
		if cname, offset, err := gpiod.FindLine(linename); err == nil {
			fmt.Printf("%s %d\n", cname, offset)
		} else {
			logErr(cmd, fmt.Errorf("'%s' %s", linename, err))
		}
	}
}
