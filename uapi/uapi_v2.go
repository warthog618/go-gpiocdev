// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

// +build linux

package uapi

import (
	"encoding/binary"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// GetLineInfoV2 returns the LineInfoV2 for one line from the GPIO character device.
//
// The fd is an open GPIO character device.
// The offset is zero based.
func GetLineInfoV2(fd uintptr, offset int) (LineInfoV2, error) {
	li := LineInfoV2{Offset: uint32(offset)}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineInfoV2Ioctl),
		uintptr(unsafe.Pointer(&li)))
	if errno != 0 {
		return LineInfoV2{}, errno
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

// GetLineValuesV2 returns the values of a set of requested lines.
//
// The fd is a requested line, as returned by GetLine.
func GetLineValuesV2(fd uintptr, values *LineValues) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineValuesV2Ioctl),
		uintptr(unsafe.Pointer(&values[0])))
	if errno != 0 {
		return errno
	}
	return nil
}

// SetLineValuesV2 sets the values of a set of requested lines.
//
// The fd is a requested line, as returned by GetLine.
func SetLineValuesV2(fd uintptr, values LineSetValues) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(setLineValuesV2Ioctl),
		uintptr(unsafe.Pointer(&values)))
	if errno != 0 {
		return errno
	}
	return nil
}

// SetLineConfigV2 sets the config of an existing handle request.
//
// The config flags in the request will be applied to all lines in the request.
func SetLineConfigV2(fd uintptr, config *LineConfig) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(setLineConfigV2Ioctl),
		uintptr(unsafe.Pointer(config)))
	if errno != 0 {
		return errno
	}
	return nil
}

// WatchLineInfoV2 sets a watch on info of a line.
//
// A watch is set on the line indicated by info.Offset. If successful the
// current line info is returned, else an error is returned.
func WatchLineInfoV2(fd uintptr, info *LineInfoV2) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(watchLineInfoV2Ioctl),
		uintptr(unsafe.Pointer(info)))
	if errno != 0 {
		return errno
	}
	return nil
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

// ReadLineInfoChangedV2 reads a line info changed event from a chip.
//
// The fd is an open GPIO character device.
//
// This function is blocking and should only be called when the fd is known to
// be ready to read.
func ReadLineInfoChangedV2(fd uintptr) (LineInfoChangedV2, error) {
	var lic LineInfoChangedV2
	err := binary.Read(fdReader(fd), nativeEndian, &lic)
	return lic, err
}

var (
	getLineInfoV2Ioctl   ioctl
	getLineIoctl         ioctl
	getLineValuesV2Ioctl ioctl
	setLineValuesV2Ioctl ioctl
	setLineConfigV2Ioctl ioctl
	watchLineInfoV2Ioctl ioctl
)

func init() {
	// ioctls require struct sizes which are only available at runtime.
	var liv2 LineInfoV2
	getLineInfoV2Ioctl = iorw(0xB4, 0x05, unsafe.Sizeof(liv2))
	watchLineInfoV2Ioctl = iorw(0xB4, 0x06, unsafe.Sizeof(liv2))
	var lr LineRequest
	getLineIoctl = iorw(0xB4, 0x07, unsafe.Sizeof(lr))
	var lc LineConfig
	setLineConfigV2Ioctl = iorw(0xB4, 0x0D, unsafe.Sizeof(lc))
	var lv LineValues
	getLineValuesV2Ioctl = iorw(0xB4, 0x0E, unsafe.Sizeof(lv))
	var lsv LineSetValues
	setLineValuesV2Ioctl = iorw(0xB4, 0x0F, unsafe.Sizeof(lsv))
}

// LineInfoV2 contains the details of a single line of a GPIO chip.
type LineInfoV2 struct {
	// The system name for this line.
	Name [nameSize]byte

	// If requested, a string added by the requester to identify the
	// owner of the request.
	Consumer [nameSize]byte

	// The offset of the line within the chip.
	Offset uint32

	NumAttrs uint32

	Flags LineFlagV2

	Attrs [10]LineAttribute

	// reserved for future use.
	Padding [lineInfoV2PadSize]uint32
}

