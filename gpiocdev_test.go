// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package gpiocdev_test

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/uapi"
	"github.com/warthog618/go-gpiosim"
	"golang.org/x/sys/unix"
)

var kernelAbiVersion int

func TestMain(m *testing.M) {
	flag.IntVar(&kernelAbiVersion, "abi", 0, "kernel uAPI version")
	flag.Parse()
	rc := m.Run()
	os.Exit(rc)
}

var (
	biasKernel               = uapi.Semver{5, 5}  // bias flags added
	setConfigKernel          = uapi.Semver{5, 5}  // setLineConfig ioctl added
	infoWatchKernel          = uapi.Semver{5, 7}  // watchLineInfo ioctl added
	uapiV2Kernel             = uapi.Semver{5, 10} // uapi v2 added
	eventClockRealtimeKernel = uapi.Semver{5, 11} // realtime event clock option added
)

func TestRequestLine(t *testing.T) {
	var opts []gpiocdev.LineReqOption
	if kernelAbiVersion != 0 {
		opts = append(opts, gpiocdev.ABIVersionOption(kernelAbiVersion))
	}

	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	offset := 3

	// non-existent
	l, err := gpiocdev.RequestLine(s.DevPath()+"not", offset, opts...)
	assert.NotNil(t, err)
	require.Nil(t, l)

	// negative
	l, err = gpiocdev.RequestLine(s.DevPath(), -1, opts...)
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	require.Nil(t, l)

	// out of range
	l, err = gpiocdev.RequestLine(s.DevPath(), s.Config().NumLines+1, opts...)
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	require.Nil(t, l)

	// success - input
	l, err = gpiocdev.RequestLine(s.DevPath(), offset)
	assert.Nil(t, err)
	require.NotNil(t, l)

	// already requested input
	l2, err := gpiocdev.RequestLine(s.DevPath(), offset)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output
	l2, err = gpiocdev.RequestLine(s.DevPath(), offset, append(opts, gpiocdev.AsOutput(0))...)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output as event
	l2, err = gpiocdev.RequestLine(s.DevPath(), offset, append(opts, gpiocdev.WithBothEdges)...)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	err = l.Close()
	assert.Nil(t, err)
}

func TestRequestLines(t *testing.T) {
	var opts []gpiocdev.LineReqOption
	if kernelAbiVersion != 0 {
		opts = append(opts, gpiocdev.ABIVersionOption(kernelAbiVersion))
	}

	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	offsets := []int{1, 4}
	// non-existent
	ll, err := gpiocdev.RequestLines(s.DevPath()+"not", offsets, opts...)
	assert.NotNil(t, err)
	require.Nil(t, ll)

	// negative
	ll, err = gpiocdev.RequestLines(s.DevPath(), append(offsets, -1), opts...)
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// out of range
	ll, err = gpiocdev.RequestLines(s.DevPath(), append(offsets, s.Config().NumLines))
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// success - output
	ll, err = gpiocdev.RequestLines(s.DevPath(), offsets, gpiocdev.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, ll)

	// already requested input
	ll2, err := gpiocdev.RequestLines(s.DevPath(), offsets)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output
	ll2, err = gpiocdev.RequestLines(s.DevPath(), offsets, append(opts, gpiocdev.AsOutput())...)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output as event
	ll2, err = gpiocdev.RequestLines(s.DevPath(), offsets, append(opts, gpiocdev.WithBothEdges)...)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	err = ll.Close()
	assert.Nil(t, err)
}

func TestNewChip(t *testing.T) {
	var chipOpts []gpiocdev.ChipOption
	if kernelAbiVersion != 0 {
		chipOpts = append(chipOpts, gpiocdev.ABIVersionOption(kernelAbiVersion))
	}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	// non-existent
	c, err := gpiocdev.NewChip(s.DevPath()+"not", chipOpts...)
	assert.NotNil(t, err)
	assert.Nil(t, c)

	// success
	c = getChip(t, s.DevPath())
	err = c.Close()
	assert.Nil(t, err)

	// name
	c, err = gpiocdev.NewChip(s.ChipName(), chipOpts...)
	assert.Nil(t, err)
	require.NotNil(t, c)
	err = c.Close()
	assert.Nil(t, err)

	// option
	c = getChip(t, s.DevPath(), gpiocdev.WithConsumer("gpiocdev_test"))
	assert.Equal(t, s.ChipName(), c.Name)
	assert.Equal(t, s.Config().Label, c.Label)
	err = c.Close()
	assert.Nil(t, err)
}

