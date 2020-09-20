// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package uapi_test

import (
	"fmt"
	"os"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod/mockup"
	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

func TestRepeatedGetLineHandle(t *testing.T) {
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	require.NotNil(t, c)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	hr := uapi.HandleRequest{
		Flags:   uapi.HandleRequestInput,
		Lines:   2,
		Offsets: [uapi.HandlesMax]uint32{1, 3},
	}

	// input
	err = uapi.GetLineHandle(f.Fd(), &hr)
	require.Nil(t, err)

	// busy
	err = uapi.GetLineHandle(f.Fd(), &hr)
	assert.Equal(t, unix.EBUSY, err)

	// output
	hr.Flags = uapi.HandleRequestOutput
	hr.DefaultValues[0] = 0
	hr.DefaultValues[1] = 1
	err = uapi.GetLineHandle(f.Fd(), &hr)
	assert.Equal(t, unix.EBUSY, err)

	unix.Close(int(hr.Fd))
}

func TestRepeatedGetLineEvent(t *testing.T) {
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	require.NotNil(t, c)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	er := uapi.EventRequest{
		Offset:      1,
		HandleFlags: uapi.HandleRequestInput,
		EventFlags:  uapi.EventRequestBothEdges,
	}

	// input
	err = uapi.GetLineEvent(f.Fd(), &er)
	assert.Nil(t, err)

	// busy
	err = uapi.GetLineEvent(f.Fd(), &er)
	assert.Equal(t, unix.EBUSY, err)

	unix.Close(int(er.Fd))
}

func TestRepeatedGetLine(t *testing.T) {
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	require.NotNil(t, c)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	lr := uapi.LineRequest{
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input,
		},
		Lines:   2,
		Offsets: [uapi.LinesMax]uint32{1, 3},
	}

	// input
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	// busy
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Equal(t, unix.EBUSY, err)

	// output
	lr.Config.Flags = uapi.LineFlagV2Output
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Equal(t, unix.EBUSY, err)

	unix.Close(int(lr.Fd))
}

func TestAsIs(t *testing.T) {
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	patterns := []uapi.HandleFlag{
		uapi.HandleRequestInput,
		uapi.HandleRequestOutput,
	}
	for _, flags := range patterns {
		label := ""
		hr := uapi.HandleRequest{
			Offsets: [uapi.HandlesMax]uint32{2},
			Lines:   uint32(1),
		}
		info := uapi.LineInfo{
			Offset: 2,
		}
		if flags.IsInput() {
			label += "input"
			hr.Flags |= uapi.HandleRequestInput
		}
		if flags.IsOutput() {
			label += "output"
			hr.Flags |= uapi.HandleRequestOutput
			info.Flags |= uapi.LineFlagIsOut
		}
		tf := func(t *testing.T) {
			testLineAsIs(t, c, hr, info)
		}
		t.Run(label, tf)
	}
}

