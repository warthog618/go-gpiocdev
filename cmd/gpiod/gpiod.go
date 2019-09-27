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

var rootCmd = &cobra.Command{
	Use:   "gpiod",
	Short: "gpiod is a tool to access and manipulate GPIO lines",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	Version: version,
}

func init() {
}

func initConfig() {
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func logErr(cmd *cobra.Command, err error) {
	fmt.Fprintf(os.Stderr, "gpiod %s: %s\n", cmd.Name(), err)
}
