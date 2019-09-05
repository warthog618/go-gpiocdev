// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package gpiod

import (
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
	Name  string
	Label string
	// The number of GPIO lines on this chip.
	lines uint
	// default consumer label for reserved lines
	consumer string
	// mutex covers the attributes below it.
	mu sync.Mutex
	// set of requests currently open.
	// This doubles as a closed flag - is nil once closed.
	rr map[uint]*request
	// watcher for events
	w *watcher
}

// LineInfo contains a summary of publicly available information about the
// line.
type LineInfo struct {
	Offset     uint
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
	err := IsCharDev(path)
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
		return nil, err
	}
	ci, err := uapi.GetChipInfo(f.Fd())
	if err != nil {
		f.Close()
		return nil, err
	}
	c := Chip{
		f:        f,
		Name:     string(ci.Name[:]),
		Label:    string(ci.Label[:]),
		lines:    uint(ci.Lines),
		consumer: co.consumer,
		rr:       make(map[uint]*request)}
	if len(c.Label) == 0 {
		c.Label = "unknown"
	}
	return &c, nil
}

// Close releases all resources associated with the Chip
func (c *Chip) Close() {
	c.mu.Lock()
	rr := c.rr
	c.rr = nil
	c.mu.Unlock()
	if c.w != nil {
		c.w.close()
	}
	for _, req := range rr {
		unix.Close(int(req.fd))
	}
	c.f.Close()
}

// LineInfo returns the publicly available information on the line.
// This is always available and does not require requesting the line.
func (c *Chip) LineInfo(offset uint) (LineInfo, error) {
	if offset >= c.lines {
		return LineInfo{}, unix.EINVAL
	}
	li, err := uapi.GetLineInfo(c.f.Fd(), uint32(offset))
	if err != nil {
		return LineInfo{}, err
	}
	return LineInfo{
		Offset:     offset,
		Name:       string(li.Name[:]),
		Consumer:   string(li.Consumer[:]),
		Requested:  li.Flags.IsRequested(),
		IsOut:      li.Flags.IsOut(),
		ActiveLow:  li.Flags.IsActiveLow(),
		OpenDrain:  li.Flags.IsOpenDrain(),
		OpenSource: li.Flags.IsOpenSource(),
	}, nil
}

// Lines returns the number of lines that exist on the GPIO chip.
func (c *Chip) Lines() uint {
	return c.lines
}

// RequestLine requests control of a single line on the chip.
// If granted, control is maintained until either the Line or Chip are closed.
func (c *Chip) RequestLine(offset uint, options ...LineOption) (*Line, error) {
	ll, err := c.RequestLines([]uint{offset}, options...)
	if err != nil {
		return nil, err
	}
	l := Line{*ll}
	return &l, nil
}

// RequestLines requests control of a collection of lines on the chip.
func (c *Chip) RequestLines(offsets []uint, options ...LineOption) (*Lines, error) {
	for _, o := range offsets {
		if o >= c.lines {
			return nil, unix.EINVAL
		}
	}
	lo := LineOptions{
		consumer: c.consumer,
	}
	for _, option := range options {
		option.applyLineOption(&lo)
	}
	ll := Lines{
		offsets: offsets,
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
				for _, ono := range offsets[:i] {
					c.removeRequest(ono)
				}
				return nil, err
			}
			fd := uintptr(er.Fd)
			if i == 0 {
				ll.vfd = fd
			}
			err = c.addRequest(request{o, ll.vfd, lo.eh})
			if err != nil {
				unix.Close(int(fd))
				for _, ono := range offsets[:i] {
					c.removeRequest(ono)
				}
				return nil, err
			}
		}
	} else {
		hr := uapi.HandleRequest{
			Lines: uint32(len(offsets)),
			Flags: lo.HandleFlags,
		}
		copy(hr.Consumer[:], lo.consumer)
		//copy(hr.Offsets[:], l.offsets) - with cast
		for i, o := range ll.offsets {
			hr.Offsets[i] = uint32(o)
		}
		err := uapi.GetLineHandle(c.f.Fd(), &hr)
		if err != nil {
			return nil, err
		}
		ll.vfd = uintptr(hr.Fd)
		err = c.addRequest(request{offsets[0], ll.vfd, lo.eh})
		if err != nil {
			return nil, err
		}
	}
	return &ll, nil
}