func TestWatchIsolation(t *testing.T) {
	requireKernel(t, infoWatchKernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	f1, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f1.Close()

	f2, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f2.Close()

	// set watch
	li := uapi.LineInfo{Offset: 3}
	lname := c.Label + "-3"
	err = uapi.WatchLineInfo(f1.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfo{Offset: 3}
	copy(xli.Name[:], lname)
	assert.Equal(t, xli, li)

	chg, err := readLineInfoChangedTimeout(f1.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f1")

	chg, err = readLineInfoChangedTimeout(f2.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f2")

	// request line
	hr := uapi.HandleRequest{Lines: 1, Flags: uapi.HandleRequestInput}
	hr.Offsets[0] = 3
	copy(hr.Consumer[:], "testwatch")
	err = uapi.GetLineHandle(f2.Fd(), &hr)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedTimeout(f1.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedRequested, chg.Type)
	xli.Flags |= uapi.LineFlagUsed
	copy(xli.Consumer[:], "testwatch")
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f2.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f2")

	err = uapi.WatchLineInfo(f2.Fd(), &li)
	require.Nil(t, err)
	err = uapi.UnwatchLineInfo(f1.Fd(), li.Offset)
	require.Nil(t, err)
	unix.Close(int(hr.Fd))

	unix.Close(int(hr.Fd))
	chg, err = readLineInfoChangedTimeout(f2.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedReleased, chg.Type)
	xli = uapi.LineInfo{Offset: 3}
	copy(xli.Name[:], lname)
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f1.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f1")
}

func TestBulkEventRead(t *testing.T) {
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	er := uapi.EventRequest{
		Offset: 1,
		HandleFlags: uapi.HandleRequestInput |
			uapi.HandleRequestActiveLow,
		EventFlags: uapi.EventRequestBothEdges,
	}
	err = uapi.GetLineEvent(f.Fd(), &er)
	require.Nil(t, err)

	evt, err := readEventTimeout(er.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	c.SetValue(1, 1)
	time.Sleep(clkTick)
	c.SetValue(1, 0)
	time.Sleep(clkTick)
	c.SetValue(1, 1)
	time.Sleep(clkTick)
	c.SetValue(1, 0)
	time.Sleep(clkTick)

	var ed uapi.EventData
	b := make([]byte, unsafe.Sizeof(ed)*3)
	n, err := unix.Read(int(er.Fd), b[:])
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)

	unix.Close(int(er.Fd))
}

func TestBulkEventReadV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{1},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	c.SetValue(1, 1)
	time.Sleep(clkTick)
	c.SetValue(1, 0)
	time.Sleep(clkTick)
	c.SetValue(1, 1)
	time.Sleep(clkTick)
	c.SetValue(1, 0)
	time.Sleep(clkTick)

	var ed uapi.LineEvent
	b := make([]byte, unsafe.Sizeof(ed)*3)
	n, err := unix.Read(int(lr.Fd), b[:])
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)

	unix.Close(int(lr.Fd))
}

func TestWatchInfoVersionLockV1(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	// test that watch locks to v1
	liv1 := uapi.LineInfo{Offset: 3}
	err = uapi.WatchLineInfo(f.Fd(), &liv1)
	require.Nil(t, err)

	li := uapi.LineInfoV2{Offset: 3}
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	assert.Equal(t, unix.EPERM, err)

	err = uapi.UnwatchLineInfo(f.Fd(), 3)
	require.Nil(t, err)

	err = uapi.WatchLineInfo(f.Fd(), &liv1)
	require.Nil(t, err)
}

func TestWatchInfoVersionLockV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	// test that watch locks to v2
	li := uapi.LineInfoV2{Offset: 3}
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Nil(t, err)

	liv1 := uapi.LineInfo{Offset: 3}
	err = uapi.WatchLineInfo(f.Fd(), &liv1)
	assert.Equal(t, unix.EPERM, err)

	err = uapi.UnwatchLineInfo(f.Fd(), 3)
	require.Nil(t, err)

	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Nil(t, err)
}

func TestSetConfigEdgeDetection(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	require.NotNil(t, c)

	patterns := []struct {
		name  string
		flags uapi.LineFlagV2
	}{
		{"input", uapi.LineFlagV2Input},
		{"output", uapi.LineFlagV2Output},
		{"rising", uapi.LineFlagV2Input | uapi.LineFlagV2EdgeRising},
		{"falling", uapi.LineFlagV2Input | uapi.LineFlagV2EdgeFalling},
		{"both", uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth},
	}

	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	for _, p1 := range patterns {
		for _, p2 := range patterns {
			tf := func(t *testing.T) {
				lr := uapi.LineRequest{
					Lines:   1,
					Offsets: [uapi.LinesMax]uint32{1},
					Config: uapi.LineConfig{
						Flags: p1.flags,
					},
				}
				err = uapi.GetLine(f.Fd(), &lr)
				require.Nil(t, err)
				defer unix.Close(int(lr.Fd))

				xevt := uapi.LineEvent{
					Offset:    1,
					LineSeqno: 1,
					Seqno:     1,
				}
				testEdgeDetectionEvents(t, c, lr.Fd, &xevt, p1.flags)

				config := uapi.LineConfig{
					Flags: p2.flags,
				}
				err = uapi.SetLineConfigV2(uintptr(lr.Fd), &config)
				require.Nil(t, err)
				testEdgeDetectionEvents(t, c, lr.Fd, &xevt, p2.flags)

				config.Flags = p1.flags
				err = uapi.SetLineConfigV2(uintptr(lr.Fd), &config)
				require.Nil(t, err)
				testEdgeDetectionEvents(t, c, lr.Fd, &xevt, p1.flags)
			}
			t.Run(fmt.Sprintf("%s-to-%s", p1.name, p2.name), tf)
		}
	}
}

func testEdgeDetectionEvents(t *testing.T, c *mockup.Chip, fd int32, xevt *uapi.LineEvent, flags uapi.LineFlagV2) {
	line := int(xevt.Offset)
	for i := 0; i < 2; i++ {
		c.SetValue(line, 1)
		if flags&uapi.LineFlagV2EdgeRising == 0 {
			evt, err := readLineEventTimeout(fd, spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, evt, "spurious event")
		} else {
			xevt.ID = uapi.LineEventRisingEdge
			evt, err := readLineEventTimeout(fd, eventWaitTimeout)
			require.Nil(t, err)
			require.NotNil(t, evt, flags)
			evt.Timestamp = 0
			assert.Equal(t, *xevt, *evt)
			xevt.LineSeqno++
			xevt.Seqno++
		}

		c.SetValue(line, 0)
		if flags&uapi.LineFlagV2EdgeFalling == 0 {
			evt, err := readLineEventTimeout(fd, spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, evt, "spurious event")
		} else {
			xevt.ID = uapi.LineEventFallingEdge
			evt, err := readLineEventTimeout(fd, eventWaitTimeout)
			require.Nil(t, err)
			require.NotNil(t, evt)
			evt.Timestamp = 0
			assert.Equal(t, *xevt, *evt)
			xevt.LineSeqno++
			xevt.Seqno++
		}
	}
}

func TestEventBufferOverflow(t *testing.T) {
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	require.NotNil(t, c)

	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 1)
	require.Nil(t, err)
	er := uapi.EventRequest{
		Offset: 1,
		HandleFlags: uapi.HandleRequestInput |
			uapi.HandleRequestActiveLow,
		EventFlags: uapi.EventRequestBothEdges,
	}
	err = uapi.GetLineEvent(f.Fd(), &er)
	require.Nil(t, err)
	defer unix.Close(int(er.Fd))

	for i := 0; i < 19; i++ {
		err = c.SetValue(1, i&1)
		require.Nil(t, err)
		time.Sleep(clkTick)
	}
	// last 3 events should be discarded by the kernel
	xevt := uapi.EventData{}
	for i := 0; i < 16; i++ {
		evt, err := readEventTimeout(er.Fd, eventWaitTimeout)
		require.Nil(t, err)
		require.NotNil(t, evt)
		evt.Timestamp = 0
		// events are out of sync due to overflow...
		if i&1 != 0 {
			xevt.ID = uapi.EventRequestFallingEdge
		} else {
			xevt.ID = uapi.EventRequestRisingEdge
		}
		assert.Equal(t, xevt, *evt)
	}
	// actual state is high while final event was falling...
	var hd uapi.HandleData
	var hdx uapi.HandleData
	hdx[0] = 1
	err = uapi.GetLineValues(uintptr(er.Fd), &hd)
	assert.Nil(t, err)
	assert.Equal(t, hdx, hd)

	evt, err := readEventTimeout(er.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")
}

func TestEventBufferOverflowV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	require.NotNil(t, c)

	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 1)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{1},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	for i := 0; i < 20; i++ {
		err = c.SetValue(1, i&1)
		require.Nil(t, err)
		time.Sleep(clkTick)
	}
	// first 4 events should be discarded by the kernel
	xevt := uapi.LineEvent{
		Offset:    1,
		LineSeqno: 5,
		Seqno:     5,
	}
	for i := 0; i < 16; i++ {
		evt, err := readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err)
		require.NotNil(t, evt)
		evt.Timestamp = 0
		if i&1 == 0 {
			xevt.ID = uapi.LineEventFallingEdge
		} else {
			xevt.ID = uapi.LineEventRisingEdge
		}
		assert.Equal(t, xevt, *evt)
		xevt.LineSeqno++
		xevt.Seqno++
	}
	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")
}

