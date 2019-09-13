// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// Package gpiod provides a library for the Linux GPIO descriptor UAPI.
// This is a Go equivalent to libgpiod.
package gpiod

import (
	"errors"
	"fmt"
	"gpiod/uapi"
	"os"
	"sync"
	"time"

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

	// default consumer label for reserved lines
	consumer string

	// mutex covers the attributes below it.
	mu sync.Mutex

	// set of requests currently open.
	// This doubles as a closed flag - is nil once closed.
	rr map[int]*request

	// watcher for events
	w *watcher
}

// LineInfo contains a summary of publicly available information about the
// line.
type LineInfo struct {
	Offset     int
	Name       string
	Consumer   string
	Requested  bool
	IsOut      bool
	ActiveLow  bool
	OpenDrain  bool
	OpenSource bool
}

// NewChip opens a GPIO character device.
func NewChip(path string, options ...ChipOption) (*Chip, error) {
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
		f:        f,
		Name:     uapi.BytesToString(ci.Name[:]),
		Label:    uapi.BytesToString(ci.Label[:]),
		lines:    int(ci.Lines),
		consumer: co.consumer,
		rr:       make(map[int]*request),
	}
	if len(c.Label) == 0 {
		c.Label = "unknown"
	}
	return &c, nil
}

// Close releases all resources associated with the Chip.
func (c *Chip) Close() error {
	c.mu.Lock()
	rr := c.rr
	c.rr = nil
	c.mu.Unlock()
	if rr == nil {
		return ErrClosed
	}
	if c.w != nil {
		c.w.close()
	}
	for _, req := range rr {
		unix.Close(int(req.fd))
	}
	return c.f.Close()
}

// LineInfo returns the publicly available information on the line.
// This is always available and does not require requesting the line.
func (c *Chip) LineInfo(offset int) (LineInfo, error) {
	if offset < 0 || offset >= c.lines {
		return LineInfo{}, ErrInvalidOffset
	}
	li, err := uapi.GetLineInfo(c.f.Fd(), offset)
	if err != nil {
		return LineInfo{}, err
	}
	return LineInfo{
		Offset:     offset,
		Name:       uapi.BytesToString(li.Name[:]),
		Consumer:   uapi.BytesToString(li.Consumer[:]),
		Requested:  li.Flags.IsRequested(),
		IsOut:      li.Flags.IsOut(),
		ActiveLow:  li.Flags.IsActiveLow(),
		OpenDrain:  li.Flags.IsOpenDrain(),
		OpenSource: li.Flags.IsOpenSource(),
	}, nil
}

// Lines returns the number of lines that exist on the GPIO chip.
func (c *Chip) Lines() int {
	return c.lines
}

// RequestLine requests control of a single line on the chip.
// If granted, control is maintained until either the Line or Chip are closed.
func (c *Chip) RequestLine(offset int, options ...LineOption) (*Line, error) {
	ll, err := c.RequestLines([]int{offset}, options...)
	if err != nil {
		return nil, err
	}
	l := Line{
		offset: offset,
		vfd:    ll.vfd,
		canset: ll.canset,
		chip:   c,
	}
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
		consumer: c.consumer,
	}
	for _, option := range options {
		option.applyLineOption(&lo)
	}
	ll := Lines{
		offsets: append([]int(nil), offsets...),
		canset:  lo.HandleFlags.IsOutput(),
		mu:      &sync.Mutex{},
		chip:    c,
	}
	if lo.eh != nil {
		for i, o := range offsets {
			er := uapi.EventRequest{
				Offset:      uint32(o),
				HandleFlags: lo.HandleFlags,
				EventFlags:  lo.EventFlags,
			}
			copy(er.Consumer[:], lo.consumer)
			err := uapi.GetLineEvent(c.f.Fd(), &er)
			if err != nil {
				c.removeRequests(offsets[:i]...)
				return nil, err
			}
			fd := uintptr(er.Fd)
			if i == 0 {
				ll.vfd = fd
			}
			err = c.addRequest(request{o, fd, lo.eh})
			if err != nil {
				// this should never happen, but just in case...
				unix.Close(int(fd))
				c.removeRequests(offsets[:i]...)
				return nil, err
			}
		}
	} else {
		hr := uapi.HandleRequest{
			Lines: uint32(len(offsets)),
			Flags: lo.HandleFlags,
		}
		copy(hr.Consumer[:], lo.consumer)
		//copy(hr.Offsets[:], ll.offsets) - with cast
		for i, o := range ll.offsets {
			hr.Offsets[i] = uint32(o)
		}
		//copy(hr.DefaultValues[:], lo.DefaultValues) - with cast
		for i, v := range lo.DefaultValues {
			hr.DefaultValues[i] = uint8(v)
		}
		err := uapi.GetLineHandle(c.f.Fd(), &hr)
		if err != nil {
			return nil, err
		}
		ll.vfd = uintptr(hr.Fd)
		err = c.addRequest(request{offsets[0], ll.vfd, lo.eh})
		if err != nil {
			// this should never happen, but just in case...
			return nil, err
		}
	}
	return &ll, nil
}

