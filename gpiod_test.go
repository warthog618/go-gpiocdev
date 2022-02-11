// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package gpiod_test

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
	"github.com/warthog618/gpiod/mockup"
	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

var platform Platform
var kernelAbiVersion int

func TestMain(m *testing.M) {
	var pname string
	flag.StringVar(&pname, "platform", "mockup", "test platform")
	flag.IntVar(&kernelAbiVersion, "abi", 0, "kernel uAPI version")
	flag.Parse()
	p, err := newPlatform(pname)
	if err != nil {
		fmt.Println("Platform not supported -", err)
		os.Exit(-1)
	}
	platform = p
	rc := m.Run()
	platform.Close()
	os.Exit(rc)
}

var (
	biasKernel               = mockup.Semver{5, 5}  // bias flags added
	setConfigKernel          = mockup.Semver{5, 5}  // setLineConfig ioctl added
	infoWatchKernel          = mockup.Semver{5, 7}  // watchLineInfo ioctl added
	uapiV2Kernel             = mockup.Semver{5, 10} // uapi v2 added
	eventClockRealtimeKernel = mockup.Semver{5, 11} // realtime event clock option added
)

func TestRequestLine(t *testing.T) {
	var opts []gpiod.LineReqOption
	if kernelAbiVersion != 0 {
		opts = append(opts, gpiod.ABIVersionOption(kernelAbiVersion))
	}

	lo := platform.FloatingLines()[0]

	// non-existent
	l, err := gpiod.RequestLine(platform.Devpath()+"not", -1, opts...)
	assert.NotNil(t, err)
	require.Nil(t, l)

	// negative
	l, err = gpiod.RequestLine(platform.Devpath(), -1, opts...)
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, l)

	// out of range
	l, err = gpiod.RequestLine(platform.Devpath(), platform.Lines())
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, l)

	// success - input
	l, err = gpiod.RequestLine(platform.Devpath(), lo)
	assert.Nil(t, err)
	require.NotNil(t, l)

	// already requested input
	l2, err := gpiod.RequestLine(platform.Devpath(), lo)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output
	l2, err = gpiod.RequestLine(platform.Devpath(), lo, append(opts, gpiod.AsOutput(0))...)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output as event
	l2, err = gpiod.RequestLine(platform.Devpath(), lo, append(opts, gpiod.WithBothEdges)...)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	err = l.Close()
	assert.Nil(t, err)
}

func TestRequestLines(t *testing.T) {
	var opts []gpiod.LineReqOption
	if kernelAbiVersion != 0 {
		opts = append(opts, gpiod.ABIVersionOption(kernelAbiVersion))
	}

	// non-existent
	ll, err := gpiod.RequestLines(platform.Devpath()+"not", []int{platform.IntrLine(), -1}, opts...)
	assert.NotNil(t, err)
	require.Nil(t, ll)

	// negative
	ll, err = gpiod.RequestLines(platform.Devpath(), []int{platform.IntrLine(), -1}, opts...)
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// out of range
	ll, err = gpiod.RequestLines(platform.Devpath(), []int{platform.IntrLine(), platform.Lines()})
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// success - output
	ll, err = gpiod.RequestLines(platform.Devpath(), platform.FloatingLines(), gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, ll)

	// already requested input
	ll2, err := gpiod.RequestLines(platform.Devpath(), platform.FloatingLines())
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output
	ll2, err = gpiod.RequestLines(platform.Devpath(), platform.FloatingLines(), append(opts, gpiod.AsOutput())...)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output as event
	ll2, err = gpiod.RequestLines(platform.Devpath(), platform.FloatingLines(), append(opts, gpiod.WithBothEdges)...)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	err = ll.Close()
	assert.Nil(t, err)
}

