// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package gpiod

import "github.com/warthog618/gpiod/uapi"

// ChipOption defines the interface required to provide a Chip option.
type ChipOption interface {
	applyChipOption(*ChipOptions)
}

// ChipOptions contains the options for a Chip.
type ChipOptions struct {
	consumer string
}

// ConsumerOption defines the consumer label for a line.
type ConsumerOption string

// WithConsumer provides the consumer label for the line.
// When applied to a chip it provides the default consumer label for all lines
// requested by the chip.
func WithConsumer(consumer string) ConsumerOption {
	return ConsumerOption(consumer)

}

func (o ConsumerOption) applyChipOption(c *ChipOptions) {
	c.consumer = string(o)
}

func (o ConsumerOption) applyLineOption(l *LineOptions) {
	l.consumer = string(o)
}

// LineOption defines the interface required to provide an option for Line and
// Lines.
type LineOption interface {
	applyLineOption(*LineOptions)
}

// LineOptions contains the options for a Line or Lines.
type LineOptions struct {
	consumer      string
	DefaultValues []int
	EventFlags    uapi.EventFlag
	HandleFlags   uapi.HandleFlag
	eh            EventHandler
}

// EventHandler is a receiver for line events.
type EventHandler func(LineEvent)

// AsIsOption indicates the line direction should be left as is.
type AsIsOption struct{}

// AsIs indicates that a line be requested as neither an input or output.
// That is its direction is left as is. This option overrides and clears any
// previous Input or Output options.
func AsIs() AsIsOption {
	return AsIsOption{}
}

func (o AsIsOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestOutput | uapi.HandleRequestInput)
}

// InputOption indicates the line direction should be set to an input.
type InputOption struct{}

// AsInput indicates that a line be requested as an input.
// This option overrides and clears any previous Output, OpenDrain, or
// OpenSource options.
func AsInput() InputOption {
	return InputOption{}
}

func (o InputOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestOutput |
		uapi.HandleRequestOpenDrain |
		uapi.HandleRequestOpenSource)
	l.HandleFlags |= uapi.HandleRequestInput
}

// OutputOption indicates the line direction should be set to an output.
type OutputOption struct {
	defaultValues []int
}

// AsOutput indicates that a line or lines be requested as an output.
// The initial active state for the line(s) can optionally be provided.
// If fewer values are provided than lines then the remaining lines default to
// inactive.
//
// This option overrides and clears any previous Input, RisingEdge, FallingEdge,
// or BothEdges options.
func AsOutput(values ...int) OutputOption {
	vv := append([]int(nil), values...)
	return OutputOption{vv}
}

func (o OutputOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^uapi.HandleRequestInput
	l.HandleFlags |= uapi.HandleRequestOutput
	l.EventFlags = 0
	l.DefaultValues = o.defaultValues
}

// ActiveLowOption indicates the line be considered active when the line level
// is low.
type ActiveLowOption struct{}

// AsActiveLow indicates that a line be considered active when the line level
// is low.
func AsActiveLow() ActiveLowOption {
	return ActiveLowOption{}
}

func (o ActiveLowOption) applyLineOption(l *LineOptions) {
	l.HandleFlags |= uapi.HandleRequestActiveLow
}

// OpenDrainOption indicates that a line be driven low but left floating for
// high.
type OpenDrainOption struct{}

// AsOpenDrain indicates that a line be driven low but left floating for high.
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, or OpenSource options.
func AsOpenDrain() OpenDrainOption {
	return OpenDrainOption{}
}

func (o OpenDrainOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestInput | uapi.HandleRequestOpenSource)
	l.HandleFlags |= (uapi.HandleRequestOpenDrain | uapi.HandleRequestOutput)
	l.EventFlags = 0
}

// OpenSourceOption indicates that a line be driven high but left floating for
// low.
type OpenSourceOption struct{}

// AsOpenSource indicates that a line be driven low but left floating for hign.
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, or OpenDrain options.
func AsOpenSource() OpenSourceOption {
	return OpenSourceOption{}
}

func (o OpenSourceOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestInput | uapi.HandleRequestOpenDrain)
	l.HandleFlags |= (uapi.HandleRequestOpenSource | uapi.HandleRequestOutput)
	l.EventFlags = 0
}

// FallingEdgeOption indicates that a line will generate events when its active
// state transitions from high to low.
type FallingEdgeOption struct {
	e EventHandler
}

// WithFallingEdge indicates that a line will generate events when its active
// state transitions from high to low.
// Events are forwarded to the provided handler function.
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithFallingEdge(e func(LineEvent)) FallingEdgeOption {
	return FallingEdgeOption{EventHandler(e)}
}

func (o FallingEdgeOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestOutput |
		uapi.HandleRequestOpenDrain |
		uapi.HandleRequestOpenSource)
	l.HandleFlags |= uapi.HandleRequestInput
	l.EventFlags |= uapi.EventRequestFallingEdge
	l.eh = o.e
}

// RisingEdgeOption indicates that a line will generate events when its active
// state transitions from low to high.
type RisingEdgeOption struct {
	e EventHandler
}

// WithRisingEdge indicates that a line will generate events when its active
// state transitions from low to high.
// Events are forwarded to the provided handler function.
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithRisingEdge(e func(LineEvent)) RisingEdgeOption {
	return RisingEdgeOption{EventHandler(e)}
}

func (o RisingEdgeOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestOutput |
		uapi.HandleRequestOpenDrain |
		uapi.HandleRequestOpenSource)
	l.HandleFlags |= uapi.HandleRequestInput
	l.EventFlags |= uapi.EventRequestRisingEdge
	l.eh = o.e
}

// BothEdgesOption indicates that a line will generate events when its active
// state transitions from low to high and from high to low.
type BothEdgesOption struct {
	e EventHandler
}

// WithBothEdges indicates that a line will generate events when its active
// state transitions from low to high and from high to low.
// Events are forwarded to the provided handler function.
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithBothEdges(e func(LineEvent)) BothEdgesOption {
	return BothEdgesOption{EventHandler(e)}
}

func (o BothEdgesOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestOutput |
		uapi.HandleRequestOpenDrain |
		uapi.HandleRequestOpenSource)
	l.HandleFlags |= uapi.HandleRequestInput
	l.EventFlags |= uapi.EventRequestBothEdges
	l.eh = o.e
}