func TestChips(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	cc := gpiocdev.Chips()
	require.GreaterOrEqual(t, len(cc), 1)
	assert.Contains(t, cc, s.ChipName())
}

func TestChipClose(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	// without lines
	c := getChip(t, s.DevPath())
	err = c.Close()
	assert.Nil(t, err)

	// closed
	err = c.Close()
	assert.Equal(t, gpiocdev.ErrClosed, err)

	// with lines
	offsets := []int{2, 5}
	c = getChip(t, s.DevPath())
	require.NotNil(t, c)
	ll, err := c.RequestLines(offsets, gpiocdev.WithBothEdges)
	assert.Nil(t, err)
	err = c.Close()
	assert.Nil(t, err)
	require.NotNil(t, ll)
	err = ll.Close()
	assert.Nil(t, err)

	// after lines closed
	c = getChip(t, s.DevPath())
	require.NotNil(t, c)
	ll, err = c.RequestLines(offsets, gpiocdev.WithBothEdges)
	assert.Nil(t, err)
	require.NotNil(t, ll)
	err = ll.Close()
	assert.Nil(t, err)
	err = c.Close()
	assert.Nil(t, err)
}

func TestChipFindLine(t *testing.T) {
	s, err := gpiosim.NewSim(
		gpiosim.WithName("gpiocdev_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8,
			gpiosim.WithNamedLine(3, "CFLBUTTON1"),
			gpiosim.WithNamedLine(5, "CFLLED6"),
		)),
		gpiosim.WithBank(gpiosim.NewBank("right", 8,
			gpiosim.WithNamedLine(1, "CFLBUTTON1"),
			gpiosim.WithNamedLine(4, "CFLLED7"),
		)),
	)
	require.Nil(t, err)
	defer s.Close()

	c, err := gpiocdev.NewChip(s.Chips[0].ChipName())
	require.Nil(t, err)
	defer c.Close()

	offset, err := c.FindLine("CFLLED6")
	require.Nil(t, err)
	require.Equal(t, 5, offset)

	offset, err = c.FindLine("CFLLED7")
	require.Equal(t, gpiocdev.ErrNotFound, err)

	offset, err = c.FindLine("CFLLED3")
	require.Equal(t, gpiocdev.ErrNotFound, err)

	c.Close()
	c, err = gpiocdev.NewChip(s.Chips[1].ChipName())
	require.Nil(t, err)

	offset, err = c.FindLine("CFLLED6")
	require.Equal(t, gpiocdev.ErrNotFound, err)

	offset, err = c.FindLine("CFLLED7")
	require.Nil(t, err)
	require.Equal(t, 4, offset)

	offset, err = c.FindLine("CFLLED3")
	require.Equal(t, gpiocdev.ErrNotFound, err)
}

func TestChipLineInfo(t *testing.T) {
	offset := 4
	s, err := gpiosim.NewSim(
		gpiosim.WithName("gpiocdev_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8,
			gpiosim.WithNamedLine(offset, "BUTTON1"),
		)),
	)
	require.Nil(t, err)
	defer s.Close()

	sc := &s.Chips[0]
	c := getChip(t, sc.DevPath())
	xli := gpiocdev.LineInfo{}
	// out of range
	li, err := c.LineInfo(sc.Config().NumLines)
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	assert.Equal(t, xli, li)

	// valid
	li, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xli = gpiocdev.LineInfo{
		Offset: offset,
		Name:   sc.Config().Names[offset],
		Config: gpiocdev.LineConfig{
			Direction: gpiocdev.LineDirectionInput,
		},
	}
	assert.Equal(t, xli, li)

	// closed
	c.Close()
	li, err = c.LineInfo(1)
	assert.NotNil(t, err)
	xli = gpiocdev.LineInfo{}
	assert.Equal(t, xli, li)
}

func TestChipLines(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	c := getChip(t, s.DevPath())
	defer c.Close()
	lines := c.Lines()
	assert.Equal(t, s.Config().NumLines, lines)
}