func TestNewChip(t *testing.T) {
	var chipOpts []gpiod.ChipOption
	if kernelAbiVersion != 0 {
		chipOpts = append(chipOpts, gpiod.ABIVersionOption(kernelAbiVersion))
	}
	// non-existent
	c, err := gpiod.NewChip(platform.Devpath()+"not", chipOpts...)
	assert.NotNil(t, err)
	assert.Nil(t, c)

	// success
	c = getChip(t)
	err = c.Close()
	assert.Nil(t, err)

	// name
	c, err = gpiod.NewChip(platform.Name(), chipOpts...)
	assert.Nil(t, err)
	require.NotNil(t, c)
	err = c.Close()
	assert.Nil(t, err)

	// option
	c = getChip(t, gpiod.WithConsumer("gpiod_test"))
	assert.Equal(t, platform.Name(), c.Name)
	assert.Equal(t, platform.Label(), c.Label)
	err = c.Close()
	assert.Nil(t, err)
}

func TestChips(t *testing.T) {
	cc := gpiod.Chips()
	require.GreaterOrEqual(t, len(cc), 1)
	assert.Contains(t, cc, platform.Name())
}

func TestChipClose(t *testing.T) {
	// without lines
	c := getChip(t)
	err := c.Close()
	assert.Nil(t, err)

	// closed
	err = c.Close()
	assert.Equal(t, gpiod.ErrClosed, err)

	// with lines
	c = getChip(t)
	require.NotNil(t, c)
	ll, err := c.RequestLines(platform.FloatingLines(), gpiod.WithBothEdges)
	assert.Nil(t, err)
	err = c.Close()
	assert.Nil(t, err)
	require.NotNil(t, ll)
	err = ll.Close()
	assert.Nil(t, err)

	// after lines closed
	c = getChip(t)
	require.NotNil(t, c)
	ll, err = c.RequestLines(platform.FloatingLines(), gpiod.WithBothEdges)
	assert.Nil(t, err)
	require.NotNil(t, ll)
	err = ll.Close()
	assert.Nil(t, err)
	err = c.Close()
	assert.Nil(t, err)
}

func TestChipLineInfo(t *testing.T) {
	c := getChip(t)
	xli := gpiod.LineInfo{}
	// out of range
	li, err := c.LineInfo(platform.Lines())
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	assert.Equal(t, xli, li)

	// valid
	li, err = c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	xli = gpiod.LineInfo{
		Offset: platform.IntrLine(),
		Name:   platform.IntrName(),
		Config: gpiod.LineConfig{
			Direction: gpiod.LineDirectionInput,
		},
	}
	assert.Equal(t, xli, li)

	// closed
	c.Close()
	li, err = c.LineInfo(1)
	assert.NotNil(t, err)
	xli = gpiod.LineInfo{}
	assert.Equal(t, xli, li)
}

func TestChipLines(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	lines := c.Lines()
	assert.Equal(t, platform.Lines(), lines)
}

func TestChipRequestLine(t *testing.T) {
	c := getChip(t)
	defer c.Close()

	lo := platform.FloatingLines()[0]

	// negative
	l, err := c.RequestLine(-1)
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, l)

	// out of range
	l, err = c.RequestLine(c.Lines())
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, l)

	// success - input
	l, err = c.RequestLine(lo)
	assert.Nil(t, err)
	require.NotNil(t, l)

	// already requested input
	l2, err := c.RequestLine(lo)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output
	l2, err = c.RequestLine(lo, gpiod.AsOutput(0))
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output as event
	l2, err = c.RequestLine(lo, gpiod.WithBothEdges)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	err = l.Close()
	assert.Nil(t, err)
}

func TestChipRequestLines(t *testing.T) {
	c := getChip(t)
	defer c.Close()

	// negative
	ll, err := c.RequestLines([]int{platform.IntrLine(), -1})
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// out of range
	ll, err = c.RequestLines([]int{platform.IntrLine(), c.Lines()})
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// success - output
	ll, err = c.RequestLines(platform.FloatingLines(), gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, ll)

	// already requested input
	ll2, err := c.RequestLines(platform.FloatingLines())
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output
	ll2, err = c.RequestLines(platform.FloatingLines(), gpiod.AsOutput())
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output as event
	ll2, err = c.RequestLines(platform.FloatingLines(), gpiod.WithBothEdges)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	err = ll.Close()
	assert.Nil(t, err)
}

