// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package gpiod

import (
	"time"

	"github.com/warthog618/gpiod/uapi"
)

// ChipOption defines the interface required to provide a Chip option.
type ChipOption interface {
	applyChipOption(*ChipOptions)
}

// ChipOptions contains the options for a Chip.
type ChipOptions struct {
	consumer string
	Config   LineConfig
	abi      int
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

// LineReconfig defines the interface required to update an option for Line and
// Lines.
type LineReconfig interface {
	applyLineReconfig(*LineOptions)
}

// LineOptions contains the options for a Line or Lines.
type LineOptions struct {
	consumer string
	Config   LineConfig
	values   []int
	eh       EventHandler
	abi      int
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
	l.Config.Flags &^= uapi.LineFlagV2Direction
	l.Config.Direction = 0
}

// InputOption indicates the line direction should be set to an input.
type InputOption struct{}

// AsInput indicates that a line be requested as an input.
//
// This option overrides and clears any previous Output, OpenDrain, or
// OpenSource options.
var AsInput = InputOption{}

func (o InputOption) applyChipOption(c *ChipOptions) {
	c.Config.Flags |= uapi.LineFlagV2Direction
	c.Config.Direction = uapi.LineDirectionInput
}

func (o InputOption) applyLineOption(l *LineOptions) {
	l.Config.Flags |= uapi.LineFlagV2Direction
	l.Config.Direction = uapi.LineDirectionInput
}

func (o InputOption) applyLineReconfig(l *LineOptions) {
	l.Config.Flags |= uapi.LineFlagV2Direction
	l.Config.Direction = uapi.LineDirectionInput
}

// OutputOption indicates the line direction should be set to an output.
type OutputOption struct {
	values []int
}

// AsOutput indicates that a line or lines be requested as an output.
//
// The initial active state for the line(s) can optionally be provided.
// If fewer values are provided than lines then the remaining lines default to
// inactive.
//
// This option overrides and clears any previous Input, RisingEdge, FallingEdge,
// BothEdges, or Debounce options.
func AsOutput(values ...int) OutputOption {
	vv := append([]int(nil), values...)
	return OutputOption{vv}
}

func (o OutputOption) applyLineOption(l *LineOptions) {
	l.Config.Flags |= uapi.LineFlagV2Direction
	l.Config.Direction = uapi.LineDirectionOutput
	l.values = o.values
	l.Config.Flags &^= (uapi.LineFlagV2EdgeDetection | uapi.LineFlagV2Debounce)
	l.Config.EdgeDetection = 0
	l.Config.Debounce = 0
}

func (o OutputOption) applyLineReconfig(l *LineOptions) {
	o.applyLineOption(l)
}

// LevelOption determines the line level that is considered active.
type LevelOption struct {
	activeLow bool
}

func (o LevelOption) updateFlags(f uapi.LineFlagV2) uapi.LineFlagV2 {
	if o.activeLow {
		return f | uapi.LineFlagV2ActiveLow
	}
	return f &^ uapi.LineFlagV2ActiveLow
}

func (o LevelOption) applyChipOption(c *ChipOptions) {
	c.Config.Flags = o.updateFlags(c.Config.Flags)
}

func (o LevelOption) applyLineOption(l *LineOptions) {
	l.Config.Flags = o.updateFlags(l.Config.Flags)
}

func (o LevelOption) applyLineReconfig(l *LineOptions) {
	l.Config.Flags = o.updateFlags(l.Config.Flags)
}

// AsActiveLow indicates that a line be considered active when the line level
// is low.
var AsActiveLow = LevelOption{true}

// AsActiveHigh indicates that a line be considered active when the line level
// is high.
//
// This is the default active level.
var AsActiveHigh = LevelOption{}

// DriveOption determines if a line is open drain, open source or push-pull.
type DriveOption struct {
	drive uapi.LineDrive
}

func (o DriveOption) applyLineOption(l *LineOptions) {
	l.Config.Flags |= uapi.LineFlagV2Direction | uapi.LineFlagV2Drive
	l.Config.Flags &^= uapi.LineFlagV2EdgeDetection | uapi.LineFlagV2Debounce
	l.Config.Direction = uapi.LineDirectionOutput
	l.Config.Drive = o.drive
	l.Config.EdgeDetection = 0
	l.Config.Debounce = 0
}

func (o DriveOption) applyLineReconfig(l *LineOptions) {
	o.applyLineOption(l)
}

// AsOpenDrain indicates that a line be driven low but left floating for high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenSource, or Debounce options.
var AsOpenDrain = DriveOption{uapi.LineDriveOpenDrain}

// AsOpenSource indicates that a line be driven low but left floating for high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenDrain, or Debounce options.
var AsOpenSource = DriveOption{uapi.LineDriveOpenSource}

// AsPushPull indicates that a line be driven both low and high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenDrain, OpenSource or Debounce
// options.
var AsPushPull = DriveOption{}

// BiasOption indicates how a line is to be biased.
//
// Bias options require Linux v5.5 or later.
type BiasOption struct {
	bias uapi.LineBias
}

func (o BiasOption) applyChipOption(c *ChipOptions) {
	c.Config.Flags |= uapi.LineFlagV2Bias
	c.Config.Bias = o.bias
}

func (o BiasOption) applyLineOption(l *LineOptions) {
	l.Config.Flags |= uapi.LineFlagV2Bias
	l.Config.Bias = o.bias
}

func (o BiasOption) applyLineReconfig(l *LineOptions) {
	o.applyLineOption(l)
}

// WithBiasDisabled indicates that a line have its internal bias disabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithBiasDisabled = BiasOption{uapi.LineBiasDisabled}

// WithPullDown indicates that a line have its internal pull-down enabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithPullDown = BiasOption{uapi.LineBiasPullDown}

// WithPullUp indicates that a line have its internal pull-up enabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithPullUp = BiasOption{uapi.LineBiasPullUp}

// EdgeOption indicates that a line will generate events when edges are detected.
type EdgeOption struct {
	eh   EventHandler
	edge uapi.LineEdge
}

func (o EdgeOption) applyLineOption(l *LineOptions) {
	l.Config.Flags |= (uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection)
	l.Config.Flags &^= uapi.LineFlagV2Drive
	l.Config.Direction = uapi.LineDirectionInput
	l.Config.EdgeDetection = o.edge
	l.Config.Drive = 0
	l.eh = o.eh
}

// WithFallingEdge indicates that a line will generate events when its active
// state transitions from high to low.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithFallingEdge(e func(LineEvent)) EdgeOption {
	return EdgeOption{EventHandler(e), uapi.LineEdgeFalling}
}

