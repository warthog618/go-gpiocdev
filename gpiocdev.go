// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// Package gpiocdev is a library for accessing GPIO pins/lines on Linux platforms
// using the GPIO character device.
//
// This is a Go equivalent of libgpiod.
//
// Supports:
//   - Line direction (input/output)
//   - Line write (active/inactive)
//   - Line read (active/inactive)
//   - Line bias (pull-up/pull-down/disabled)
//   - Line drive (push-pull/open-drain/open-source)
//   - Line level (active-high/active-low)
//   - Line edge detection (rising/falling/both)
//   - Line labels
//   - Collections of lines for near simultaneous reads and writes on multiple lines
//
// Example of use:
//
//	v := 0
//	l, err := gpiocdev.RequestLine("gpiochip0", 4, gpiocdev.AsOutput(v))
//	if err != nil {
//		panic(err)
//	}
//	for {
//		<-time.After(time.Second)
//		v ^= 1
//		l.SetValue(v)
//	}
package gpiocdev

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/warthog618/go-gpiocdev/uapi"
	"golang.org/x/sys/unix"
)

// Chip represents a single GPIO chip that controls a set of lines.
type Chip struct {
	f *os.File
	// The system name for this chip.
	Name string

	// A more individual label for the chip.
	Label string

	// The number of GPIO lines on this chip.
	lines int

	// default options for reserved lines.
	options ChipOptions

	// mutex covers the attributes below it.
	mu sync.Mutex

	// watcher for line info changes
	iw *infoWatcher

	// handlers for info changes in watched lines, keyed by offset.
	ich map[int]InfoChangeHandler

	// indicates the chip has been closed.
	closed bool
}

// LineConfig contains the configuration parameters for the line.
type LineConfig struct {
	// A flag indicating if the line is active low.
	ActiveLow bool

	// The line direction.
	Direction LineDirection

	// The line drive.
	Drive LineDrive

	// The line bias.
	Bias LineBias

	// The line edge detection.
	EdgeDetection LineEdge

	// A flag indicating if the line is debounced.
	Debounced bool

	// The line debounce period.
	DebouncePeriod time.Duration

	// The source clock for events on the line.
	EventClock LineEventClock
}

// LineDirection indicates the direction of a line.
type LineDirection int

const (
	// LineDirectionUnknown indicate the line direction is unknown.
	LineDirectionUnknown LineDirection = iota

	// LineDirectionInput indicates the line is an input.
	LineDirectionInput

	// LineDirectionOutput indicates the line is an output.
	LineDirectionOutput
)

// LineDrive indicates the drive of an output line.
type LineDrive int

const (
	// LineDrivePushPull indicates the line is driven in both directions.
	LineDrivePushPull LineDrive = iota

	// LineDriveOpenDrain indicates the line is an open drain output.
	LineDriveOpenDrain

	// LineDriveOpenSource indicates the line is an open source output.
	LineDriveOpenSource
)

// LineBias indicates the bias applied to a line.
type LineBias int

const (
	// LineBiasUnknown indicates the line bias is unknown.
	LineBiasUnknown LineBias = iota

	// LineBiasDisabled indicates the line bias is disabled.
	LineBiasDisabled

	// LineBiasPullUp indicates the line has pull up enabled.
	LineBiasPullUp

	// LineBiasPullDown indicates the line has pull down enabled.
	LineBiasPullDown
)

// LineEdge indicates the edges detected by the line.
type LineEdge int

const (
	// LineEdgeNone indicates the line edge detection is disabled.
	LineEdgeNone LineEdge = iota

	// LineEdgeRising indicates the line has rising edge detection enabled.
	LineEdgeRising

	// LineEdgeFalling indicates the line has falling edge detection enabled.
	LineEdgeFalling

	// LineEdgeBoth indicates the line has both rising and falling edge
	// detection enabled.
	LineEdgeBoth = LineEdgeRising | LineEdgeFalling
)

// LineEventClock indicates the source clock used to timestamp edge events.
type LineEventClock int

const (
	// LineEventClockMonotonic indicates the source clock is CLOCK_MONOTONIC.
	LineEventClockMonotonic LineEventClock = iota

	// LineEventClockRealtime indicates the source clock is CLOCK_REALTIME.
	LineEventClockRealtime
)

