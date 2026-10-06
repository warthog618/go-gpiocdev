// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

package uapi

import (
	"bytes"
	"encoding/binary"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// GetChipInfo returns the ChipInfo for the GPIO character device.
//
// The fd is an open GPIO character device.
func GetChipInfo(fd uintptr) (ChipInfo, error) {
	var ci ChipInfo
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getChipInfoIoctl),
		uintptr(unsafe.Pointer(&ci)))
	if errno != 0 {
		return ci, errno
	}
	return ci, nil
}

// GetLineInfo returns the LineInfo for one line from the GPIO character device.
//
// The fd is an open GPIO character device.
// The offset is zero based.
func GetLineInfo(fd uintptr, offset int) (LineInfo, error) {
	li := LineInfo{Offset: uint32(offset)}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineInfoV2Ioctl),
		uintptr(unsafe.Pointer(&li)))
	if errno != 0 {
		return LineInfo{}, errno
	}
	return li, nil
}

// GetLine requests a line from the GPIO character device.
//
// The fd is an open GPIO character device.
// The lines must not already be requested.
// The flags in the request will be applied to all lines in the request.
// If successful, the fd for the line is returned in the request.fd.
func GetLine(fd uintptr, request *LineRequest) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineIoctl),
		uintptr(unsafe.Pointer(request)))
	if errno != 0 {
		return errno
	}
	return nil
}

// GetLineValues returns the values of a set of requested lines.
//
// The fd is a requested line, as returned by GetLine.
//
// The values returned are the logical values, with inactive being 0.
func GetLineValues(fd uintptr, values *LineValues) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineValuesV2Ioctl),
		uintptr(unsafe.Pointer(values)))
	if errno != 0 {
		return errno
	}
	return nil
}

// SetLineValues sets the values of a set of requested lines.
//
// The fd is a requested line, as returned by GetLine.
func SetLineValues(fd uintptr, values LineValues) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(setLineValuesV2Ioctl),
		uintptr(unsafe.Pointer(&values)))
	if errno != 0 {
		return errno
	}
	return nil
}

// SetLineConfig sets the config of an existing handle request.
//
// The config flags in the request will be applied to all lines in the request.
func SetLineConfig(fd uintptr, config *LineConfig) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(setLineConfigV2Ioctl),
		uintptr(unsafe.Pointer(config)))
	if errno != 0 {
		return errno
	}
	return nil
}

// WatchLineInfo sets a watch on info of a line.
//
// A watch is set on the line indicated by info.Offset. If successful the
// current line info is returned, else an error is returned.
func WatchLineInfo(fd uintptr, info *LineInfo) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(watchLineInfoV2Ioctl),
		uintptr(unsafe.Pointer(info)))
	if errno != 0 {
		return errno
	}
	return nil
}

// UnwatchLineInfo clears a watch on info of a line.
//
// Disables the watch on info for the line.
func UnwatchLineInfo(fd uintptr, offset uint32) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(unwatchLineInfoIoctl),
		uintptr(unsafe.Pointer(&offset)))
	if errno != 0 {
		return errno
	}
	return nil
}

type fdReader int

func (fd fdReader) Read(b []byte) (int, error) {
	return unix.Read(int(fd), b[:])
}

// ReadLineEvent reads a single event from a requested line.
//
// The fd is a requested line, as returned by GetLine.
//
// This function is blocking and should only be called when the fd is known to
// be ready to read.
func ReadLineEvent(fd uintptr) (LineEvent, error) {
	var le LineEvent
	err := binary.Read(fdReader(fd), nativeEndian, &le)
	return le, err
}

// ReadLineInfoChanged reads a line info changed event from a chip.
//
// The fd is an open GPIO character device.
//
// This function is blocking and should only be called when the fd is known to
// be ready to read.
func ReadLineInfoChanged(fd uintptr) (LineInfoChanged, error) {
	var lic LineInfoChanged
	err := binary.Read(fdReader(fd), nativeEndian, &lic)
	return lic, err
}

// IOCTL command codes
type ioctl uintptr