func TestChipWatchLineInfo(t *testing.T) {
	requireKernel(t, infoWatchKernel)

	c := getChip(t)

	lo := platform.FloatingLines()[0]
	wc1 := make(chan gpiod.LineInfoChangeEvent, 5)
	watcher1 := func(info gpiod.LineInfoChangeEvent) {
		wc1 <- info
	}

	// closed
	c.Close()
	_, err := c.WatchLineInfo(lo, watcher1)
	require.Equal(t, gpiod.ErrClosed, err)

	c = getChip(t)
	defer c.Close()

	// unwatched
	_, err = c.WatchLineInfo(lo, watcher1)
	require.Nil(t, err)

	l, err := c.RequestLine(lo)
	assert.Nil(t, err)
	require.NotNil(t, l)
	waitInfoEvent(t, wc1, gpiod.LineRequested)
	l.Reconfigure(gpiod.AsActiveLow)
	waitInfoEvent(t, wc1, gpiod.LineReconfigured)
	l.Close()
	waitInfoEvent(t, wc1, gpiod.LineReleased)

	wc2 := make(chan gpiod.LineInfoChangeEvent, 2)

	// watched
	watcher2 := func(info gpiod.LineInfoChangeEvent) {
		wc2 <- info
	}
	_, err = c.WatchLineInfo(lo, watcher2)
	assert.Equal(t, unix.EBUSY, err)

	l, err = c.RequestLine(lo)
	assert.Nil(t, err)
	require.NotNil(t, l)
	waitInfoEvent(t, wc1, gpiod.LineRequested)
	waitNoInfoEvent(t, wc2)
	l.Close()
	waitInfoEvent(t, wc1, gpiod.LineReleased)
	waitNoInfoEvent(t, wc2)
}

func TestChipUnwatchLineInfo(t *testing.T) {
	requireKernel(t, infoWatchKernel)

	c := getChip(t)
	c.Close()

	lo := platform.FloatingLines()[0]

	// closed
	err := c.UnwatchLineInfo(lo)
	assert.Nil(t, err)

	c = getChip(t)
	defer c.Close()

	// Unwatched
	err = c.UnwatchLineInfo(lo)
	assert.Equal(t, unix.EBUSY, err)

	// Watched
	wc := 0
	watcher := func(info gpiod.LineInfoChangeEvent) {
		wc++
	}
	_, err = c.WatchLineInfo(lo, watcher)
	require.Nil(t, err)
	err = c.UnwatchLineInfo(lo)
	assert.Nil(t, err)

	l, err := c.RequestLine(lo)
	assert.Nil(t, err)
	require.NotNil(t, l)
	assert.Zero(t, wc)
	l.Close()
	assert.Zero(t, wc)
}

func TestLineChip(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	cname := l.Chip()
	assert.Equal(t, c.Name, cname)
}

func TestLineClose(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)

	err = l.Close()
	assert.Equal(t, gpiod.ErrClosed, err)
}

func TestLineInfo(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine(), gpiod.WithBothEdges)
	assert.Nil(t, err)
	require.NotNil(t, l)
	cli, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)

	li, err := l.Info()
	assert.Nil(t, err)
	require.NotNil(t, li)
	assert.Equal(t, cli, li)

	// cached
	li, err = l.Info()
	assert.Nil(t, err)
	require.NotNil(t, li)
	assert.Equal(t, cli, li)

	// closed
	l.Close()
	_, err = l.Info()
	assert.Equal(t, gpiod.ErrClosed, err)
}

func TestLineOffset(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lo := l.Offset()
	assert.Equal(t, platform.IntrLine(), lo)
}

