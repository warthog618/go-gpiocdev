// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package gpiocdev

import (
	"time"

	"github.com/warthog618/go-gpiocdev/uapi"
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
	eh       EventHandler
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

// LineConfigOption defines the interface required to update an option for
// Line and Lines.
type LineConfigOption interface {
	applyLineConfigOption(*lineConfigOptions)
}

// SubsetLineConfigOption defines the interface required to update an option for a
// subset of requested lines.
type SubsetLineConfigOption interface {
	applySubsetLineConfigOption([]int, *lineConfigOptions)
}

// lineReqOptions contains the options for a Line(s) request.
type lineReqOptions struct {
	lineConfigOptions
	consumer        string
	abi             int
	eh              EventHandler
	eventBufferSize int
}

// lineConfigOptions contains the configuration options for a Line(s) reconfigure.
type lineConfigOptions struct {
	offsets []int
	values  map[int]int
	defCfg  LineConfig
	lineCfg map[int]*LineConfig
}

func (lco *lineConfigOptions) lineConfig(offset int) *LineConfig {
	if lco.lineCfg == nil {
		lco.lineCfg = map[int]*LineConfig{}
	}
	lc := lco.lineCfg[offset]
	if lc == nil {
		tlc := lco.defCfg
		lc = &tlc
		lco.lineCfg[offset] = lc
	}
	return lc
}

func (lco lineConfigOptions) outputValues() uapi.OutputValues {
	ov := uapi.LineBitmap(0)
	for idx, val := range lco.offsets {
		ov = ov.Set(idx, lco.values[val])
	}
	return uapi.OutputValues(ov)
}

type lineConfigAttributes []uapi.LineConfigAttribute

func (lca lineConfigAttributes) append(attr uapi.LineAttribute, mask uapi.LineBitmap) lineConfigAttributes {
	for idx, cae := range lca {
		if cae.Attr.ID == attr.ID {
			lca[idx].Mask &^= mask
		}
	}
	for idx, cae := range lca {
		if cae.Attr == attr {
			lca[idx].Mask |= mask
			return lca
		}
	}
	return append(lca, uapi.LineConfigAttribute{Attr: attr, Mask: mask})
}

func (lco lineConfigOptions) toULineConfig() (ulc uapi.LineConfig, err error) {

	mask := uapi.NewLineBitMask(len(lco.offsets))
	cfgAttrs := lineConfigAttributes{
		// first cfg slot reserved for default flags
		uapi.LineConfigAttribute{Attr: uapi.LineFlagV2(0).Encode(), Mask: mask},
	}
	attrs := lco.defCfg.toLineAttributes()
	for _, attr := range attrs {
		if attr.ID == uapi.LineAttributeIDFlags {
			cfgAttrs[0].Attr = attr
		} else {
			cfgAttrs = cfgAttrs.append(attr, mask)
		}
	}

	var outputMask uapi.LineBitmap
	if lco.defCfg.Direction == LineDirectionOutput {
		outputMask = mask
	}

	for idx, offset := range lco.offsets {
		cfg := lco.lineCfg[offset]
		if cfg == nil {
			continue
		}
		mask = uapi.LineBitmap(1) << uint(idx)
		attrs = cfg.toLineAttributes()
		for _, attr := range attrs {
			cfgAttrs = cfgAttrs.append(attr, mask)
		}
		if cfg.Direction == LineDirectionOutput {
			outputMask |= mask
		} else {
			outputMask &^= mask
		}
	}
	var defFlags uapi.LineFlagV2
	defFlags.Decode(cfgAttrs[0].Attr)
	// replace default flags in slot 0 with outputValues
	cfgAttrs[0].Attr = lco.outputValues().Encode()
	cfgAttrs[0].Mask = outputMask

	// filter mask==0 entries
	loopAttrs := cfgAttrs
	cfgAttrs = cfgAttrs[:0]
	for _, attr := range loopAttrs {
		if attr.Mask != 0 {
			cfgAttrs = append(cfgAttrs, attr)
		}
	}

	if len(cfgAttrs) > 10 {
		err = ErrConfigOverflow
		return
	}

	ulc.Flags = defFlags
	ulc.NumAttrs = uint32(len(cfgAttrs))
	copy(ulc.Attrs[:], cfgAttrs)
	return
}

// EventHandler is a receiver for line events.
type EventHandler func(LineEvent)

// AsIsOption indicates the line direction should be left as is.
type AsIsOption int

// AsIs indicates that a line be requested as neither an input or output.
//
// That is its direction is left as is. This option overrides and clears any
// previous Input or Output options.
const AsIs = AsIsOption(0)

func (o AsIsOption) applyLineReqOption(l *lineReqOptions) {
	l.defCfg.Direction = LineDirectionUnknown
}

// InputOption indicates the line direction should be set to an input.
type InputOption int

// AsInput indicates that a line be requested as an input.
//
// This option overrides and clears any previous Output, OpenDrain, or
// OpenSource options.
const AsInput = InputOption(0)

func (o InputOption) applyLineConfig(lc *LineConfig) {
	lc.Direction = LineDirectionInput
	lc.Drive = LineDrivePushPull
}

func (o InputOption) applyChipOption(c *ChipOptions) {
	c.config.Direction = LineDirectionInput
}

func (o InputOption) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfigOption(&lro.lineConfigOptions)
}

