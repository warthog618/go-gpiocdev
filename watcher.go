// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package gpiod

import (
	"fmt"
	"time"

	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

type watcher struct {
	epfd int

	// the handler for detected events
	eh EventHandler

	// pipe to signal watcher to shutdown
	donefds []int

	// closed once watcher exits
	doneCh chan struct{}
}

func newWatcher(fd int32, eh EventHandler) (w *watcher, err error) {
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
	err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, int(fd), &epv)
	if err != nil {
		return
	}
	w = &watcher{
		epfd:    epfd,
		eh:      eh,
		donefds: p,
		doneCh:  make(chan struct{}),
	}
	go w.watch()
	return
}

func (w *watcher) Close() error {
	unix.Write(w.donefds[1], []byte("bye"))
	<-w.doneCh
	unix.Close(w.donefds[0])
	unix.Close(w.donefds[1])
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
			if fd == int32(w.donefds[0]) {
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
			}
			w.eh(le)
		}
	}
}

type watcherV1 struct {
	watcher

	// fd to offset mapping
	evtfds map[int]int
}

func newWatcherV1(fds map[int]int, eh EventHandler) (w *watcherV1, err error) {
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
	for fd := range fds {
		epv.Fd = int32(fd)
		err = unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &epv)
		if err != nil {
			return
		}
	}
	w = &watcherV1{
		watcher: watcher{
			epfd:    epfd,
			eh:      eh,
			donefds: p,
			doneCh:  make(chan struct{}),
		},
		evtfds: fds,
	}
	go w.watch()
	return
}

func (w *watcherV1) Close() error {
	unix.Write(w.donefds[1], []byte("bye"))
	<-w.doneCh
	for fd := range w.evtfds {
		unix.Close(fd)
	}
	unix.Close(w.donefds[0])
	unix.Close(w.donefds[1])
	return nil
}

func (w *watcherV1) watch() {
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