func TestChipRequestLine(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	c := getChip(t, s.DevPath())
	defer c.Close()

	offset := 3

	// negative
	l, err := c.RequestLine(-1)
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	require.Nil(t, l)

	// out of range
	l, err = c.RequestLine(c.Lines())
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	require.Nil(t, l)

	// success - input
	l, err = c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)

	// already requested input
	l2, err := c.RequestLine(offset)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output
	l2, err = c.RequestLine(offset, gpiocdev.AsOutput(0))
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output as event
	l2, err = c.RequestLine(offset, gpiocdev.WithBothEdges)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	err = l.Close()
	assert.Nil(t, err)
}

func TestChipRequestLines(t *testing.T) {
	offsets := []int{4, 2}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	c := getChip(t, s.DevPath())
	defer c.Close()

	// negative
	ll, err := c.RequestLines(append(offsets, -1))
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// out of range
	ll, err = c.RequestLines(append(offsets, c.Lines()))
	assert.Equal(t, gpiocdev.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// success - output
	ll, err = c.RequestLines(offsets, gpiocdev.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, ll)

	// already requested input
	ll2, err := c.RequestLines(offsets)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output
	ll2, err = c.RequestLines(offsets, gpiocdev.AsOutput())
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output as event
	ll2, err = c.RequestLines(offsets, gpiocdev.WithBothEdges)
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	err = ll.Close()
	assert.Nil(t, err)
}

func TestChipWatchLineInfo(t *testing.T) {
	requireKernel(t, infoWatchKernel)

	offset := 4
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())

	wc1 := make(chan gpiocdev.LineInfoChangeEvent, 5)
	watcher1 := func(info gpiocdev.LineInfoChangeEvent) {
		wc1 <- info
	}

	// closed
	c.Close()
	_, err = c.WatchLineInfo(offset, watcher1)
	require.Equal(t, gpiocdev.ErrClosed, err)

	c = getChip(t, s.DevPath())
	defer c.Close()

	// unwatched
	_, err = c.WatchLineInfo(offset, watcher1)
	require.Nil(t, err)

	l, err := c.RequestLine(offset, gpiocdev.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, l)
	waitInfoEvent(t, wc1, gpiocdev.LineRequested)
	l.Reconfigure(gpiocdev.AsActiveLow)
	waitInfoEvent(t, wc1, gpiocdev.LineReconfigured)
	l.Close()
	waitInfoEvent(t, wc1, gpiocdev.LineReleased)

	wc2 := make(chan gpiocdev.LineInfoChangeEvent, 2)

	// watched
	watcher2 := func(info gpiocdev.LineInfoChangeEvent) {
		wc2 <- info
	}
	_, err = c.WatchLineInfo(offset, watcher2)
	assert.Equal(t, unix.EBUSY, err)

	l, err = c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	waitInfoEvent(t, wc1, gpiocdev.LineRequested)
	waitNoInfoEvent(t, wc2)
	l.Close()
	waitInfoEvent(t, wc1, gpiocdev.LineReleased)
	waitNoInfoEvent(t, wc2)
}

func TestChipUnwatchLineInfo(t *testing.T) {
	requireKernel(t, infoWatchKernel)

	offset := 3
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()

	c := getChip(t, s.DevPath())
	c.Close()

	// closed
	err = c.UnwatchLineInfo(offset)
	assert.Nil(t, err)

	c = getChip(t, s.DevPath())
	defer c.Close()

	// Unwatched
	err = c.UnwatchLineInfo(offset)
	assert.Equal(t, unix.EBUSY, err)

	// Watched
	wc := 0
	watcher := func(info gpiocdev.LineInfoChangeEvent) {
		wc++
	}
	_, err = c.WatchLineInfo(offset, watcher)
	require.Nil(t, err)
	err = c.UnwatchLineInfo(offset)
	assert.Nil(t, err)

	l, err := c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	assert.Zero(t, wc)
	l.Close()
	assert.Zero(t, wc)
}

func TestLineChip(t *testing.T) {
	offset := 3
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()
	l, err := c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	cname := l.Chip()
	assert.Equal(t, c.Name, cname)
}

func TestLineClose(t *testing.T) {
	offset := 3
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()
	l, err := c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)

	err = l.Close()
	assert.Equal(t, gpiocdev.ErrClosed, err)
}

func TestLineInfo(t *testing.T) {
	offset := 3
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()
	l, err := c.RequestLine(offset, gpiocdev.WithBothEdges)
	assert.Nil(t, err)
	require.NotNil(t, l)
	cli, err := c.LineInfo(offset)
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
	assert.Equal(t, gpiocdev.ErrClosed, err)
}