// LineInfo contains a summary of publicly available information about the
// line.
type LineInfo struct {
	// The line offset within the chip.
	Offset int

	// The system name for the line.
	Name string

	// A string identifying the requester of the line, if requested.
	Consumer string

	// The line is in use.
	Used bool

	// The configuration parameters for the line.
	Config LineConfig
}

func naturalLess(lhs, rhs string) bool {
	llhs := len(lhs)
	lrhs := len(rhs)
	if llhs == lrhs {
		return lhs < rhs
	}
	if llhs < lrhs {
		return true
	}
	return false
}

// Chips returns the names of the available GPIO devices.
func Chips() []string {
	ee, err := os.ReadDir("/dev") // returns chips in ascii order
	if err != nil {
		return nil
	}
	cc := []string(nil)
	for _, e := range ee {
		name := e.Name()
		if strings.HasPrefix(name, "gpiochip") {
			cc = append(cc, name)
		}
	}
	// sort in numeric order
	sort.Slice(cc, func(i, j int) bool { return naturalLess(cc[i], cc[j]) })
	return cc
}

// RequestLine requests control of a single line on a chip.
//
// If granted, control is maintained until the Line is closed.
func RequestLine(chip string, offset int, options ...LineReqOption) (*Line, error) {
	c, err := NewChip(chip)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	return c.RequestLine(offset, options...)
}

// RequestLines requests control of a collection of lines on a chip.
//
// If granted, control is maintained until the Lines are closed.
func RequestLines(chip string, offsets []int, options ...LineReqOption) (*Lines, error) {
	c, err := NewChip(chip)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	return c.RequestLines(offsets, options...)
}

// NewChip opens a GPIO character device.
func NewChip(name string, options ...ChipOption) (*Chip, error) {
	path := nameToPath(name)
	err := IsChip(path)
	if err != nil {
		return nil, err
	}
	co := ChipOptions{
		consumer: "gpiocdev-" + strconv.Itoa(os.Getpid()),
	}
	for _, option := range options {
		option.applyChipOption(&co)
	}
	f, err := os.OpenFile(path, unix.O_CLOEXEC, unix.O_RDONLY)
	if err != nil {
		// only happens if device removed/locked since IsChip call.
		return nil, err
	}
	ci, err := uapi.GetChipInfo(f.Fd())
	if err != nil {
		// only occurs if IsChip was wrong?
		f.Close()
		return nil, err
	}
	c := Chip{
		f:       f,
		Name:    uapi.BytesToString(ci.Name[:]),
		Label:   uapi.BytesToString(ci.Label[:]),
		lines:   int(ci.Lines),
		options: co,
	}
	if c.options.abi == 0 {
		// probe v2 - should only throw an error if v2 is not supported.
		if _, err = c.LineInfo(0); err == nil {
			c.options.abi = 2
		} else {
			c.options.abi = 1
		}
	}
	if len(c.Label) == 0 {
		c.Label = "unknown"
	}
	return &c, nil
}

// Close releases the Chip.
//
// It does not release any lines which may be requested - they must be closed
// independently.
func (c *Chip) Close() error {
	c.mu.Lock()
	closed := c.closed
	c.closed = true
	c.mu.Unlock()
	if closed {
		return ErrClosed
	}
	if c.iw != nil {
		c.iw.close()
	}
	return c.f.Close()
}

// FindLine returns the offset of the named line, if found on the chip.
//
// If multiple lines have the same name then the lowest matching offset is returned.
func (c *Chip) FindLine(name string) (offset int, err error) {
	for o := 0; o < c.lines; o++ {
		inf, err := c.LineInfo(o)
		if err == nil && inf.Name == name {
			return o, nil
		}
	}
	return 0, ErrNotFound
}

// LineInfo returns the publicly available information on the line.
//
// This is always available and does not require requesting the line.
func (c *Chip) LineInfo(offset int) (info LineInfo, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		err = ErrClosed
		return
	}
	if offset < 0 || offset >= c.lines {
		err = ErrInvalidOffset
		return
	}
	if c.options.abi == 1 {
		var li uapi.LineInfo
		li, err = uapi.GetLineInfo(c.f.Fd(), offset)
		if err == nil {
			info = newLineInfo(li)
		}
		return
	}
	var li uapi.LineInfoV2
	li, err = uapi.GetLineInfoV2(c.f.Fd(), offset)
	if err == nil {
		info = newLineInfoV2(li)
	}
	return
}