type request struct {
	offset int
	fd     uintptr
	eh     eventHandler
}

func (c *Chip) addRequest(r request) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// just in case of a race on Chip.Close and Chip.GetLine(s)
	if c.rr == nil {
		return ErrClosed
	}
	c.rr[r.offset] = &r
	if r.eh != nil {
		if c.w == nil {
			w, err := newWatcher()
			if err != nil {
				return err
			}
			c.w = w
		}
		c.w.add(&r)
	}
	return nil
}

func (c *Chip) removeRequests(offsets ...int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rr == nil {
		return
	}
	for _, offset := range offsets {
		r := c.rr[offset]
		delete(c.rr, offset)
		if r != nil {
			if r.eh != nil {
				c.w.del(r)
			}
			unix.Close(int(r.fd))
		}
	}
}

// Line represents a single requested line.
type Line struct {
	offset int
	vfd    uintptr
	canset bool
	mu     sync.Mutex
	chip   *Chip
}

// Value returns the current value (active state) of the line.
func (l *Line) Value() (int, error) {
	var values uapi.HandleData
	err := uapi.GetLineValues(l.vfd, &values)
	return int(values[0]), err
}

// SetValue sets the current active state of the line.
// Only valid for output lines.
func (l *Line) SetValue(value int) error {
	if l.canset == false {
		return ErrPermissionDenied
	}
	var values uapi.HandleData
	values[0] = uint8(value)
	return uapi.SetLineValues(l.vfd, values)
}

// Close releases all resources held by the requested line.
func (l *Line) Close() error {
	l.mu.Lock()
	c := l.chip
	l.chip = nil
	l.mu.Unlock()
	if c == nil {
		return ErrClosed
	}
	c.removeRequests(l.offset)
	return nil
}

// Lines represents a collection of requested lines.
type Lines struct {
	offsets []int
	vfd     uintptr
	canset  bool
	mu      *sync.Mutex
	chip    *Chip
}

// Close releases all resources held by the requested lines.
func (l *Lines) Close() error {
	l.mu.Lock()
	c := l.chip
	l.chip = nil
	l.mu.Unlock()
	if c == nil {
		return ErrClosed
	}
	c.removeRequests(l.offsets...)
	return nil
}

// Values returns the current value (active state) of the collection of
// lines.
func (l *Lines) Values() ([]int, error) {
	var values uapi.HandleData
	err := uapi.GetLineValues(l.vfd, &values)
	if err != nil {
		return nil, err
	}
	vv := make([]int, len(l.offsets))
	for i := 0; i < len(l.offsets); i++ {
		vv[i] = int(values[i])
	}
	return vv, nil
}

// SetValues sets the current active state of the collection of lines.
// Only valid for output lines.
// All lines in the set are set at once.  If insufficient values are provided
// then the remaining lines are set to inactive.
func (l *Lines) SetValues(values ...int) error {
	if l.canset == false {
		return ErrPermissionDenied
	}
	if len(values) > len(l.offsets) {
		return ErrInvalidOffset
	}
	var vv uapi.HandleData
	for i, v := range values {
		vv[i] = uint8(v)
	}
	return uapi.SetLineValues(l.vfd, vv)
}

// LineEventType indicates the type of change to the line active state.
// Note that for active low lines a low line level results in a high active
// state.
type LineEventType int

const (
	_ LineEventType = iota
	// LineEventRisingEdge indicates a low to high event.
	LineEventRisingEdge

	// LineEventFallingEdge indicates a high to low event.
	LineEventFallingEdge
)

// LineEvent represents a change in state to a monitored line.
type LineEvent struct {
	// The line offset within the GPIO chip.
	Offset int

	// Timestamp is the time the event was detected.
	// This is the Unix epoch - nsec since Jan 1 1970.
	Timestamp time.Duration

	// The type of event this structure represents.
	Type LineEventType
}

// IsChip checks if the device at path is an accessible GPIO chardev.
// Returns an error if not.
func IsChip(path string) error {
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
	devstr := fmt.Sprintf("%d:%d", unix.Major(stat.Rdev), unix.Minor(stat.Rdev))
	sysstr := string(sysfsdev[:n-1])
	if devstr != sysstr {
		return ErrNotCharacterDevice
	}
	return nil
}

var (
	// ErrClosed indicates the chip or line has already been closed.
	ErrClosed = errors.New("already closed")
	// ErrInvalidOffset indicates a line offset is invalid.
	ErrInvalidOffset = errors.New("invalid offset")
	// ErrNotCharacterDevice indicates the device is not a character device.
	ErrNotCharacterDevice = errors.New("not a character device")
	// ErrPermissionDenied indicates caller does not have required permissions
	// for the operation.
	ErrPermissionDenied = errors.New("permission denied")
)
