// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

package gpiod

import (
	"fmt"
	"time"

	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

type infoWatcher struct {
	epfd int

	// the handler for detected events
	ch InfoChangeHandler

	// pipe to signal watcher to shutdown
	donefds []int

	// closed once watcher exits
	doneCh chan struct{}
}

func newInfoWatcher(fd int, ch InfoChangeHandler) (iw *infoWatcher, err error) {
	var epfd int
	epfd, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			unix.Close(epfd)
		}
	}()
	p := []int{0, 0}
	err = unix.Pipe2(p, unix.O_CLOEXEC)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			unix.Close(p[0])
			unix.Close(p[1])
		}
	}()
	epv := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(p[0])}
	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(p[0]), &epv)
	if err != nil {
		return
	}
	epv.Fd = int32(fd)
	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &epv)
	if err != nil {
		return
	}
	iw = &infoWatcher{
		epfd:    epfd,
		ch:      ch,
		donefds: p,
		doneCh:  make(chan struct{}),
	}
	go iw.watch()
	return
}

func (iw *infoWatcher) close() {
	unix.Write(iw.donefds[1], []byte("bye"))
	<-iw.doneCh
	unix.Close(iw.donefds[0])
	unix.Close(iw.donefds[1])
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
			if fd == int32(iw.donefds[0]) {
				unix.Close(iw.epfd)
				return
			}
			lic, err := uapi.ReadLineInfoChanged(uintptr(fd))
			if err != nil {
				continue
			}
			lice := LineInfoChangeEvent{
				Info:      newLineInfo(lic.Info),
				Timestamp: time.Duration(lic.Timestamp),
				Type:      LineInfoChangeType(lic.Type),
			}
			iw.ch(lice)
		}
	}
}
