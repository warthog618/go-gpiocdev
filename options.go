// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package gpiod

import "github.com/warthog618/gpiod/uapi"

// ChipOption defines the interface required to provide a Chip option.
type ChipOption interface {
	applyChipOption(*ChipOptions)
}

// ChipOptions contains the options for a Chip.
type ChipOptions struct {
	consumer    string
	HandleFlags uapi.HandleFlag
}

// ConsumerOption defines the consumer label for a line.
type ConsumerOption string

// WithConsumer provides the consumer label for the line.
//
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

// LineConfig defines the interface required to update an option for Line and
// Lines.
type LineConfig interface {
	applyLineConfig(*LineOptions)
}

// LineOptions contains the options for a Line or Lines.
type LineOptions struct {
	consumer      string
	InitialValues []int
	EventFlags    uapi.EventFlag
	HandleFlags   uapi.HandleFlag
	eh            EventHandler
}

// EventHandler is a receiver for line events.
type EventHandler func(LineEvent)

// AsIsOption indicates the line direction should be left as is.
type AsIsOption struct{}

// AsIs indicates that a line be requested as neither an input or output.
//
// That is its direction is left as is. This option overrides and clears any
// previous Input or Output options.
var AsIs = AsIsOption{}

func (o AsIsOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestOutput | uapi.HandleRequestInput)
}

// InputOption indicates the line direction should be set to an input.
type InputOption struct{}

// AsInput indicates that a line be requested as an input.
//
// This option overrides and clears any previous Output, OpenDrain, or
// OpenSource options.
var AsInput = InputOption{}

func (o InputOption) updateFlags(f uapi.HandleFlag) uapi.HandleFlag {
	f &= ^(uapi.HandleRequestOutput |
		uapi.HandleRequestOpenDrain |
		uapi.HandleRequestOpenSource)
	f |= uapi.HandleRequestInput
	return f
}

func (o InputOption) applyChipOption(c *ChipOptions) {
	c.HandleFlags = o.updateFlags(c.HandleFlags)
}

func (o InputOption) applyLineOption(l *LineOptions) {
	l.HandleFlags = o.updateFlags(l.HandleFlags)
}

func (o InputOption) applyLineConfig(l *LineOptions) {
	o.applyLineOption(l)
}

// OutputOption indicates the line direction should be set to an output.
type OutputOption struct {
	initialValues []int
}

// AsOutput indicates that a line or lines be requested as an output.
//
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
	l.InitialValues = o.initialValues
}

func (o OutputOption) applyLineConfig(l *LineOptions) {
	o.applyLineOption(l)
}

// LevelOption determines the line level that is considered active.
type LevelOption struct {
	flag uapi.HandleFlag
}

func (o LevelOption) updateFlags(f uapi.HandleFlag) uapi.HandleFlag {
	f &= ^uapi.HandleRequestActiveLow
	f |= o.flag
	return f
}

func (o LevelOption) applyChipOption(c *ChipOptions) {
	c.HandleFlags = o.updateFlags(c.HandleFlags)
}

func (o LevelOption) applyLineOption(l *LineOptions) {
	l.HandleFlags = o.updateFlags(l.HandleFlags)
}

func (o LevelOption) applyLineConfig(l *LineOptions) {
	o.applyLineOption(l)
}

// AsActiveLow indicates that a line be considered active when the line level
// is low.
var AsActiveLow = LevelOption{uapi.HandleRequestActiveLow}

// AsActiveHigh indicates that a line be considered active when the line level
// is high.
//
// This is the default active level.
var AsActiveHigh = LevelOption{}

// DriveOption determines if a line is open drain, open source or push-pull.
type DriveOption struct {
	flag uapi.HandleFlag
}

func (o DriveOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestInput |
		uapi.HandleRequestOpenDrain |
		uapi.HandleRequestOpenSource)
	l.HandleFlags |= (o.flag | uapi.HandleRequestOutput)
	l.EventFlags = 0
}

func (o DriveOption) applyLineConfig(l *LineOptions) {
	o.applyLineOption(l)
}

// AsOpenDrain indicates that a line be driven low but left floating for high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, or OpenSource options.
var AsOpenDrain = DriveOption{uapi.HandleRequestOpenDrain}

// AsOpenSource indicates that a line be driven low but left floating for high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, or OpenDrain options.
var AsOpenSource = DriveOption{uapi.HandleRequestOpenSource}

// AsPushPull indicates that a line be driven both low and high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenDrain, or OpenSource options.
var AsPushPull = DriveOption{}

// BiasOption indicates how a line is to be biased.
//
// Bias options require Linux v5.5 or later.
type BiasOption struct {
	flag uapi.HandleFlag
}

func (o BiasOption) updateFlags(f uapi.HandleFlag) uapi.HandleFlag {
	f &= ^(uapi.HandleRequestBiasDisable |
		uapi.HandleRequestPullDown |
		uapi.HandleRequestPullUp)
	f |= o.flag
	return f
}

func (o BiasOption) applyChipOption(c *ChipOptions) {
	c.HandleFlags = o.updateFlags(c.HandleFlags)
}

func (o BiasOption) applyLineOption(l *LineOptions) {
	l.HandleFlags = o.updateFlags(l.HandleFlags)
}

func (o BiasOption) applyLineConfig(l *LineOptions) {
	o.applyLineOption(l)
}

// WithBiasDisable indicates that a line have its internal bias disabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithBiasDisable = BiasOption{uapi.HandleRequestBiasDisable}

// WithPullDown indicates that a line have its internal pull-down enabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithPullDown = BiasOption{uapi.HandleRequestPullDown}

// WithPullUp indicates that a line have its internal pull-up enabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithPullUp = BiasOption{uapi.HandleRequestPullUp}

// EdgeOption indicates that a line will generate events when edges are detected.
type EdgeOption struct {
	e    EventHandler
	edge uapi.EventFlag
}

func (o EdgeOption) applyLineOption(l *LineOptions) {
	l.HandleFlags &= ^(uapi.HandleRequestOutput |
		uapi.HandleRequestOpenDrain |
		uapi.HandleRequestOpenSource)
	l.HandleFlags |= uapi.HandleRequestInput
	l.EventFlags = o.edge
	l.eh = o.e
}

// WithFallingEdge indicates that a line will generate events when its active
// state transitions from high to low.
//
// Events are forwarded to the provided handler function.
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithFallingEdge(e func(LineEvent)) EdgeOption {
	return EdgeOption{EventHandler(e), uapi.EventRequestFallingEdge}
}

// WithRisingEdge indicates that a line will generate events when its active
// state transitions from low to high.
//
// Events are forwarded to the provided handler function.
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithRisingEdge(e func(LineEvent)) EdgeOption {
	return EdgeOption{EventHandler(e), uapi.EventRequestRisingEdge}
}

// WithBothEdges indicates that a line will generate events when its active
// state transitions from low to high and from high to low.
//
// Events are forwarded to the provided handler function.
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithBothEdges(e func(LineEvent)) EdgeOption {
	return EdgeOption{EventHandler(e), uapi.EventRequestBothEdges}
}