// LineInfoChangedV2 contains the details of a change to line info.
//
// This is returned via the chip fd in response to changes to watched lines.
type LineInfoChangedV2 struct {
	// The updated info.
	Info LineInfoV2

	// The time the change occured.
	Timestamp uint64

	// The type of change.
	Type ChangeType

	// reserved for future use.
	Padding [lineInfoChangedV2PadSize]uint32
}

// LineFlagV2 are the flags for a line.
type LineFlagV2 uint64

const (
	// LineFlagV2Used indicates that the line is already in use.
	// It may have been requested by this process or another process,
	// or may be reserved by the kernel.
	//
	// The line cannot be requested until this flag is clear.
	LineFlagV2Used LineFlagV2 = 1 << iota

	// LineFlagV2ActiveLow indicates that the line is active low.
	LineFlagV2ActiveLow

	// LineFlagV2Input indicates that the line direction is an input.
	LineFlagV2Input

	// LineFlagV2Output indicates that the line direction is an output.
	LineFlagV2Output

	// LineFlagV2EdgeRising indicates that edge detection is enabled for rising
	// edges.
	LineFlagV2EdgeRising

	// LineFlagV2EdgeFalling indicates that edge detection is enabled for
	// falling edges.
	LineFlagV2EdgeFalling

	// LineFlagV2OpenDrain indicates that the line drive is open drain.
	LineFlagV2OpenDrain

	// LineFlagV2OpenSource indicates that the line drive is open source.
	LineFlagV2OpenSource

	// LineFlagV2BiasPullUp indicates that the line bias is pull-up.
	LineFlagV2BiasPullUp

	// LineFlagV2BiasPullDown indicates that the line bias is set pull-down.
	LineFlagV2BiasPullDown

	// LineFlagV2BiasDisabled indicates that the line bias is disabled.
	LineFlagV2BiasDisabled

	LineFlagV2DirectionMask = LineFlagV2Input | LineFlagV2Output

	// LineFlagV2EdgeMask is a helper value for selecting edge detection on
	// both edges.
	LineFlagV2EdgeMask = LineFlagV2EdgeRising | LineFlagV2EdgeFalling
	LineFlagV2EdgeBoth = LineFlagV2EdgeMask

	LineFlagV2DriveMask = LineFlagV2OpenDrain | LineFlagV2OpenSource

	LineFlagV2BiasMask = LineFlagV2BiasDisabled | LineFlagV2BiasPullUp | LineFlagV2BiasPullDown
)

// IsAvailable returns true if the line is available to be requested.
func (f LineFlagV2) IsAvailable() bool {
	return f&LineFlagV2Used == 0
}

// IsUsed returns true if the line is not available to be requested.
func (f LineFlagV2) IsUsed() bool {
	return f&LineFlagV2Used != 0
}

// IsActiveLow returns true if the line is active low.
func (f LineFlagV2) IsActiveLow() bool {
	return f&LineFlagV2ActiveLow != 0
}

// IsInput returns true if the line is an input.
func (f LineFlagV2) IsInput() bool {
	return f&LineFlagV2Input != 0
}

// IsOutput returns true if the line is an output.
func (f LineFlagV2) IsOutput() bool {
	return f&LineFlagV2Output != 0
}

func (f LineFlagV2) IsOpenDrain() bool {
	return f&LineFlagV2OpenDrain != 0
}

func (f LineFlagV2) IsOpenSource() bool {
	return f&LineFlagV2OpenSource != 0
}

func (f LineFlagV2) IsRisingEdge() bool {
	return f&LineFlagV2EdgeRising != 0
}

func (f LineFlagV2) IsFallingEdge() bool {
	return f&LineFlagV2EdgeFalling != 0
}

