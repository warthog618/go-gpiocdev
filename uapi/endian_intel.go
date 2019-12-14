// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build 386 amd64

package uapi

import (
	"encoding/binary"
)

// endian to use to decode reads from the local kernel.
var nativeEndian binary.ByteOrder = binary.LittleEndian
