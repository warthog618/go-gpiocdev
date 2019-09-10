// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package uapi

import (
	"encoding/binary"
	"unsafe"
)

func findEndian() binary.ByteOrder {
	// the standard hack to determine native Endianness.
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)
	switch buf {
	case [2]byte{0xCD, 0xAB}:
		return binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		return binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}