var (
	getChipInfoIoctl     ioctl
	getLineInfoV2Ioctl   ioctl
	getLineIoctl         ioctl
	getLineValuesV2Ioctl ioctl
	setLineValuesV2Ioctl ioctl
	setLineConfigV2Ioctl ioctl
	watchLineInfoV2Ioctl ioctl
	unwatchLineInfoIoctl ioctl
)

func init() {
	// ioctls require struct sizes which are only available at runtime.
	var ci ChipInfo
	getChipInfoIoctl = ior(0xB4, 0x01, unsafe.Sizeof(ci))
	var li LineInfo
	getLineInfoV2Ioctl = iorw(0xB4, 0x05, unsafe.Sizeof(li))
	watchLineInfoV2Ioctl = iorw(0xB4, 0x06, unsafe.Sizeof(li))
	var lr LineRequest
	getLineIoctl = iorw(0xB4, 0x07, unsafe.Sizeof(lr))
	var offset uint32
	unwatchLineInfoIoctl = iorw(0xB4, 0x0C, unsafe.Sizeof(offset))
	var lc LineConfig
	setLineConfigV2Ioctl = iorw(0xB4, 0x0D, unsafe.Sizeof(lc))
	var lv LineValues
	getLineValuesV2Ioctl = iorw(0xB4, 0x0E, unsafe.Sizeof(lv))
	setLineValuesV2Ioctl = iorw(0xB4, 0x0F, unsafe.Sizeof(lv))
}

// Size of name and consumer strings.
const nameSize = 32

// ChipInfo contains the details of a GPIO chip.
type ChipInfo struct {
	// The system name of the device.
	Name [nameSize]byte

	// An identifying label added by the device driver.
	Label [nameSize]byte

	// The number of lines supported by this chip.
	Lines uint32
}

// LineInfo contains the details of a single line of a GPIO chip.
type LineInfo struct {
	// The system name for this line.
	Name [nameSize]byte

	// If requested, a string added by the requester to identify the
	// owner of the request.
	Consumer [nameSize]byte

	// The offset of the line within the chip.
	Offset uint32

	NumAttrs uint32

	Flags LineFlag

	Attrs [10]LineAttribute

	// reserved for future use.
	Padding [lineInfoPadSize]uint32
}

// ChangeType indicates the type of change that has occurred to a line.
type ChangeType uint32

const (
	_ ChangeType = iota

	// LineChangedRequested indicates the line has been requested.
	LineChangedRequested

	// LineChangedReleased indicates the line has been released.
	LineChangedReleased

	// LineChangedConfig indicates the line configuration has changed.
	LineChangedConfig
)

// LineInfoChanged contains the details of a change to line info.
//
// This is returned via the chip fd in response to changes to watched lines.
type LineInfoChanged struct {
	// The updated info.
	Info LineInfo

	// The time the change occurred.
	Timestamp uint64

	// The type of change.
	Type ChangeType

	// reserved for future use.
	Padding [lineInfoChangedPadSize]uint32
}

// LineFlag are the flags for a line.
type LineFlag uint64