func lineInfoToLineConfig(li uapi.LineInfo) LineConfig {
	lc := LineConfig{}
	lc.ActiveLow = li.Flags.IsActiveLow()

	if li.Flags.IsOut() {
		lc.Direction = LineDirectionOutput
		if li.Flags.IsOpenDrain() {
			lc.Drive = LineDriveOpenDrain
		} else if li.Flags.IsOpenSource() {
			lc.Drive = LineDriveOpenSource
		}
	} else {
		lc.Direction = LineDirectionInput
	}

	if li.Flags.IsPullUp() {
		lc.Bias = LineBiasPullUp
	} else if li.Flags.IsPullDown() {
		lc.Bias = LineBiasPullDown
	} else if li.Flags.IsBiasDisable() {
		lc.Bias = LineBiasDisabled
	}
	return lc
}

func lineInfoV2ToLineConfig(li uapi.LineInfoV2) LineConfig {
	lc := LineConfig{}
	lc.ActiveLow = li.Flags.IsActiveLow()

	if li.Flags.IsOutput() {
		lc.Direction = LineDirectionOutput
		if li.Flags.IsOpenDrain() {
			lc.Drive = LineDriveOpenDrain
		} else if li.Flags.IsOpenSource() {
			lc.Drive = LineDriveOpenSource
		}
	} else {
		lc.Direction = LineDirectionInput
	}

	if li.Flags.IsBothEdges() {
		lc.EdgeDetection = LineEdgeBoth
	} else if li.Flags.IsRisingEdge() {
		lc.EdgeDetection = LineEdgeRising
	} else if li.Flags.IsFallingEdge() {
		lc.EdgeDetection = LineEdgeFalling
	}

	if li.Flags.IsBiasPullUp() {
		lc.Bias = LineBiasPullUp
	} else if li.Flags.IsBiasPullDown() {
		lc.Bias = LineBiasPullDown
	} else if li.Flags.IsBiasDisabled() {
		lc.Bias = LineBiasDisabled
	}

	for i := 0; i < int(li.NumAttrs); i++ {
		if li.Attrs[i].ID == uapi.LineAttributeIDDebounce {
			lc.Debounced = true
			lc.DebouncePeriod = time.Duration(li.Attrs[i].Value32()) * time.Microsecond
		}
	}
	return lc
}

func newLineInfo(li uapi.LineInfo) LineInfo {
	return LineInfo{
		Offset:   int(li.Offset),
		Name:     uapi.BytesToString(li.Name[:]),
		Consumer: uapi.BytesToString(li.Consumer[:]),
		Used:     li.Flags.IsUsed(),
		Config:   lineInfoToLineConfig(li),
	}
}

func newLineInfoV2(li uapi.LineInfoV2) LineInfo {
	return LineInfo{
		Offset:   int(li.Offset),
		Name:     uapi.BytesToString(li.Name[:]),
		Consumer: uapi.BytesToString(li.Consumer[:]),
		Used:     li.Flags.IsUsed(),
		Config:   lineInfoV2ToLineConfig(li),
	}
}

// Lines returns the number of lines that exist on the GPIO chip.
func (c *Chip) Lines() int {
	return c.lines
}

// RequestLine requests control of a single line on the chip.
//
// If granted, control is maintained until the Line is closed.
func (c *Chip) RequestLine(offset int, options ...LineReqOption) (*Line, error) {
	ll, err := c.RequestLines([]int{offset}, options...)
	if err != nil {
		return nil, err
	}
	l := Line{
		baseLine: baseLine{
			offsets: ll.offsets,
			values:  ll.values,
			vfd:     ll.vfd,
			isEvent: ll.isEvent,
			chip:    ll.chip,
			abi:     ll.abi,
			defCfg:  ll.defCfg,
			watcher: ll.watcher,
		},
	}
	return &l, nil
}

