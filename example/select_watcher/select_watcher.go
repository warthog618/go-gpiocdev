// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// A simple example that watches an input pin and reports edge events.
// This is a version of the watcher example that performs the watching within
// a select.
package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/device/rpi"
)

func printEvent(evt gpiocdev.LineEvent) {
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

// Watches GPIO 23 (Raspberry Pi J8-16) and reports when it changes state.
func main() {
	echan := make(chan gpiocdev.LineEvent, 6)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	eh := func(evt gpiocdev.LineEvent) {
		select {
		case echan <- evt: // the expected path
		default:
			// if you want the handler to block, rather than dropping
			// events when the channel fills then <- ctx.Done() instead
			// to ensure that the handler can't be left blocked
			fmt.Printf("event chan overflow - discarding event")
		}
	}

	offset := rpi.J8p16
	l, err := gpiocdev.RequestLine("gpiochip0", offset,
		gpiocdev.WithPullUp,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(eh))
	if err != nil {
		fmt.Printf("RequestLine returned error: %s\n", err)
		if err == syscall.Errno(22) {
			fmt.Println("Note that the WithPullUp option requires kernel V5.5 or later - check your kernel version.")
		}
		os.Exit(1)
	}

	fmt.Printf("Watching Pin %d...\n", offset)
	done := false
	for !done {
		select {
		// depending on the application other cases could deal with other channels
		case evt := <-echan:
			printEvent(evt)
		case <-ctx.Done():
			fmt.Println("exiting...")
			l.Close()
			done = true
		}
	}
	cancel()
}