func (o InputOption) applyLineConfigOption(lco *lineConfigOptions) {
	o.applyLineConfig(&lco.defCfg)
}

func (o InputOption) applySubsetLineConfigOption(offsets []int, l *lineConfigOptions) {
	for _, offset := range offsets {
		o.applyLineConfig(l.lineConfig(offset))
	}
}

// OutputOption indicates the line direction should be set to an output.
type OutputOption []int

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
	return OutputOption(vv)
}

func (o OutputOption) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfigOption(&lro.lineConfigOptions)
}

func (o OutputOption) applyLineConfig(lc *LineConfig) {
	lc.Direction = LineDirectionOutput
	lc.Debounced = false
	lc.DebouncePeriod = 0
	lc.EdgeDetection = LineEdgeNone
}

func (o OutputOption) applyLineConfigOption(lco *lineConfigOptions) {
	o.applyLineConfig(&lco.defCfg)
	for idx, value := range o {
		lco.values[lco.offsets[idx]] = value
	}
}

func (o OutputOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for idx, offset := range offsets {
		o.applyLineConfig(lco.lineConfig(offset))
		lco.values[offset] = o[idx]
	}
}

// LevelOption determines the line level that is considered active.
type LevelOption bool

func (o LevelOption) applyChipOption(c *ChipOptions) {
	c.config.ActiveLow = bool(o)
}

func (o LevelOption) applyLineReqOption(lro *lineReqOptions) {
	lro.defCfg.ActiveLow = bool(o)
}

func (o LevelOption) applyLineConfigOption(lco *lineConfigOptions) {
	lco.defCfg.ActiveLow = bool(o)
}

func (o LevelOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		lco.lineConfig(offset).ActiveLow = bool(o)
	}
}

// AsActiveLow indicates that a line be considered active when the line level
// is low.
const AsActiveLow = LevelOption(true)

// AsActiveHigh indicates that a line be considered active when the line level
// is high.
//
// This is the default active level.
const AsActiveHigh = LevelOption(false)

func (o LineDrive) applyLineConfig(lc *LineConfig) {
	lc.Drive = o
	lc.Direction = LineDirectionOutput
	lc.Debounced = false
	lc.DebouncePeriod = 0
	lc.EdgeDetection = LineEdgeNone
}

func (o LineDrive) applyChipOption(c *ChipOptions) {
	o.applyLineConfig(&c.config)
}

func (o LineDrive) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfig(&lro.defCfg)
}

func (o LineDrive) applyLineConfigOption(lco *lineConfigOptions) {
	o.applyLineConfig(&lco.defCfg)
}

func (o LineDrive) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		o.applyLineConfig(lco.lineConfig(offset))
	}
}

// AsOpenDrain indicates that a line be driven low but left floating for high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenSource, or Debounce options.
const AsOpenDrain = LineDriveOpenDrain

