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
	c := getChip(t, gpiod.WithConsumer("gpiod-test-chip"))
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
	contraOption, option gpiod.LineReqOption, config gpiod.LineConfig) {

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

func testLineDirectionReconfigure(t *testing.T, createOption gpiod.LineReqOption,
	reconfigOption gpiod.LineConfigOption, config gpiod.LineConfig) {

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

	c := getChip(t, gpiod.AsInput)
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

	c := getChip(t, option)
	defer c.Close()

	platform.TriggerIntr(activeLevel)
	ich := make(chan gpiod.LineEvent, 3)
	l, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
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

func testLineLevelOptionInput(t *testing.T, option gpiod.LineReqOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	platform.TriggerIntr(activeLevel)
	ich := make(chan gpiod.LineEvent, 3)
	l, err := c.RequestLine(platform.IntrLine(),
		option,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
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

func testLineLevelOptionOutput(t *testing.T, option gpiod.LineReqOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	c := getChip(t)
	defer c.Close()

	l, err := c.RequestLine(platform.OutLine(), option, gpiod.AsOutput(1))
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

func testLineLevelReconfigure(t *testing.T, createOption gpiod.LineReqOption,
	reconfigOption gpiod.LineConfigOption, isActiveLow bool, activeLevel int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		c := getChip(t)
		defer c.Close()

		l, err := c.RequestLine(platform.OutLine(), createOption, gpiod.AsOutput(1))
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

func testLineDriveOption(t *testing.T, option gpiod.LineReqOption,
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

func testLineDriveReconfigure(t *testing.T, createOption gpiod.LineReqOption,
	reconfigOption gpiod.LineConfigOption, drive gpiod.LineDrive, values ...int) {

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

		c := getChip(t, option)
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

func testLineBiasOption(t *testing.T, option gpiod.LineReqOption,
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

func testLineBiasReconfigure(t *testing.T, createOption gpiod.LineReqOption,
	reconfigOption gpiod.LineConfigOption, bias gpiod.LineBias, expval int) {

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

func TestWithBiasDisabled(t *testing.T) {
	bias := gpiod.LineBiasDisabled
	// can't test value - is indeterminate without external bias.
	testChipBiasOption(t, gpiod.WithBiasDisabled, bias, -1)
	testLineBiasOption(t, gpiod.WithBiasDisabled, bias, -1)
	testLineBiasReconfigure(t, gpiod.WithPullDown, gpiod.WithBiasDisabled, bias, -1)
}

func TestWithBiasAsIs(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t, gpiod.WithConsumer("TestWithBiasAsIs"))
	defer c.Close()
	requireABI(t, c, 2)

	ll := platform.FloatingLines()
	require.GreaterOrEqual(t, len(ll), 5)

	l, err := c.RequestLines(ll,
		gpiod.AsInput,
		gpiod.WithPullDown,
		gpiod.WithLines(
			[]int{ll[2], ll[4]},
			gpiod.WithBiasAsIs,
		),
	)
	assert.Nil(t, err)
	require.NotNil(t, l)

	xinf := gpiod.LineInfo{
		Used:     true,
		Consumer: "TestWithBiasAsIs",
		Offset:   ll[0],
		Config: gpiod.LineConfig{
			Bias:      gpiod.LineBiasPullDown,
			Direction: gpiod.LineDirectionInput,
		},
	}
	inf, err := c.LineInfo(ll[0])
	assert.Nil(t, err)
	xinf.Name = inf.Name // don't care about line name
	assert.Equal(t, xinf, inf)

	inf, err = c.LineInfo(ll[2])
	assert.Nil(t, err)
	xinf.Offset = ll[2]
	xinf.Config.Bias = gpiod.LineBiasUnknown
	xinf.Name = inf.Name // don't care about line name
	assert.Equal(t, xinf, inf)

	l.Close()
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

func TestWithEventHandler(t *testing.T) {
	platform.TriggerIntr(0)

	// via chip options
	ich := make(chan gpiod.LineEvent, 3)
	eh := func(evt gpiod.LineEvent) {
		ich <- evt
	}
	chipOpts := []gpiod.ChipOption{gpiod.WithEventHandler(eh)}
	if kernelAbiVersion != 0 {
		chipOpts = append(chipOpts, gpiod.ABIVersionOption(kernelAbiVersion))
	}
	c := getChip(t, chipOpts...)
	defer c.Close()

	r, err := c.RequestLine(platform.IntrLine(), gpiod.WithBothEdges)
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

	r.Close()

	// via line options
	ich2 := make(chan gpiod.LineEvent, 3)
	eh2 := func(evt gpiod.LineEvent) {
		ich2 <- evt
	}
	r, err = c.RequestLine(platform.IntrLine(),
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(eh2))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich2)
	platform.TriggerIntr(1)
	waitEvent(t, ich2, gpiod.LineEventRisingEdge)
	platform.TriggerIntr(0)
	waitEvent(t, ich2, gpiod.LineEventFallingEdge)
	platform.TriggerIntr(1)
	waitEvent(t, ich2, gpiod.LineEventRisingEdge)
	platform.TriggerIntr(0)
	waitEvent(t, ich2, gpiod.LineEventFallingEdge)
	waitNoEvent(t, ich2)

	r.Close()

	// stub out inherted event handler
	r, err = c.RequestLine(platform.IntrLine(),
		gpiod.WithEventHandler(nil))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	platform.TriggerIntr(1)
	waitNoEvent(t, ich)
	platform.TriggerIntr(0)
	waitNoEvent(t, ich)
}

func TestWithFallingEdge(t *testing.T) {
	platform.TriggerIntr(1)
	c := getChip(t)
	defer c.Close()

	ich := make(chan gpiod.LineEvent, 3)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithFallingEdge,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
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
		gpiod.WithRisingEdge,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
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
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
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

func TestWithLines(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t, gpiod.WithConsumer("TestWithLines"))
	defer c.Close()
	requireABI(t, c, 2)

	ll := platform.FloatingLines()
	require.GreaterOrEqual(t, len(ll), 5)

	patterns := []struct {
		name       string
		reqOptions []gpiod.LineReqOption
		info       map[int]gpiod.LineInfo
	}{
		{"in+out",
			[]gpiod.LineReqOption{
				gpiod.AsInput,
				gpiod.WithPullDown,
				gpiod.WithLines(
					[]int{ll[2], ll[4]},
					gpiod.AsOutput(1, 1),
					gpiod.AsActiveLow,
					gpiod.WithPullUp,
					gpiod.AsOpenDrain,
				),
			},
			map[int]gpiod.LineInfo{
				ll[0]: {
					Config: gpiod.LineConfig{
						Bias:      gpiod.LineBiasPullDown,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Bias:      gpiod.LineBiasPullUp,
						Direction: gpiod.LineDirectionOutput,
						Drive:     gpiod.LineDriveOpenDrain,
					},
				},
			},
		},
		{"in+debounced",
			[]gpiod.LineReqOption{
				gpiod.AsInput,
				gpiod.WithLines(
					[]int{ll[2], ll[4]},
					gpiod.WithDebounce(1234*time.Microsecond),
				),
				gpiod.AsActiveLow,
			},
			map[int]gpiod.LineInfo{
				ll[1]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiod.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1234 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
			},
		},
		{"out+debounced",
			[]gpiod.LineReqOption{
				gpiod.AsOutput(1, 0, 1, 1),
				gpiod.WithLines(
					[]int{ll[2], ll[4]},
					gpiod.WithDebounce(1432*time.Microsecond),
				),
			},
			map[int]gpiod.LineInfo{
				ll[3]: {
					Config: gpiod.LineConfig{
						Direction: gpiod.LineDirectionOutput,
					},
				},
				ll[4]: {
					Config: gpiod.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
			},
		},
		{"debounced+debounced",
			[]gpiod.LineReqOption{
				gpiod.WithDebounce(1234 * time.Microsecond),
				gpiod.WithLines(
					[]int{ll[2], ll[1]},
					gpiod.WithDebounce(1432*time.Microsecond),
				),
			},
			map[int]gpiod.LineInfo{
				ll[0]: {
					Config: gpiod.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1234 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
				ll[1]: {
					Config: gpiod.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
			},
		},
		{"in+out+debounced",
			[]gpiod.LineReqOption{
				gpiod.AsInput,
				gpiod.WithLines(
					[]int{ll[2], ll[4]},
					gpiod.AsOutput(1, 1),
					gpiod.AsActiveLow,
					gpiod.WithPullUp,
					gpiod.AsOpenDrain,
				),
				gpiod.WithLines(
					[]int{ll[3], ll[4]},
					gpiod.WithDebounce(1432*time.Microsecond),
				),
				gpiod.WithPullDown,
			},
			map[int]gpiod.LineInfo{
				ll[0]: {
					Config: gpiod.LineConfig{
						Bias:      gpiod.LineBiasPullDown,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Bias:      gpiod.LineBiasPullUp,
						Direction: gpiod.LineDirectionOutput,
						Drive:     gpiod.LineDriveOpenDrain,
					},
				},
				ll[3]: {
					Config: gpiod.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiod.LineConfig{
						ActiveLow:      true,
						Bias:           gpiod.LineBiasPullUp,
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
			},
		},
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			l, err := c.RequestLines(ll, p.reqOptions...)
			assert.Nil(t, err)
			require.NotNil(t, l)
			for offset, xinf := range p.info {
				inf, err := c.LineInfo(offset)
				xinf.Consumer = "TestWithLines"
				xinf.Used = true
				xinf.Offset = offset
				inf.Name = "" // don't care about line name
				assert.Nil(t, err)
				assert.Equal(t, xinf, inf, offset)
			}
			l.Close()
		}
		t.Run(p.name, tf)
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			l, err := c.RequestLines(ll)
			assert.Nil(t, err)
			require.NotNil(t, l)
			reconfigOpts := []gpiod.LineConfigOption(nil)
			for _, opt := range p.reqOptions {
				// look away - hideous casting in progress
				lco, ok := interface{}(opt).(gpiod.LineConfigOption)
				if ok {
					reconfigOpts = append(reconfigOpts, lco)
				}
			}
			err = l.Reconfigure(reconfigOpts...)
			assert.Nil(t, err)
			for offset, xinf := range p.info {
				inf, err := c.LineInfo(offset)
				xinf.Consumer = "TestWithLines"
				xinf.Used = true
				xinf.Offset = offset
				inf.Name = "" // don't care about line name
				assert.Nil(t, err)
				assert.Equal(t, xinf, inf, offset)
			}
			l.Close()
		}
		t.Run("reconfig-"+p.name, tf)
	}
}

func TestDefaulted(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t, gpiod.WithConsumer("TestDefaulted"))
	defer c.Close()

	ll := platform.FloatingLines()
	require.GreaterOrEqual(t, len(ll), 5)

	patterns := []struct {
		name       string
		reqOptions []gpiod.LineReqOption
		info       map[int]gpiod.LineInfo
		abi        int
	}{
		{"top level",
			[]gpiod.LineReqOption{
				gpiod.AsActiveLow,
				gpiod.WithPullDown,
				gpiod.Defaulted,
				gpiod.AsInput,
			},
			map[int]gpiod.LineInfo{
				ll[0]: {
					Config: gpiod.LineConfig{
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiod.LineConfig{
						Direction: gpiod.LineDirectionInput,
					},
				},
			},
			1,
		},
		{"WithLines",
			[]gpiod.LineReqOption{
				gpiod.AsInput,
				gpiod.WithLines(
					[]int{ll[2], ll[4]},
					gpiod.WithDebounce(1234*time.Microsecond),
				),
				gpiod.WithLines(
					[]int{ll[2]},
					gpiod.Defaulted,
				),
				gpiod.AsActiveLow,
			},
			map[int]gpiod.LineInfo{
				ll[1]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiod.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1234 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
			},
			2,
		},
		{"WithLines nil",
			[]gpiod.LineReqOption{
				gpiod.AsInput,
				gpiod.WithLines(
					[]int{ll[2], ll[4]},
					gpiod.WithDebounce(1234*time.Microsecond),
				),
				gpiod.WithLines(
					[]int(nil),
					gpiod.Defaulted,
				),
				gpiod.AsActiveLow,
			},
			map[int]gpiod.LineInfo{
				ll[1]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
			},
			2,
		},
		{"WithLines empty",
			[]gpiod.LineReqOption{
				gpiod.AsInput,
				gpiod.WithLines(
					[]int{ll[2], ll[4]},
					gpiod.WithDebounce(1234*time.Microsecond),
				),
				gpiod.WithLines(
					[]int{},
					gpiod.Defaulted,
				),
				gpiod.AsActiveLow,
			},
			map[int]gpiod.LineInfo{
				ll[1]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				ll[4]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
			},
			2,
		},
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			if c.UapiAbiVersion() < p.abi {
				t.Skip(ErrorBadABIVersion{p.abi, c.UapiAbiVersion()})
			}
			l, err := c.RequestLines(ll, p.reqOptions...)
			assert.Nil(t, err)
			require.NotNil(t, l)
			for offset, xinf := range p.info {
				inf, err := c.LineInfo(offset)
				xinf.Consumer = "TestDefaulted"
				xinf.Used = true
				xinf.Offset = offset
				inf.Name = "" // don't care about line name
				assert.Nil(t, err)
				assert.Equal(t, xinf, inf, offset)
			}
			l.Close()
		}
		t.Run(p.name, tf)
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			if c.UapiAbiVersion() < p.abi {
				t.Skip(ErrorBadABIVersion{p.abi, c.UapiAbiVersion()})
			}
			l, err := c.RequestLines(ll)
			assert.Nil(t, err)
			require.NotNil(t, l)
			reconfigOpts := []gpiod.LineConfigOption(nil)
			for _, opt := range p.reqOptions {
				// look away - hideous casting in progress
				lco, ok := interface{}(opt).(gpiod.LineConfigOption)
				if ok {
					reconfigOpts = append(reconfigOpts, lco)
				}
			}
			err = l.Reconfigure(reconfigOpts...)
			assert.Nil(t, err)
			for offset, xinf := range p.info {
				inf, err := c.LineInfo(offset)
				xinf.Consumer = "TestDefaulted"
				xinf.Used = true
				xinf.Offset = offset
				inf.Name = "" // don't care about line name
				assert.Nil(t, err)
				assert.Equal(t, xinf, inf, offset)
			}
			l.Close()
		}
		t.Run("reconfig-"+p.name, tf)
	}
}

func TestWithEventBufferSize(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	c := getChip(t)
	defer c.Close()
	requireABI(t, c, 2)

	ll := platform.FloatingLines()

	patterns := []struct {
		name     string
		size     int
		numLines int
	}{
		{"one smaller",
			5,
			1,
		},
		{"one larger",
			25,
			1,
		},
		{"one default",
			0,
			1,
		},
		{"two smaller",
			5,
			1,
		},
		{"two larger",
			35,
			1,
		},
		{"two default",
			0,
			1,
		},
	}

	for _, p := range patterns {
		if p.numLines == 1 {
			t.Run(p.name, func(t *testing.T) {
				l, err := c.RequestLine(ll[0], gpiod.WithEventBufferSize(p.size))
				assert.Nil(t, err)
				require.NotNil(t, l)
				l.Close()
			})
		} else {
			t.Run(p.name, func(t *testing.T) {
				l, err := c.RequestLines(ll[:p.numLines], gpiod.WithEventBufferSize(p.size))
				assert.Nil(t, err)
				require.NotNil(t, l)
				l.Close()
			})
		}
	}
}
