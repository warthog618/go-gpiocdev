// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// don't build on platforms with fixed endianness
// +build !amd64
// +build !386

package uapi

import (
	"encoding/binary"
	"unsafe"
)

// endian to use to decode reads from the local kernel.
var nativeEndian binary.ByteOrder

func init() {
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
