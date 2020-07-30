// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

// +build mips mipsle mips64 mips64le ppc64 ppc64le sparc sparc64

package uapi

// ioctl constants
const (
	iocNRBits    = 8
	iocTypeBits  = 8
	iocDirBits   = 3
	iocSizeBits  = 13
	iocNRShift   = 0
	iocTypeShift = iocNRShift + iocNRBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits
	iocWrite     = 4
	iocRead      = 2
	// iocNone = 1
)
