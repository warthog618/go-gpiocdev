// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.
//
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// +build linux

package gpiod

import (
	"fmt"
	"gpiod/uapi"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type watcher struct {
	epfd    int
	donefds []int
	mu      sync.Mutex
	hh      map[int32]*request
}

func newWatcher() (*watcher, error) {
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
	w := watcher{
		epfd:    epfd,
		hh:      make(map[int32]*request),
		donefds: p,
	}
	go w.watch()
	return &w, nil
}

func (w *watcher) add(r *request) {
	w.mu.Lock()
	w.hh[int32(r.fd)] = r
	w.mu.Unlock()
	epv := unix.EpollEvent{Events: unix.EPOLLIN, Fd: int32(r.fd)}
	unix.EpollCtl(w.epfd, unix.EPOLL_CTL_ADD, int(r.fd), &epv)
}

func (w *watcher) del(r *request) {
	unix.EpollCtl(w.epfd, unix.EPOLL_CTL_DEL, int(r.fd), nil)
	w.mu.Lock()
	delete(w.hh, int32(r.fd))
	w.mu.Unlock()
}

func (w *watcher) close() {
	unix.Write(w.donefds[1], []byte("bye"))
	unix.Close(w.donefds[1])
}

func (w *watcher) watch() {
	var epollEvents [1]unix.EpollEvent
	for {
		n, err := unix.EpollWait(w.epfd, epollEvents[:], -1)
		if err != nil {
			fmt.Println("epoll:", n, err)
			if err == unix.EBADF || err == unix.EINVAL {
				// fd closed so exit
				return
			}
			if err == unix.EINTR {
				continue
			}
			panic(fmt.Sprintf("EpollWait unexpected error: %v", err))
		}
		for _, ev := range epollEvents {
			fd := ev.Fd
			if fd == int32(w.donefds[0]) {
				unix.Close(w.epfd)
				unix.Close(w.donefds[0])
				fmt.Println("watcher exitting")
				return
			}
			w.mu.Lock()
			req := w.hh[fd]
			w.mu.Unlock()
			if req == nil {
				continue
			}
			evt, err := uapi.ReadEvent(uintptr(fd))
			if err != nil {
				fmt.Println("event read error:", err)
				continue
			}
			le := LineEvent{
				Offset:    req.offset,
				Timestamp: time.Duration(evt.Timestamp),
				Type:      LineEventType(evt.ID),
			}
			req.eh(le)
		}
	}
}