const (
	// LineFlagUsed indicates that the line is already in use.
	// It may have been requested by this process or another process,
	// or may be reserved by the kernel.
	//
	// The line cannot be requested until this flag is clear.
	LineFlagUsed LineFlag = 1 << iota

	// LineFlagActiveLow indicates that the line is active low.
	LineFlagActiveLow

	// LineFlagInput indicates that the line direction is an input.
	LineFlagInput

	// LineFlagOutput indicates that the line direction is an output.
	LineFlagOutput

	// LineFlagEdgeRising indicates that edge detection is enabled for rising
	// edges.
	LineFlagEdgeRising

	// LineFlagEdgeFalling indicates that edge detection is enabled for
	// falling edges.
	LineFlagEdgeFalling

	// LineFlagOpenDrain indicates that the line drive is open drain.
	LineFlagOpenDrain

	// LineFlagOpenSource indicates that the line drive is open source.
	LineFlagOpenSource

	// LineFlagBiasPullUp indicates that the line bias is pull-up.
	LineFlagBiasPullUp

	// LineFlagBiasPullDown indicates that the line bias is set pull-down.
	LineFlagBiasPullDown

	// LineFlagBiasDisabled indicates that the line bias is disabled.
	LineFlagBiasDisabled

	// LineFlagEventClockRealtime indicates that the CLOCK_REALTIME will be
	// the source for event timestamps.
	LineFlagEventClockRealtime

	// LineFlagDirectionMask is a mask for all direction flags.
	LineFlagDirectionMask = LineFlagInput | LineFlagOutput

	// LineFlagEdgeMask is a mask for all edge flags.
	LineFlagEdgeMask = LineFlagEdgeRising | LineFlagEdgeFalling

	// LineFlagEdgeBoth is a helper value for selecting edge detection on
	// both edges.
	LineFlagEdgeBoth = LineFlagEdgeMask

	// LineFlagDriveMask is a mask for all drive flags.
	LineFlagDriveMask = LineFlagOpenDrain | LineFlagOpenSource

	// LineFlagBiasMask is a mask for all bias flags.
	LineFlagBiasMask = LineFlagBiasDisabled | LineFlagBiasPullUp | LineFlagBiasPullDown
)

// IsAvailable returns true if the line is available to be requested.
func (f LineFlag) IsAvailable() bool {
	return f&LineFlagUsed == 0
}

// IsUsed returns true if the line is not available to be requested.
func (f LineFlag) IsUsed() bool {
	return f&LineFlagUsed != 0
}

// IsActiveLow returns true if the line is active low.
func (f LineFlag) IsActiveLow() bool {
	return f&LineFlagActiveLow != 0
}

// IsInput returns true if the line is an input.
func (f LineFlag) IsInput() bool {
	return f&LineFlagInput != 0
}

// IsOutput returns true if the line is an output.
func (f LineFlag) IsOutput() bool {
	return f&LineFlagOutput != 0
}

// IsOpenDrain returns true if the line is an open drain.
func (f LineFlag) IsOpenDrain() bool {
	return f&LineFlagOpenDrain != 0
}

// IsOpenSource returns true if the line is an open source.
func (f LineFlag) IsOpenSource() bool {
	return f&LineFlagOpenSource != 0
}

// IsRisingEdge returns true if the line has edge detection on the rising edge.
func (f LineFlag) IsRisingEdge() bool {
	return f&LineFlagEdgeRising != 0
}

// IsFallingEdge returns true if the line has edge detection on the falling edge.
func (f LineFlag) IsFallingEdge() bool {
	return f&LineFlagEdgeFalling != 0
}

// IsBothEdges returns true if the line has edge detection on both edges.
func (f LineFlag) IsBothEdges() bool {
	return f&LineFlagEdgeBoth == LineFlagEdgeBoth
}

// IsBiasDisabled returns true if the line has bias disabled.
func (f LineFlag) IsBiasDisabled() bool {
	return f&LineFlagBiasDisabled != 0
}

// IsBiasPullUp returns true if the line has pull-up bias enabled.
func (f LineFlag) IsBiasPullUp() bool {
	return f&LineFlagBiasPullUp != 0
}

// IsBiasPullDown returns true if the line has pull-down bias enabled.
func (f LineFlag) IsBiasPullDown() bool {
	return f&LineFlagBiasPullDown != 0
}

// IsRealtimeEventClock returns true if the line events will contain real-time
// timestamps.
func (f LineFlag) IsRealtimeEventClock() bool {
	return f&LineFlagEventClockRealtime != 0
}

// Encode creates a LineAttribute with the value from the LineFlag.
func (f LineFlag) Encode() (la LineAttribute) {
	la.Encode64(LineAttributeIDFlags, uint64(f))
	return
}

// Decode populates the LineFlag with value from the LineAttribute.
func (f *LineFlag) Decode(la LineAttribute) {
	*f = LineFlag(la.Value64())
}

const (
	// LinesMax is the maximum number of lines that can be requested in a single
	// request.
	LinesMax int = 64

	// the pad sizes of each struct
	lineConfigPadSize      int = 5
	lineRequestPadSize     int = 5
	lineEventPadSize       int = 6
	lineInfoPadSize        int = 4
	lineInfoChangedPadSize int = 5
)

