// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

// Package uapi provides the Linux GPIO UAPI definitions for gpiocdev.
package uapi

import (
	"bytes"
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

// IOCTL command codes
type ioctl uintptr

var (
	getChipInfoIoctl     ioctl
	unwatchLineInfoIoctl ioctl
)

// Size of name and consumer strings.
const nameSize = 32

func init() {
	// ioctls require struct sizes which are only available at runtime.
	var ci ChipInfo
	getChipInfoIoctl = ior(0xB4, 0x01, unsafe.Sizeof(ci))
	var offset uint32
	unwatchLineInfoIoctl = iorw(0xB4, 0x0C, unsafe.Sizeof(offset))
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
