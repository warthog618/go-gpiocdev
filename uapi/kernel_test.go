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

func TestRepeatedLines(t *testing.T) {
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	require.NotNil(t, c)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	hr := uapi.HandleRequest{
		Lines: 2,
	}
	hr.Offsets[0] = 1
	hr.Offsets[1] = 1

	// input
	err = uapi.GetLineHandle(f.Fd(), &hr)
	assert.NotNil(t, err)

	// output
	hr.Flags = uapi.HandleRequestOutput
	hr.DefaultValues[0] = 0
	hr.DefaultValues[1] = 1
	err = uapi.GetLineHandle(f.Fd(), &hr)
	assert.NotNil(t, err)

	unix.Close(int(hr.Fd))
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

	chg, err := readLineInfoChangedTimeout(f1.Fd(), time.Second)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f1")

	chg, err = readLineInfoChangedTimeout(f2.Fd(), time.Second)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f2")

	// request line
	hr := uapi.HandleRequest{Lines: 1, Flags: uapi.HandleRequestInput}
	hr.Offsets[0] = 3
	copy(hr.Consumer[:], "testwatch")
	err = uapi.GetLineHandle(f2.Fd(), &hr)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedTimeout(f1.Fd(), time.Second)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedRequested, chg.Type)
	xli.Flags |= uapi.LineFlagRequested
	copy(xli.Consumer[:], "testwatch")
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f2.Fd(), time.Second)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f2")

	err = uapi.WatchLineInfo(f2.Fd(), &li)
	require.Nil(t, err)
	err = uapi.UnwatchLineInfo(f1.Fd(), li.Offset)
	require.Nil(t, err)
	unix.Close(int(hr.Fd))

	unix.Close(int(hr.Fd))
	chg, err = readLineInfoChangedTimeout(f2.Fd(), time.Second)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedReleased, chg.Type)
	xli = uapi.LineInfo{Offset: 3}
	copy(xli.Name[:], lname)
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f1.Fd(), time.Second)
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

	evt, err := readEventTimeout(uintptr(er.Fd), time.Second)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	c.SetValue(1, 1)
	c.SetValue(1, 0)
	c.SetValue(1, 1)
	c.SetValue(1, 0)

	var ed uapi.EventData
	b := make([]byte, unsafe.Sizeof(ed)*3)
	n, err := unix.Read(int(er.Fd), b[:])
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)

	unix.Close(int(er.Fd))
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

	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Nil(t, err)
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
			Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
			Direction:     uapi.LineDirectionInput,
			EdgeDetection: uapi.LineEdgeBoth,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(uintptr(lr.Fd), time.Second)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	for i := 0; i < uapi.LinesMax; i++ {
		c.SetValue(i, 1)
		evt, err = readLineEventTimeout(uintptr(lr.Fd), time.Second)
		require.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
		assert.Equal(t, uint32(i), evt.Offset)
	}

	for i := 0; i < uapi.LinesMax; i++ {
		c.SetValue(i, 0)
		evt, err = readLineEventTimeout(uintptr(lr.Fd), time.Second)
		assert.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
		assert.Equal(t, uint32(i), evt.Offset)
	}

	evt, err = readLineEventTimeout(uintptr(lr.Fd), time.Second)
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
	xli.Flags |= uapi.LineFlagRequested
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
	xli.Flags |= uapi.LineFlagRequested
	li.Name = xli.Name // don't care about name
	assert.Equal(t, xli, li)
	unix.Close(int(hr.Fd))
}
