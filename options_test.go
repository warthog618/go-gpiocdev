// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package gpiod_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiosim"
	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

func TestWithConsumer(t *testing.T) {
	offset := 3
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	// default from chip
	c := getChip(t, s.DevPath(), gpiod.WithConsumer("gpiod-test-chip"))
	defer c.Close()
	l, err := c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, "gpiod-test-chip", inf.Consumer)
	err = l.Close()
	assert.Nil(t, err)

	// overridden by line
	l, err = c.RequestLine(offset,
		gpiod.WithConsumer("gpiod-test-line"))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, "gpiod-test-line", inf.Consumer)
}

func TestAsIs(t *testing.T) {
	offset := 0
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	// leave input as input
	l, err := c.RequestLine(offset, gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionInput, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(offset, gpiod.AsIs)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionInput, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)

	// leave output as output
	l, err = c.RequestLine(offset, gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionOutput, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(offset, gpiod.AsIs)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionOutput, inf.Config.Direction)
}

func testLineDirectionOption(t *testing.T,
	contraOption, option gpiod.LineReqOption, config gpiod.LineConfig) {

	t.Helper()

	offset := 2
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	// change direction
	l, err := c.RequestLine(offset, contraOption)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.NotEqual(t, config.Direction, inf.Config.Direction)
	l.Close()
	l, err = c.RequestLine(offset, option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, config.Direction, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)

	// same direction
	l, err = c.RequestLine(offset, option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, config.Direction, inf.Config.Direction)
	err = l.Close()
	assert.Nil(t, err)
}

func testLineDirectionReconfigure(t *testing.T, createOption gpiod.LineReqOption,
	reconfigOption gpiod.LineConfigOption, config gpiod.LineConfig) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		offset := 4
		s, err := gpiosim.NewSimpleton(6)
		require.Nil(t, err)
		defer s.Close()
		c := getChip(t, s.DevPath())
		defer c.Close()
		// reconfigure direction change
		l, err := c.RequestLine(offset, createOption)
		assert.Nil(t, err)
		require.NotNil(t, l)
		inf, err := c.LineInfo(offset)
		assert.Nil(t, err)
		assert.NotEqual(t, config.Direction, inf.Config.Direction)
		l.Reconfigure(reconfigOption)
		inf, err = c.LineInfo(offset)
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

func testEdgeEventPolarity(t *testing.T, s *gpiosim.Simpleton, l *gpiod.Line,
	ich <-chan gpiod.LineEvent, activeLevel int, seqno uint32) {
	t.Helper()

	evtSeqno = seqno
	s.SetPull(l.Offset(), activeLevel^1)
	waitEvent(t, ich, nextEvent(l, 0))
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	s.SetPull(l.Offset(), activeLevel)
	waitEvent(t, ich, nextEvent(l, 1))
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	err = l.Close()
	assert.Nil(t, err)
}

func testChipAsInputOption(t *testing.T) {
	t.Helper()

	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath(), gpiod.AsInput)
	defer c.Close()

	// force line to output
	l, err := c.RequestLine(offset, gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	l.Close()

	// request with Chip default
	l, err = c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionInput, inf.Config.Direction)
}

func testChipLevelOption(t *testing.T, option gpiod.ChipOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath(), gpiod.AsInput, option)
	defer c.Close()

	s.SetPull(offset, activeLevel)
	ich := make(chan gpiod.LineEvent, 3)
	l, err := c.RequestLine(offset,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)

	// can get initial state events on some platforms (e.g. RPi AsActiveHigh)
	seqno := clearEvents(ich)

	// test correct edge polarity in events
	testEdgeEventPolarity(t, s, l, ich, activeLevel, seqno)
}

func testLineLevelOptionInput(t *testing.T, option gpiod.LineReqOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	s.SetPull(offset, activeLevel)
	ich := make(chan gpiod.LineEvent, 3)
	l, err := c.RequestLine(offset,
		option,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)

	testEdgeEventPolarity(t, s, l, ich, activeLevel, 0)
}