func (f LineFlagV2) IsBothEdges() bool {
	return f&LineFlagV2EdgeBoth == LineFlagV2EdgeBoth
}

func (f LineFlagV2) IsBiasDisabled() bool {
	return f&LineFlagV2BiasDisabled != 0
}

func (f LineFlagV2) IsBiasPullUp() bool {
	return f&LineFlagV2BiasPullUp != 0
}

func (f LineFlagV2) IsBiasPullDown() bool {
	return f&LineFlagV2BiasPullDown != 0
}

const (
	// LinesMax is the maximum number of lines that can be requested in a single
	// request.
	LinesMax int = 64

	// LinesBitmapSize is number of uint64 in a bitmap large enough for
	// LinesMax.
	LinesBitmapSize int = (LinesMax + 63) / 64

	// the pad sizes of each struct
	lineConfigPadSize        int = 5
	lineRequestPadSize       int = 5
	lineEventPadSize         int = 6
	lineInfoV2PadSize        int = 4
	lineInfoChangedV2PadSize int = 5
)

type LineAttribute struct {
	ID LineAttributeID

	Padding [1]uint32

	Value [8]byte
}

func (la LineAttribute) Value32() uint32 {
	return nativeEndian.Uint32(la.Value[:])
}

func (la LineAttribute) Value64() uint64 {
	return nativeEndian.Uint64(la.Value[:])
}

type LineAttributeID uint32

const (
	LineAttributeIDFlags LineAttributeID = iota + 1

	LineAttributeIDOutputValues

	LineAttributeIDDebounce
)

type DebouncePeriod time.Duration

func (d DebouncePeriod) Encode(attr *LineAttribute) {
	attr.ID = LineAttributeIDDebounce
	nativeEndian.PutUint32(attr.Value[:], uint32(d/1000))
}

func (d DebouncePeriod) Decode(attr LineAttribute) {
	d = DebouncePeriod(attr.Value32() * 1000)
}

func (v LineValues) Encode(attr *LineAttribute) {
	attr.ID = LineAttributeIDOutputValues
	nativeEndian.PutUint64(attr.Value[:], uint64(v[0]))
}

func (v LineValues) Decode(attr LineAttribute) {
	v[0] = attr.Value64()
}

type LineConfigAttribute struct {
	Mask [LinesBitmapSize]uint64

	Attr LineAttribute
}

// LineConfig contains the configuration of a line.
type LineConfig struct {
	// The flags to be applied to the lines.
	Flags LineFlagV2

	NumAttrs uint32

	// reserved for future use.
	Padding [lineConfigPadSize]uint32

	Attrs [10]LineConfigAttribute
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

// LineValues is a bitmap containing the logical value for each line.
//
// Zero is a logical low (inactive) and 1 is a logical high (active).
type LineValues [LinesBitmapSize]uint64

// NewLineValues creates a new LineValues from an array of values.
func NewLineValues(vv ...int) LineValues {
	var lv LineValues
	for i, v := range vv {
		lv.Set(i, v)
	}
	return lv
}

// Get returns the value of the nth bit.
func (lv *LineValues) Get(n int) int {
	idx := n >> 6
	mask := uint64(1) << uint(n%64)
	if lv[idx]&mask != 0 {
		return 1
	}
	return 0
}

// Set sets the value of the nth bit.
func (lv *LineValues) Set(n, v int) {
	idx := n >> 6
	mask := uint64(1) << uint(n%64)
	if v == 0 {
		lv[idx] &^= mask
	} else {
		lv[idx] |= mask
	}
}

type LineSetValues struct {
	Mask [LinesBitmapSize]uint64
	Bits [LinesBitmapSize]uint64
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

	// The seqno for this event in all events on all lines in this line request.
	Seqno uint32

	// The seqno for this event in all events in this line.
	LineSeqno uint32

	// reserved for future use
	Padding [lineEventPadSize]uint32
}
