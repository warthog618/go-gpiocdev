// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// Package uapi provides the Linux GPIO UAPI definitions for gpiod.
package uapi

import (
	"bytes"
	"encoding/binary"
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
	var li LineInfo
	li.Offset = uint32(offset)
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
//
// The fd is an open GPIO character device.
// The line must be an input and must not already be requested.
// If successful, the fd for the line is returned in the request.fd.
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
//
// This request is without event reporting.
// The fd is an open GPIO character device.
// The lines must not already be requested.
// The flags in the request will be applied to all lines in the request.
// If successful, the fd for the line is returned in the request.fd.
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
//
// The fd is a requested line, as returned by GetLineHandle or GetLineEvent.
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
//
// The fd is a requested line, as returned by GetLineHandle or GetLineEvent.
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

// SetLineConfig sets the config of an existing handle request.
//
// The config flags in the request will be applied to all lines in the handle
// request.
func SetLineConfig(fd uintptr, config *HandleConfig) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(setLineConfigIoctl),
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
		uintptr(watchLineInfoIoctl),
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

// BytesToString is a helper function that converts strings stored in byte
// arrays, as returned by GetChipInfo and GetLineInfo, into strings.
func BytesToString(a []byte) string {
	n := bytes.IndexByte(a, 0)
	if n == -1 {
		return string(a)
	}
	return string(a[:n])
}

type fdReader int

func (fd fdReader) Read(b []byte) (int, error) {
	return unix.Read(int(fd), b[:])
}

// ReadEvent reads a single event from a requested line.
//
// The fd is a requested line, as returned by GetLineEvent.
//
// This function is blocking and should only be called when the fd is known to
// be ready to read.
func ReadEvent(fd uintptr) (EventData, error) {
	var ed EventData
	err := binary.Read(fdReader(fd), nativeEndian, &ed)
	return ed, err
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
	getLineInfoIoctl     ioctl
	getLineHandleIoctl   ioctl
	getLineEventIoctl    ioctl
	getLineValuesIoctl   ioctl
	setLineValuesIoctl   ioctl
	setLineConfigIoctl   ioctl
	watchLineInfoIoctl   ioctl
	unwatchLineInfoIoctl ioctl
)

// Size of name and consumer strings.
const nameSize = 32

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
	var hc HandleConfig
	setLineConfigIoctl = iorw(0xB4, 0x0a, unsafe.Sizeof(hc))
	watchLineInfoIoctl = iorw(0xB4, 0x0b, unsafe.Sizeof(li))
	unwatchLineInfoIoctl = iorw(0xB4, 0x0c, unsafe.Sizeof(li.Offset))
}

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
	// The offset of the line within the chip.
	Offset uint32

	// The line flags applied to this line.
	Flags LineFlag

	// The system name for this line.
	Name [nameSize]byte

	// If requested, a string added by the requester to identify the
	// owner of the request.
	Consumer [nameSize]byte
}

// LineInfoChanged contains the details of a change to line info.
//
// This is returned via the chip fd in response to changes to watched lines.
type LineInfoChanged struct {
	// The updated info.
	Info LineInfo

	// The time the change occured.
	Timestamp uint64

	// The type of change.
	Type ChangeType

	// reserved for future use.
	_ [5]uint32
}

// ChangeType indicates the type of change that has occured to a line.
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

// LineFlag are the flags for a line.
type LineFlag uint32

