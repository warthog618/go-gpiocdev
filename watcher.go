// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package gpiod

import (
	"fmt"
	"time"

	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

type watcher struct {
	epfd int

	// fd to offset mapping
	evtfds map[int]int

	// the handler for detected events
	eh EventHandler

	// pipe to signal watcher to shutdown
	donefds []int

	// closed once watcher exits
	doneCh chan struct{}
}

func newWatcher(fds map[int]int, eh EventHandler) (*watcher, error) {
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}
	p := []int{0, 0}
	err = unix.Pipe2(p, unix.O_CLOEXEC)
	if err != nil {
		return nil, err
	}
	epv := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(p[0])}
	unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(p[0]), &epv)
	for fd := range fds {
		epv.Fd = int32(fd)
		unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &epv)
	}
	w := watcher{
		epfd:    epfd,
		donefds: p,
		evtfds:  fds,
		eh:      eh,
		doneCh:  make(chan struct{}),
	}
	go w.watch()
	return &w, nil
}

func (w *watcher) close() {
	for fd := range w.evtfds {
		unix.EpollCtl(w.epfd, unix.EPOLL_CTL_DEL, fd, nil)
		unix.Close(fd)
	}
	unix.Write(w.donefds[1], []byte("bye"))
	<-w.doneCh
	unix.Close(w.donefds[1])
}

func (w *watcher) watch() {
	epollEvents := make([]unix.EpollEvent, len(w.evtfds))
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
			if fd == int32(w.donefds[0]) {
				unix.Close(w.epfd)
				unix.Close(w.donefds[0])
				return
			}
			evt, err := uapi.ReadEvent(uintptr(fd))
			if err != nil {
				continue
			}
			le := LineEvent{
				Offset:    w.evtfds[int(fd)],
				Timestamp: time.Duration(evt.Timestamp),
				Type:      LineEventType(evt.ID),
			}
			w.eh(le)
		}
	}
}
