// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package gpiod_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod"
)

func TestWithConsumer(t *testing.T) {
	requirePlatform(t)

	assert.Implements(t, (*gpiod.ChipOption)(nil), gpiod.WithConsumer(""))
	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.WithConsumer(""))

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

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.AsIs)

	// leave input as input
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsIs)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	err = l.Close()
	assert.Nil(t, err)

	// leave output as output
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsIs)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	// !!! this fails on Raspberry Pi, but passes on mockup...
	assert.True(t, inf.IsOut)
}

func TestAsInput(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.AsInput)
	assert.Implements(t, (*gpiod.LineConfig)(nil), gpiod.AsInput)

	// change output to input
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)

	// leave input as input
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	err = l.Close()
	assert.Nil(t, err)

	// reconfigure output to input
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	l.Reconfigure(gpiod.AsInput)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	err = l.Close()
	assert.Nil(t, err)
}

func TestAsOutput(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.AsOutput())
	assert.Implements(t, (*gpiod.LineConfig)(nil), gpiod.AsOutput())

	// change input to output
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput)
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
	err = l.Close()
	assert.Nil(t, err)

	// leave output as input
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	err = l.Close()
	assert.Nil(t, err)

	// reconfigure input to output
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.False(t, inf.IsOut)
	l.Reconfigure(gpiod.AsOutput(1))
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.IsOut)
	err = l.Close()
	assert.Nil(t, err)
}

func TestAsActiveLow(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.AsActiveLow)

	// input - both Value and Events are reverse polarity
	platform.TriggerIntr(0)
	ich := make(chan gpiod.LineEvent)
	l, err := c.RequestLine(platform.IntrLine(),
		gpiod.AsActiveLow,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
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

	// output - initial value and SetValue are reverse polarity
	l, err = c.RequestLine(platform.OutLine(),
		gpiod.AsActiveLow,
		gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.True(t, inf.ActiveLow)
	v = platform.ReadOut()
	assert.Equal(t, 0, v)
	err = l.SetValue(0)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, 1, v)
	err = l.SetValue(1)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, 0, v)
	err = l.Close()
	assert.Nil(t, err)

	// reconfigure as active low
	l, err = c.RequestLine(platform.OutLine(),
		gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.False(t, inf.ActiveLow)
	l.Reconfigure(gpiod.AsActiveLow)
	inf, err = c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.True(t, inf.ActiveLow)
	err = l.Close()
	assert.Nil(t, err)
}

func TestAsActiveHigh(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.AsActiveLow)

	// input - both Value and Events are same polarity
	platform.TriggerIntr(0)
	ich := make(chan gpiod.LineEvent)
	l, err := c.RequestLine(platform.IntrLine(),
		gpiod.AsActiveHigh,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.False(t, inf.ActiveLow)
	start := time.Now()
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge, start)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	start = time.Now()
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge, start)
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	err = l.Close()
	assert.Nil(t, err)

	// output - initial value and SetValue are same polarity
	l, err = c.RequestLine(platform.OutLine(),
		gpiod.AsActiveHigh,
		gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.False(t, inf.ActiveLow)
	v = platform.ReadOut()
	assert.Equal(t, 1, v)
	err = l.SetValue(0)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, 0, v)
	err = l.SetValue(1)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, 1, v)
	err = l.Close()
	assert.Nil(t, err)

	// reconfigure as active high
	l, err = c.RequestLine(platform.OutLine(),
		gpiod.AsActiveLow,
		gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.True(t, inf.ActiveLow)
	l.Reconfigure(gpiod.AsActiveHigh)
	inf, err = c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.False(t, inf.ActiveLow)
	err = l.Close()
	assert.Nil(t, err)
}

func TestAsOpenDrain(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.AsOpenDrain)
	assert.Implements(t, (*gpiod.LineConfig)(nil), gpiod.AsOpenDrain)

	l, err := c.RequestLine(platform.OutLine(),
		gpiod.AsOutput(1),
		gpiod.AsOpenDrain)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.True(t, inf.OpenDrain)
	err = l.SetValue(0)
	assert.Nil(t, err)
	v := platform.ReadOut()
	assert.Equal(t, 0, v)
	// Testing float high requires specific hardware, so assume that is
	// covered by the kernel anyway.
}

func TestAsOpenSource(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.AsOpenSource)
	assert.Implements(t, (*gpiod.LineConfig)(nil), gpiod.AsOpenSource)

	l, err := c.RequestLine(platform.OutLine(),
		gpiod.AsOutput(0),
		gpiod.AsOpenSource)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.True(t, inf.OpenSource)
	err = l.SetValue(1)
	assert.Nil(t, err)
	v := platform.ReadOut()
	assert.Equal(t, 1, v)
	// Testing float low requires specific hardware, so assume that is
	// covered by the kernel anyway.
}

func TestAsPushPull(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.AsPushPull)
	assert.Implements(t, (*gpiod.LineConfig)(nil), gpiod.AsPushPull)

	l, err := c.RequestLine(platform.OutLine(),
		gpiod.AsOutput(0),
		gpiod.AsPushPull)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.False(t, inf.OpenSource)
	assert.False(t, inf.OpenDrain)
	v := platform.ReadOut()
	assert.Equal(t, 0, v)
	err = l.SetValue(1)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, 1, v)
	err = l.SetValue(0)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, 0, v)
}

func TestWithBiasDisable(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.WithBiasDisable)
	assert.Implements(t, (*gpiod.LineConfig)(nil), gpiod.WithBiasDisable)

	l, err := c.RequestLine(platform.FloatingLines()[0],
		gpiod.AsInput,
		gpiod.WithBiasDisable)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.BiasDisable)
	// can't test value - is indeterminate without external bias.
}

func TestWithPullDown(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.WithPullDown)
	assert.Implements(t, (*gpiod.LineConfig)(nil), gpiod.WithPullDown)

	l, err := c.RequestLine(platform.FloatingLines()[0],
		gpiod.AsInput,
		gpiod.WithPullDown)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.PullDown)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
}
func TestWithPullUp(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.WithPullUp)
	assert.Implements(t, (*gpiod.LineConfig)(nil), gpiod.WithPullUp)

	l, err := c.RequestLine(platform.FloatingLines()[0],
		gpiod.AsInput,
		gpiod.WithPullUp)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.True(t, inf.PullUp)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
}

func TestWithFallingEdge(t *testing.T) {
	requirePlatform(t)
	platform.TriggerIntr(1)
	c := getChip(t)
	defer c.Close()

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.WithFallingEdge(nil))

	ich := make(chan gpiod.LineEvent)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithFallingEdge(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	start := time.Now()
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge, start)
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
	start = time.Now()
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

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.WithRisingEdge(nil))

	ich := make(chan gpiod.LineEvent)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithRisingEdge(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	start := time.Now()
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge, start)
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
	start = time.Now()
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

	assert.Implements(t, (*gpiod.LineOption)(nil), gpiod.WithBothEdges(nil))

	ich := make(chan gpiod.LineEvent)
	lines := append(platform.FloatingLines(), platform.IntrLine())
	r, err := c.RequestLines(lines,
		gpiod.WithPullDown,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	start := time.Now()
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge, start)
	start = time.Now()
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge, start)
	start = time.Now()
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge, start)
	start = time.Now()
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge, start)
	waitNoEvent(t, ich)
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