func TestSetConfigDebouncedEdges(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	require.NotNil(t, c)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{1},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	periods := []int{-1, 1, 0, 2}
	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
		Offset:    1,
	}

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	for _, period := range periods {
		if period >= 0 {
			config := uapi.LineConfig{
				Flags: lr.Config.Flags,
			}
			config.NumAttrs = 1
			config.Attrs[0].Mask = 1
			uapi.DebouncePeriod(period).Encode(&config.Attrs[0].Attr)
			err = uapi.SetLineConfigV2(uintptr(lr.Fd), &config)
			require.Nil(t, err, period)
		}

		for i := 0; i < 2; i++ {
			xevt.ID = uapi.LineEventRisingEdge
			c.SetValue(1, 1)
			evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
			require.Nil(t, err, i)
			require.NotNil(t, evt, i)
			evt.Timestamp = 0
			assert.Equal(t, xevt, *evt, i)

			xevt.LineSeqno++
			xevt.Seqno++
			xevt.ID = uapi.LineEventFallingEdge
			c.SetValue(1, 0)
			evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
			require.Nil(t, err, i)
			require.NotNil(t, evt, i)
			evt.Timestamp = 0
			assert.Equal(t, xevt, *evt, i)

			xevt.LineSeqno++
			xevt.Seqno++
		}
	}
}