func TestLineOffset(t *testing.T) {
	offset := 3
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()
	l, err := c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lo := l.Offset()
	assert.Equal(t, offset, lo)
}

func TestLineReconfigure(t *testing.T) {
	requireKernel(t, setConfigKernel)

	offset := 3
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath(), gpiocdev.WithConsumer("TestLineReconfigure"))
	defer c.Close()

	xinf := gpiocdev.LineInfo{
		Used:     true,
		Consumer: "TestLineReconfigure",
		Offset:   offset,
		Config: gpiocdev.LineConfig{
			Direction: gpiocdev.LineDirectionInput,
		},
	}
	l, err := c.RequestLine(offset, gpiocdev.AsInput)
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
	err = l.Reconfigure(gpiocdev.AsActiveLow)
	assert.Nil(t, err)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Config.ActiveLow = true
	assert.Equal(t, xinf, inf)

	// closed
	l.Close()
	err = l.Reconfigure(gpiocdev.AsActiveLow)
	assert.Equal(t, gpiocdev.ErrClosed, err)

	// event request
	l, err = c.RequestLine(offset,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(func(gpiocdev.LineEvent) {}))
	assert.Nil(t, err)
	require.NotNil(t, l)

	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Config.ActiveLow = false
	if l.UapiAbiVersion() != 1 {
		// uAPI v1 does not return edge detection status in info
		xinf.Config.EdgeDetection = gpiocdev.LineEdgeBoth
	}
	assert.Equal(t, xinf, inf)

	err = l.Reconfigure(gpiocdev.AsActiveLow)
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

	offsets := []int{1, 3, 0, 2}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath(), gpiocdev.WithConsumer("TestLinesReconfigure"))
	defer c.Close()

	offset := offsets[1]
	ll, err := c.RequestLines(offsets, gpiocdev.AsInput)
	assert.Nil(t, err)
	require.NotNil(t, ll)

	xinf := gpiocdev.LineInfo{
		Used:     true,
		Consumer: "TestLinesReconfigure",
		Offset:   offset,
		Config: gpiocdev.LineConfig{
			Direction: gpiocdev.LineDirectionInput,
		},
	}
	inf, err := c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, xinf, inf)

	// no options
	err = ll.Reconfigure()
	assert.Nil(t, err)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	assert.Equal(t, xinf, inf)

	// one option
	err = ll.Reconfigure(gpiocdev.AsActiveLow)
	assert.Nil(t, err)
	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Config.ActiveLow = true
	assert.Equal(t, xinf, inf)

	if ll.UapiAbiVersion() != 1 {
		inner := []int{offsets[3], offsets[0]}

		// WithLines
		err = ll.Reconfigure(
			gpiocdev.WithLines(inner, gpiocdev.WithPullUp),
			gpiocdev.AsActiveHigh,
		)
		assert.Nil(t, err)

		inf, err = c.LineInfo(offset)
		assert.Nil(t, err)
		xinf.Config.ActiveLow = false
		assert.Equal(t, xinf, inf)

		xinfi := gpiocdev.LineInfo{
			Used:     true,
			Consumer: "TestLinesReconfigure",
			Offset:   inner[0],
			Config: gpiocdev.LineConfig{
				ActiveLow: true,
				Bias:      gpiocdev.LineBiasPullUp,
				Direction: gpiocdev.LineDirectionInput,
			},
		}
		inf, err = c.LineInfo(inner[0])
		assert.Nil(t, err)
		assert.Equal(t, xinfi, inf)

		inf, err = c.LineInfo(inner[1])
		assert.Nil(t, err)
		xinfi.Offset = inner[1]
		assert.Equal(t, xinfi, inf)

		// single WithLines -> 3 distinct configs
		err = ll.Reconfigure(
			gpiocdev.WithLines(inner[:1], gpiocdev.WithPullDown),
		)
		assert.Nil(t, err)

		inf, err = c.LineInfo(offset)
		assert.Nil(t, err)
		xinf.Config.ActiveLow = false
		assert.Equal(t, xinf, inf)

		inf, err = c.LineInfo(inner[1])
		assert.Nil(t, err)
		xinfi.Offset = inner[1]
		assert.Equal(t, xinfi, inf)

		inf, err = c.LineInfo(inner[0])
		assert.Nil(t, err)
		xinfi.Offset = inner[0]
		xinfi.Config.Bias = gpiocdev.LineBiasPullDown
		assert.Equal(t, xinfi, inf)
	}

	// closed
	ll.Close()
	err = ll.Reconfigure(gpiocdev.AsActiveLow)
	assert.Equal(t, gpiocdev.ErrClosed, err)

	// event request
	ll, err = c.RequestLines(offsets,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(func(gpiocdev.LineEvent) {}))
	assert.Nil(t, err)
	require.NotNil(t, ll)

	inf, err = c.LineInfo(offset)
	assert.Nil(t, err)
	xinf.Config.ActiveLow = false
	if ll.UapiAbiVersion() != 1 {
		// uAPI v1 does not return edge detection status in info
		xinf.Config.EdgeDetection = gpiocdev.LineEdgeBoth
	}
	assert.Equal(t, xinf, inf)

	err = ll.Reconfigure(gpiocdev.AsActiveLow)
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
	offset := 3
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	s.SetPull(offset, 0)
	l, err := c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	s.SetPull(offset, 1)
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	l.Close()
	_, err = l.Value()
	assert.Equal(t, gpiocdev.ErrClosed, err)
}

