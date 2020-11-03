// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package gpiod_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod"
)

func TestWithConsumer(t *testing.T) {
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
	if !platform.SupportsAsIs() {
		t.Skip("platform doesn't support as-is")
	}
	c := getChip(t)
	defer c.Close()

	// leave input as input
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionInput, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsIs)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionInput, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)

	// leave output as output
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionOutput, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsIs)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionOutput, inf.Config.Direction)
}

func testLineDirectionOption(t *testing.T,
	contraOption, option gpiod.LineOption, config gpiod.LineConfig) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	// change direction
	l, err := c.RequestLine(platform.FloatingLines()[0], contraOption)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.NotEqual(t, config.Direction, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(platform.FloatingLines()[0], option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, config.Direction, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)

	// same direction
	l, err = c.RequestLine(platform.FloatingLines()[0], option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(platform.FloatingLines()[0])
	assert.Nil(t, err)
	assert.Equal(t, config.Direction, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)
}

func testLineDirectionReconfigure(t *testing.T, createOption gpiod.LineOption,
	reconfigOption gpiod.LineReconfig, config gpiod.LineConfig) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		c := getChip(t)
		defer c.Close()
		// reconfigure direction change
		l, err := c.RequestLine(platform.FloatingLines()[0], createOption)
		assert.Nil(t, err)
		require.NotNil(t, l)
		inf, err := c.LineInfo(platform.FloatingLines()[0])
		assert.Nil(t, err)
		assert.NotEqual(t, config.Direction, inf.Config.Direction)
		l.Reconfigure(reconfigOption)
		inf, err = c.LineInfo(platform.FloatingLines()[0])
		assert.Nil(t, err)
		assert.Equal(t, config.Direction, inf.Config.Direction)
		err = l.Close()
		assert.Nil(t, err)
	}
	t.Run("Reconfigure", tf)
}

func TestAsInput(t *testing.T) {
	config := gpiod.LineConfig{
		Direction: gpiod.LineDirectionInput,
	}
	testChipAsInputOption(t)
	testLineDirectionOption(t, gpiod.AsOutput(), gpiod.AsInput, config)
	testLineDirectionReconfigure(t, gpiod.AsOutput(), gpiod.AsInput, config)
}

func TestAsOutput(t *testing.T) {
	config := gpiod.LineConfig{
		Direction: gpiod.LineDirectionOutput,
	}
	testLineDirectionOption(t, gpiod.AsInput, gpiod.AsOutput(), config)
	testLineDirectionReconfigure(t, gpiod.AsInput, gpiod.AsOutput(), config)
}

func testEdgeEventPolarity(t *testing.T, l *gpiod.Line,
	ich <-chan gpiod.LineEvent, activeLevel int) {

	t.Helper()

	platform.TriggerIntr(activeLevel ^ 1)
	waitEvent(t, ich, gpiod.LineEventFallingEdge)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	platform.TriggerIntr(activeLevel)
	waitEvent(t, ich, gpiod.LineEventRisingEdge)
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	err = l.Close()
	assert.Nil(t, err)
}

func testChipAsInputOption(t *testing.T) {

	t.Helper()

	c, err := gpiod.NewChip(platform.Devpath(), gpiod.AsInput)

	assert.Nil(t, err)
	require.NotNil(t, c)
	defer c.Close()

	// force line to output
	l, err := c.RequestLine(platform.OutLine(),
		gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	l.Close()

	// request with Chip default
	l, err = c.RequestLine(platform.OutLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionInput, inf.Config.Direction)
}

func testChipLevelOption(t *testing.T, option gpiod.ChipOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	c, err := gpiod.NewChip(platform.Devpath(), option)

	assert.Nil(t, err)
	require.NotNil(t, c)
	defer c.Close()

	platform.TriggerIntr(activeLevel)
	ich := make(chan gpiod.LineEvent, 3)
	l, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)

	// can get initial state events on some platforms (e.g. RPi AsActiveHigh)
	clearEvents(ich)

	// test correct edge polarity in events
	testEdgeEventPolarity(t, l, ich, activeLevel)
}

func testLineLevelOptionInput(t *testing.T, option gpiod.LineOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	platform.TriggerIntr(activeLevel)
	ich := make(chan gpiod.LineEvent, 3)
	l, err := c.RequestLine(platform.IntrLine(),
		option,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)

	testEdgeEventPolarity(t, l, ich, activeLevel)
}

