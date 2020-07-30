// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// Package gpiod is a library for accessing GPIO pins/lines on Linux platforms
// using the GPIO character device.
//
// This is a Go equivalent of libgpiod.
//
// Supports:
// - Line direction (input/output)
// - Line write (active/inactive)
// - Line read (active/inactive)
// - Line bias (pull-up/pull-down/disabled)
// - Line drive (push-pull/open-drain/open-source)
// - Line level (active-high/active-low)
// - Line edge detection (rising/falling/both)
// - Line labels
// - Collections of lines for near simultaneous reads and writes on multiple lines
//
// Example of use:
//
//  c, err := gpiod.NewChip("gpiochip0")
//  if err != nil {
//  	panic(err)
//  }
//  v := 0
//  l, err := c.RequestLine(4, gpiod.AsOutput(v))
//  if err != nil {
//  	panic(err)
//  }
//  for {
//  	<-time.After(time.Second)
//  	v ^= 1
//  	l.SetValue(v)
//  }
//
package gpiod

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/warthog618/gpiod/uapi"
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

// LineInfo contains a summary of publicly available information about the
// line.
type LineInfo struct {
	// The line offset within the chip.
	Offset int

	// The system name for the line.
	Name string

	// A string identifying the requester of the line, if requested.
	Consumer string

	// True if the line is requested.
	Requested bool

	// True if the line was requested as an output.
	IsOut bool

	// True if the line was requested as active low.
	ActiveLow bool

	// True if the line was requested as open drain.
	//
	// Only valid for outputs.
	OpenDrain bool

	// True if the line was requested as open source.
	//
	// Only valid for outputs.
	OpenSource bool

	// True if the line was requested with bias disabled.
	BiasDisable bool

	// True if the line was requested with pull-down.
	PullDown bool

	// True if the line was requested with pull-up.
	PullUp bool
}

// Chips returns the names of the available GPIO devices.
func Chips() []string {
	cc := []string(nil)
	for _, name := range chipNames() {
		if IsChip(name) == nil {
			cc = append(cc, name)
		}
	}
	return cc
}

// FindLine finds the chip and offset of the named line.
//
// Returns an error if the line cannot be found.
func FindLine(lname string) (string, int, error) {
	c, o, err := findLine(lname)
	if err != nil {
		return "", 0, err
	}
	c.Close()
	return c.Name, o, nil
}

