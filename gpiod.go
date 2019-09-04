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

// LineInfo contains a summary of publically available information about the
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
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	ci, err := uapi.GetChipInfo(f.Fd())
	if err != nil {
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
		c.w.Close()
	}
	for _, req := range rr {
		unix.Close(int(req.fd))
	}
	c.f.Close()
}

// LineInfo returns the publically available information on the line.
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

// GetLine requests control of a single line on the chip.
// If granted, control is maintained until either the Line or Chip are closed.
func (c *Chip) GetLine(offset uint, options ...LineOption) (*Line, error) {
	ll, err := c.GetLines([]uint{offset}, options...)
	if err != nil {
		return nil, err
	}
	l := Line{*ll}
	return &l, nil
}

// GetLines requests control of a correction of lines on the chip.
func (c *Chip) GetLines(offsets []uint, options ...LineOption) (*Lines, error) {
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

// GetValue returns the current value (active state) of the line.
func (l *Line) GetValue() (int, error) {
	v, err := l.GetValues()
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

// Lines represents a correction of requested lines.
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

// GetValues returns the current value (active state) of the correction of
// lines.
func (l *Lines) GetValues() ([]uint8, error) {
	var values uapi.HandleData
	err := uapi.GetLineValues(l.vfd, &values)
	if err != nil {
		return nil, err
	}
	return append(values[:0:0], values[:len(l.offsets)]...), nil
}

// SetValues sets the current active state of the correction of lines. Only
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
	// LineEventRisingEdge indicates a low to high event.
	LineEventRisingEdge = LineEventType(1)
	// LineEventFallingEdge indicates a high to low event.
	LineEventFallingEdge = LineEventType(iota)
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

type watcher struct {
	epfd    int
	donefds []int
	mu      sync.Mutex
	hh      map[int32]*request
}

func (w *watcher) add(r *request) {
	w.mu.Lock()
	w.hh[int32(r.fd)] = r
	w.mu.Unlock()
	epv := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(r.fd)}
	unix.EpollCtl(w.epfd, unix.EPOLL_CTL_ADD, int(r.fd), &epv)
}

func (w *watcher) del(r *request) {
	unix.EpollCtl(w.epfd, unix.EPOLL_CTL_DEL, int(r.fd), nil)
	w.mu.Lock()
	delete(w.hh, int32(r.fd))
	w.mu.Unlock()
}

func newWatcher() (*watcher, error) {
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	p := []int{0, 0}
	err = unix.Pipe2(p, unix.O_CLOEXEC)
	if err != nil {
		return nil, err
	}
	epv := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(p[0])}
	unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(p[0]), &epv)
	w := watcher{
		epfd:    epfd,
		hh:      make(map[int32]*request),
		donefds: p,
	}
	go w.watch()
	return &w, nil
}

// Close - His watch has ended.
func (w *watcher) Close() {
	unix.Write(w.donefds[1], []byte("bye"))
	unix.Close(w.donefds[1])
}

func (w *watcher) watch() {
	var epollEvents [1]unix.EpollEvent
	for {
		n, err := unix.EpollWait(w.epfd, epollEvents[:], -1)
		if err != nil {
			fmt.Println("epoll:", n, err)
			if err == unix.EBADF || err == unix.EINVAL {
				// fd closed so exit
				return
			}
			if err == unix.EINTR {
				continue
			}
			panic(fmt.Sprintf("EpollWait unexpected error: %v", err))
		}
		for _, ev := range epollEvents {
			fd := ev.Fd
			if fd == int32(w.donefds[0]) {
				unix.Close(w.epfd)
				unix.Close(w.donefds[0])
				fmt.Println("watcher exitting")
				return
			}
			w.mu.Lock()
			req := w.hh[fd]
			w.mu.Unlock()
			if req == nil {
				continue
			}
			evt, err := uapi.ReadEvent(uintptr(fd))
			if err != nil {
				fmt.Println("event read error:", err)
				continue
			}
			le := LineEvent{
				Offset:    req.offset,
				Timestamp: time.Duration(evt.Timestamp),
				Type:      LineEventType(evt.ID),
			}
			req.eh(le)
		}
	}
}