func testLineLevelOptionOutput(t *testing.T, option gpiod.LineOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(),
		option, gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(platform.OutLine())
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)
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
	reconfigOption gpiod.LineReconfig, isActiveLow bool, activeLevel int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

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
		assert.NotEqual(t, isActiveLow, inf.Config.ActiveLow)
		l.Reconfigure(reconfigOption)
		inf, err = c.LineInfo(platform.OutLine())
		assert.Nil(t, err)
		assert.Equal(t, isActiveLow, inf.Config.ActiveLow)
		v = platform.ReadOut()
		assert.Equal(t, activeLevel, v)
		err = l.Close()
		assert.Nil(t, err)
	}
	t.Run("Reconfigure", tf)
}

func TestAsActiveLow(t *testing.T) {
	testChipLevelOption(t, gpiod.AsActiveLow, true, 0)
	testLineLevelOptionInput(t, gpiod.AsActiveLow, true, 0)
	testLineLevelOptionOutput(t, gpiod.AsActiveLow, true, 0)
	testLineLevelReconfigure(t, gpiod.AsActiveHigh, gpiod.AsActiveLow, true, 0)
}

func TestAsActiveHigh(t *testing.T) {
	testChipLevelOption(t, gpiod.AsActiveHigh, false, 1)
	testLineLevelOptionInput(t, gpiod.AsActiveHigh, false, 1)
	testLineLevelOptionOutput(t, gpiod.AsActiveHigh, false, 1)
	testLineLevelReconfigure(t, gpiod.AsActiveLow, gpiod.AsActiveHigh, false, 1)
}

func testLineDriveOption(t *testing.T, option gpiod.LineOption,
	drive gpiod.LineDrive, values ...int) {

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
	assert.Equal(t, drive, inf.Config.Drive)
	for _, sv := range values {
		err = l.SetValue(sv)
		assert.Nil(t, err)
		v := platform.ReadOut()
		assert.Equal(t, sv, v)
	}
}

func testLineDriveReconfigure(t *testing.T, createOption gpiod.LineOption,
	reconfigOption gpiod.LineReconfig, drive gpiod.LineDrive, values ...int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

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
		assert.Equal(t, drive, inf.Config.Drive)
		for _, sv := range values {
			err = l.SetValue(sv)
			assert.Nil(t, err)
			v := platform.ReadOut()
			assert.Equal(t, sv, v)
		}
	}
	t.Run("Reconfigure", tf)
}

func TestAsOpenDrain(t *testing.T) {
	drive := gpiod.LineDriveOpenDrain
	// Testing float high requires specific hardware, so assume that is
	// covered by the kernel anyway...
	testLineDriveOption(t, gpiod.AsOpenDrain, drive, 0)
	testLineDriveReconfigure(t, gpiod.AsOpenSource, gpiod.AsOpenDrain, drive, 0)
}

func TestAsOpenSource(t *testing.T) {
	drive := gpiod.LineDriveOpenSource
	// Testing float low requires specific hardware, so assume that is
	// covered by the kernel anyway.
	testLineDriveOption(t, gpiod.AsOpenSource, drive, 1)
	testLineDriveReconfigure(t, gpiod.AsOpenDrain, gpiod.AsOpenSource, drive, 1)
}

func TestAsPushPull(t *testing.T) {
	drive := gpiod.LineDrivePushPull
	testLineDriveOption(t, gpiod.AsPushPull, drive, 0, 1)
	testLineDriveReconfigure(t, gpiod.AsOpenDrain, gpiod.AsPushPull, drive, 0, 1)
}

func testChipBiasOption(t *testing.T, option gpiod.ChipOption,
	bias gpiod.LineBias, expval int) {

	tf := func(t *testing.T) {
		requireKernel(t, biasKernel)

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
		assert.Equal(t, bias, inf.Config.Bias)

		if expval == -1 {
			return
		}
		v, err := l.Value()
		assert.Nil(t, err)
		assert.Equal(t, expval, v)
	}
	t.Run("Chip", tf)
}

