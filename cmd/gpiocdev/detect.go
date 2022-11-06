// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/warthog618/go-gpiocdev"
)

func init() {
	rootCmd.AddCommand(detectCmd)
}

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect available GPIO chips",
	Long:  `List all GPIO chips, print their labels and number of GPIO lines.`,
	Run:   detect,
}

func detect(cmd *cobra.Command, args []string) {
	rc := 0
	cc := gpiocdev.Chips()
	for _, path := range cc {
		c, err := gpiocdev.NewChip(path)
		if err != nil {
			logErr(cmd, err)
			rc = 1
			continue
		}
		fmt.Printf("%s [%s] (%d lines) using kernel uAPI v%d\n",
			c.Name, c.Label, c.Lines(), c.UapiAbiVersion())
		c.Close()
	}
	os.Exit(rc)
}
