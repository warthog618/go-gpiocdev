// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

// +build arm arm64 386 amd64

package uapi

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
	iocWrite     = 1
	iocRead      = 2
)