// AsOpenSource indicates that a line be driven high but left floating for low.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenDrain, or Debounce options.
const AsOpenSource = LineDriveOpenSource

// AsPushPull indicates that a line be driven both low and high.
//
// This option sets the Output option and overrides and clears any previous
// Input, RisingEdge, FallingEdge, BothEdges, OpenDrain, OpenSource or Debounce
// options.
const AsPushPull = LineDrivePushPull

func (o LineBias) applyChipOption(c *ChipOptions) {
	c.config.Bias = o
}

func (o LineBias) applyLineReqOption(lro *lineReqOptions) {
	lro.defCfg.Bias = o
}

func (o LineBias) applyLineConfigOption(lco *lineConfigOptions) {
	lco.defCfg.Bias = o
}

func (o LineBias) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		lco.lineConfig(offset).Bias = o
	}
}

// WithBiasAsIs indicates that a line have its internal bias left unchanged.
//
// This option corresponds to the default bias configuration and its only useful
// application is to clear any previous bias option in a chain of LineOptions,
// before that configuration is applied.
//
// Requires Linux v5.5 or later.
const WithBiasAsIs = LineBiasUnknown

// WithBiasDisabled indicates that a line have its internal bias disabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
const WithBiasDisabled = LineBiasDisabled

// WithPullDown indicates that a line have its internal pull-down enabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
const WithPullDown = LineBiasPullDown

// WithPullUp indicates that a line have its internal pull-up enabled.
//
// This option overrides and clears any previous bias options.
//
// Requires Linux v5.5 or later.
const WithPullUp = LineBiasPullUp

func (o EventHandler) applyChipOption(c *ChipOptions) {
	c.eh = o
}

func (o EventHandler) applyLineReqOption(lro *lineReqOptions) {
	lro.eh = o
}

// WithEventHandler indicates that a line will generate events when its active
// state transitions from high to low.
//
// Events are forwarded to the provided handler function.
//
// To maintain event ordering, the event handler is called serially for each
// event from the requested lines.  To minimize the possiblity of overflowing
// the queue of events in the kernel, the event handler should handle or
// hand-off the event and return as soon as possible.
//
// Note that calling Close on the requested line from within the event handler
// will result in deadlock, as the Close waits for the event handler to
// return.  Therefore the Close must be called from a different goroutine.
func WithEventHandler(e EventHandler) EventHandler {
	return e
}

func (o LineEdge) applyLineConfig(lc *LineConfig) {
	lc.EdgeDetection = o
	lc.Direction = LineDirectionInput
}

func (o LineEdge) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfig(&lro.defCfg)
}

func (o LineEdge) applyLineConfigOption(lco *lineConfigOptions) {
	o.applyLineConfig(&lco.defCfg)
}

func (o LineEdge) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		o.applyLineConfig(lco.lineConfig(offset))
	}
}

// WithFallingEdge indicates that a line will generate events when its active
// state transitions from high to low.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
const WithFallingEdge = LineEdgeFalling

// WithRisingEdge indicates that a line will generate events when its active
// state transitions from low to high.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
const WithRisingEdge = LineEdgeRising

// WithBothEdges indicates that a line will generate events when its active
// state transitions from low to high and from high to low.
//
// Events are forwarded to the provided handler function.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
const WithBothEdges = LineEdgeBoth

// WithoutEdges indicates that a line will not generate events due to active
// state transitions.
//
// This is the default for line requests, but allows the removal of edge
// detection by reconfigure.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
//
// The WithoutEdges option requires Linux v5.10 or later.
const WithoutEdges = LineEdgeNone

func (o LineEventClock) applyChipOption(c *ChipOptions) {
	c.config.EventClock = LineEventClock(o)
}

func (o LineEventClock) applyLineReqOption(lro *lineReqOptions) {
	lro.defCfg.EventClock = LineEventClock(o)
}

func (o LineEventClock) applyLineConfigOption(lco *lineConfigOptions) {
	lco.defCfg.EventClock = LineEventClock(o)
}

func (o LineEventClock) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		lco.lineConfig(offset).EventClock = LineEventClock(o)
	}
}

