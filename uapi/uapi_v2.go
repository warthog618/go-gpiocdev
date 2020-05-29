// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

// +build linux

package uapi

import (
	"encoding/binary"
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
func SetLineValuesV2(fd uintptr, values LineValues) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL,
		fd,
		uintptr(setLineValuesV2Ioctl),
		uintptr(unsafe.Pointer(&values[0])))
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
	getLineInfoV2Ioctl = iorw(0xB4, 0x0D, unsafe.Sizeof(liv2))
	watchLineInfoV2Ioctl = iorw(0xB4, 0x0E, unsafe.Sizeof(liv2))
	var lr LineRequest
	getLineIoctl = iorw(0xB4, 0x0F, unsafe.Sizeof(lr))
	var lc LineConfig
	setLineConfigV2Ioctl = iorw(0xB4, 0x10, unsafe.Sizeof(lc))
	var lv LineValues
	getLineValuesV2Ioctl = iorw(0xB4, 0x11, unsafe.Sizeof(lv))
	setLineValuesV2Ioctl = iorw(0xB4, 0x12, unsafe.Sizeof(lv))
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

	// The line flags applied to this line.
	Flags LineFlagV2

	// The line direction, if LineFlagV2Direction is set.
	Direction LineDirection

	// The line drive, if LineFlagV2Drive is set.
	Drive LineDrive

	// The line bias, if LineFlagV2Bias is set.
	Bias LineBias

	// The line edge detection, if LineFlagV2EdgeDetection is set.
	EdgeDetection LineEdge

	// The line debounce value, in microseconds, if LineFlagV2Debounce is set.
	Debounce uint32

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
type LineFlagV2 uint32

const (
	// LineFlagV2Busy indicates that the line is already in use.
	// It may have been requested by this process or another process,
	// or may be reserved by the kernel.
	//
	// The line cannot be requested until this flag is clear.
	LineFlagV2Busy LineFlagV2 = 1 << iota

	// LineFlagV2ActiveLow indicates that the line is active low.
	LineFlagV2ActiveLow

	// LineFlagV2Direction indicates that the line direction is set.
	LineFlagV2Direction

	// LineFlagV2Drive indicates that the line drive is set.
	LineFlagV2Drive

	// LineFlagV2Bias indicates that the line bias is set.
	LineFlagV2Bias

	// LineFlagV2EdgeDetection indicates that the line edge detection is set.
	LineFlagV2EdgeDetection

	// LineFlagV2Debounce indicates that the line debounce is set.
	LineFlagV2Debounce
)

// IsAvailable returns true if the line is available to be requested.
func (f LineFlagV2) IsAvailable() bool {
	return f&LineFlagV2Busy == 0
}

// IsBusy returns true if the line is not available to be requested.
func (f LineFlagV2) IsBusy() bool {
	return f&LineFlagV2Busy != 0
}

// IsActiveLow returns true if the line is active low.
func (f LineFlagV2) IsActiveLow() bool {
	return f&LineFlagV2ActiveLow != 0
}

// HasDirection returns true if the line has direction set.
func (f LineFlagV2) HasDirection() bool {
	return f&LineFlagV2Direction != 0
}

// HasDrive returns true if the line has drive set.
func (f LineFlagV2) HasDrive() bool {
	return f&LineFlagV2Drive != 0
}

// HasBias returns true if the line has bias set.
func (f LineFlagV2) HasBias() bool {
	return f&LineFlagV2Bias != 0
}

// HasEdgeDetection returns true if the line has edge detection set.
func (f LineFlagV2) HasEdgeDetection() bool {
	return f&LineFlagV2EdgeDetection != 0
}

// HasDebounce returns true if the line has debounce set.
func (f LineFlagV2) HasDebounce() bool {
	return f&LineFlagV2Debounce != 0
}

// LineDirection indicates the direction of a line.
type LineDirection uint8

const (
	// LineDirectionInput indicates the line is an input.
	LineDirectionInput LineDirection = iota

	// LineDirectionOutput indicates the line is an output.
	LineDirectionOutput
)

// LineDrive indicates the drive of an output line.
type LineDrive uint8

const (
	// LineDrivePushPull indicatges the line is driven in both directions.
	LineDrivePushPull LineDrive = iota

	// LineDriveOpenDrain indicates the line is an open drain output.
	LineDriveOpenDrain

	// LineDriveOpenSource indicates the line is an open souce output.
	LineDriveOpenSource
)

// LineBias indicates the bias of a line.
type LineBias uint8

const (
	// LineBiasDisabled indicates the line bias is disabled.
	LineBiasDisabled LineBias = iota

	// LineBiasPullUp indicates the line has pull up enabled.
	LineBiasPullUp

	// LineBiasPullDown indicates the line has pull down enabled.
	LineBiasPullDown
)

// LineEdge indicates the edges to be detected by edge detection.
type LineEdge uint8

const (
	// LineEdgeNone indicates the line edge detection is disabled.
	LineEdgeNone LineEdge = iota

	// LineEdgeRising indicates the line has rising edge detection enabled.
	LineEdgeRising

	// LineEdgeFalling indicates the line has falling edge detection enabled.
	LineEdgeFalling

	// LineEdgeBoth indicates the line has both rising and falling edge
	// detection enabled.
	LineEdgeBoth
)

const (
	// LinesMax is the maximum number of lines that can be requested in a single
	// request.
	LinesMax int = 64

	// the pad sizes of each struct
	lineConfigPadSize        int = 7
	lineRequestPadSize       int = 4
	lineEventPadSize         int = 2
	lineInfoV2PadSize        int = 12
	lineInfoChangedV2PadSize int = 5
)

// LineConfig contains the configuration of a line.
type LineConfig struct {
	// The default values to be applied to output lines (when
	// HandleRequestOutput is set in the Flags).
	Values [LinesMax]uint8

	// The flags to be applied to the lines.
	Flags LineFlagV2

	// The line direction, if LineFlagV2Direction is set.
	Direction LineDirection

	// The line drive, if LineFlagV2Drive is set.
	Drive LineDrive

	// The line bias, if LineFlagV2Bias is set.
	Bias LineBias

	// The line edge detection, if LineFlagV2EdgeDetection is set.
	EdgeDetection LineEdge

	// The line debounce value, if LineFlagV2Debounce is set.
	Debounce uint32

	// reserved for future use.
	Padding [lineConfigPadSize]uint32
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

	// reserved for future use.
	Padding [lineRequestPadSize]uint32

	// The file handle for the requested lines.
	// Set if the request is successful.
	Fd int32
}

// LineValues contains the logical value for each line.
// Zero is a logical low and any other value is a logical high.
type LineValues [LinesMax]uint8

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

	// reserved for future use
	Padding [lineEventPadSize]uint32
}