// NewChip opens a GPIO character device.
func NewChip(name string, options ...ChipOption) (*Chip, error) {
	path := nameToPath(name)
	err := IsChip(path)
	if err != nil {
		return nil, err
	}
	co := ChipOptions{
		consumer: fmt.Sprintf("gpiod-%d", os.Getpid()),
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

// FindLine returns the offset of the named line, or an error if not found.
func (c *Chip) FindLine(name string) (int, error) {
	for o := 0; o < c.lines; o++ {
		inf, err := c.LineInfo(o)
		if err != nil {
			return 0, err
		}
		if inf.Name == name {
			return o, nil
		}
	}
	return 0, ErrLineNotFound
}

// FindLines returns the offsets of the named lines, or an error unless all are
// found.
func (c *Chip) FindLines(names ...string) (oo []int, err error) {
	ioo := make([]int, len(names))
	for i, name := range names {
		var o int
		o, err = c.FindLine(name)
		if err != nil {
			return
		}
		ioo[i] = o
	}
	oo = ioo
	return
}

// LineInfo returns the publically available information on the line.
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
	var li uapi.LineInfo
	li, err = uapi.GetLineInfo(c.f.Fd(), offset)
	if err == nil {
		info = newLineInfo(li)
	}
	return
}

func newLineInfo(li uapi.LineInfo) LineInfo {
	return LineInfo{
		Offset:      int(li.Offset),
		Name:        uapi.BytesToString(li.Name[:]),
		Consumer:    uapi.BytesToString(li.Consumer[:]),
		Requested:   li.Flags.IsRequested(),
		IsOut:       li.Flags.IsOut(),
		ActiveLow:   li.Flags.IsActiveLow(),
		OpenDrain:   li.Flags.IsOpenDrain(),
		OpenSource:  li.Flags.IsOpenSource(),
		BiasDisable: li.Flags.IsBiasDisable(),
		PullDown:    li.Flags.IsPullDown(),
		PullUp:      li.Flags.IsPullUp(),
	}
}

// Lines returns the number of lines that exist on the GPIO chip.
func (c *Chip) Lines() int {
	return c.lines
}

// RequestLine requests control of a single line on the chip.
//
// If granted, control is maintained until either the Line or Chip are closed.
func (c *Chip) RequestLine(offset int, options ...LineOption) (*Line, error) {
	ll, err := c.RequestLines([]int{offset}, options...)
	if err != nil {
		return nil, err
	}
	l := Line{baseLine{
		offsets:      ll.offsets,
		vfd:          ll.vfd,
		isEvent:      ll.isEvent,
		chip:         c.Name,
		flags:        ll.flags,
		outputValues: ll.outputValues,
		w:            ll.w,
	}}
	return &l, nil
}

// RequestLines requests control of a collection of lines on the chip.
func (c *Chip) RequestLines(offsets []int, options ...LineOption) (*Lines, error) {
	for _, o := range offsets {
		if o < 0 || o >= c.lines {
			return nil, ErrInvalidOffset
		}
	}
	lo := LineOptions{
		consumer:    c.options.consumer,
		HandleFlags: c.options.HandleFlags,
	}
	for _, option := range options {
		option.applyLineOption(&lo)
	}
	ll := Lines{baseLine{
		offsets:      append([]int(nil), offsets...),
		chip:         c.Name,
		flags:        lo.HandleFlags,
		outputValues: lo.InitialValues,
	}}
	var err error
	if lo.eh != nil {
		ll.isEvent = true
		ll.vfd, ll.w, err = c.getEventRequest(ll.offsets, lo)
	} else {
		ll.vfd, err = c.getHandleRequest(ll.offsets, lo)
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
		})
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
// Requires Linux v5.7 or later.
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
	li := uapi.LineInfo{Offset: uint32(offset)}
	err = uapi.WatchLineInfo(c.f.Fd(), &li)
	if err != nil {
		return
	}
	c.ich[offset] = lich
	info = newLineInfo(li)
	return
}

// UnwatchLineInfo disables watching changes to line info.
//
// Requires Linux v5.7 or later.
func (c *Chip) UnwatchLineInfo(offset int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	delete(c.ich, offset)
	return uapi.UnwatchLineInfo(c.f.Fd(), uint32(offset))
}