// RequestLines requests control of a collection of lines on the chip.
//
// If granted, control is maintained until the Lines are closed.
func (c *Chip) RequestLines(offsets []int, options ...LineReqOption) (*Lines, error) {
	for _, o := range offsets {
		if o < 0 || o >= c.lines {
			return nil, ErrInvalidOffset
		}
	}
	offsets = append([]int(nil), offsets...)
	lro := lineReqOptions{
		lineConfigOptions: lineConfigOptions{
			offsets: offsets,
			values:  map[int]int{},
			defCfg:  c.options.config,
		},
		consumer: c.options.consumer,
		abi:      c.options.abi,
		eh:       c.options.eh,
	}
	for _, option := range options {
		option.applyLineReqOption(&lro)
	}
	ll := Lines{
		baseLine: baseLine{
			offsets: offsets,
			values:  lro.values,
			chip:    c.Name,
			abi:     lro.abi,
			defCfg:  lro.defCfg,
		},
	}
	var err error
	if ll.abi == 2 {
		ll.vfd, ll.watcher, err = c.getLine(ll.offsets, lro)
	} else {
		err = lro.defCfg.v1Validate()
		if err != nil {
			return nil, err
		}
		if lro.eh == nil {
			ll.vfd, err = c.getHandleRequest(ll.offsets, lro)
		} else {
			ll.isEvent = true
			ll.vfd, ll.watcher, err = c.getEventRequest(ll.offsets, lro)
		}
	}
	if err != nil {
		return nil, err
	}
	return &ll, nil
}

// creates the iw and ich
//
// Assumes c is locked.
func (c *Chip) createInfoWatcher() error {
	iw, err := newInfoWatcher(int(c.f.Fd()),
		func(lic LineInfoChangeEvent) {
			c.mu.Lock()
			ich := c.ich[lic.Info.Offset]
			c.mu.Unlock() // handler called outside lock
			if ich != nil {
				ich(lic)
			}
		},
		c.options.abi)
	if err != nil {
		return err
	}
	c.iw = iw
	c.ich = map[int]InfoChangeHandler{}
	return nil
}

// WatchLineInfo enables watching changes to line info for the specified lines.
//
// The changes are reported via the chip InfoChangeHandler.
// Repeated calls replace the InfoChangeHandler.
//
// Requires Linux 5.7 or later.
func (c *Chip) WatchLineInfo(offset int, lich InfoChangeHandler) (info LineInfo, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		err = ErrClosed
		return
	}
	if c.iw == nil {
		err = c.createInfoWatcher()
		if err != nil {
			return
		}
	}
	if c.options.abi == 1 {
		li := uapi.LineInfo{Offset: uint32(offset)}
		err = uapi.WatchLineInfo(c.f.Fd(), &li)
		if err != nil {
			return
		}
		c.ich[offset] = lich
		info = newLineInfo(li)
		return
	}
	li := uapi.LineInfoV2{Offset: uint32(offset)}
	err = uapi.WatchLineInfoV2(c.f.Fd(), &li)
	if err != nil {
		return
	}
	c.ich[offset] = lich
	info = newLineInfoV2(li)
	return
}

// UnwatchLineInfo disables watching changes to line info.
//
// Requires Linux 5.7 or later.
func (c *Chip) UnwatchLineInfo(offset int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	delete(c.ich, offset)
	return uapi.UnwatchLineInfo(c.f.Fd(), uint32(offset))
}

func (c *Chip) getLine(offsets []int, lro lineReqOptions) (uintptr, io.Closer, error) {

	config, err := lro.toULineConfig()
	if err != nil {
		return 0, nil, err
	}
	lr := uapi.LineRequest{
		Lines:  uint32(len(offsets)),
		Config: config,
	}
	copy(lr.Consumer[:len(lr.Consumer)-1], lro.consumer)
	// copy(hr.Offsets[:], offsets) - with cast
	for i, o := range offsets {
		lr.Offsets[i] = uint32(o)
	}
	err = uapi.GetLine(c.f.Fd(), &lr)
	if err != nil {
		return 0, nil, err
	}
	var w io.Closer
	if lro.eh != nil {
		w, err = newWatcher(lr.Fd, lro.eh)
		if err != nil {
			unix.Close(int(lr.Fd))
			return 0, nil, err
		}
	}
	return uintptr(lr.Fd), w, nil
}

func (lc LineConfig) toHandleFlags() uapi.HandleFlag {
	var flags uapi.HandleFlag

	if lc.ActiveLow {
		flags |= uapi.HandleRequestActiveLow
	}

	switch lc.Direction {
	case LineDirectionOutput:
		flags |= uapi.HandleRequestOutput
	case LineDirectionInput:
		flags |= uapi.HandleRequestInput
	}

	switch lc.Drive {
	case LineDriveOpenDrain:
		flags |= uapi.HandleRequestOpenDrain
	case LineDriveOpenSource:
		flags |= uapi.HandleRequestOpenSource
	}

	switch lc.Bias {
	case LineBiasPullUp:
		flags |= uapi.HandleRequestPullUp
	case LineBiasPullDown:
		flags |= uapi.HandleRequestPullDown
	case LineBiasDisabled:
		flags |= uapi.HandleRequestBiasDisable
	}

	return flags
}

