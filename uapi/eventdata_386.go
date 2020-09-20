// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

// +build linux

// Package uapi provides the Linux GPIO UAPI definitions for gpiod.
package uapi

// EventData contains the details of a particular line event.
//
// This is returned via the event request fd in response to events.
type EventData struct {
	// The time the event was detected.
	Timestamp uint64

	// The type of event detected.
	ID EventFlag

	// No pad required for i386.
}