func (c *Chip) getEventRequest(offsets []int, lo LineOptions) (uintptr, *watcher, error) {
	var vfd uintptr
	fds := make(map[int]int)
	for i, o := range offsets {
		er := uapi.EventRequest{
			Offset:      uint32(o),
			HandleFlags: lo.HandleFlags,
			EventFlags:  lo.EventFlags,
		}
		copy(er.Consumer[:len(er.Consumer)-1], lo.consumer)
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
	w, err := newWatcher(fds, lo.eh)
	if err != nil {
		for fd := range fds {
			unix.Close(fd)
		}
		return 0, nil, err
	}
	return vfd, w, nil
}

func (c *Chip) getHandleRequest(offsets []int, lo LineOptions) (uintptr, error) {
	hr := uapi.HandleRequest{
		Lines: uint32(len(offsets)),
		Flags: lo.HandleFlags,
	}
	copy(hr.Consumer[:len(hr.Consumer)-1], lo.consumer)
	// copy(hr.Offsets[:], offsets) - with cast
	for i, o := range offsets {
		hr.Offsets[i] = uint32(o)
	}
	// copy(hr.DefaultValues[:], lo.InitialValues) - with cast
	for i, v := range lo.InitialValues {
		hr.DefaultValues[i] = uint8(v)
	}
	err := uapi.GetLineHandle(c.f.Fd(), &hr)
	if err != nil {
		return 0, err
	}
	return uintptr(hr.Fd), nil
}

type baseLine struct {
	offsets []int
	vfd     uintptr
	isEvent bool
	chip    string
	// mu covers all that follow - those above are immutable
	mu           sync.Mutex
	flags        uapi.HandleFlag
	outputValues []int
	info         []*LineInfo
	closed       bool
	w            *watcher
}

// Chip returns the name of the chip from which the line was requested.
func (l *baseLine) Chip() string {
	return l.chip
}

// Close releases all resources held by the requested line.
func (l *baseLine) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	l.closed = true
	if l.w != nil {
		l.w.close()
	} else {
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
// Requires Linux v5.5 or later.
func (l *baseLine) Reconfigure(options ...LineConfig) error {
	if l.isEvent {
		return ErrPermissionDenied
	}
	if len(options) == 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return ErrClosed
	}
	lo := LineOptions{
		HandleFlags:   l.flags,
		InitialValues: l.outputValues,
	}
	for _, option := range options {
		option.applyLineConfig(&lo)
	}
	hc := uapi.HandleConfig{Flags: lo.HandleFlags}
	for i, v := range lo.InitialValues {
		hc.DefaultValues[i] = uint8(v)
	}
	err := uapi.SetLineConfig(l.vfd, &hc)
	if err == nil {
		l.flags = hc.Flags
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
	c, err := NewChip(l.chip)
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
	var values uapi.HandleData
	err := uapi.GetLineValues(l.vfd, &values)
	return int(values[0]), err
}

// SetValue sets the current active state of the line.
//
// Only valid for output lines.
func (l *Line) SetValue(value int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.flags.IsOutput() {
		return ErrPermissionDenied
	}
	if l.closed {
		return ErrClosed
	}
	l.outputValues = []int{value}
	var values uapi.HandleData
	values[0] = uint8(value)
	return uapi.SetLineValues(l.vfd, values)
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
	c, err := NewChip(l.chip)
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
	var uvv uapi.HandleData
	err := uapi.GetLineValues(l.vfd, &uvv)
	if err != nil {
		return err
	}
	lines := len(l.offsets)
	if len(values) < lines {
		lines = len(values)
	}
	for i := 0; i < lines; i++ {
		values[i] = int(uvv[i])
	}
	return nil
}

// SetValues sets the current active state of the collection of lines.
//
// Only valid for output lines.
//
// All lines in the set are set at once.  If insufficient values are provided
// then the remaining lines are set to inactive.
func (l *Lines) SetValues(values []int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.flags.IsOutput() {
		return ErrPermissionDenied
	}
	if len(values) > len(l.offsets) {
		return ErrInvalidOffset
	}
	if l.closed {
		return ErrClosed
	}
	l.outputValues = append([]int(nil), values...)
	var vv uapi.HandleData
	for i, v := range values {
		vv[i] = uint8(v)
	}
	return uapi.SetLineValues(l.vfd, vv)
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
	// been based on CLOCK_REALTIME, but from Linux v5.7 it is based on
	// CLOCK_MONOTONIC.
	Timestamp time.Duration

	// The type of state change event this structure represents.
	Type LineEventType
}

// LineInfoChangeEvent represents a change in the info a line.
type LineInfoChangeEvent struct {
	// Info is the updated line info.
	Info LineInfo

	// Timestamp indicates the time the event was detected.
	//
	// The timestamp is intended for accurately measuring intervals between
	// events. It is not guaranteed to be based on a particular clock, but from
	// Linux v5.7 it is based on CLOCK_MONOTONIC.
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

// chipNames returns the name of potential gpiochips.
//
// Does not open them or check if they are valid.
func chipNames() []string {
	ee, err := ioutil.ReadDir("/dev")
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
	return cc
}

// helper that finds the chip and offset corresponding to a named line.
//
// If found returns the chip and offset, else an error.
func findLine(lname string) (*Chip, int, error) {
	for _, name := range chipNames() {
		c, err := NewChip(name)
		if err != nil {
			continue
		}
		o, err := c.FindLine(lname)
		if err == nil {
			return c, o, nil
		}
	}
	return nil, 0, ErrLineNotFound
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

	// ErrInvalidOffset indicates a line offset is invalid.
	ErrInvalidOffset = errors.New("invalid offset")

	// ErrNotCharacterDevice indicates the device is not a character device.
	ErrNotCharacterDevice = errors.New("not a character device")

	// ErrLineNotFound indicates the line was not found.
	ErrLineNotFound = errors.New("line not found")

	// ErrPermissionDenied indicates caller does not have required permissions
	// for the operation.
	ErrPermissionDenied = errors.New("permission denied")
)