func (lc LineConfig) toEventFlags() uapi.EventFlag {
	switch lc.EdgeDetection {
	case LineEdgeBoth:
		return uapi.EventRequestBothEdges
	case LineEdgeRising:
		return uapi.EventRequestRisingEdge
	case LineEdgeFalling:
		return uapi.EventRequestFallingEdge
	default:
		return 0
	}
}

func (lc LineConfig) toLineFlagV2() (flags uapi.LineFlagV2) {
	if lc.ActiveLow {
		flags |= uapi.LineFlagV2ActiveLow
	}
	switch lc.Direction {
	case LineDirectionOutput:
		flags |= uapi.LineFlagV2Output
		switch lc.Drive {
		case LineDriveOpenDrain:
			flags |= uapi.LineFlagV2OpenDrain
		case LineDriveOpenSource:
			flags |= uapi.LineFlagV2OpenSource
		}
	case LineDirectionInput:
		flags |= uapi.LineFlagV2Input
		if lc.EdgeDetection&LineEdgeRising != 0 {
			flags |= uapi.LineFlagV2EdgeRising
		}
		if lc.EdgeDetection&LineEdgeFalling != 0 {
			flags |= uapi.LineFlagV2EdgeFalling
		}
		if lc.EventClock == LineEventClockRealtime {
			flags |= uapi.LineFlagV2EventClockRealtime
		}
	}

	switch lc.Bias {
	case LineBiasDisabled:
		flags |= uapi.LineFlagV2BiasDisabled
	case LineBiasPullUp:
		flags |= uapi.LineFlagV2BiasPullUp
	case LineBiasPullDown:
		flags |= uapi.LineFlagV2BiasPullDown
	}
	return
}

func (lc LineConfig) toLineAttributes() (attrs []uapi.LineAttribute) {
	flags := lc.toLineFlagV2()
	attr := uapi.LineAttribute{}
	if flags != 0 {
		attr.Encode64(uapi.LineAttributeIDFlags, uint64(flags))
		attrs = append(attrs, attr)
	}
	if lc.Debounced {
		attr = uapi.DebouncePeriod(lc.DebouncePeriod).Encode()
		attrs = append(attrs, attr)
	}
	return
}

func (lc LineConfig) v1Validate() error {
	if lc.Debounced {
		return ErrUapiIncompatibility{"debounce", 1}
	}
	if lc.EventClock != LineEventClockMonotonic {
		return ErrUapiIncompatibility{"event clock", 1}
	}
	return nil
}

func (c *Chip) getEventRequest(offsets []int, lro lineReqOptions) (uintptr, io.Closer, error) {
	var vfd uintptr
	fds := make(map[int]int)
	for i, o := range offsets {
		er := uapi.EventRequest{
			Offset:      uint32(o),
			HandleFlags: lro.defCfg.toHandleFlags(),
			EventFlags:  lro.defCfg.toEventFlags(),
		}
		copy(er.Consumer[:len(er.Consumer)-1], lro.consumer)
		err := uapi.GetLineEvent(c.f.Fd(), &er)
		if err != nil {
			return 0, nil, err
		}
		fd := uintptr(er.Fd)
		if i == 0 {
			vfd = fd
		}
		fds[int(fd)] = o
	}
	w, err := newWatcherV1(fds, lro.eh)
	if err != nil {
		for fd := range fds {
			unix.Close(fd)
		}
		return 0, nil, err
	}
	return vfd, w, nil
}

func (c *Chip) getHandleRequest(offsets []int, lro lineReqOptions) (uintptr, error) {
	hr := uapi.HandleRequest{
		Lines: uint32(len(offsets)),
		Flags: lro.defCfg.toHandleFlags(),
	}
	copy(hr.Consumer[:len(hr.Consumer)-1], lro.consumer)
	// copy(hr.Offsets[:], offsets) - with cast
	for i, o := range offsets {
		hr.Offsets[i] = uint32(o)
	}
	for idx, offset := range lro.offsets {
		hr.DefaultValues[idx] = uint8(lro.values[offset])
	}
	err := uapi.GetLineHandle(c.f.Fd(), &hr)
	if err != nil {
		return 0, err
	}
	return uintptr(hr.Fd), nil
}