// LineAttribute defines a configuration attribute for a line.
type LineAttribute struct {
	ID LineAttributeID

	Padding [1]uint32

	Value [8]byte
}

// Encode32 populates the LineAttribute using the id and 32-bit value.
func (la *LineAttribute) Encode32(id LineAttributeID, value uint32) {
	la.ID = id
	nativeEndian.PutUint32(la.Value[:], value)
}

// Encode64 populates the LineAttribute using the id and 64-bit value.
func (la *LineAttribute) Encode64(id LineAttributeID, value uint64) {
	la.ID = id
	nativeEndian.PutUint64(la.Value[:], value)
}

// Value32 returns the 32-bit value from the LineAttribute.
func (la LineAttribute) Value32() uint32 {
	return nativeEndian.Uint32(la.Value[:])
}

// Value64 returns the 64-bit value from the LineAttribute.
func (la LineAttribute) Value64() uint64 {
	return nativeEndian.Uint64(la.Value[:])
}

// LineAttributeID identifies the type of a configuration attribute.
type LineAttributeID uint32

const (
	// LineAttributeIDFlags indicates the attribute contains LineFlag flags.
	LineAttributeIDFlags LineAttributeID = iota + 1

	// LineAttributeIDOutputValues indicates the attribute contains line output values.
	LineAttributeIDOutputValues

	// LineAttributeIDDebounce indicates the attribute contains a debounce period.
	LineAttributeIDDebounce
)

// DebouncePeriod specifies the time the line must be stable before a level
// transition is recognized.
type DebouncePeriod time.Duration

// Encode creates a LineAttribute with the value from the DebouncePeriod.
func (d DebouncePeriod) Encode() (la LineAttribute) {
	la.Encode32(LineAttributeIDDebounce, uint32(d/1000))
	return
}

// Decode populates the DebouncePeriod with value from the LineAttribute.
func (d *DebouncePeriod) Decode(la LineAttribute) {
	*d = DebouncePeriod(la.Value32() * 1000)
}

// OutputValues specify the active level of output lines.
type OutputValues LineBitmap

// Encode creates a LineAttribute with the values from the OutputValues.
func (ov OutputValues) Encode() (la LineAttribute) {
	la.Encode64(LineAttributeIDOutputValues, uint64(ov))
	return
}

// Decode populates the OutputValues with values from the LineAttribute.
func (ov *OutputValues) Decode(la LineAttribute) {
	*ov = OutputValues(la.Value64())
}

// LineConfigAttribute associates a configuration attribute with one or more
// requested lines.
type LineConfigAttribute struct {
	// Attr contains the configuration attribute.
	Attr LineAttribute

	// Mask identifies the lines to which this attribute applies.
	//
	// This is a bitmap of lines in LineRequest.Offsets.
	Mask LineBitmap
}

// LineConfig contains the configuration of a line.
type LineConfig struct {
	// The flags to be applied to the lines.
	Flags LineFlag

	NumAttrs uint32

	// reserved for future use.
	Padding [lineConfigPadSize]uint32

	Attrs [10]LineConfigAttribute
}

// AddAttribute adds an attribute to the configuration.
//
// This is an unconditional add - it performs no filtering or consistency
// checking other than limiting the number of attributes.
func (lc *LineConfig) AddAttribute(lca LineConfigAttribute) {
	if lc.NumAttrs < 10 {
		lc.Attrs[lc.NumAttrs] = lca
		lc.NumAttrs++
	}
}

// RemoveAttribute removes an attribute from the configuration.
func (lc *LineConfig) RemoveAttribute(lca LineConfigAttribute) {
	d := 0
	for s := 0; s < int(lc.NumAttrs); s++ {
		if lc.Attrs[s] != lca {
			if d != s {
				lc.Attrs[d] = lc.Attrs[s]
			}
			d++
		}
	}
	lc.NumAttrs = uint32(d)
}

