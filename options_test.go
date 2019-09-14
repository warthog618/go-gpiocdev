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

	// default from chip
	c, err := gpiod.NewChip(platform.Devpath(),
		gpiod.WithConsumer("gpiod-test-chip"))
	assert.Nil(t, err)
	require.NotNil(t, c)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, "gpiod-test-chip", inf.Consumer)
	err = l.Close()
	assert.Nil(t, err)

	// overridden by line
	l, err = c.RequestLine(platform.IntrLine(),
		gpiod.WithConsumer("gpiod-test-line"))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err = c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, "gpiod-test-line", inf.Consumer)
}

func TestAsIs(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	// leave input as input
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsIs())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	l.Close()

	// leave output as output
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsIs())
	assert.Nil(t, err)
	require.NotNil(t, l)
	l.Close()
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	// !!! this fails on Raspberry Pi, but passes on mockup...
	assert.True(t, inf.IsOut)
	l.Close()
}

func TestAsInput(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	// leave input as input
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	l.Close()

	// change output to input
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	l.Close()
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	l.Close()
}

func TestAsOutput(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	// change input to output
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	l.Close()

	// leave output as input
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	l.Close()
}

func TestAsActiveLow(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	// input - both Value and Events are reverse polarity
	platform.TriggerIntr(0)
	ich := make(chan gpiod.LineEvent)
	l, err := c.RequestLine(platform.IntrLine(),
		gpiod.AsActiveLow(),
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.True(t, inf.ActiveLow)
	start := time.Now()
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventFallingEdge, start)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	start = time.Now()
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventRisingEdge, start)
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)

	err = l.Close()
	assert.Nil(t, err)

	// output - Value and SetValue are reverse polarity
	// need platform to have a loopback or raw read of floating line??
}

func TestAsOpenDrain(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.FloatingLines()[0],
		gpiod.AsOpenDrain())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.OpenDrain)
	// Testing physical behaviour requires specific hardware, so assume that is
	// covered by the kernel anyway.
}

func TestAsOpenSource(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.FloatingLines()[0],
		gpiod.AsOpenSource())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.OpenSource)
	// Testing physical behaviour requires specific hardware, so assume that is
	// covered by the kernel anyway.
}

func TestWithFallingEdge(t *testing.T) {
	requirePlatform(t)
	platform.TriggerIntr(1)
	c := getChip(t)
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
	c := getChip(t)
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
	c := getChip(t)
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
