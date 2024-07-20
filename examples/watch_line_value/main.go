// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

// A simple example that watches an input pin and reports edge events.
package main

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/warthog618/go-gpiocdev"
)

func eventHandler(evt gpiocdev.LineEvent) {
	t := time.Now()
	edge := "rising"
	if evt.Type == gpiocdev.LineEventFallingEdge {
		edge = "falling"
	}
	if evt.Seqno != 0 {
		// only uAPI v2 populates the sequence numbers
		fmt.Printf("event: #%d(%d)%3d %-7s %s (%s)\n",
			evt.Seqno,
			evt.LineSeqno,
			evt.Offset,
			edge,
			t.Format(time.RFC3339Nano),
			evt.Timestamp)
	} else {
		fmt.Printf("event:%3d %-7s %s (%s)\n",
			evt.Offset,
			edge,
			t.Format(time.RFC3339Nano),
			evt.Timestamp)
	}
}

// Watches gpiochip0:23 and reports when it changes state.
func main() {
	offset := 23
	chip := "gpiochip0"
	l, err := gpiocdev.RequestLine(chip, offset,
		gpiocdev.WithPullUp,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(eventHandler))
	if err != nil {
		fmt.Printf("RequestLine returned error: %s\n", err)
		if err == syscall.Errno(22) {
			fmt.Println("Note that the WithPullUp option requires Linux 5.5 or later - check your kernel version.")
		}
		os.Exit(1)
	}
	defer l.Close()

	// In a real application the main thread would do something useful.
	// But we'll just run for a minute then exit.
	fmt.Printf("Watching Pin %s:%d...\n", chip, offset)
	time.Sleep(time.Minute)
	fmt.Println("watch_line_value exiting...")
}