func TestLineReconfigure(t *testing.T) {
	requireKernel(t, setConfigKernel)

	c := getChip(t, gpiod.WithConsumer("TestLineReconfigure"))
	defer c.Close()

	offset := platform.IntrLine()
	xinf := gpiod.LineInfo{
		Used:     true,
		Consumer: "TestLineReconfigure",
		Offset:   offset,
		Config: gpiod.LineConfig{
			Direction: gpiod.LineDirectionInput,
		},
	}
	l, err := c.RequestLine(offset, gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)

	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Name = inf.Name // don't care about line name
	assert.Equal(t, xinf, inf)

	// no options
	err = l.Reconfigure()
	assert.Nil(t, err)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, xinf, inf)

	// an option
	err = l.Reconfigure(gpiod.AsActiveLow)
	assert.Nil(t, err)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Config.ActiveLow = true
	assert.Equal(t, xinf, inf)

	// closed
	l.Close()
	err = l.Reconfigure(gpiod.AsActiveLow)
	assert.Equal(t, gpiod.ErrClosed, err)

	// event request
	l, err = c.RequestLine(offset,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(gpiod.LineEvent) {}))
	assert.Nil(t, err)
	require.NotNil(t, l)

	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Config.ActiveLow = false
	if l.UapiAbiVersion() != 1 {
		// uAPI v1 does not return edge detection status in info
		xinf.Config.EdgeDetection = gpiod.LineEdgeBoth
	}
	assert.Equal(t, xinf, inf)

	err = l.Reconfigure(gpiod.AsActiveLow)
	switch l.UapiAbiVersion() {
	case 1:
		assert.Equal(t, unix.EINVAL, err)
	case 2:
		assert.Nil(t, err)
		xinf.Config.ActiveLow = true
	}
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, xinf, inf)
	l.Close()
}

func TestLinesReconfigure(t *testing.T) {
	requireKernel(t, setConfigKernel)

	c := getChip(t, gpiod.WithConsumer("TestLinesReconfigure"))
	defer c.Close()

	offsets := platform.FloatingLines()
	offset := offsets[1]
	ll, err := c.RequestLines(offsets, gpiod.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, ll)

	xinf := gpiod.LineInfo{
		Used:     true,
		Consumer: "TestLinesReconfigure",
		Offset:   offset,
		Config: gpiod.LineConfig{
			Direction: gpiod.LineDirectionInput,
		},
	}
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Name = inf.Name // don't care about line name
	assert.Equal(t, xinf, inf)

	// no options
	err = ll.Reconfigure()
	assert.Nil(t, err)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, xinf, inf)

	// one option
	err = ll.Reconfigure(gpiod.AsActiveLow)
	assert.Nil(t, err)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Config.ActiveLow = true
	assert.Equal(t, xinf, inf)

	if ll.UapiAbiVersion() != 1 {
		inner := []int{offsets[3], offsets[0]}

		// WithLines
		err = ll.Reconfigure(
			gpiod.WithLines(inner, gpiod.WithPullUp),
			gpiod.AsActiveHigh,
		)
		assert.Nil(t, err)

		inf, err = c.LineInfo(offset)
		assert.Nil(t, err)
		xinf.Config.ActiveLow = false
		assert.Equal(t, xinf, inf)

		xinfi := gpiod.LineInfo{
			Used:     true,
			Consumer: "TestLinesReconfigure",
			Offset:   inner[0],
			Config: gpiod.LineConfig{
				ActiveLow: true,
				Bias:      gpiod.LineBiasPullUp,
				Direction: gpiod.LineDirectionInput,
			},
		}
		inf, err = c.LineInfo(inner[0])
		assert.Nil(t, err)
		xinfi.Name = inf.Name // don't care about line name
		assert.Equal(t, xinfi, inf)

		inf, err = c.LineInfo(inner[1])
		assert.Nil(t, err)
		xinfi.Offset = inner[1]
		xinfi.Name = inf.Name // don't care about line name
		assert.Equal(t, xinfi, inf)

		// single WithLines -> 3 distinct configs
		err = ll.Reconfigure(
			gpiod.WithLines(inner[:1], gpiod.WithPullDown),
		)
		assert.Nil(t, err)

		inf, err = c.LineInfo(offset)
		assert.Nil(t, err)
		xinf.Config.ActiveLow = false
		assert.Equal(t, xinf, inf)

		inf, err = c.LineInfo(inner[1])
		assert.Nil(t, err)
		xinfi.Offset = inner[1]
		xinfi.Name = inf.Name // don't care about line name
		assert.Equal(t, xinfi, inf)

		inf, err = c.LineInfo(inner[0])
		assert.Nil(t, err)
		xinfi.Offset = inner[0]
		xinfi.Config.Bias = gpiod.LineBiasPullDown
		xinfi.Name = inf.Name // don't care about line name
		assert.Equal(t, xinfi, inf)
	}

	// closed
	ll.Close()
	err = ll.Reconfigure(gpiod.AsActiveLow)
	assert.Equal(t, gpiod.ErrClosed, err)

	// event request
	ll, err = c.RequestLines(offsets,
		gpiod.WithBothEdges,
		gpiod.WithEventHandler(func(gpiod.LineEvent) {}))
	assert.Nil(t, err)
	require.NotNil(t, ll)

	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Config.ActiveLow = false
	if ll.UapiAbiVersion() != 1 {
		// uAPI v1 does not return edge detection status in info
		xinf.Config.EdgeDetection = gpiod.LineEdgeBoth
	}
	assert.Equal(t, xinf, inf)

	err = ll.Reconfigure(gpiod.AsActiveLow)
	switch ll.UapiAbiVersion() {
	case 1:
		assert.Equal(t, unix.EINVAL, err)
	case 2:
		assert.Nil(t, err)
		xinf.Config.ActiveLow = true
	}
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, xinf, inf)
	ll.Close()
}