func TestLineSetValue(t *testing.T) {
	offset := 0
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	// input
	l, err := c.RequestLine(offset)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValue(1)
	assert.Equal(t, gpiocdev.ErrPermissionDenied, err)
	l.Close()

	// output
	l, err = c.RequestLine(offset, gpiocdev.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValue(1)
	assert.Nil(t, err)
	l.Close()
	err = l.SetValue(1)
	assert.Equal(t, gpiocdev.ErrClosed, err)
}

func TestLinesChip(t *testing.T) {
	offsets := []int{5, 4, 3}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()
	l, err := c.RequestLines(offsets)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lc := l.Chip()
	assert.Equal(t, c.Name, lc)
}

func TestLinesClose(t *testing.T) {
	offsets := []int{5, 0, 3}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()
	l, err := c.RequestLines(offsets)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)

	err = l.Close()
	assert.Equal(t, gpiocdev.ErrClosed, err)
}

func TestLinesInfo(t *testing.T) {
	offsets := []int{5, 1, 3}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()
	l, err := c.RequestLines(offsets)
	assert.Nil(t, err)
	require.NotNil(t, l)

	// initial
	li, err := l.Info()
	assert.Nil(t, err)
	for i, o := range offsets {
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
	for i, o := range offsets {
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
	assert.Equal(t, gpiocdev.ErrClosed, err)
	assert.Nil(t, li)
}

func TestLineOffsets(t *testing.T) {
	offsets := []int{1, 4, 3}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()
	l, err := c.RequestLines(offsets)
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lo := l.Offsets()
	assert.Equal(t, offsets, lo)
}

func TestLinesValues(t *testing.T) {
	offsets := []int{1, 2, 3, 4, 5}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	offset := offsets[1]
	// input
	s.SetPull(offset, 0)
	l, err := c.RequestLines(offsets)
	assert.Nil(t, err)
	require.NotNil(t, l)
	vv := make([]int, len(offsets))
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, []int{0, 0, 0, 0, 0}, vv)
	s.SetPull(offset, 1)
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, []int{0, 1, 0, 0, 0}, vv)

	// subset
	vv = make([]int, len(offsets)-2)
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, []int{0, 1, 0}, vv)

	l.Close()

	// after close
	err = l.Values(vv)
	assert.NotNil(t, err)

	// output
	l, err = c.RequestLines(offsets, gpiocdev.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	vv = make([]int, len(offsets))
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, []int{0, 0, 0, 0, 0}, vv)

	l.Close()
}

func checkLevels(t *testing.T, s *gpiosim.Simpleton, offsets, values []int) {
	for i, o := range offsets {
		v, err := s.Level(o)
		assert.Nil(t, err)
		assert.Equal(t, values[i], v, i)
	}
}