const (
	// LineFlagRequested indicates that the line has been requested.
	// It may have been requested by this process or another process.
	// The line cannot be requested again until this flag is clear.
	LineFlagRequested LineFlag = 1 << iota

	// LineFlagIsOut indicates that the line is an output.
	LineFlagIsOut

	// LineFlagActiveLow indicates that the line is active low.
	LineFlagActiveLow

	// LineFlagOpenDrain indicates that the line will pull low when set low but
	// float when set high. This flag only applies to output lines.
	// An output cannot be both open drain and open source.
	LineFlagOpenDrain

	// LineFlagOpenSource indicates that the line will pull high when set high
	// but float when set low. This flag only applies to output lines.
	// An output cannot be both open drain and open source.
	LineFlagOpenSource

	// LineFlagPullUp indicates that the internal line pull up is enabled.
	LineFlagPullUp

	// LineFlagPullDown indicates that the internal line pull down is enabled.
	LineFlagPullDown

	// LineFlagBiasDisable indicates that the internal line bias is disabled.
	LineFlagBiasDisable
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

// IsBiasDisable returns true if the line has bias disabled.
func (f LineFlag) IsBiasDisable() bool {
	return f&LineFlagBiasDisable != 0
}

// IsPullDown returns true if the line has pull-down enabled.
func (f LineFlag) IsPullDown() bool {
	return f&LineFlagPullDown != 0
}

// IsPullUp returns true if the line has pull-up enabled.
func (f LineFlag) IsPullUp() bool {
	return f&LineFlagPullUp != 0
}

// HandleConfig is a request to change the config of an existing request.
//
// Can be applied to both handle and event requests.
// Event requests cannot be reconfigured to outputs.
type HandleConfig struct {
	// The flags to be applied to the lines.
	Flags HandleFlag

	// The default values to be applied to output lines (when
	// HandleRequestOutput is set in the Flags).
	DefaultValues [HandlesMax]uint8

	// reserved for future use.
	_ [4]uint32
}

// HandleRequest is a request for control of a set of lines.
// The lines must all be on the same GPIO chip.
type HandleRequest struct {
	// The lines to be requested.
	Offsets [HandlesMax]uint32

	// The flags to be applied to the lines.
	Flags HandleFlag

	// The default values to be applied to output lines.
	DefaultValues [HandlesMax]uint8

	// The string identifying the requester to be applied to the lines.
	Consumer [nameSize]byte

	// The number of lines being requested.
	Lines uint32

	// The file handle for the requested lines.
	// Set if the request is successful.
	Fd int32
}

// HandleFlag contains the
type HandleFlag uint32

const (
	// HandleRequestInput requests the line as an input.
	//
	// This is ignored if Output is also set.
	HandleRequestInput HandleFlag = 1 << iota

	// HandleRequestOutput requests the line as an output.
	//
	// This takes precedence over Input, if both are set.
	HandleRequestOutput

	// HandleRequestActiveLow requests the line be made active low.
	HandleRequestActiveLow

	// HandleRequestOpenDrain requests the line be made open drain.
	//
	// This option requires the line to be requested as an Output.
	// This cannot be set at the same time as OpenSource.
	HandleRequestOpenDrain

	// HandleRequestOpenSource requests the line be made open source.
	//
	// This option requires the line to be requested as an Output.
	// This cannot be set at the same time as OpenDrain.
	HandleRequestOpenSource

	// HandleRequestPullUp requests the line have pull-up enabled.
	HandleRequestPullUp

	// HandleRequestPullDown requests the line have pull-down enabled.
	HandleRequestPullDown

	// HandleRequestBiasDisable requests the line have bias disabled.
	HandleRequestBiasDisable

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

// IsBiasDisable returns true if the line is requested with bias disabled.
func (f HandleFlag) IsBiasDisable() bool {
	return f&HandleRequestBiasDisable != 0
}

// IsPullDown returns true if the line is requested with pull-down enabled.
func (f HandleFlag) IsPullDown() bool {
	return f&HandleRequestPullDown != 0
}

// IsPullUp returns true if the line is requested with pull-up enabled.
func (f HandleFlag) IsPullUp() bool {
	return f&HandleRequestPullUp != 0
}

// HandleData contains the logical value for each line.
// Zero is a logical low and any other value is a logical high.
type HandleData [HandlesMax]uint8

// EventRequest is a request for control of a line with event reporting enabled.
type EventRequest struct {
	// The line to be requested.
	Offset uint32

	// The line flags applied to this line.
	HandleFlags HandleFlag

	// The type of events to report.
	EventFlags EventFlag

	// The string identifying the requester to be applied to the line.
	Consumer [nameSize]byte

	// The file handle for the requested line.
	// Set if the request is successful.
	Fd int32
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
	EventRequestRisingEdge EventFlag = 1 << iota

	// EventRequestFallingEdge requests falling edge events.
	// This means a transition from a high logical state to a low logical state.
	// For active high lines (the default) this means a transition from a
	// physical high to a physical low.
	// Note that for active low lines this means a transition from a physical
	// low to a physical high.
	EventRequestFallingEdge

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
//
// This is returned via the event request fd in response to events.
type EventData struct {
	// The time the event was detected.
	Timestamp uint64

	// The type of event detected.
	ID uint32

	// pad to workaround 64bit OS padding
	_ uint32
}

