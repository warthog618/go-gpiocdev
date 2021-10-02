// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A utility to control GPIO lines.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gpiodctl",
	Short: "gpiodctl is a utility to control GPIO lines",
	Long:  "gpiodctl is a utility to control GPIO lines on Linux GPIO character devices",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func logErr(cmd *cobra.Command, err error) {
	fmt.Fprintf(os.Stderr, "gpiodctl %s: %s\n", cmd.Name(), err)
}