func testLineBiasOption(t *testing.T, option gpiod.LineOption,
	bias gpiod.LineBias, expval int) {

	tf := func(t *testing.T) {
		requireKernel(t, biasKernel)

		c := getChip(t)
		defer c.Close()
		l, err := c.RequestLine(platform.FloatingLines()[0],
			gpiod.AsInput, option)
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		inf, err := c.LineInfo(platform.FloatingLines()[0])
		assert.Nil(t, err)
		assert.Equal(t, bias, inf.Config.Bias)
		if expval == -1 {
			return
		}
		v, err := l.Value()
		assert.Nil(t, err)
		assert.Equal(t, expval, v)
	}
	t.Run("Line", tf)
}

func testLineBiasReconfigure(t *testing.T, createOption gpiod.LineOption,
	reconfigOption gpiod.LineReconfig, bias gpiod.LineBias, expval int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

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
		assert.Equal(t, bias, inf.Config.Bias)
		if expval == -1 {
			return
		}
		v, err := l.Value()
		assert.Nil(t, err)
		assert.Equal(t, expval, v)
	}
	t.Run("Reconfigure", tf)
}

func TestWithBiasDisable(t *testing.T) {
	bias := gpiod.LineBiasDisabled
	// can't test value - is indeterminate without external bias.
	testChipBiasOption(t, gpiod.WithBiasDisabled, bias, -1)
	testLineBiasOption(t, gpiod.WithBiasDisabled, bias, -1)
	testLineBiasReconfigure(t, gpiod.WithPullDown, gpiod.WithBiasDisabled, bias, -1)
}

func TestWithPullDown(t *testing.T) {
	bias := gpiod.LineBiasPullDown
	testChipBiasOption(t, gpiod.WithPullDown, bias, 0)
	testLineBiasOption(t, gpiod.WithPullDown, bias, 0)
	testLineBiasReconfigure(t, gpiod.WithPullUp, gpiod.WithPullDown, bias, 0)
}
func TestWithPullUp(t *testing.T) {
	bias := gpiod.LineBiasPullUp
	testChipBiasOption(t, gpiod.WithPullUp, bias, 1)
	testLineBiasOption(t, gpiod.WithPullUp, bias, 1)
	testLineBiasReconfigure(t, gpiod.WithPullDown, gpiod.WithPullUp, bias, 1)
}

func TestWithFallingEdge(t *testing.T) {
	platform.TriggerIntr(1)
	c := getChip(t)
	defer c.Close()

	ich := make(chan gpiod.LineEvent, 3)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithFallingEdge(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge)
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge)
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
}

func TestWithRisingEdge(t *testing.T) {
	platform.TriggerIntr(0)
	c := getChip(t)
	defer c.Close()

	ich := make(chan gpiod.LineEvent, 3)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithRisingEdge(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge)
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge)
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
}

func TestWithBothEdges(t *testing.T) {
	platform.TriggerIntr(0)
	c := getChip(t)
	defer c.Close()

	ich := make(chan gpiod.LineEvent, 3)
	lines := append(platform.FloatingLines(), platform.IntrLine())
	r, err := c.RequestLines(lines,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge)
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge)
	platform.TriggerIntr(1)
	waitEvent(t, ich, gpiod.LineEventRisingEdge)
	platform.TriggerIntr(0)
	waitEvent(t, ich, gpiod.LineEventFallingEdge)
	waitNoEvent(t, ich)
}

func waitEvent(t *testing.T, ch <-chan gpiod.LineEvent, etype gpiod.LineEventType) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Equal(t, etype, evt.Type)
	case <-time.After(time.Second):
		assert.Fail(t, "timeout waiting for event")
	}
}

func waitNoEvent(t *testing.T, ch <-chan gpiod.LineEvent) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Fail(t, "received unexpected event", evt)
	case <-time.After(20 * time.Millisecond):
	}
}

func clearEvents(ch <-chan gpiod.LineEvent) {
	select {
	case <-ch:
	case <-time.After(20 * time.Millisecond):
	}
}

func TestWithDebounce(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithDebounce(10*time.Microsecond))

	if c.UapiAbiVersion() == 1 {
		xerr := gpiod.ErrUapiIncompatibility{"debounce", 1}
		assert.Equal(t, xerr, err)
		assert.Nil(t, l)
		return
	}
	require.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()

	inf, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionInput, inf.Config.Direction)
	assert.True(t, inf.Config.Debounced)
	assert.Equal(t, 10*time.Microsecond, inf.Config.DebouncePeriod)
}
