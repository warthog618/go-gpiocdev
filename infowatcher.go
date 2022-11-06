// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package gpiocdev

import (
	"fmt"
	"time"

	"github.com/warthog618/go-gpiocdev/uapi"
	"golang.org/x/sys/unix"
)

type infoWatcher struct {
	epfd int

	// eventfd to signal watcher to shutdown
	donefd int

	// the handler for detected events
	ch InfoChangeHandler

	// closed once watcher exits
	doneCh chan struct{}

	abi int
}

func newInfoWatcher(fd int, ch InfoChangeHandler, abi int) (iw *infoWatcher, err error) {
	var epfd, donefd int
	epfd, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			unix.Close(epfd)
		}
	}()
	donefd, err = unix.Eventfd(0, unix.EFD_CLOEXEC)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			unix.Close(donefd)
		}
	}()
	epv := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(donefd)}
	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(donefd), &epv)
	if err != nil {
		return
	}
	epv.Fd = int32(fd)
	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &epv)
	if err != nil {
		return
	}
	iw = &infoWatcher{
		epfd:   epfd,
		donefd: donefd,
		ch:     ch,
		doneCh: make(chan struct{}),
		abi:    abi,
	}
	go iw.watch()
	return
}

func (iw *infoWatcher) close() {
	unix.Write(iw.donefd, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	<-iw.doneCh
	unix.Close(iw.donefd)
}

func (iw *infoWatcher) watch() {
	epollEvents := make([]unix.EpollEvent, 2)
	defer close(iw.doneCh)
	for {
		n, err := unix.EpollWait(iw.epfd, epollEvents[:], -1)
		if err != nil {
			if err == unix.EBADF || err == unix.EINVAL {
				// fd closed so exit
				return
			}
			if err == unix.EINTR {
				continue
			}
			panic(fmt.Sprintf("EpollWait unexpected error: %v", err))
		}
		for i := 0; i < n; i++ {
			ev := epollEvents[i]
			fd := ev.Fd
			if fd == int32(iw.donefd) {
				unix.Close(iw.epfd)
				return
			}
			if iw.abi == 1 {
				iw.readInfoChanged(fd)
			} else {
				iw.readInfoChangedV2(fd)
			}
		}
	}
}

func (iw *infoWatcher) readInfoChanged(fd int32) {
	lic, err := uapi.ReadLineInfoChanged(uintptr(fd))
	if err != nil {
		fmt.Printf("error reading line change:%s\n", err)
		return
	}
	lice := LineInfoChangeEvent{
		Info:      newLineInfo(lic.Info),
		Timestamp: time.Duration(lic.Timestamp),
		Type:      LineInfoChangeType(lic.Type),
	}
	iw.ch(lice)

}

func (iw *infoWatcher) readInfoChangedV2(fd int32) {
	lic, err := uapi.ReadLineInfoChangedV2(uintptr(fd))
	if err != nil {
		fmt.Printf("error reading line change:%s\n", err)
		return
	}
	lice := LineInfoChangeEvent{
		Info:      newLineInfoV2(lic.Info),
		Timestamp: time.Duration(lic.Timestamp),
		Type:      LineInfoChangeType(lic.Type),
	}
	iw.ch(lice)
}