func TestLinesSetValues(t *testing.T) {
	offsets := []int{2, 3, 1}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	defer s.Close()
	c := getChip(t, s.DevPath())
	defer c.Close()

	// input
	l, err := c.RequestLines(offsets)
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValues([]int{0, 1})
	assert.Equal(t, gpiocdev.ErrPermissionDenied, err)
	l.Close()

	// output
	l, err = c.RequestLines(offsets, gpiocdev.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValues([]int{1, 0, 1})
	assert.Nil(t, err)
	checkLevels(t, s, offsets, []int{1, 0, 1})

	// subset
	err = l.SetValues([]int{0, 1})
	assert.Nil(t, err)
	checkLevels(t, s, offsets, []int{0, 1, 0})

	// too many values
	err = l.SetValues([]int{1, 0, 1, 0})
	assert.Nil(t, err)
	checkLevels(t, s, offsets, []int{1, 0, 1})

	// closed
	l.Close()
	err = l.SetValues([]int{0, 1})
	assert.Equal(t, gpiocdev.ErrClosed, err)
}

func naturalLess(lhs, rhs string) bool {
	llhs := len(lhs)
	lrhs := len(rhs)
	if llhs == lrhs {
		return lhs < rhs
	}
	if llhs < lrhs {
		return true
	}
	return false
}

func TestFindLine(t *testing.T) {
	s, err := gpiosim.NewSim(
		gpiosim.WithName("gpiocdev_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8,
			gpiosim.WithNamedLine(3, "FLBUTTON1"),
			gpiosim.WithNamedLine(5, "FLLED6"),
		)),
		gpiosim.WithBank(gpiosim.NewBank("right", 8,
			gpiosim.WithNamedLine(1, "FLBUTTON1"),
			gpiosim.WithNamedLine(4, "FLLED7"),
		)),
	)
	require.Nil(t, err)
	defer s.Close()

	chip, offset, err := gpiocdev.FindLine("FLLED6")
	require.Nil(t, err)
	require.Equal(t, s.Chips[0].ChipName(), chip)
	require.Equal(t, 5, offset)

	chip, offset, err = gpiocdev.FindLine("FLLED7")
	require.Nil(t, err)
	require.Equal(t, s.Chips[1].ChipName(), chip)
	require.Equal(t, 4, offset)

	chip, offset, err = gpiocdev.FindLine("FLLED3")
	require.Equal(t, gpiocdev.ErrNotFound, err)

	chip, offset, err = gpiocdev.FindLine("FLBUTTON1")
	require.Nil(t, err)
	// behaviour depends on chip number assigned by gpio-sim
	if naturalLess(s.Chips[0].ChipName(), s.Chips[1].ChipName()) {
		require.Equal(t, s.Chips[0].ChipName(), chip)
		require.Equal(t, 3, offset)
	} else {
		require.Equal(t, s.Chips[1].ChipName(), chip)
		require.Equal(t, 1, offset)
	}
}

func TestIsChip(t *testing.T) {
	// non-existent
	err := gpiocdev.IsChip("/dev/nonexistent")
	assert.NotNil(t, err)

	// wrong mode
	err = gpiocdev.IsChip("/dev/zero")
	assert.Equal(t, gpiocdev.ErrNotCharacterDevice, err)

	// no sysfs
	err = gpiocdev.IsChip("/dev/null")
	assert.Equal(t, gpiocdev.ErrNotCharacterDevice, err)

	// not sure how to test the remaining conditions...
}

func waitInfoEvent(t *testing.T, ch <-chan gpiocdev.LineInfoChangeEvent, etype gpiocdev.LineInfoChangeType) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Equal(t, etype, evt.Type)
	case <-time.After(time.Second):
		assert.Fail(t, "timeout waiting for event")
	}
}

func waitNoInfoEvent(t *testing.T, ch <-chan gpiocdev.LineInfoChangeEvent) {
	t.Helper()
	select {
	case evt := <-ch:
		assert.Fail(t, "received unexpected event", evt)
	case <-time.After(20 * time.Millisecond):
	}
}

func getChip(t *testing.T, chipPath string, chipOpts ...gpiocdev.ChipOption) *gpiocdev.Chip {
	if kernelAbiVersion != 0 {
		chipOpts = append(chipOpts, gpiocdev.ABIVersionOption(kernelAbiVersion))
	}
	c, err := gpiocdev.NewChip(chipPath, chipOpts...)
	require.Nil(t, err)
	require.NotNil(t, c)
	return c
}

func requireKernel(t *testing.T, min uapi.Semver) {
	t.Helper()
	if err := uapi.CheckKernelVersion(min); err != nil {
		t.Skip(err)
	}
}

func requireABI(t *testing.T, chip *gpiocdev.Chip, abi int) {
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