func TestLineValue(t *testing.T) {
	c := getChip(t)
	defer c.Close()

	platform.TriggerIntr(0)
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	platform.TriggerIntr(1)
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	l.Close()
	_, err = l.Value()
	assert.Equal(t, gpiod.ErrClosed, err)
}

func TestLineSetValue(t *testing.T) {
	c := getChip(t)
	defer c.Close()

	lo := platform.FloatingLines()[0]

	// input
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValue(1)
	assert.Equal(t, gpiod.ErrPermissionDenied, err)
	l.Close()

	// output
	l, err = c.RequestLine(lo,
		gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValue(1)
	assert.Nil(t, err)
	l.Close()
	err = l.SetValue(1)
	assert.Equal(t, gpiod.ErrClosed, err)
}

func TestLinesChip(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lc := l.Chip()
	assert.Equal(t, c.Name, lc)
}

func TestLinesClose(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)

	err = l.Close()
	assert.Equal(t, gpiod.ErrClosed, err)
}

func TestLinesInfo(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)

	// initial
	li, err := l.Info()
	assert.Nil(t, err)
	for i, o := range platform.FloatingLines() {
		cli, err := c.LineInfo(o)
		assert.Nil(t, err)
		assert.NotNil(t, li[i])
		if li[0] != nil {
			assert.Equal(t, cli, *li[i])
		}
	}

	// cached
	li, err = l.Info()
	assert.Nil(t, err)
	for i, o := range platform.FloatingLines() {
		cli, err := c.LineInfo(o)
		assert.Nil(t, err)
		assert.NotNil(t, li[i])
		if li[0] != nil {
			assert.Equal(t, cli, *li[i])
		}
	}

	// closed
	l.Close()
	li, err = l.Info()
	assert.Equal(t, gpiod.ErrClosed, err)
	assert.Nil(t, li)
}

func TestLineOffsets(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lo := l.Offsets()
	assert.Equal(t, platform.FloatingLines(), lo)
}

func TestLinesValues(t *testing.T) {
	c := getChip(t)
	defer c.Close()

	// input
	platform.TriggerIntr(0)
	lines := append([]int{platform.IntrLine()}, platform.FloatingLines()...)
	l, err := c.RequestLines(lines)
	assert.Nil(t, err)
	require.NotNil(t, l)
	vv := make([]int, len(lines))
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, 0, vv[0])
	platform.TriggerIntr(1)
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, 1, vv[0])

	// subset
	vv = vv[:len(lines)-2]
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, 1, vv[0])

	l.Close()

	// after close
	err = l.Values(vv)
	assert.NotNil(t, err)

	// output
	lines = platform.FloatingLines()
	l, err = c.RequestLines(lines, gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Values(vv)
	assert.Nil(t, err)
	// actual values are indeterminate

	l.Close()
}