type request struct {
	offset uint
	fd     uintptr
	eh     eventHandler
}

func (c *Chip) addRequest(r request) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// just in case of a race on Chip.Close and Chip.GetLine(s)
	if c.rr == nil {
		return unix.ECANCELED
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

func (c *Chip) removeRequest(offset uint) {
	c.mu.Lock()
	if c.rr != nil {
		r := c.rr[offset]
		delete(c.rr, offset)
		if r != nil {
			if r.eh != nil {
				c.w.del(r)
			}
			unix.Close(int(r.fd))
		}
	}
	c.mu.Unlock()
}

// Line represents a single requested line.
type Line struct {
	Lines
}

// Value returns the current value (active state) of the line.
func (l *Line) Value() (int, error) {
	v, err := l.Values()
	if err != nil {
		return 0, err
	}
	return int(v[0]), nil
}

// SetValue sets the current active state of the line.
// Only valid for output lines.
func (l *Line) SetValue(value int) error {
	if l.canset == false {
		return unix.EPERM
	}
	var values uapi.HandleData
	values[0] = uint8(value)
	return uapi.SetLineValues(l.vfd, values)
}

// Lines represents a collection of requested lines.
type Lines struct {
	offsets []uint
	vfd     uintptr
	canset  bool
	mu      *sync.Mutex
	chip    *Chip
}

// Close releases all resources held by the requested lines.
func (l *Lines) Close() {
	l.mu.Lock()
	c := l.chip
	l.chip = nil
	l.mu.Unlock()
	if c == nil {
		return
	}
	for _, o := range l.offsets {
		c.removeRequest(o)
	}
}

// Values returns the current value (active state) of the collection of
// lines.
func (l *Lines) Values() ([]uint8, error) {
	var values uapi.HandleData
	err := uapi.GetLineValues(l.vfd, &values)
	if err != nil {
		return nil, err
	}
	return append(values[:0:0], values[:len(l.offsets)]...), nil
}

// SetValues sets the current active state of the collection of lines. Only
// valid for output lines.
// All lines in the set are set at once and the provided values must contain a
// value for each line.
func (l *Lines) SetValues(values []uint8) error {
	if l.canset == false {
		return unix.EPERM
	}
	if len(values) < len(l.offsets) {
		return unix.EINVAL
	}
	var vv uapi.HandleData
	copy(vv[:], values)
	return uapi.SetLineValues(l.vfd, vv)
}

// LineEventType indicates the type of change to the line active state.
// Note that for active low lines a low line level results in a high active
// state.
type LineEventType uint

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
	Offset uint
	// Timestamp is the best guess as to the time the event was detected.
	// This is the Unix epoch - nsec since Jan 1 1970.
	Timestamp time.Duration
	// The type of event this structure represents.
	Type LineEventType
}

// IsCharDev checks if the device at path is an accessible GPIO chardev.
// Returns nil if it is and an error if not.
func IsCharDev(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeCharDevice == 0 {
		return unix.ENOTTY
	}
	sysfspath := fmt.Sprintf("/sys/bus/gpio/devices/%s/dev", fi.Name())
	if err = unix.Access(sysfspath, unix.R_OK); err != nil {
		return unix.ENOTTY
	}
	sysfsf, err := os.Open(sysfspath)
	if err != nil {
		return unix.ENODEV
	}
	var sysfsdev [16]byte
	n, err := sysfsf.Read(sysfsdev[:])
	sysfsf.Close()
	if err != nil || n <= 0 {
		return unix.ENODEV
	}
	var stat unix.Stat_t
	if err = unix.Lstat(path, &stat); err != nil {
		return err
	}
	devstr := fmt.Sprintf("%d:%d", unix.Major(stat.Rdev), unix.Minor(stat.Rdev))
	sysstr := string(sysfsdev[:n-1])
	if devstr != sysstr {
		return unix.ENODEV
	}
	return nil
}
