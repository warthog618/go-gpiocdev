// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

// Package uapi provides the Linux GPIO UAPI definitions for gpiod.
package uapi

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/unix"
)

// GetChipInfo returns the ChipInfo for the GPIO character device.
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
// Offsets are zero based.
func GetLineInfo(fd uintptr, offset uint32) (LineInfo, error) {
	var li LineInfo
	li.Offset = offset
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineInfoIoctl),
		uintptr(unsafe.Pointer(&li)))
	if errno != 0 {
		return LineInfo{}, errno
	}
	return li, nil
}

// GetLineEvent requests a line from the GPIO character device with event
// reporting enabled.
func GetLineEvent(fd uintptr, request *EventRequest) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineEventIoctl),
		uintptr(unsafe.Pointer(request)))
	if errno != 0 {
		return errno
	}
	return nil
}

// GetLineHandle requests a line from the GPIO character device.
// This request is without event reporting.
func GetLineHandle(fd uintptr, request *HandleRequest) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineHandleIoctl),
		uintptr(unsafe.Pointer(request)))
	if errno != 0 {
		return errno
	}
	return nil
}

// GetLineValues returns the values of a set of requested lines.
func GetLineValues(fd uintptr, values *HandleData) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(getLineValuesIoctl),
		uintptr(unsafe.Pointer(&values[0])))
	if errno != 0 {
		return errno
	}
	return nil
}

// SetLineValues sets the values of a set of requested lines.
func SetLineValues(fd uintptr, values HandleData) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(setLineValuesIoctl),
		uintptr(unsafe.Pointer(&values[0])))
	if errno != 0 {
		return errno
	}
	return nil
}

type eventReader int

func (fd eventReader) Read(b []byte) (int, error) {
	return unix.Read(int(fd), b[:])
}

// ReadEvent reads a single event from a requested line.
func ReadEvent(fd uintptr) (EventData, error) {
	var ed EventData
	err := binary.Read(eventReader(fd), nativeEndian, &ed)
	return ed, err
}

// IOCTL command codes
type ioctl uintptr

var (
	getChipInfoIoctl   ioctl
	getLineInfoIoctl   ioctl
	getLineHandleIoctl ioctl
	getLineEventIoctl  ioctl
	getLineValuesIoctl ioctl
	setLineValuesIoctl ioctl
)

// Size of name and consumer strings.
const nameSize = 32

// endian to use to decode reads from the local kernel.
var nativeEndian binary.ByteOrder

func init() {
	// ioctls require struct sizes which are only available at runtime.
	var ci ChipInfo
	getChipInfoIoctl = ior(0xB4, 0x01, unsafe.Sizeof(ci))
	var li LineInfo
	getLineInfoIoctl = iorw(0xB4, 0x02, unsafe.Sizeof(li))
	var hr HandleRequest
	getLineHandleIoctl = iorw(0xB4, 0x03, unsafe.Sizeof(hr))
	var le EventRequest
	getLineEventIoctl = iorw(0xB4, 0x04, unsafe.Sizeof(le))
	var hd HandleData
	getLineValuesIoctl = iorw(0xB4, 0x08, unsafe.Sizeof(hd))
	setLineValuesIoctl = iorw(0xB4, 0x09, unsafe.Sizeof(hd))

	// the standard hack to determine native Endianness.
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)
	switch buf {
	case [2]byte{0xCD, 0xAB}:
		nativeEndian = binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		nativeEndian = binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}

// ChipInfo contains the details of a GPIO chip.
type ChipInfo struct {
	Name  [nameSize]byte
	Label [nameSize]byte
	Lines uint32
}

// LineInfo contains the details of a single line of a GPIO chip.
type LineInfo struct {
	Offset   uint32
	Flags    LineFlag
	Name     [nameSize]byte
	Consumer [nameSize]byte
}

// LineFlag are the flags for a line.
type LineFlag uint32

const (
	// LineFlagRequested indicates that the line has been requested.
	// It may have been requested by this process or another process.
	// The line cannot be requested again until this flag is clear.
	LineFlagRequested = LineFlag(1) << iota
	// LineFlagIsOut indicates that the line is an output.
	LineFlagIsOut = LineFlag(1) << iota
	// LineFlagActiveLow indicates that the line is active low.
	LineFlagActiveLow = LineFlag(1) << iota
	// LineFlagOpenDrain indicates that the line will pull low when set low but
	// float when set high. This flag only applies to output lines.
	// An output cannot be both open drain and open source.
	LineFlagOpenDrain = LineFlag(1) << iota
	// LineFlagOpenSource indicates that the line will pull high when set high
	// but float when set low. This flag only applies to output lines.
	// An output cannot be both open drain and open source.
	LineFlagOpenSource = LineFlag(1) << iota
)

// IsRequested returns true if the line is requested.
func (f LineFlag) IsRequested() bool {
	return f&LineFlagRequested != 0
}

// IsOut returns true if the line is an output.
func (f LineFlag) IsOut() bool {
	return f&LineFlagIsOut != 0
}

// IsActiveLow returns true if the line is active low.
func (f LineFlag) IsActiveLow() bool {
	return f&LineFlagActiveLow != 0
}