func TestLinesSetValues(t *testing.T) {
	c := getChip(t)
	defer c.Close()

	// input
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValues([]int{0, 1})
	assert.Equal(t, gpiod.ErrPermissionDenied, err)
	l.Close()

	// output
	l, err = c.RequestLines(platform.FloatingLines(),
		gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValues([]int{1, 0})
	assert.Nil(t, err)

	// too many values
	err = l.SetValues([]int{1, 1, 1})
	assert.Nil(t, err)

	// closed
	l.Close()
	err = l.SetValues([]int{0, 1})
	assert.Equal(t, gpiod.ErrClosed, err)
}

func TestIsChip(t *testing.T) {
	// nonexistent
	err := gpiod.IsChip("/dev/nonexistent")
	assert.NotNil(t, err)

	// wrong mode
	err = gpiod.IsChip("/dev/zero")
	assert.Equal(t, gpiod.ErrNotCharacterDevice, err)

	// no sysfs
	err = gpiod.IsChip("/dev/null")
	assert.Equal(t, gpiod.ErrNotCharacterDevice, err)

	// not sure how to test the remaining conditions...
}

func waitInfoEvent(t *testing.T, ch <-chan gpiod.LineInfoChangeEvent, etype gpiod.LineInfoChangeType) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Equal(t, etype, evt.Type)
	case <-time.After(time.Second):
		assert.Fail(t, "timeout waiting for event")
	}
}

func waitNoInfoEvent(t *testing.T, ch <-chan gpiod.LineInfoChangeEvent) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Fail(t, "received unexpected event", evt)
	case <-time.After(20 * time.Millisecond):
	}
}

func getChip(t *testing.T, chipOpts ...gpiod.ChipOption) *gpiod.Chip {
	if kernelAbiVersion != 0 {
		chipOpts = append(chipOpts, gpiod.ABIVersionOption(kernelAbiVersion))
	}
	c, err := gpiod.NewChip(platform.Devpath(), chipOpts...)
	require.Nil(t, err)
	require.NotNil(t, c)
	return c
}

type gpiochip struct {
	name    string
	label   string
	devpath string
	lines   int
	// line triggered by TriggerIntr.
	intro     int
	introName string
	outo      int
	// floating lines - can be harmlessly set to outputs.
	ff []int
}

func (c *gpiochip) Name() string {
	return c.name
}

func (c *gpiochip) Label() string {
	return c.label
}
func (c *gpiochip) Devpath() string {
	return c.devpath
}

func (c *gpiochip) Lines() int {
	return c.lines
}

func (c *gpiochip) IntrLine() int {
	return c.intro
}

func (c *gpiochip) IntrName() string {
	return c.introName
}

func (c *gpiochip) OutLine() int {
	return c.outo
}

func (c *gpiochip) FloatingLines() []int {
	return c.ff
}

// two flavours of chip, raspberry and mockup.
type Platform interface {
	Name() string
	Label() string
	Devpath() string
	Lines() int
	IntrLine() int
	IntrName() string
	OutLine() int
	FloatingLines() []int
	TriggerIntr(int)
	ReadOut() int
	SupportsAsIs() bool
	Close()
}

type RaspberryPi struct {
	gpiochip
	chip  *gpiod.Chip
	wline *gpiod.Line
}

func isPi(path string) error {
	if err := gpiod.IsChip(path); err != nil {
		return err
	}
	f, err := os.OpenFile(path, unix.O_CLOEXEC, unix.O_RDONLY)
	if err != nil {
		return err
	}
	defer f.Close()
	ci, err := uapi.GetChipInfo(f.Fd())
	if err != nil {
		return err
	}
	label := uapi.BytesToString(ci.Label[:])
	if label != "pinctrl-bcm2835" && label != "pinctrl-bcm2711" {
		return fmt.Errorf("unsupported gpiochip - %s", label)
	}
	return nil
}

func newPi(path string) (*RaspberryPi, error) {
	if err := isPi(path); err != nil {
		return nil, err
	}
	ch, err := gpiod.NewChip(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			ch.Close()
		}
	}()
	pi := RaspberryPi{
		gpiochip: gpiochip{
			name:      "gpiochip0",
			label:     ch.Label,
			devpath:   path,
			lines:     int(ch.Lines()),
			intro:     rpi.J8p15,
			introName: "",
			outo:      rpi.J8p16,
			ff:        []int{rpi.J8p11, rpi.J8p12, rpi.J8p7, rpi.J8p13, rpi.J8p22},
		},
		chip: ch,
	}
	if ch.Label == "pinctrl-bcm2711" {
		pi.introName = "GPIO22"
	}
	// check J8p15 and J8p16 are tied
	w, err := ch.RequestLine(pi.outo, gpiod.AsOutput(1),
		gpiod.WithConsumer("gpiod-test-w"))
	if err != nil {
		return nil, err
	}
	defer w.Close()
	r, err := ch.RequestLine(pi.intro,
		gpiod.WithConsumer("gpiod-test-r"))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	v, _ := r.Value()
	if v != 1 {
		return nil, errors.New("J8p15 and J8p16 must be tied")
	}
	w.SetValue(0)
	v, _ = r.Value()
	if v != 0 {
		return nil, errors.New("J8p15 and J8p16 must be tied")
	}
	return &pi, nil
}

