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
	consumer string
	abi      int
	eh       EventHandler
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
		lc = &LineConfig{}
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

func (o InputOption) applySubsetLineConfigOption(offsets []int, l *lineConfigOptions) {
	for _, offset := range offsets {
		l.lineConfig(offset).Direction = LineDirectionInput
	}
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
	for idx, value := range o.values {
		lco.values[lco.offsets[idx]] = value
	}
}

func (o OutputOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for idx, offset := range offsets {
		o.applyLineConfig(lco.lineConfig(offset))
		lco.values[offset] = o.values[idx]
	}
}

// LevelOption determines the line level that is considered active.
type LevelOption struct {
	activeLow bool
}

func (o LevelOption) applyChipOption(c *ChipOptions) {
	c.config.ActiveLow = o.activeLow
}

func (o LevelOption) applyLineReqOption(lro *lineReqOptions) {
	lro.defCfg.ActiveLow = o.activeLow
}

func (o LevelOption) applyLineConfigOption(lco *lineConfigOptions) {
	lco.defCfg.ActiveLow = o.activeLow
}

func (o LevelOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		lco.lineConfig(offset).ActiveLow = o.activeLow
	}
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

func (o DriveOption) applyLineConfig(lc *LineConfig) {
	lc.Drive = o.drive
	lc.Direction = LineDirectionOutput
	lc.Debounced = false
	lc.DebouncePeriod = 0
}

func (o DriveOption) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfig(&lro.defCfg)
}

func (o DriveOption) applyLineConfigOption(lco *lineConfigOptions) {
	o.applyLineConfig(&lco.defCfg)
}

func (o DriveOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		o.applyLineConfig(lco.lineConfig(offset))
	}
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

func (o BiasOption) applyLineReqOption(lro *lineReqOptions) {
	lro.defCfg.Bias = o.bias
}

func (o BiasOption) applyLineConfigOption(lco *lineConfigOptions) {
	lco.defCfg.Bias = o.bias
}

func (o BiasOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
	for _, offset := range offsets {
		lco.lineConfig(offset).Bias = o.bias
	}
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

func (o EventHandlerOption) applyChipOption(c *ChipOptions) {
	c.eh = o.eh
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

func (o EdgeOption) applyLineConfig(lc *LineConfig) {
	lc.EdgeDetection = o.edge
	lc.Direction = LineDirectionInput
}

func (o EdgeOption) applyLineReqOption(lro *lineReqOptions) {
	o.applyLineConfig(&lro.defCfg)
}

func (o EdgeOption) applyLineConfigOption(lco *lineConfigOptions) {
	o.applyLineConfig(&lco.defCfg)
}

func (o EdgeOption) applySubsetLineConfigOption(offsets []int, lco *lineConfigOptions) {
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

func (o DebounceOption) applyLineConfig(lc *LineConfig) {
	lc.Direction = LineDirectionInput
	lc.Debounced = true
	lc.DebouncePeriod = o.period
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