// IsOpenDrain returns true if the line is open-drain.
func (f LineFlag) IsOpenDrain() bool {
	return f&LineFlagOpenDrain != 0
}

// IsOpenSource returns true if the line is open-source.
func (f LineFlag) IsOpenSource() bool {
	return f&LineFlagOpenSource != 0
}

// HandleRequest is a request for control of a set of lines.
// The lines must all be on the same GPIO chip.
type HandleRequest struct {
	Offsets       [HandlesMax]uint32
	Flags         HandleFlag
	DefaultValues [HandlesMax]byte
	Consumer      [nameSize]byte
	Lines         uint32
	Fd            int32
}

// HandleFlag contains the
type HandleFlag uint32

const (
	// HandleRequestInput requests the line as an input.
	// This cannot be set at the same time as Output, OpenDrain or OpenSource.
	HandleRequestInput = HandleFlag(1) << iota
	// HandleRequestOutput requests the line as an output.
	HandleRequestOutput = HandleFlag(1) << iota
	// HandleRequestActiveLow requests the line be made active low.
	HandleRequestActiveLow = HandleFlag(1) << iota
	// HandleRequestOpenDrain requests the line be made open drain.
	// This cannot be set at the same time as OpenSource.
	HandleRequestOpenDrain = HandleFlag(1) << iota
	// HandleRequestOpenSource requests the line be made open source.
	// This cannot be set at the same time as OpenDrain.
	HandleRequestOpenSource = HandleFlag(1) << iota
	// HandlesMax is the maximum number of lines that can be requested in a
	// single request.
	HandlesMax = 64
)

// IsInput returns true if the line is requested as an input.
func (f HandleFlag) IsInput() bool {
	return f&HandleRequestInput != 0
}

// IsOutput returns true if the line is requested as an output.
func (f HandleFlag) IsOutput() bool {
	return f&HandleRequestOutput != 0
}

// IsActiveLow returns true if the line is requested as a active low.
func (f HandleFlag) IsActiveLow() bool {
	return f&HandleRequestActiveLow != 0
}

// IsOpenDrain returns true if the line is requested as an open drain.
func (f HandleFlag) IsOpenDrain() bool {
	return f&HandleRequestOpenDrain != 0
}

// IsOpenSource returns true if the line is requested as an open source.
func (f HandleFlag) IsOpenSource() bool {
	return f&HandleRequestOpenSource != 0
}

// HandleData contains the logical value for each line.
// Zero is a logical low and any other value is a logical high.
type HandleData [HandlesMax]uint8

// EventRequest is a request for control of a line with event reporting enabled.
type EventRequest struct {
	Offset      uint32
	HandleFlags HandleFlag
	EventFlags  EventFlag
	Consumer    [nameSize]byte
	Fd          int32
}

// EventFlag indicates the types of events that will be reported.
type EventFlag uint32

const (
	// EventRequestRisingEdge requests rising edge events.
	// This means a transition from a low logical state to a high logical state.
	// For active high lines (the default) this means a transition from a
	// physical low to a physical high.
	// Note that for active low lines this means a transition from a physical
	// high to a physical low.
	EventRequestRisingEdge = EventFlag(1) << iota
	// EventRequestFallingEdge requests falling edge events.
	// This means a transition from a high logical state to a low logical state.
	// For active high lines (the default) this means a transition from a
	// physical high to a physical low.
	// Note that for active low lines this means a transition from a physical
	// low to a physical high.
	EventRequestFallingEdge = EventFlag(1) << iota
	// EventRequestBothEdges requests both rising and falling edge events.
	// This is equivalent to requesting both EventRequestRisingEdge and
	// EventRequestRisingEdge.
	EventRequestBothEdges = EventRequestRisingEdge | EventRequestFallingEdge
)

// IsRisingEdge returns true if rising edge events have been requested.
func (f EventFlag) IsRisingEdge() bool {
	return f&EventRequestRisingEdge != 0
}

// IsFallingEdge returns true if falling edge events have been requested.
func (f EventFlag) IsFallingEdge() bool {
	return f&EventRequestFallingEdge != 0
}

// IsBothEdges returns true if both rising and falling edge events have been
// requested.
func (f EventFlag) IsBothEdges() bool {
	return f&EventRequestBothEdges == EventRequestBothEdges
}

// EventData contains the details of a particular line event.
type EventData struct {
	// The time the event was detected.
	Timestamp uint64
	// The type of event detected.
	ID uint32
	// pad
	_ uint32
}

// ioctl constants
const (
	iocNRBits    = 8
	iocTypeBits  = 8
	iocDirBits   = 2
	iocSizeBits  = 14
	iocNRShift   = 0
	iocTypeShift = iocNRShift + iocNRBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits
	iocWrite     = uintptr(1)
	iocRead      = uintptr(2)
)

func ior(t, nr, size uintptr) ioctl {
	return ioctl((iocRead << iocDirShift) |
		(size << iocSizeShift) |
		(t << iocTypeShift) |
		(nr << iocNRShift))
}

func iorw(t, nr, size uintptr) ioctl {
	return ioctl(((iocRead | iocWrite) << iocDirShift) |
		(size << iocSizeShift) |
		(t << iocTypeShift) |
		(nr << iocNRShift))
}