func TestGetLineDebouncedEdges(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	require.NotNil(t, c)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{1},
		Config: uapi.LineConfig{
			Flags:    uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
			NumAttrs: 1,
		},
	}
	lr.Config.Attrs[0].Mask = 1
	uapi.DebouncePeriod(20).Encode(&lr.Config.Attrs[0].Attr)
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
		Offset:    1,
	}

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	for i := 0; i < 2; i++ {
		xevt.ID = uapi.LineEventRisingEdge
		c.SetValue(1, 1)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err, i)
		require.NotNil(t, evt, i)
		evt.Timestamp = 0
		assert.Equal(t, xevt, *evt, i)

		xevt.LineSeqno++
		xevt.Seqno++
		xevt.ID = uapi.LineEventFallingEdge
		c.SetValue(1, 0)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err, i)
		require.NotNil(t, evt, i)
		evt.Timestamp = 0
		assert.Equal(t, xevt, *evt, i)

		xevt.LineSeqno++
		xevt.Seqno++
	}
}

func TestSetConfigEdgeDetectionPolarity(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	require.NotNil(t, c)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{1},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeRising,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	flags := []uapi.LineFlagV2{0, uapi.LineFlagV2ActiveLow, 0, uapi.LineFlagV2ActiveLow}
	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
		Offset:    1,
		ID:        uapi.LineEventRisingEdge,
	}

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	for _, flag := range flags {
		config := uapi.LineConfig{
			Flags: lr.Config.Flags | flag,
		}
		err = uapi.SetLineConfigV2(uintptr(lr.Fd), &config)
		require.Nil(t, err, flag)

		if flag == 0 {
			for i := 0; i < 2; i++ {
				c.SetValue(1, 1)
				evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
				require.Nil(t, err, i)
				require.NotNil(t, evt, i)
				evt.Timestamp = 0
				assert.Equal(t, xevt, *evt, i)

				xevt.LineSeqno++
				xevt.Seqno++
				c.SetValue(1, 0)
				evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
				assert.Nil(t, err, i)
				assert.Nil(t, evt, "spurious event", i)
			}
		} else {
			for i := 0; i < 2; i++ {
				c.SetValue(1, 1)
				evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
				assert.Nil(t, err, i)
				assert.Nil(t, evt, "spurious event", i)

				c.SetValue(1, 0)
				evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
				require.Nil(t, err, i)
				require.NotNil(t, evt, i)
				evt.Timestamp = 0
				assert.Equal(t, xevt, *evt, i)
				xevt.LineSeqno++
				xevt.Seqno++
			}
		}
	}
}

func TestOutputSets(t *testing.T) {
	t.Skip("contains known failures as of 5.4-rc1")
	requireMockup(t)
	patterns := []struct {
		name string
		flag uapi.HandleFlag
	}{
		{"o", uapi.HandleRequestOutput},
		{"od", uapi.HandleRequestOutput | uapi.HandleRequestOpenDrain},
		{"os", uapi.HandleRequestOutput | uapi.HandleRequestOpenSource},
	}
	c, err := mock.Chip(0)
	require.Nil(t, err)
	line := 0
	for _, p := range patterns {
		for initial := 0; initial <= 1; initial++ {
			for toggle := 0; toggle <= 1; toggle++ {
				for activeLow := 0; activeLow <= 1; activeLow++ {
					final := initial
					if toggle == 1 {
						final ^= 0x01
					}
					flags := p.flag
					if activeLow == 1 {
						flags |= uapi.HandleRequestActiveLow
					}
					label := fmt.Sprintf("%s-%d-%d-%d(%d)", p.name, initial^1, initial, final, activeLow)
					tf := func(t *testing.T) {
						testLine(t, c, line, flags, initial, toggle)
					}
					t.Run(label, tf)
				}
			}
		}
	}
}