// WithMonotonicEventClock specifies that the edge event timestamps are sourced
// from CLOCK_MONOTONIC.
//
// This option corresponds to the default event clock configuration and its only
// useful application is to clear any previous event clock option in a chain of
// LineOptions, before that configuration is applied.
const WithMonotonicEventClock = LineEventClockMonotonic

// WithRealtimeEventClock specifies that the edge event timestamps are sourced
// from CLOCK_REALTIME.
//
// Requires Linux v5.11 or later.
const WithRealtimeEventClock = LineEventClockRealtime

// DebounceOption indicates that a line will be debounced.
//
// The DebounceOption requires Linux v5.10 or later.
type DebounceOption time.Duration

func (o DebounceOption) applyLineConfig(lc *LineConfig) {
	lc.Direction = LineDirectionInput
	lc.Debounced = true
	lc.DebouncePeriod = time.Duration(o)
}

func (o DebounceOption) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfig(&lro.defCfg)
}

func (o DebounceOption) applyLineConfigOption(lco *lineConfigOptions) {
	o.applyLineConfig(&lco.defCfg)
}

func (o DebounceOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		o.applyLineConfig(lco.lineConfig(offset))
	}
}

// WithDebounce indicates that a line will be debounced with the specified
// debounce period.
//
// This option sets the Input option and overrides and clears any previous
// Output, OpenDrain, or OpenSource options.
//
// Requires Linux v5.10 or later.
func WithDebounce(period time.Duration) DebounceOption {
	return DebounceOption(period)
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

// LinesOption specifies line options that are to be applied to a subset of
// the lines in a request.
type LinesOption struct {
	offsets []int
	options []SubsetLineConfigOption
}

func (o LinesOption) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfigOption(&lro.lineConfigOptions)
}

func (o LinesOption) applyLineConfigOption(lco *lineConfigOptions) {
	for _, option := range o.options {
		option.applySubsetLineConfigOption(o.offsets, lco)
	}
}

// WithLines specifies line options to be applied to a subset of the lines in a
// request.
//
// The offsets should be a strict subset of the offsets provided to
// RequestLines().
// Any offsets outside that set are ignored.
func WithLines(offsets []int, options ...SubsetLineConfigOption) LinesOption {
	return LinesOption{offsets, options}
}

// DefaultedOption resets the configuration to default values.
type DefaultedOption int

func (o DefaultedOption) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfigOption(&lro.lineConfigOptions)
}

func (o DefaultedOption) applyLineConfigOption(lco *lineConfigOptions) {
	lco.defCfg = LineConfig{}
	lco.values = map[int]int{}
}

func (o DefaultedOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	if len(offsets) == 0 {
		lco.lineCfg = nil
	}
	for _, offset := range offsets {
		delete(lco.values, offset)
		delete(lco.lineCfg, offset)
	}
}

// Defaulted resets all configuration options to default values.
//
// This option provides the means to simply reset all configuration options to
// their default values.  This is rarely necessary but is made available for
// completeness.
//
// When applied within WithLines() it resets the configuration of the lines to
// the default for the request, effectively clearing all previous WithLines()
// options for the specified offsets.  If no offsets are specified then the
// configurarion for all offsets is reset to the request default.
//
// When applied outside WithLines() it resets the default configuration for the
// request itself to default values but leaves any configuration set within
// WithLines() unchanged.
const Defaulted = DefaultedOption(0)

// EventBufferSizeOption provides a suggested minimum number of events the
// kernel will buffer for the line request.
//
// The EventBufferSizeOption requires Linux v5.10 or later.
type EventBufferSizeOption int

func (o EventBufferSizeOption) applyLineReqOption(lro *lineReqOptions) {
	lro.eventBufferSize = int(o)
}

// WithEventBufferSize suggests a minimum number of events the kernel will
// buffer for the line request.
//
// Note that the value is only a suggestion, and the kernel may set higher
// values or place a cap on the buffer size.
//
// A zero value (the default) indicates that the kernel should use its default
// buffer size (the number of requested lines * 16).
//
// Requires Linux v5.10 or later.
func WithEventBufferSize(size int) EventBufferSizeOption {
	return EventBufferSizeOption(size)
}
