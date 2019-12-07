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

func testLineDirectionOption(t *testing.T,
	contraOption, option gpiod.LineOption, info gpiod.LineInfo) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	// change direction
	l, err := c.RequestLine(platform.FloatingLines()[0], contraOption)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.NotEqual(t, info.IsOut, inf.IsOut)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, info.IsOut, inf.IsOut)
	err = l.Close()
	assert.Nil(t, err)

	// same direction
	l, err = c.RequestLine(platform.FloatingLines()[0], option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, info.IsOut, inf.IsOut)
	err = l.Close()
	assert.Nil(t, err)
}

func testLineDirectionReconfigure(t *testing.T, createOption gpiod.LineOption,
	reconfigOption gpiod.LineConfig, info gpiod.LineInfo) {

	t.Helper()

	c := getChip(t)
	defer c.Close()
	// reconfigure direction change
	l, err := c.RequestLine(platform.FloatingLines()[0], createOption)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.NotEqual(t, info.IsOut, inf.IsOut)
	l.Reconfigure(reconfigOption)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, info.IsOut, inf.IsOut)
	err = l.Close()
	assert.Nil(t, err)
}

func TestAsInput(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{IsOut: false}
	testLineDirectionOption(t, gpiod.AsOutput(), gpiod.AsInput, info)
	testLineDirectionReconfigure(t, gpiod.AsOutput(), gpiod.AsInput, info)
}

func TestAsOutput(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{IsOut: true}
	testLineDirectionOption(t, gpiod.AsInput, gpiod.AsOutput(), info)
	testLineDirectionReconfigure(t, gpiod.AsInput, gpiod.AsOutput(), info)

}

func testEdgeEventPolarity(t *testing.T, l *gpiod.Line,
	ich <-chan gpiod.LineEvent, activeLevel int) {

	t.Helper()

	start := time.Now()
	platform.TriggerIntr(activeLevel ^ 1)
	waitEvent(t, ich, gpiod.LineEventFallingEdge, start)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	start = time.Now()
	platform.TriggerIntr(activeLevel)
	waitEvent(t, ich, gpiod.LineEventRisingEdge, start)
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	err = l.Close()
	assert.Nil(t, err)
}

func testChipLevelOption(t *testing.T, option gpiod.ChipOption,
	info gpiod.LineInfo, activeLevel int) {

	t.Helper()

	c, err := gpiod.NewChip(platform.Devpath(), option)

	assert.Nil(t, err)
	require.NotNil(t, c)
	defer c.Close()

	platform.TriggerIntr(activeLevel)
	ich := make(chan gpiod.LineEvent)
	l, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, info.ActiveLow, inf.ActiveLow)

	// test correct edge polarity in events
	testEdgeEventPolarity(t, l, ich, activeLevel)
}

func testLineLevelOptionInput(t *testing.T, option gpiod.LineOption,
	info gpiod.LineInfo, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	platform.TriggerIntr(activeLevel)
	ich := make(chan gpiod.LineEvent)
	l, err := c.RequestLine(platform.IntrLine(),
		option,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()

	testEdgeEventPolarity(t, l, ich, activeLevel)
}

func testLineLevelOptionOutput(t *testing.T, option gpiod.LineOption,
	info gpiod.LineInfo, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(),
		option, gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, info.ActiveLow, inf.ActiveLow)
	v := platform.ReadOut()
	assert.Equal(t, activeLevel, v)
	err = l.SetValue(0)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, activeLevel^1, v)
	err = l.SetValue(1)
	assert.Nil(t, err)
	v = platform.ReadOut()
	assert.Equal(t, activeLevel, v)
	err = l.Close()
	assert.Nil(t, err)
}

func testLineLevelReconfigure(t *testing.T, createOption gpiod.LineOption,
	reconfigOption gpiod.LineConfig, info gpiod.LineInfo, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(),
		createOption, gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	v := platform.ReadOut()
	assert.Equal(t, activeLevel^1, v)
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.NotEqual(t, info.ActiveLow, inf.ActiveLow)
	l.Reconfigure(reconfigOption)
	inf, err = c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, info.ActiveLow, inf.ActiveLow)
	v = platform.ReadOut()
	assert.Equal(t, activeLevel, v)
	err = l.Close()
	assert.Nil(t, err)
}

func TestAsActiveLow(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{ActiveLow: true}
	testChipLevelOption(t, gpiod.AsActiveLow, info, 0)
	testLineLevelOptionInput(t, gpiod.AsActiveLow, info, 0)
	testLineLevelOptionOutput(t, gpiod.AsActiveLow, info, 0)
	testLineLevelReconfigure(t, gpiod.AsActiveHigh, gpiod.AsActiveLow, info, 0)
}

func TestAsActiveHigh(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{ActiveLow: false}
	testChipLevelOption(t, gpiod.AsActiveHigh, info, 1)
	testLineLevelOptionInput(t, gpiod.AsActiveHigh, info, 1)
	testLineLevelOptionOutput(t, gpiod.AsActiveHigh, info, 1)
	testLineLevelReconfigure(t, gpiod.AsActiveLow, gpiod.AsActiveHigh, info, 1)
}