// WithRisingEdge indicates that a line will generate events when its active
// state transitions from low to high.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithRisingEdge(e func(LineEvent)) EdgeOption {
	return EdgeOption{EventHandler(e), uapi.LineEdgeRising}
}

// WithBothEdges indicates that a line will generate events when its active
// state transitions from low to high and from high to low.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithBothEdges(e func(LineEvent)) EdgeOption {
	return EdgeOption{EventHandler(e), uapi.LineEdgeBoth}
}

// DebounceOption indicates that a line will be debounced.
type DebounceOption struct {
	period time.Duration
}

func (o DebounceOption) applyLineOption(l *LineOptions) {
	l.Config.Debounce = uint32(o.period.Microseconds())
}

// WithDebounce indicates that a line will be debounced with the specified
// debounce period.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
func WithDebounce(period time.Duration) DebounceOption {
	return DebounceOption{period}
}

// ABIVersionOption selects the version of the GPIO ioctl commands to use.
//
// The default is to use the latest version supported by the kernel.
type ABIVersionOption int

func (o ABIVersionOption) applyChipOption(c *ChipOptions) {
	c.abi = int(o)
}

func (o ABIVersionOption) applyLineOption(l *LineOptions) {
	l.abi = int(o)
}

// WithABIVersion indicates the version of the GPIO ioctls to use.
//
// The default is to use the latest version supported by the kernel.
func WithABIVersion(version int) ABIVersionOption {
	return ABIVersionOption(version)
}
