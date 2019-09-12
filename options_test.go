// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package gpiod_test

import (
	"gpiod"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithConsumer(t *testing.T) {
	requirePlatform(t)
}

func TestAsIs(t *testing.T) {
	requirePlatform(t)
}

func TestAsInput(t *testing.T) {
	requirePlatform(t)
}

func TestAsOutput(t *testing.T) {
	requirePlatform(t)
}

func TestAsActiveLow(t *testing.T) {
	requirePlatform(t)
}

func TestAsOpenDrain(t *testing.T) {
	requirePlatform(t)
}

func TestAsOpenSource(t *testing.T) {
	requirePlatform(t)
}

func TestWithFallingEdge(t *testing.T) {
	requirePlatform(t)
	platform.TriggerIntr(1)
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(t, err)
	require.NotNil(t, c)
	defer c.Close()
	ich := make(chan gpiod.LineEvent)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithFallingEdge(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	start := time.Now()
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge, start)
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
}

func TestWithRisingEdge(t *testing.T) {
	requirePlatform(t)
	platform.TriggerIntr(0)
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(t, err)
	require.NotNil(t, c)
	defer c.Close()
	ich := make(chan gpiod.LineEvent)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithRisingEdge(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	start := time.Now()
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge, start)
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
}

func TestWithBothEdges(t *testing.T) {
	requirePlatform(t)
	platform.TriggerIntr(0)
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(t, err)
	require.NotNil(t, c)
	defer c.Close()
	ich := make(chan gpiod.LineEvent)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	start := time.Now()
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge, start)
	start = time.Now()
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge, start)
}

func waitEvent(t *testing.T, ch chan gpiod.LineEvent, etype gpiod.LineEventType, start time.Time) {
	t.Helper()
	select {
	case evt := <-ch:
		end := time.Now()
		assert.Equal(t, etype, evt.Type)
		assert.LessOrEqual(t, start.UnixNano(), evt.Timestamp.Nanoseconds())
		assert.GreaterOrEqual(t, end.UnixNano(), evt.Timestamp.Nanoseconds())
	case <-time.After(10 * time.Millisecond):
		assert.Fail(t, "timeout waiting for event")
	}
}

func waitNoEvent(t *testing.T, ch chan gpiod.LineEvent) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Fail(t, "received unexpected event", evt)
	case <-time.After(20 * time.Millisecond):
	}
}
