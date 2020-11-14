// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package gpiod

import (
	"time"
)

// ChipOption defines the interface required to provide a Chip option.
type ChipOption interface {
	applyChipOption(*ChipOptions)
}

// ChipOptions contains the options for a Chip.
type ChipOptions struct {
	consumer string
	config   LineConfig
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

func (o ConsumerOption) applyLineReqOption(l *lineReqOptions) {
	l.consumer = string(o)
}

// LineReqOption defines the interface required to provide an option for Line and
// Lines as part of a line request.
type LineReqOption interface {
	applyLineReqOption(*lineReqOptions)
}

// LineConfigOption defines the interface required to update an option for Line and
// Lines.
type LineConfigOption interface {
	applyLineConfigOption(*lineConfigOptions)
}

// lineReqOptions contains the options for a Line(s) request.
type lineReqOptions struct {
	lineConfigOptions
	consumer string
	abi      int
}

// lineConfigOptions contains the configuration options for a Line(s) reconfigure.
type lineConfigOptions struct {
	offsets []int
	values  []int
	defCfg  LineConfig
	lineCfg map[int]LineConfig
	eh      EventHandler
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

func (o AsIsOption) applyLineReqOption(l *lineReqOptions) {
	l.defCfg.Direction = LineDirectionUnknown
}

// InputOption indicates the line direction should be set to an input.
type InputOption struct{}

// AsInput indicates that a line be requested as an input.
//
// This option overrides and clears any previous Output, OpenDrain, or
// OpenSource options.
var AsInput = InputOption{}

func (o InputOption) applyChipOption(c *ChipOptions) {
	c.config.Direction = LineDirectionInput
}

func (o InputOption) applyLineReqOption(l *lineReqOptions) {
	l.defCfg.Direction = LineDirectionInput
}

func (o InputOption) applyLineConfigOption(l *lineConfigOptions) {
	l.defCfg.Direction = LineDirectionInput
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

func (o OutputOption) applyLineReqOption(l *lineReqOptions) {
	o.applyLineConfigOption(&l.lineConfigOptions)
}

func (o OutputOption) applyLineConfigOption(l *lineConfigOptions) {
	l.defCfg.Direction = LineDirectionOutput
	l.values = o.values
	l.defCfg.Debounced = false
	l.defCfg.DebouncePeriod = 0
}

// LevelOption determines the line level that is considered active.
type LevelOption struct {
	activeLow bool
}

func (o LevelOption) applyChipOption(c *ChipOptions) {
	c.config.ActiveLow = o.activeLow
}

func (o LevelOption) applyLineReqOption(l *lineReqOptions) {
	l.defCfg.ActiveLow = o.activeLow
}

func (o LevelOption) applyLineConfigOption(l *lineConfigOptions) {
	l.defCfg.ActiveLow = o.activeLow
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
	drive LineDrive
}

func (o DriveOption) applyLineReqOption(l *lineReqOptions) {
	o.applyLineConfigOption(&l.lineConfigOptions)
}

func (o DriveOption) applyLineConfigOption(l *lineConfigOptions) {
	l.defCfg.Drive = o.drive
	l.defCfg.Direction = LineDirectionOutput
	l.defCfg.Debounced = false
	l.defCfg.DebouncePeriod = 0
}

// AsOpenDrain indicates that a line be driven low but left floating for high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenSource, or Debounce options.
var AsOpenDrain = DriveOption{LineDriveOpenDrain}

// AsOpenSource indicates that a line be driven low but left floating for high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenDrain, or Debounce options.
var AsOpenSource = DriveOption{LineDriveOpenSource}

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
	bias LineBias
}

func (o BiasOption) applyChipOption(c *ChipOptions) {
	c.config.Bias = o.bias
}

func (o BiasOption) applyLineReqOption(l *lineReqOptions) {
	o.applyLineConfigOption(&l.lineConfigOptions)
}

func (o BiasOption) applyLineConfigOption(l *lineConfigOptions) {
	l.defCfg.Bias = o.bias
}

// WithBiasDisabled indicates that a line have its internal bias disabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithBiasDisabled = BiasOption{LineBiasDisabled}

// WithPullDown indicates that a line have its internal pull-down enabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithPullDown = BiasOption{LineBiasPullDown}

// WithPullUp indicates that a line have its internal pull-up enabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
var WithPullUp = BiasOption{LineBiasPullUp}

// EventHandlerOption provides the handler for events on requested lines.
type EventHandlerOption struct {
	eh EventHandler
}

func (o EventHandlerOption) applyLineReqOption(lro *lineReqOptions) {
	lro.eh = o.eh
}

// WithEventHandler indicates that a line will generate events when its active
// state transitions from high to low.
//
// Events are forwarded to the provided handler function.
func WithEventHandler(e func(LineEvent)) EventHandlerOption {
	return EventHandlerOption{e}
}

// EdgeOption indicates that a line will generate events when edges are detected.
type EdgeOption struct {
	edge LineEdge
}

func (o EdgeOption) applyLineReqOption(lro *lineReqOptions) {
	lro.defCfg.EdgeDetection = o.edge
	lro.defCfg.Direction = LineDirectionInput
}

func (o EdgeOption) applyLineConfigOption(lco *lineConfigOptions) {
	lco.defCfg.EdgeDetection = o.edge
	lco.defCfg.Direction = LineDirectionInput
}

// WithFallingEdge indicates that a line will generate events when its active
// state transitions from high to low.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
var WithFallingEdge = EdgeOption{LineEdgeFalling}

// WithRisingEdge indicates that a line will generate events when its active
// state transitions from low to high.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
var WithRisingEdge = EdgeOption{LineEdgeRising}

// WithBothEdges indicates that a line will generate events when its active
// state transitions from low to high and from high to low.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
var WithBothEdges = EdgeOption{LineEdgeBoth}

// DebounceOption indicates that a line will be debounced.
//
// The DebounceOption requires Linux v5.10 or later.
type DebounceOption struct {
	period time.Duration
}

func (o DebounceOption) applyLineReqOption(l *lineReqOptions) {
	o.applyLineConfigOption(&l.lineConfigOptions)
}

func (o DebounceOption) applyLineConfigOption(l *lineConfigOptions) {
	l.defCfg.Direction = LineDirectionInput
	l.defCfg.Debounced = true
	l.defCfg.DebouncePeriod = o.period
}

// WithDebounce indicates that a line will be debounced with the specified
// debounce period.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
//
// Requires Linux v5.10 or later.
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

func (o ABIVersionOption) applyLineReqOption(l *lineReqOptions) {
	l.abi = int(o)
}

// WithABIVersion indicates the version of the GPIO ioctls to use.
//
// The default is to use the latest version supported by the kernel.
//
// ABI version 2 requires Linux v5.10 or later.
func WithABIVersion(version int) ABIVersionOption {
	return ABIVersionOption(version)
}