func testLineDriveOption(t *testing.T, option gpiod.LineOption,
	info gpiod.LineInfo, values ...int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(),
		gpiod.AsOutput(1), option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, info.OpenDrain, inf.OpenDrain)
	assert.Equal(t, info.OpenSource, inf.OpenSource)
	for _, sv := range values {
		err = l.SetValue(sv)
		assert.Nil(t, err)
		v := platform.ReadOut()
		assert.Equal(t, sv, v)
	}
}

func testLineDriveReconfigure(t *testing.T, createOption gpiod.LineOption,
	reconfigOption gpiod.LineConfig, info gpiod.LineInfo, values ...int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(),
		createOption, gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	err = l.Reconfigure(reconfigOption)
	assert.Nil(t, err)
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, info.OpenDrain, inf.OpenDrain)
	assert.Equal(t, info.OpenSource, inf.OpenSource)
	for _, sv := range values {
		err = l.SetValue(sv)
		assert.Nil(t, err)
		v := platform.ReadOut()
		assert.Equal(t, sv, v)
	}
}

func TestAsOpenDrain(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{OpenDrain: true}
	// Testing float high requires specific hardware, so assume that is
	// covered by the kernel anyway...
	testLineDriveOption(t, gpiod.AsOpenDrain, info, 0)
	testLineDriveReconfigure(t, gpiod.AsOpenSource, gpiod.AsOpenDrain, info, 0)
}

func TestAsOpenSource(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{OpenSource: true}
	// Testing float low requires specific hardware, so assume that is
	// covered by the kernel anyway.
	testLineDriveOption(t, gpiod.AsOpenSource, info, 1)
	testLineDriveReconfigure(t, gpiod.AsOpenDrain, gpiod.AsOpenSource, info, 1)
}

func TestAsPushPull(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{}
	testLineDriveOption(t, gpiod.AsPushPull, info, 0, 1)
	testLineDriveReconfigure(t, gpiod.AsOpenDrain, gpiod.AsPushPull, info, 0, 1)
}

func testChipBiasOption(t *testing.T, option gpiod.ChipOption,
	info gpiod.LineInfo, expval int) {

	t.Helper()

	c, err := gpiod.NewChip(platform.Devpath(), option)
	assert.Nil(t, err)
	require.NotNil(t, c)
	defer c.Close()

	l, err := c.RequestLine(platform.FloatingLines()[0],
		gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, info.BiasDisable, inf.BiasDisable)
	assert.Equal(t, info.PullUp, inf.PullUp)
	assert.Equal(t, info.PullDown, inf.PullDown)

	if expval == -1 {
		return
	}
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, expval, v)
}

func testLineBiasOption(t *testing.T, option gpiod.LineOption,
	info gpiod.LineInfo, expval int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.FloatingLines()[0],
		gpiod.AsInput, option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, info.BiasDisable, inf.BiasDisable)
	assert.Equal(t, info.PullUp, inf.PullUp)
	assert.Equal(t, info.PullDown, inf.PullDown)
	if expval == -1 {
		return
	}
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, expval, v)
}

func testLineBiasReconfigure(t *testing.T, createOption gpiod.LineOption,
	reconfigOption gpiod.LineConfig, info gpiod.LineInfo, expval int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.FloatingLines()[0],
		createOption, gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	l.Reconfigure(reconfigOption)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, info.BiasDisable, inf.BiasDisable)
	assert.Equal(t, info.PullUp, inf.PullUp)
	assert.Equal(t, info.PullDown, inf.PullDown)
	if expval == -1 {
		return
	}
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, expval, v)
}

func TestWithBiasDisable(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{BiasDisable: true}
	// can't test value - is indeterminate without external bias.
	testChipBiasOption(t, gpiod.WithBiasDisable, info, -1)
	testLineBiasOption(t, gpiod.WithBiasDisable, info, -1)
	testLineBiasReconfigure(t, gpiod.WithPullDown, gpiod.WithBiasDisable, info, -1)
}

func TestWithPullDown(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{PullDown: true}
	testChipBiasOption(t, gpiod.WithPullDown, info, 0)
	testLineBiasOption(t, gpiod.WithPullDown, info, 0)
	testLineBiasReconfigure(t, gpiod.WithPullUp, gpiod.WithPullDown, info, 0)
}
func TestWithPullUp(t *testing.T) {
	requirePlatform(t)

	info := gpiod.LineInfo{PullUp: true}
	testChipBiasOption(t, gpiod.WithPullUp, info, 1)
	testLineBiasOption(t, gpiod.WithPullUp, info, 1)
	testLineBiasReconfigure(t, gpiod.WithPullDown, gpiod.WithPullUp, info, 1)
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

func waitEvent(t *testing.T, ch <-chan gpiod.LineEvent, etype gpiod.LineEventType, start time.Time) {
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
