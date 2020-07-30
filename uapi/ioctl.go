// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

// +build linux

package uapi

// ioctl constants defined in ioctl_XXX

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

func iow(t, nr, size uintptr) ioctl {
	return ioctl((iocWrite << iocDirShift) |
		(size << iocSizeShift) |
		(t << iocTypeShift) |
		(nr << iocNRShift))
}