// UapiAbiVersion returns the version of the GPIO uAPI the chip is using.
func (c *Chip) UapiAbiVersion() int {
	return c.options.abi
}

type baseLine struct {
	offsets []int
	vfd     uintptr
	isEvent bool
	chip    string
	abi     int
	// mu covers all that follow - those above are immutable
	mu      sync.Mutex
	values  map[int]int
	defCfg  LineConfig
	lineCfg map[int]*LineConfig
	info    []*LineInfo
	closed  bool
	watcher io.Closer
}

// UapiAbiVersion returns the version of the GPIO uAPI the line is using.
func (l *baseLine) UapiAbiVersion() int {
	return l.abi
}

// Chip returns the name of the chip from which the line was requested.
func (l *baseLine) Chip() string {
	return l.chip
}

// Close releases all resources held by the requested line.
//
// Note that this includes waiting for any running event handler to return.
// As a consequence the Close must not be called from the context of the event
// handler - the Close should be called from a different goroutine.
func (l *baseLine) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	l.closed = true
	if l.watcher != nil {
		l.watcher.Close()
	}
	if !l.isEvent { // isEvent => v1 => closed by watcher
		unix.Close(int(l.vfd))
	}
	return nil
}

// Reconfigure updates the configuration of the requested line(s).
//
// Configuration for options other than those passed in remain unchanged.
//
// Not valid for lines with edge detection enabled.
//
// Requires Linux 5.5 or later.
func (l *baseLine) Reconfigure(options ...LineConfigOption) error {
	if l.isEvent {
		return unix.EINVAL
	}
	if len(options) == 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	lro := lineReqOptions{
		lineConfigOptions: lineConfigOptions{
			offsets: l.offsets,
			values:  l.values,
			defCfg:  l.defCfg,
			lineCfg: l.lineCfg,
		},
	}
	for _, option := range options {
		option.applyLineConfigOption(&lro.lineConfigOptions)
	}
	if l.abi == 1 {
		err := lro.defCfg.v1Validate()
		if err != nil {
			return err
		}
		hc := uapi.HandleConfig{Flags: lro.defCfg.toHandleFlags()}
		for idx, offset := range lro.offsets {
			hc.DefaultValues[idx] = uint8(lro.values[offset])
		}
		err = uapi.SetLineConfig(l.vfd, &hc)
		if err == nil {
			l.defCfg = lro.defCfg
		}
		return err
	}
	config, err := lro.toULineConfig()
	if err != nil {
		return err
	}
	err = uapi.SetLineConfigV2(l.vfd, &config)
	if err == nil {
		l.defCfg = lro.defCfg
		l.lineCfg = lro.lineCfg
	}
	return err
}

// Line represents a single requested line.
type Line struct {
	baseLine
}

// Offset returns the offset of the line within the chip.
func (l *Line) Offset() int {
	return l.offsets[0]
}

// Info returns the information about the line.
func (l *Line) Info() (info LineInfo, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		err = ErrClosed
		return
	}
	if l.info != nil {
		info = *l.info[0]
		return
	}
	c, err := NewChip(l.chip, WithABIVersion(l.abi))
	if err != nil {
		return
	}
	defer c.Close()
	inf, err := c.LineInfo(l.offsets[0])
	if err != nil {
		return
	}
	l.info = []*LineInfo{&inf}
	info = *l.info[0]
	return
}

// Value returns the current value (active state) of the line.
func (l *Line) Value() (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return 0, ErrClosed
	}
	if l.abi == 1 {
		hd := uapi.HandleData{}
		err := uapi.GetLineValues(l.vfd, &hd)
		return int(hd[0]), err
	}
	lv := uapi.LineValues{Mask: 1}
	err := uapi.GetLineValuesV2(l.vfd, &lv)
	return lv.Get(0), err
}