func (c *RaspberryPi) Close() {
	if c.wline != nil {
		c.wline.Close()
		c.wline = nil
	}
	// revert intr trigger line to input
	l, _ := c.chip.RequestLine(c.outo)
	l.Close()
	// revert floating lines to inputs
	ll, _ := c.chip.RequestLines(platform.FloatingLines())
	ll.Close()
	c.chip.Close()
}

func (c *RaspberryPi) OutLine() int {
	if c.wline != nil {
		c.wline.Close()
		c.wline = nil
	}
	return c.outo
}

func (c *RaspberryPi) ReadOut() int {
	r, err := c.chip.RequestLine(c.intro,
		gpiod.WithConsumer("gpiod-test-r"))
	if err != nil {
		return -1
	}
	defer r.Close()
	v, err := r.Value()
	if err != nil {
		return -1
	}
	return v
}

func (c *RaspberryPi) SupportsAsIs() bool {
	// RPi pinctrl-bcm2835 returns lines to input on release.
	return false
}

func (c *RaspberryPi) TriggerIntr(value int) {
	if c.wline != nil {
		c.wline.SetValue(value)
		return
	}
	w, _ := c.chip.RequestLine(c.outo, gpiod.AsOutput(value),
		gpiod.WithConsumer("gpiod-test-w"))
	c.wline = w
}

type Mockup struct {
	gpiochip
	m *mockup.Mockup
	c *mockup.Chip
}

func newMockup() (*Mockup, error) {
	m, err := mockup.New([]int{20}, true)
	if err != nil {
		return nil, err
	}
	c, err := m.Chip(0)
	if err != nil {
		return nil, err
	}
	return &Mockup{
		gpiochip{
			name:      c.Name,
			label:     c.Label,
			devpath:   c.DevPath,
			lines:     20,
			intro:     10,
			introName: "gpio-mockup-A-10",
			outo:      9,
			ff:        []int{11, 12, 15, 16, 9},
		}, m, c}, nil
}

func (c *Mockup) Close() {
	c.m.Close()
}

func (c *Mockup) ReadOut() int {
	v, err := c.c.Value(c.outo)
	if err != nil {
		return -1
	}
	return v
}

func (c *Mockup) SupportsAsIs() bool {
	return true
}

func (c *Mockup) TriggerIntr(value int) {
	c.c.SetValue(c.intro, value)
}

func newPlatform(pname string) (Platform, error) {
	switch pname {
	case "mockup":
		p, err := newMockup()
		if err != nil {
			return nil, fmt.Errorf("error loading gpio-mockup: %w", err)
		}
		return p, nil
	case "rpi":
		return newPi("/dev/gpiochip0")
	default:
		return nil, fmt.Errorf("unknown platform '%s'", pname)
	}
}

func requireKernel(t *testing.T, min mockup.Semver) {
	t.Helper()
	if err := mockup.CheckKernelVersion(min); err != nil {
		t.Skip(err)
	}
}

func requireABI(t *testing.T, chip *gpiod.Chip, abi int) {
	t.Helper()
	if chip.UapiAbiVersion() != abi {
		t.Skip(ErrorBadABIVersion{abi, chip.UapiAbiVersion()})
	}
}

// ErrorBadVersion indicates the kernel version is insufficient.
type ErrorBadABIVersion struct {
	Need int
	Have int
}

func (e ErrorBadABIVersion) Error() string {
	return fmt.Sprintf("require kernel ABI %d, but using %d", e.Need, e.Have)
}
