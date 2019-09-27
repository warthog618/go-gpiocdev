// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "undefined"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version",
	Long:  `All software has versions. This is gpiod's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s (gpiod) %s\n", os.Args[0], version)
	},
}