// SetValue sets the current value (active state) of the line.
//
// Only valid for output lines.
func (l *Line) SetValue(value int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	info, err := l.Info()
	if err != nil {
		return err
	}
	if info.Config.Direction != LineDirectionOutput {
		return ErrPermissionDenied
	}
	if l.closed {
		return ErrClosed
	}
	if l.abi == 1 {
		hd := uapi.HandleData{}
		hd[0] = uint8(value)
		err := uapi.SetLineValues(l.vfd, hd)
		if err == nil {
			l.values[l.offsets[0]] = value
		}
		return err
	}
	lsv := uapi.LineValues{
		Mask: 1,
		Bits: uapi.NewLineBitmap(value),
	}
	err = uapi.SetLineValuesV2(l.vfd, lsv)
	if err == nil {
		l.values[l.offsets[0]] = value
	}
	return err
}

// Lines represents a collection of requested lines.
type Lines struct {
	baseLine
}

// Offsets returns the offsets of the lines within the chip.
func (l *Lines) Offsets() []int {
	return l.offsets
}

// Info returns the information about the lines.
func (l *Lines) Info() ([]*LineInfo, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, ErrClosed
	}
	if l.info != nil {
		return l.info, nil
	}
	c, err := NewChip(l.chip, WithABIVersion(l.abi))
	if err != nil {
		return nil, err
	}
	defer c.Close()
	info := make([]*LineInfo, len(l.offsets))
	for i, o := range l.offsets {
		inf, err := c.LineInfo(o)
		if err != nil {
			return nil, err
		}
		info[i] = &inf
	}
	l.info = info
	return l.info, nil
}

// Values returns the current values (active state) of the collection of lines.
//
// Gets as many values from the set, in order, as can be fit in values, up to
// the full set.
func (l *Lines) Values(values []int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	lines := len(values)
	if lines > len(l.offsets) {
		lines = len(l.offsets)
	}
	if l.abi == 1 {
		hd := uapi.HandleData{}
		err := uapi.GetLineValues(l.vfd, &hd)
		if err != nil {
			return err
		}
		for i := 0; i < lines; i++ {
			values[i] = int(hd[i])
		}
		return nil
	}
	lv := uapi.LineValues{Mask: uapi.NewLineBitMask(lines)}
	err := uapi.GetLineValuesV2(l.vfd, &lv)
	if err != nil {
		return err
	}
	for i := 0; i < lines; i++ {
		values[i] = lv.Get(i)
	}
	return nil
}

// SetValues sets the current active state of the collection of lines.
//
// Only valid for output lines.
//
// All lines in the set are set at once.  If insufficient values are provided
// then the remaining lines are set to inactive. If too many values are provided
// then the surplus values are ignored.
func (l *Lines) SetValues(values []int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range values {
		info, err := l.Info()
		if err != nil {
			return err
		}
		if info[i].Config.Direction != LineDirectionOutput {
			return ErrPermissionDenied
		}
	}
	if l.closed {
		return ErrClosed
	}
	if len(values) > len(l.offsets) {
		values = values[:len(l.offsets)]
	}
	if l.abi == 1 {
		hd := uapi.HandleData{}
		for i, v := range values {
			hd[i] = uint8(v)
		}
		err := uapi.SetLineValues(l.vfd, hd)
		if err == nil {
			for i, v := range values {
				l.values[l.offsets[i]] = v
			}
		}
		return err
	}
	lv := uapi.LineValues{
		Mask: uapi.NewLineBitMask(len(l.offsets)),
		Bits: uapi.NewLineBitmap(values...),
	}
	err := uapi.SetLineValuesV2(l.vfd, lv)
	if err == nil {
		for i, v := range values {
			l.values[l.offsets[i]] = v
		}
	}

	return err
}

// LineEventType indicates the type of change to the line active state.
//
// Note that for active low lines a low line level results in a high active
// state.
type LineEventType int

const (
	_ LineEventType = iota
	// LineEventRisingEdge indicates an inactive to active event.
	LineEventRisingEdge

	// LineEventFallingEdge indicates an active to inactive event.
	LineEventFallingEdge
)

// LineEvent represents a change in the state of a line.
type LineEvent struct {
	// The line offset within the GPIO chip.
	Offset int

	// Timestamp indicates the time the event was detected.
	//
	// The timestamp is intended for accurately measuring intervals between
	// events. It is not guaranteed to be based on a particular clock. It has
	// been based on CLOCK_REALTIME, but from Linux 5.7 it is based on
	// CLOCK_MONOTONIC.
	Timestamp time.Duration

	// The type of state change event this structure represents.
	Type LineEventType

	// The sequence number for this event in all events on all lines in this line request.
	//
	// Requires uAPI v2.
	Seqno uint32

	// The sequence number for this event in all events in this line.
	//
	// Requires uAPI v2.
	LineSeqno uint32
}