func testLineLevelOptionOutput(t *testing.T, option gpiod.LineReqOption,
	isActiveLow bool, activeLevel int) {

	t.Helper()

	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	l, err := c.RequestLine(offset, option, gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, isActiveLow, inf.Config.ActiveLow)
	v, err := s.Level(offset)
	assert.Nil(t, err)
	assert.Equal(t, activeLevel, v)
	err = l.SetValue(0)
	assert.Nil(t, err)
	v, err = s.Level(offset)
	assert.Nil(t, err)
	assert.Equal(t, activeLevel^1, v)
	err = l.SetValue(1)
	assert.Nil(t, err)
	v, err = s.Level(offset)
	assert.Nil(t, err)
	assert.Equal(t, activeLevel, v)
	err = l.Close()
	assert.Nil(t, err)
}

func testLineLevelReconfigure(t *testing.T, createOption gpiod.LineReqOption,
	reconfigOption gpiod.LineConfigOption, isActiveLow bool, activeLevel int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		offset := 4
		s, err := gpiosim.NewSimpleton(6)
		require.Nil(t, err)
		defer s.Close()
		c := getChip(t, s.DevPath())
		defer c.Close()

		l, err := c.RequestLine(offset, createOption, gpiod.AsOutput(1))
		assert.Nil(t, err)
		require.NotNil(t, l)
		v, err := s.Level(offset)
		assert.Nil(t, err)
		assert.Equal(t, activeLevel^1, v)
		inf, err := c.LineInfo(offset)
		assert.Nil(t, err)
		assert.NotEqual(t, isActiveLow, inf.Config.ActiveLow)
		l.Reconfigure(reconfigOption)
		inf, err = c.LineInfo(offset)
		assert.Nil(t, err)
		assert.Equal(t, isActiveLow, inf.Config.ActiveLow)
		v, err = s.Level(offset)
		assert.Nil(t, err)
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

func testChipDriveOption(t *testing.T, option gpiod.ChipOption,
	drive gpiod.LineDrive, values ...int) {

	t.Helper()

	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath(), option)
	defer c.Close()

	l, err := c.RequestLine(offset, gpiod.AsOutput(1))
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(offset)
	t.Logf("li: %v", inf)
	assert.Nil(t, err)
	assert.Equal(t, drive, inf.Config.Drive)
	for _, sv := range values {
		err = l.SetValue(sv)
		assert.Nil(t, err)
		v, err := s.Level(offset)
		assert.Nil(t, err)
		assert.Equal(t, sv, v)
	}
}

func testLineDriveOption(t *testing.T, option gpiod.LineReqOption,
	drive gpiod.LineDrive, values ...int) {

	t.Helper()

	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	l, err := c.RequestLine(offset, gpiod.AsOutput(1), option)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, drive, inf.Config.Drive)
	for _, sv := range values {
		err = l.SetValue(sv)
		assert.Nil(t, err)
		v, err := s.Level(offset)
		assert.Nil(t, err)
		assert.Equal(t, sv, v)
	}
}

func testLineDriveReconfigure(t *testing.T, createOption gpiod.LineReqOption,
	reconfigOption gpiod.LineConfigOption, drive gpiod.LineDrive, values ...int) {

	tf := func(t *testing.T) {
		requireKernel(t, setConfigKernel)

		offset := 4
		s, err := gpiosim.NewSimpleton(6)
		require.Nil(t, err)
		defer s.Close()
		c := getChip(t, s.DevPath())
		defer c.Close()

		l, err := c.RequestLine(offset,	createOption, gpiod.AsOutput(1))
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		err = l.Reconfigure(reconfigOption)
		assert.Nil(t, err)
		inf, err := c.LineInfo(offset)
		assert.Nil(t, err)
		assert.Equal(t, drive, inf.Config.Drive)
		for _, sv := range values {
			err = l.SetValue(sv)
			assert.Nil(t, err)
			v, err := s.Level(offset)
			assert.Nil(t, err)
			assert.Equal(t, sv, v)
		}
	}
	t.Run("Reconfigure", tf)
}

func TestAsOpenDrain(t *testing.T) {
	drive := gpiod.LineDriveOpenDrain
	// Testing float high requires specific hardware, so assume that is
	// covered by the kernel anyway...
	testChipDriveOption(t, gpiod.AsOpenDrain, drive, 0)
	testLineDriveOption(t, gpiod.AsOpenDrain, drive, 0)
	testLineDriveReconfigure(t, gpiod.AsOpenSource, gpiod.AsOpenDrain, drive, 0)
}

