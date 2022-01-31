// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/warthog618/gpiod"
)

func init() {
	getCmd.Flags().BoolVarP(&getOpts.ActiveLow, "active-low", "l", false, "treat the line state as active low")
	getCmd.Flags().BoolVarP(&getOpts.AsIs, "as-is", "a", false, "request the line as-is rather than as an input")
	getCmd.Flags().StringVarP(&getOpts.Bias, "bias", "b", "as-is", "set the line bias.")
	getCmd.Flags().IntVar(&getOpts.AbiV, "abiv", 0, "use specified ABI version.")
	getCmd.Flags().MarkHidden("abiv")
	getCmd.SetHelpTemplate(getCmd.HelpTemplate() + extendedGetHelp)
	rootCmd.AddCommand(getCmd)
}

var extendedGetHelp = `
Biases:
  as-is:        leave bias unchanged
  disable:      disable bias
  pull-up:      enable pull-up
  pull-down:    enable pull-down
`

var (
	getCmd = &cobra.Command{
		Use:                   "get [flags] <chip> <offset1>...",
		Short:                 "Get the state of a line or lines",
		Long:                  `Read the state of a line or lines from a GPIO chip.`,
		Args:                  cobra.MinimumNArgs(2),
		RunE:                  get,
		DisableFlagsInUseLine: true,
	}
	getOpts = struct {
		ActiveLow bool
		AsIs      bool
		Bias      string
		AbiV      int
	}{}
)

func get(cmd *cobra.Command, args []string) error {
	name := args[0]
	oo, err := parseOffsets(args[1:])
	if err != nil {
		return err
	}
	c, err := gpiod.NewChip(name, gpiod.WithConsumer("gpiodctl-get"))
	if err != nil {
		return err
	}
	defer c.Close()
	opts := makeGetOpts()
	l, err := c.RequestLines(oo, opts...)
	if err != nil {
		return fmt.Errorf("error requesting GPIO line: %s", err)
	}
	defer l.Close()
	vv := make([]int, len(l.Offsets()))
	err = l.Values(vv)
	if err != nil {
		return fmt.Errorf("error reading GPIO state: %s", err)
	}
	vstr := fmt.Sprintf("%d", vv[0])
	for _, v := range vv[1:] {
		vstr += fmt.Sprintf(" %d", v)
	}
	fmt.Println(vstr)
	return nil
}

func makeGetOpts() []gpiod.LineReqOption {
	opts := []gpiod.LineReqOption{}
	if getOpts.ActiveLow {
		opts = append(opts, gpiod.AsActiveLow)
	}
	if !getOpts.AsIs {
		opts = append(opts, gpiod.AsInput)
	}
	bias := strings.ToLower(getOpts.Bias)
	switch bias {
	case "pull-up":
		opts = append(opts, gpiod.WithPullUp)
	case "pull-down":
		opts = append(opts, gpiod.WithPullDown)
	case "disable":
		opts = append(opts, gpiod.WithBiasDisabled)
	case "as-is":
		fallthrough
	default:
	}
	if getOpts.AbiV != 0 {
		opts = append(opts, gpiod.WithABIVersion(getOpts.AbiV))
	}
	return opts
}

func parseOffsets(args []string) ([]int, error) {
	oo := []int(nil)
	for _, arg := range args {
		o, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse offset '%s'", arg)
		}
		oo = append(oo, int(o))
	}
	return oo, nil
}