func TestEdgeDetectionLinesMax(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	if mock != nil {
		mock.Close()
	}
	defer reloadMockup()
	mock, err := mockup.New([]int{int(uapi.LinesMax)}, false)
	require.Nil(t, err)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	offsets := [uapi.LinesMax]uint32{}
	for i := 0; i < uapi.LinesMax; i++ {
		offsets[i] = uint32(i)
		err = c.SetValue(i, 0)
		require.Nil(t, err)
	}
	lr := uapi.LineRequest{
		Lines:   uint32(uapi.LinesMax),
		Offsets: offsets,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	lv := uapi.LineValues{
		Mask: uapi.NewLineBitMask(uapi.LinesMax),
	}
	for i := 0; i < uapi.LinesMax; i++ {
		c.SetValue(i, 1)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
		assert.Equal(t, uint32(i), evt.Offset)

		err = uapi.GetLineValuesV2(uintptr(lr.Fd), &lv)
		assert.Nil(t, err)
		assert.Equal(t, 1, lv.Bits.Get(i))
	}

	for i := 0; i < uapi.LinesMax; i++ {
		c.SetValue(i, 0)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		assert.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
		assert.Equal(t, uint32(i), evt.Offset)

		err = uapi.GetLineValuesV2(uintptr(lr.Fd), &lv)
		assert.Nil(t, err)
		assert.Equal(t, 0, lv.Bits.Get(i))
	}

	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))
}

func testLine(t *testing.T, c *mockup.Chip, line int, flags uapi.HandleFlag, initial, toggle int) {
	t.Helper()
	// set mock initial - opposing default
	c.SetValue(line, initial^0x01)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	// request line output
	hr := uapi.HandleRequest{
		Flags: flags,
		Lines: uint32(1),
	}
	hr.Offsets[0] = uint32(line)
	hr.DefaultValues[0] = uint8(initial)
	err = uapi.GetLineHandle(f.Fd(), &hr)
	require.Nil(t, err)
	if toggle != 0 {
		var hd uapi.HandleData
		hd[0] = uint8(initial ^ 0x01)
		err = uapi.SetLineValues(uintptr(hr.Fd), hd)
		assert.Nil(t, err, "can't set value 1")
		err = uapi.GetLineValues(uintptr(hr.Fd), &hd)
		assert.Nil(t, err, "can't get value 1")
		assert.Equal(t, uint8(initial^1), hd[0], "get value 1")
		hd[0] = uint8(initial)
		err = uapi.SetLineValues(uintptr(hr.Fd), hd)
		assert.Nil(t, err, "can't set value 2")
		err = uapi.GetLineValues(uintptr(hr.Fd), &hd)
		assert.Nil(t, err, "can't get value 2")
		assert.Equal(t, uint8(initial), hd[0], "get value 2")
		hd[0] = uint8(initial ^ 0x01)
		err = uapi.SetLineValues(uintptr(hr.Fd), hd)
		assert.Nil(t, err, "can't set value 3")
		err = uapi.GetLineValues(uintptr(hr.Fd), &hd)
		assert.Nil(t, err, "can't get value 3")
		assert.Equal(t, uint8(initial^1), hd[0], "get value 3")
	}
	// release
	unix.Close(int(hr.Fd))
}

func testLineAsIs(t *testing.T, c *mockup.Chip, hr uapi.HandleRequest, info uapi.LineInfo) {
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	line := int(hr.Offsets[0])
	copy(hr.Consumer[:31], "test-as-is")

	// initial request to set expected state
	err = uapi.GetLineHandle(f.Fd(), &hr)
	require.Nil(t, err)
	li, err := uapi.GetLineInfo(f.Fd(), line)
	assert.Nil(t, err)
	var xli uapi.LineInfo = info
	xli.Flags |= uapi.LineFlagUsed
	copy(xli.Consumer[:31], "test-as-is")
	li.Name = xli.Name // don't care about name
	assert.Equal(t, xli, li)
	unix.Close(int(hr.Fd))

	// check released
	li, err = uapi.GetLineInfo(f.Fd(), line)
	assert.Nil(t, err)
	xli = info
	xli.Flags &^= (uapi.LineFlagActiveLow)
	li.Name = xli.Name // don't care about name
	assert.Equal(t, xli, li)

	// request as-is and check state and value
	copy(hr.Consumer[:31], "test-as-is")
	hr.Flags &^= (uapi.HandleRequestInput | uapi.HandleRequestOutput)
	err = uapi.GetLineHandle(f.Fd(), &hr)
	require.Nil(t, err)
	li, err = uapi.GetLineInfo(f.Fd(), line)
	assert.Nil(t, err)
	xli = info
	copy(xli.Consumer[:31], "test-as-is")
	xli.Flags |= uapi.LineFlagUsed
	li.Name = xli.Name // don't care about name
	assert.Equal(t, xli, li)
	unix.Close(int(hr.Fd))
}