func TestAsOpenSource(t *testing.T) {
	drive := gpiod.LineDriveOpenSource
	// Testing float low requires specific hardware, so assume that is
	// covered by the kernel anyway.
	testChipDriveOption(t, gpiod.AsOpenSource, drive, 1)
	testLineDriveOption(t, gpiod.AsOpenSource, drive, 1)
	testLineDriveReconfigure(t, gpiod.AsOpenDrain, gpiod.AsOpenSource, drive, 1)
}

func TestAsPushPull(t *testing.T) {
	drive := gpiod.LineDrivePushPull
	testChipDriveOption(t, gpiod.AsPushPull, drive, 0, 1)
	testLineDriveOption(t, gpiod.AsPushPull, drive, 0, 1)
	testLineDriveReconfigure(t, gpiod.AsOpenDrain, gpiod.AsPushPull, drive, 0, 1)
}

func testChipBiasOption(t *testing.T, option gpiod.ChipOption,
	bias gpiod.LineBias, expval int) {

	tf := func(t *testing.T) {
		requireKernel(t, biasKernel)

		offset := 4
		s, err := gpiosim.NewSimpleton(6)
		require.Nil(t, err)
		defer s.Close()
		c := getChip(t, s.DevPath(), option)
		defer c.Close()

		l, err := c.RequestLine(offset, gpiod.AsInput)
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		inf, err := c.LineInfo(offset)
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

		offset := 4
		s, err := gpiosim.NewSimpleton(6)
		require.Nil(t, err)
		defer s.Close()
		c := getChip(t, s.DevPath())
		defer c.Close()

		l, err := c.RequestLine(offset, gpiod.AsInput, option)
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		inf, err := c.LineInfo(offset)
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

		offset := 4
		s, err := gpiosim.NewSimpleton(6)
		require.Nil(t, err)
		defer s.Close()
		c := getChip(t, s.DevPath())
		defer c.Close()

		l, err := c.RequestLine(offset, createOption, gpiod.AsInput)
		assert.Nil(t, err)
		require.NotNil(t, l)
		defer l.Close()
		l.Reconfigure(reconfigOption)
		inf, err := c.LineInfo(offset)
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

	offsets := []int{1, 2, 3, 4, 5}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath(), gpiod.WithConsumer("TestWithBiasAsIs"))
	defer c.Close()
	requireABI(t, c, 2)

	l, err := c.RequestLines(offsets,
		gpiod.AsInput,
		gpiod.WithPullDown,
		gpiod.WithLines(
			[]int{offsets[2], offsets[4]},
			gpiod.WithBiasAsIs,
		),
	)
	assert.Nil(t, err)
	require.NotNil(t, l)

	xinf := gpiod.LineInfo{
		Used:     true,
		Consumer: "TestWithBiasAsIs",
		Offset:   offsets[0],
		Config: gpiod.LineConfig{
			Bias:      gpiod.LineBiasPullDown,
			Direction: gpiod.LineDirectionInput,
		},
	}
	inf, err := c.LineInfo(offsets[0])
	assert.Nil(t, err)
	assert.Equal(t, xinf, inf)

	inf, err = c.LineInfo(offsets[2])
	assert.Nil(t, err)
	xinf.Offset = offsets[2]
	xinf.Config.Bias = gpiod.LineBiasUnknown
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

var evtSeqno uint32

type AbiVersioner interface {
	UapiAbiVersion() int
}

func nextEvent(r AbiVersioner, active int) gpiod.LineEvent {
	if r.UapiAbiVersion() != 1 {
		evtSeqno++
	}
	typ := gpiod.LineEventFallingEdge
	if active != 0 {
		typ = gpiod.LineEventRisingEdge
	}
	return gpiod.LineEvent{
		Type:      typ,
		Seqno:     evtSeqno,
		LineSeqno: evtSeqno,
	}
}

func TestWithEventHandler(t *testing.T) {
	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	s.SetPull(offset, 0)

	// via chip options
	ich := make(chan gpiod.LineEvent, 3)
	eh := func(evt gpiod.LineEvent) {
		ich <- evt
	}
	chipOpts := []gpiod.ChipOption{gpiod.WithEventHandler(eh)}
	if kernelAbiVersion != 0 {
		chipOpts = append(chipOpts, gpiod.ABIVersionOption(kernelAbiVersion))
	}
	c := getChip(t, s.DevPath(), chipOpts...)
	defer c.Close()

	r, err := c.RequestLine(offset, gpiod.WithBothEdges)
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	s.SetPull(offset, 1)
	waitEvent(t, ich, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitEvent(t, ich, nextEvent(r, 0))
	s.SetPull(offset, 1)
	waitEvent(t, ich, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitEvent(t, ich, nextEvent(r, 0))
	waitNoEvent(t, ich)

	r.Close()

	// via line options
	ich2 := make(chan gpiod.LineEvent, 3)
	eh2 := func(evt gpiod.LineEvent) {
		ich2 <- evt
	}
	r, err = c.RequestLine(offset,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(eh2))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich2)
	s.SetPull(offset, 1)
	waitEvent(t, ich2, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitEvent(t, ich2, nextEvent(r, 0))
	s.SetPull(offset, 1)
	waitEvent(t, ich2, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitEvent(t, ich2, nextEvent(r, 0))
	waitNoEvent(t, ich2)

	r.Close()

	// stub out inherted event handler
	r, err = c.RequestLine(offset,
		gpiod.WithEventHandler(nil))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	waitNoEvent(t, ich)
	s.SetPull(offset, 1)
	waitNoEvent(t, ich)
	s.SetPull(offset, 0)
	waitNoEvent(t, ich)
}

func TestWithFallingEdge(t *testing.T) {
	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	s.SetPull(offset, 1)
	c := getChip(t, s.DevPath())
	defer c.Close()

	ich := make(chan gpiod.LineEvent, 3)
	r, err := c.RequestLine(offset,
		gpiod.WithFallingEdge,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	s.SetPull(offset, 0)
	waitEvent(t, ich, nextEvent(r, 0))
	s.SetPull(offset, 1)
	waitNoEvent(t, ich)
	s.SetPull(offset, 0)
	waitEvent(t, ich, nextEvent(r, 0))
	s.SetPull(offset, 1)
	waitNoEvent(t, ich)
}

func TestWithRisingEdge(t *testing.T) {
	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	s.SetPull(offset, 0)
	c := getChip(t, s.DevPath())
	defer c.Close()

	ich := make(chan gpiod.LineEvent, 3)
	r, err := c.RequestLine(offset,
		gpiod.WithRisingEdge,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	s.SetPull(offset, 1)
	waitEvent(t, ich, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitNoEvent(t, ich)
	s.SetPull(offset, 1)
	waitEvent(t, ich, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitNoEvent(t, ich)
}

func TestWithBothEdges(t *testing.T) {
	offsets := []int{4, 3, 2, 1}
	offset := offsets[1]
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	s.SetPull(offset, 0)
	c := getChip(t, s.DevPath())
	defer c.Close()

	ich := make(chan gpiod.LineEvent, 3)
	r, err := c.RequestLines(offsets,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	s.SetPull(offset, 1)
	waitEvent(t, ich, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitEvent(t, ich, nextEvent(r, 0))
	s.SetPull(offset, 1)
	waitEvent(t, ich, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitEvent(t, ich, nextEvent(r, 0))
	waitNoEvent(t, ich)
}

func TestWithoutEdges(t *testing.T) {
	offsets := []int{4, 3, 2, 1}
	offset := offsets[1]
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	s.SetPull(offset, 0)
	c := getChip(t, s.DevPath())
	defer c.Close()

	ich := make(chan gpiod.LineEvent, 3)
	r, err := c.RequestLines(offsets,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
			ich <- evt
		}))
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	s.SetPull(offset, 1)
	waitEvent(t, ich, nextEvent(r, 1))
	s.SetPull(offset, 0)
	waitEvent(t, ich, nextEvent(r, 0))

	err = r.Reconfigure(gpiod.WithoutEdges)
	if c.UapiAbiVersion() == 1 {
		// uapi v2 required for edge reconfiguration
		assert.Equal(t, unix.EINVAL, err)
		return
	}
	require.Nil(t, err)
	waitNoEvent(t, ich)
	s.SetPull(offset, 1)
	waitNoEvent(t, ich)
	s.SetPull(offset, 0)
	waitNoEvent(t, ich)
}

func TestWithRealtimeEventClock(t *testing.T) {
	offsets := []int{4, 3, 2, 1}
	offset := offsets[1]
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	s.SetPull(offset, 0)
	c := getChip(t, s.DevPath())
	defer c.Close()

	var evtTimestamp time.Duration
	ich := make(chan gpiod.LineEvent, 3)
	r, err := c.RequestLines(offsets,
		gpiod.WithBothEdges,
		gpiod.WithRealtimeEventClock,
		gpiod.WithEventHandler(func(evt gpiod.LineEvent) {
			evtTimestamp = evt.Timestamp
			ich <- evt
		}))
	if c.UapiAbiVersion() == 1 {
		// uapi v2 required for event clock option
		assert.Equal(t, gpiod.ErrUapiIncompatibility{Feature: "event clock", AbiVersion: 1}, err)
		assert.Nil(t, r)
		return
	}
	if uapi.CheckKernelVersion(eventClockRealtimeKernel) != nil {
		// old kernels should reject the realtime request
		assert.Equal(t, unix.EINVAL, err)
		assert.Nil(t, r)
		if r != nil {
			r.Close()
		}
		return
	}
	require.Nil(t, err)
	require.NotNil(t, r)
	defer r.Close()
	evtSeqno = 0
	waitNoEvent(t, ich)
	start := time.Now()
	s.SetPull(offset, 1)
	waitEvent(t, ich, nextEvent(r, 1))
	end := time.Now()
	// with time converted to nanoseconds duration
	assert.LessOrEqual(t, start.UnixNano(), evtTimestamp.Nanoseconds())
	assert.GreaterOrEqual(t, end.UnixNano(), evtTimestamp.Nanoseconds())
	// with timestamp converted to time
	evtTime := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC).Add(evtTimestamp)
	assert.False(t, evtTime.Before(start))
	assert.False(t, evtTime.After(end))

	start = time.Now()
	s.SetPull(offset, 0)
	waitEvent(t, ich, nextEvent(r, 0))
	end = time.Now()
	assert.LessOrEqual(t, start.UnixNano(), evtTimestamp.Nanoseconds())
	assert.GreaterOrEqual(t, end.UnixNano(), evtTimestamp.Nanoseconds())
	evtTime = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC).Add(evtTimestamp)
	assert.False(t, evtTime.Before(start))
	assert.False(t, evtTime.After(end))
}

func waitEvent(t *testing.T, ch <-chan gpiod.LineEvent, xevt gpiod.LineEvent) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Equal(t, xevt.Type, evt.Type)
		assert.Equal(t, xevt.Seqno, evt.Seqno)
		assert.Equal(t, xevt.LineSeqno, evt.LineSeqno)
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

func clearEvents(ch <-chan gpiod.LineEvent) uint32 {
	var seqno uint32
	select {
	case evt := <-ch:
		seqno = evt.Seqno
	case <-time.After(20 * time.Millisecond):
	}
	return seqno
}

func TestWithDebounce(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	offset := 1
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	l, err := c.RequestLine(offset,
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

	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, gpiod.LineDirectionInput, inf.Config.Direction)
	assert.True(t, inf.Config.Debounced)
	assert.Equal(t, 10*time.Microsecond, inf.Config.DebouncePeriod)
}

func TestWithLines(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	offsets := []int{4, 3, 2, 1, 0}
	offset := offsets[1]
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	s.SetPull(offset, 0)
	c := getChip(t, s.DevPath(), gpiod.WithConsumer("TestWithLines"))
	defer c.Close()
	requireABI(t, c, 2)

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
					[]int{offsets[2], offsets[4]},
					gpiod.AsOutput(1, 1),
					gpiod.AsActiveLow,
					gpiod.WithPullUp,
					gpiod.AsOpenDrain,
				),
			},
			map[int]gpiod.LineInfo{
				offsets[0]: {
					Config: gpiod.LineConfig{
						Bias:      gpiod.LineBiasPullDown,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[2]: {
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
					[]int{offsets[2], offsets[4]},
					gpiod.WithDebounce(1234*time.Microsecond),
				),
				gpiod.AsActiveLow,
			},
			map[int]gpiod.LineInfo{
				offsets[1]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[4]: {
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
					[]int{offsets[2], offsets[4]},
					gpiod.WithDebounce(1432*time.Microsecond),
				),
			},
			map[int]gpiod.LineInfo{
				offsets[3]: {
					Config: gpiod.LineConfig{
						Direction: gpiod.LineDirectionOutput,
					},
				},
				offsets[4]: {
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
					[]int{offsets[2], offsets[1]},
					gpiod.WithDebounce(1432*time.Microsecond),
				),
			},
			map[int]gpiod.LineInfo{
				offsets[0]: {
					Config: gpiod.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1234 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
				offsets[1]: {
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
					[]int{offsets[2], offsets[4]},
					gpiod.AsOutput(1, 1),
					gpiod.AsActiveLow,
					gpiod.WithPullUp,
					gpiod.AsOpenDrain,
				),
				gpiod.WithLines(
					[]int{offsets[3], offsets[4]},
					gpiod.WithDebounce(1432*time.Microsecond),
				),
				gpiod.WithPullDown,
			},
			map[int]gpiod.LineInfo{
				offsets[0]: {
					Config: gpiod.LineConfig{
						Bias:      gpiod.LineBiasPullDown,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Bias:      gpiod.LineBiasPullUp,
						Direction: gpiod.LineDirectionOutput,
						Drive:     gpiod.LineDriveOpenDrain,
					},
				},
				offsets[3]: {
					Config: gpiod.LineConfig{
						Debounced:      true,
						DebouncePeriod: 1432 * time.Microsecond,
						Direction:      gpiod.LineDirectionInput,
					},
				},
				offsets[4]: {
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
			l, err := c.RequestLines(offsets, p.reqOptions...)
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
			l, err := c.RequestLines(offsets)
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
	offsets := []int{4, 3, 2, 1, 0}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath(), gpiod.WithConsumer("TestDefaulted"))
	defer c.Close()

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
				offsets[0]: {
					Config: gpiod.LineConfig{
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[2]: {
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
					[]int{offsets[2], offsets[4]},
					gpiod.WithDebounce(1234*time.Microsecond),
				),
				gpiod.WithLines(
					[]int{offsets[2]},
					gpiod.Defaulted,
				),
				gpiod.AsActiveLow,
			},
			map[int]gpiod.LineInfo{
				offsets[1]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[4]: {
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
					[]int{offsets[2], offsets[4]},
					gpiod.WithDebounce(1234*time.Microsecond),
				),
				gpiod.WithLines(
					[]int(nil),
					gpiod.Defaulted,
				),
				gpiod.AsActiveLow,
			},
			map[int]gpiod.LineInfo{
				offsets[1]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[4]: {
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
					[]int{offsets[2], offsets[4]},
					gpiod.WithDebounce(1234*time.Microsecond),
				),
				gpiod.WithLines(
					[]int{},
					gpiod.Defaulted,
				),
				gpiod.AsActiveLow,
			},
			map[int]gpiod.LineInfo{
				offsets[1]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[2]: {
					Config: gpiod.LineConfig{
						ActiveLow: true,
						Direction: gpiod.LineDirectionInput,
					},
				},
				offsets[4]: {
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
			l, err := c.RequestLines(offsets, p.reqOptions...)
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
			l, err := c.RequestLines(offsets)
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
	offsets := []int{4, 3, 2, 1}
	offset := offsets[1]
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	s.SetPull(offset, 0)
	c := getChip(t, s.DevPath())
	defer c.Close()
	requireABI(t, c, 2)

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
				l, err := c.RequestLine(offsets[0], gpiod.WithEventBufferSize(p.size))
				assert.Nil(t, err)
				require.NotNil(t, l)
				l.Close()
			})
		} else {
			t.Run(p.name, func(t *testing.T) {
				l, err := c.RequestLines(offsets[:p.numLines], gpiod.WithEventBufferSize(p.size))
				assert.Nil(t, err)
				require.NotNil(t, l)
				l.Close()
			})
		}
	}
}