// RemoveAttributeID removes all attributes with a given ID from the configuration.
func (lc *LineConfig) RemoveAttributeID(id LineAttributeID) {
	d := 0
	for s := 0; s < int(lc.NumAttrs); s++ {
		if lc.Attrs[s].Attr.ID != id {
			if d != s {
				lc.Attrs[d] = lc.Attrs[s]
			}
			d++
		}
	}
	lc.NumAttrs = uint32(d)
}

// LineRequest is a request for control of a set of lines.
// The lines must all be on the same GPIO chip.
type LineRequest struct {
	// The lines to be requested.
	Offsets [LinesMax]uint32

	// The string identifying the requester to be applied to the lines.
	Consumer [nameSize]byte

	// The configuration for the requested lines
	Config LineConfig

	// The number of lines being requested.
	Lines uint32

	// Minimum size of the event buffer.
	EventBufferSize uint32

	// reserved for future use.
	Padding [lineRequestPadSize]uint32

	// The file handle for the requested lines.
	// Set if the request is successful.
	Fd int32
}

// LineBitmap is a bitmap containing a bit for each line.
type LineBitmap uint64

// NewLineBits creates a new LineBitmap from an array of bit numbers.
func NewLineBits(vv ...int) LineBitmap {
	var lb LineBitmap
	for _, bit := range vv {
		lb = lb.Set(bit, 1)
	}
	return lb
}

// NewLineBitmap creates a bitmap from an array of bit values.
func NewLineBitmap(vv ...int) LineBitmap {
	var lb LineBitmap
	for i, v := range vv {
		lb = lb.Set(i, v)
	}
	return lb
}

// NewLineBitMask returns a mask of n bits.
func NewLineBitMask(n int) LineBitmap {
	if n >= LinesMax {
		n = LinesMax
	}
	if n == LinesMax {
		return 0xffffffffffffffff
	}
	return (LineBitmap(1) << uint(n)) - 1
}

// Get returns the value of the nth bit.
func (lb LineBitmap) Get(n int) int {
	mask := LineBitmap(1) << uint(n)
	if lb&mask != 0 {
		return 1
	}
	return 0
}

// Set sets the value of the nth bit.
func (lb LineBitmap) Set(n, v int) LineBitmap {
	mask := LineBitmap(1) << uint(n)
	if v == 0 {
		return lb &^ mask
	}
	return lb | mask
}

// LineValues contains the output values for a set of lines.
type LineValues struct {
	// Bits contains the logical value of the the lines.
	//
	// Zero is a logical low (inactive) and 1 is a logical high (active).
	//
	// This is a bitmap of lines in LineRequest.Offsets.
	Bits LineBitmap

	// Mask identifies the lines to which this attribute applies.
	//
	// This is a bitmap of lines in LineRequest.Offsets.
	Mask LineBitmap
}

// Get returns the value of the nth bit.
func (lv LineValues) Get(n int) int {
	mask := LineBitmap(1) << uint(n)
	if lv.Bits&mask != 0 {
		return 1
	}
	return 0
}

// LineEventID indicates the type of event detected.
type LineEventID uint32

const (
	// LineEventRisingEdge indicates the event is a rising edge.
	LineEventRisingEdge LineEventID = iota + 1

	// LineEventFallingEdge indicates the event is a falling edge.
	LineEventFallingEdge
)

// LineEvent contains the details of a particular line event.
//
// This is returned via the event request fd in response to events.
type LineEvent struct {
	// The time the event was detected.
	Timestamp uint64

	// The type of event detected.
	ID LineEventID

	// The line that triggered the event.
	Offset uint32

	// The sequence number for this event in all events on all lines in this line request.
	Seqno uint32

	// The sequence number for this event in all events in this line.
	LineSeqno uint32

	// reserved for future use
	Padding [lineEventPadSize]uint32
}

// BytesToString is a helper function that converts strings stored in byte
// arrays, as returned by GetChipInfo and GetLineInfo, into strings.
func BytesToString(a []byte) string {
	n := bytes.IndexByte(a, 0)
	if n == -1 {
		return string(a)
	}
	return string(a[:n])
}
