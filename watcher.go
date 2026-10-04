// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package gpiocdev

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"

	"github.com/warthog618/go-gpiocdev/uapi"
)

type watcher struct {
	epfd int

	// eventfd to signal watcher to shutdown
	donefd int

	// the handler for detected events
	eh EventHandler

	// closed once watcher exits
	doneCh chan struct{}
}

func newWatcher(fd int32, eh EventHandler) (w *watcher, err error) {
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
	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(fd), &epv)
	if err != nil {
		return
	}
	w = &watcher{
		epfd:   epfd,
		donefd: donefd,
		eh:     eh,
		doneCh: make(chan struct{}),
	}
	go w.watch()
	return
}

func (w *watcher) Close() error {
	unix.Write(w.donefd, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	<-w.doneCh
	unix.Close(w.donefd)
	return nil
}

func (w *watcher) watch() {
	epollEvents := make([]unix.EpollEvent, 2)
	defer close(w.doneCh)
	for {
		n, err := unix.EpollWait(w.epfd, epollEvents[:], -1)
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
			if fd == int32(w.donefd) {
				unix.Close(w.epfd)
				return
			}
			evt, err := uapi.ReadLineEvent(uintptr(fd))
			if err != nil {
				continue
			}
			le := LineEvent{
				Offset:    int(evt.Offset),
				Timestamp: time.Duration(evt.Timestamp),
				Type:      LineEventType(evt.ID),
				Seqno:     evt.Seqno,
				LineSeqno: evt.LineSeqno,
			}
			w.eh(le)
		}
	}
}