// LineInfoChangeEvent represents a change in the info a line.
type LineInfoChangeEvent struct {
	// Info is the updated line info.
	Info LineInfo

	// Timestamp indicates the time the event was detected.
	//
	// The timestamp is intended for accurately measuring intervals between
	// events. It is not guaranteed to be based on a particular clock, but from
	// Linux 5.7 it is based on CLOCK_MONOTONIC.
	Timestamp time.Duration

	// The type of info change event this structure represents.
	Type LineInfoChangeType
}

// LineInfoChangeType indicates the type of change to the line info.
type LineInfoChangeType int

const (
	_ LineInfoChangeType = iota

	// LineRequested indicates the line has been requested.
	LineRequested

	// LineReleased indicates the line has been released.
	LineReleased

	// LineReconfigured indicates the line configuration has changed.
	LineReconfigured
)

// InfoChangeHandler is a receiver for line info change events.
type InfoChangeHandler func(LineInfoChangeEvent)

// FindLine returns the offset of the named line, if found on available chips.
//
// If multiple lines have the same name then the first one found (lowest chip
// and lowest offset) is returned.
func FindLine(name string) (chip string, offset int, err error) {
	var c *Chip
	for _, chip = range Chips() {
		c, err = NewChip(chip)
		if err != nil {
			continue
		}
		offset, err = c.FindLine(name)
		if err == nil {
			return
		}
	}
	return "", 0, ErrNotFound
}

// IsChip checks if the named device is an accessible GPIO character device.
//
// Returns an error if not.
func IsChip(name string) error {
	path := nameToPath(name)
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeCharDevice == 0 {
		return ErrNotCharacterDevice
	}
	sysfspath := fmt.Sprintf("/sys/bus/gpio/devices/%s/dev", fi.Name())
	if err = unix.Access(sysfspath, unix.R_OK); err != nil {
		return ErrNotCharacterDevice
	}
	sysfsf, err := os.Open(sysfspath)
	if err != nil {
		// changed since Access?
		return ErrNotCharacterDevice
	}
	var sysfsdev [16]byte
	n, err := sysfsf.Read(sysfsdev[:])
	sysfsf.Close()
	if err != nil || n <= 0 {
		return ErrNotCharacterDevice
	}
	var stat unix.Stat_t
	if err = unix.Lstat(path, &stat); err != nil {
		return err
	}
	devstr := fmt.Sprintf("%d:%d", unix.Major(uint64(stat.Rdev)), unix.Minor(uint64(stat.Rdev)))
	sysstr := string(sysfsdev[:n-1])
	if devstr != sysstr {
		return ErrNotCharacterDevice
	}
	return nil
}

func nameToPath(name string) string {
	if strings.HasPrefix(name, "/dev/") {
		return name
	}
	return "/dev/" + name
}

var (
	// ErrClosed indicates the chip or line has already been closed.
	ErrClosed = errors.New("already closed")

	// ErrConfigOverflow indicates the provided configuration is too complicated
	// to be mapped to the kernel uAPI.
	//
	// Reduce the number of line options or split the request into multiple
	// requests for smaller sets of lines.
	ErrConfigOverflow = errors.New("configuration too complex to map to kernel uAPI")

	// ErrInvalidOffset indicates a line offset is invalid.
	ErrInvalidOffset = errors.New("invalid offset")

	// ErrNotCharacterDevice indicates the device is not a character device.
	ErrNotCharacterDevice = errors.New("not a character device")

	// ErrNotFound indicates the named line cannot be found.
	ErrNotFound = errors.New("not found")

	// ErrPermissionDenied indicates caller does not have required permissions
	// for the operation.
	ErrPermissionDenied = errors.New("permission denied")
)

// ErrUapiIncompatibility indicates the feature is not supported by the given
// kernel uAPI version.
type ErrUapiIncompatibility struct {
	Feature    string
	AbiVersion int
}

func (e ErrUapiIncompatibility) Error() string {
	return fmt.Sprintf("%s not available in kernel GPIO uAPI v%d", e.Feature, e.AbiVersion)
}
