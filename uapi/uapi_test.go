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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod/mockup"
	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

var (
	mock       *mockup.Mockup
	setupError error
)

func TestMain(m *testing.M) {
	mock, setupError = mockup.New([]int{4, 8}, true)
	rc := m.Run()
	if mock != nil {
		mock.Close()
	}
	os.Exit(rc)
}

func mockupRequired(t *testing.T) {
	t.Helper()
	if setupError != nil {
		t.Fail()
		t.Skip(setupError)
	}
}

func TestGetChipInfo(t *testing.T) {
	mockupRequired(t)
	for n := 0; n < mock.Chips(); n++ {
		c, err := mock.Chip(n)
		require.Nil(t, err)
		f := func(t *testing.T) {
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			cix := uapi.ChipInfo{
				Lines: uint32(c.Lines),
			}
			copy(cix.Name[:], c.Name)
			copy(cix.Label[:], c.Label)
			ci, err := uapi.GetChipInfo(f.Fd())
			assert.Nil(t, err)
			assert.Equal(t, cix, ci)
		}
		t.Run(c.Name, f)
	}
	// badfd
	ci, err := uapi.GetChipInfo(0)
	cix := uapi.ChipInfo{}
	assert.NotNil(t, err)
	assert.Equal(t, cix, ci)
}

func TestGetLineInfo(t *testing.T) {
	mockupRequired(t)
	for n := 0; n < mock.Chips(); n++ {
		c, err := mock.Chip(n)
		require.Nil(t, err)
		for l := 0; l < c.Lines; l++ {
			f := func(t *testing.T) {
				f, err := os.Open(c.DevPath)
				require.Nil(t, err)
				defer f.Close()
				lix := uapi.LineInfo{
					Offset: uint32(l),
					Flags:  0,
				}
				copy(lix.Name[:], fmt.Sprintf("%s-%d", c.Label, l))
				copy(lix.Consumer[:], "")
				li, err := uapi.GetLineInfo(f.Fd(), l)
				assert.Nil(t, err)
				assert.Equal(t, lix, li)
			}
			t.Run(fmt.Sprintf("%s-%d", c.Name, l), f)
		}
	}
	// badfd
	li, err := uapi.GetLineInfo(0, 1)
	lix := uapi.LineInfo{}
	assert.NotNil(t, err)
	assert.Equal(t, lix, li)
}

func TestGetLineEvent(t *testing.T) {
	mockupRequired(t)
	patterns := []struct {
		name       string // unique name for pattern (hf/ef/offsets/xval combo)
		cnum       int
		handleFlag uapi.HandleFlag
		eventFlag  uapi.EventFlag
		offset     uint32
		err        error
	}{
		{"activeLow", 1, uapi.HandleRequestActiveLow, uapi.EventRequestBothEdges, 2, nil},
		{"as is", 0, 0, uapi.EventRequestBothEdges, 2, nil},
		{"input", 0, uapi.HandleRequestInput, uapi.EventRequestBothEdges, 2, nil},
		// expected errors
		{"output", 0, uapi.HandleRequestOutput, uapi.EventRequestBothEdges, 2, unix.EINVAL},
		{"oorange", 0, uapi.HandleRequestInput, uapi.EventRequestBothEdges, 6, unix.EINVAL},
		// unexpected successes...
		{"input drain", 0, uapi.HandleRequestInput | uapi.HandleRequestOpenDrain, uapi.EventRequestBothEdges, 2, unix.EINVAL},
		{"input source", 0, uapi.HandleRequestInput | uapi.HandleRequestOpenSource, uapi.EventRequestBothEdges, 2, unix.EINVAL},
		{"as is drain", 0, uapi.HandleRequestOpenDrain, uapi.EventRequestBothEdges, 2, unix.EINVAL},
		{"as is source", 0, uapi.HandleRequestOpenSource, uapi.EventRequestBothEdges, 2, unix.EINVAL},
	}
	for _, p := range patterns {
		c, err := mock.Chip(p.cnum)
		require.Nil(t, err)
		tf := func(t *testing.T) {
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			er := uapi.EventRequest{
				Offset:      p.offset,
				HandleFlags: p.handleFlag,
			}
			copy(er.Consumer[:], p.name)
			err = uapi.GetLineEvent(f.Fd(), &er)
			assert.Equal(t, p.err, err)
			if p.offset > uint32(c.Lines) {
				return
			}
			li, err := uapi.GetLineInfo(f.Fd(), int(p.offset))
			assert.Nil(t, err)
			if p.err != nil {
				assert.False(t, li.Flags.IsRequested())
				return
			}
			xli := uapi.LineInfo{
				Offset: p.offset,
				Flags:  uapi.LineFlagRequested | lineFromHandle(p.handleFlag),
			}
			copy(xli.Name[:], li.Name[:]) // don't care about name
			copy(xli.Consumer[:], p.name)
			assert.Equal(t, xli, li)
			unix.Close(int(er.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineHandle(t *testing.T) {
	mockupRequired(t)
	patterns := []struct {
		name       string // unique name for pattern (hf/ef/offsets/xval combo)
		cnum       int
		handleFlag uapi.HandleFlag
		offsets    []uint32
		err        error
	}{
		{"activeLow", 1, uapi.HandleRequestActiveLow, []uint32{2}, nil},
		{"as is", 0, 0, []uint32{2}, nil},
		{"input", 0, uapi.HandleRequestInput, []uint32{2}, nil},
		{"output", 0, uapi.HandleRequestOutput, []uint32{2}, nil},
		{"output drain", 0, uapi.HandleRequestOutput | uapi.HandleRequestOpenDrain, []uint32{2}, nil},
		{"output source", 0, uapi.HandleRequestOutput | uapi.HandleRequestOpenSource, []uint32{3}, nil},
		// expected errors
		{"both io", 0, uapi.HandleRequestInput | uapi.HandleRequestOutput, []uint32{2}, unix.EINVAL},
		{"overlength", 0, uapi.HandleRequestInput, []uint32{0, 1, 2, 3, 4}, unix.EINVAL},
		{"oorange", 0, uapi.HandleRequestInput, []uint32{6}, unix.EINVAL},
		{"input drain", 0, uapi.HandleRequestInput | uapi.HandleRequestOpenDrain, []uint32{1}, unix.EINVAL},
		{"input source", 0, uapi.HandleRequestInput | uapi.HandleRequestOpenSource, []uint32{2}, unix.EINVAL},
		{"as is drain", 0, uapi.HandleRequestOpenDrain, []uint32{2}, unix.EINVAL},
		{"as is source", 0, uapi.HandleRequestOpenSource, []uint32{1}, unix.EINVAL},
	}
	for _, p := range patterns {
		c, err := mock.Chip(p.cnum)
		require.Nil(t, err)
		tf := func(t *testing.T) {
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			hr := uapi.HandleRequest{
				Flags: p.handleFlag,
				Lines: uint32(len(p.offsets)),
			}
			copy(hr.Offsets[:], p.offsets)
			copy(hr.Consumer[:], p.name)
			err = uapi.GetLineHandle(f.Fd(), &hr)
			assert.Equal(t, p.err, err)
			if p.offsets[0] > uint32(c.Lines) {
				return
			}
			// check line info
			li, err := uapi.GetLineInfo(f.Fd(), int(p.offsets[0]))
			assert.Nil(t, err)
			if p.err != nil {
				assert.False(t, li.Flags.IsRequested())
				return
			}
			xli := uapi.LineInfo{
				Offset: p.offsets[0],
				Flags:  uapi.LineFlagRequested | lineFromHandle(p.handleFlag),
			}
			copy(xli.Name[:], li.Name[:]) // don't care about name
			copy(xli.Consumer[:], p.name)
			assert.Equal(t, xli, li)
			unix.Close(int(hr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineValues(t *testing.T) {
	mockupRequired(t)
	patterns := []struct {
		name       string // unique name for pattern (hf/ef/offsets/xval combo)
		cnum       int
		handleFlag uapi.HandleFlag
		evtFlag    uapi.EventFlag
		offsets    []uint32
		val        []uint8
	}{
		{"activeLow lo", 1, uapi.HandleRequestActiveLow, 0, []uint32{2}, []uint8{0}},
		{"activeLow hi", 1, uapi.HandleRequestActiveLow, 0, []uint32{2}, []uint8{1}},
		{"as is lo", 0, 0, 0, []uint32{2}, []uint8{0}},
		{"as is hi", 0, 0, 0, []uint32{1}, []uint8{1}},
		{"input lo", 0, uapi.HandleRequestInput, 0, []uint32{2}, []uint8{0}},
		{"input hi", 0, uapi.HandleRequestInput, 0, []uint32{1}, []uint8{1}},
		{"output lo", 0, uapi.HandleRequestOutput, 0, []uint32{2}, []uint8{0}},
		{"output hi", 0, uapi.HandleRequestOutput, 0, []uint32{1}, []uint8{1}},
		{"both lo", 1, 0, uapi.EventRequestBothEdges, []uint32{2}, []uint8{0}},
		{"both hi", 1, 0, uapi.EventRequestBothEdges, []uint32{1}, []uint8{1}},
		{"falling lo", 0, 0, uapi.EventRequestFallingEdge, []uint32{2}, []uint8{0}},
		{"falling hi", 0, 0, uapi.EventRequestFallingEdge, []uint32{1}, []uint8{1}},
		{"rising lo", 0, 0, uapi.EventRequestRisingEdge, []uint32{2}, []uint8{0}},
		{"rising hi", 0, 0, uapi.EventRequestRisingEdge, []uint32{1}, []uint8{1}},
		{"input 2a", 0, uapi.HandleRequestInput, 0, []uint32{0, 1}, []uint8{1, 0}},
		{"input 2b", 0, uapi.HandleRequestInput, 0, []uint32{2, 1}, []uint8{0, 1}},
		{"input 3a", 0, uapi.HandleRequestInput, 0, []uint32{0, 1, 2}, []uint8{0, 1, 1}},
		{"input 3b", 0, uapi.HandleRequestInput, 0, []uint32{0, 2, 1}, []uint8{0, 1, 0}},
		{"input 4a", 0, uapi.HandleRequestInput, 0, []uint32{0, 1, 2, 3}, []uint8{0, 1, 1, 1}},
		{"input 4b", 0, uapi.HandleRequestInput, 0, []uint32{3, 2, 1, 0}, []uint8{1, 1, 0, 1}},
		{"input 8a", 1, uapi.HandleRequestInput, 0, []uint32{0, 1, 2, 3, 4, 5, 6, 7}, []uint8{0, 1, 1, 1, 1, 1, 0, 0}},
		{"input 8b", 1, uapi.HandleRequestInput, 0, []uint32{3, 2, 1, 0, 4, 5, 6, 7}, []uint8{1, 1, 0, 1, 1, 1, 0, 1}},
		{"activeLow 8b", 1, uapi.HandleRequestInput | uapi.HandleRequestActiveLow, 0, []uint32{3, 2, 1, 0, 4, 6, 7}, []uint8{1, 1, 0, 1, 1, 1, 0, 0}},
	}
	for _, p := range patterns {
		c, err := mock.Chip(p.cnum)
		require.Nil(t, err)
		// set vals in mock
		require.LessOrEqual(t, len(p.offsets), len(p.val))
		for i, o := range p.offsets {
			v := int(p.val[i])
			if p.handleFlag.IsActiveLow() {
				v ^= 0x01 // assumes using 1 for high
			}
			err := c.SetValue(int(o), v)
			assert.Nil(t, err)
		}
		tf := func(t *testing.T) {
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			var fd int32
			xval := p.val
			if p.evtFlag == 0 {
				hr := uapi.HandleRequest{
					Flags: p.handleFlag,
					Lines: uint32(len(p.offsets)),
				}
				copy(hr.Offsets[:], p.offsets)
				err := uapi.GetLineHandle(f.Fd(), &hr)
				require.Nil(t, err)
				fd = hr.Fd
				if p.handleFlag.IsOutput() {
					// mock is ignored for outputs
					xval = make([]uint8, len(p.val))
				}
			} else {
				assert.Equal(t, 1, len(p.offsets)) // reminder that events are limited to one line
				er := uapi.EventRequest{
					Offset:      p.offsets[0],
					HandleFlags: p.handleFlag,
					EventFlags:  p.evtFlag,
				}
				err = uapi.GetLineEvent(f.Fd(), &er)
				require.Nil(t, err)
				fd = er.Fd
			}
			var hdx uapi.HandleData
			copy(hdx[:], xval)
			var hd uapi.HandleData
			err = uapi.GetLineValues(uintptr(fd), &hd)
			assert.Nil(t, err)
			assert.Equal(t, hdx, hd)
			unix.Close(int(fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	var hdx uapi.HandleData
	var hd uapi.HandleData
	err := uapi.GetLineValues(0, &hd)
	assert.NotNil(t, err)
	assert.Equal(t, hdx, hd)
}

func TestSetLineValues(t *testing.T) {
	mockupRequired(t)
	patterns := []struct {
		name       string // unique name for pattern (hf/ef/offsets/xval combo)
		cnum       int
		handleFlag uapi.HandleFlag
		offsets    []uint32
		val        []uint8
		err        error
	}{
		{"activeLow lo", 1, uapi.HandleRequestActiveLow | uapi.HandleRequestOutput, []uint32{2}, []uint8{0}, nil},
		{"activeLow hi", 1, uapi.HandleRequestActiveLow | uapi.HandleRequestOutput, []uint32{2}, []uint8{1}, nil},
		{"as is lo", 0, uapi.HandleRequestOutput, []uint32{2}, []uint8{0}, nil},
		{"as is hi", 0, uapi.HandleRequestOutput, []uint32{1}, []uint8{1}, nil},
		{"output lo", 0, uapi.HandleRequestOutput, []uint32{2}, []uint8{0}, nil},
		{"output hi", 0, uapi.HandleRequestOutput, []uint32{1}, []uint8{1}, nil},
		{"output 2a", 0, uapi.HandleRequestOutput, []uint32{0, 1}, []uint8{1, 0}, nil},
		{"output 2b", 0, uapi.HandleRequestOutput, []uint32{2, 1}, []uint8{0, 1}, nil},
		{"output 3a", 0, uapi.HandleRequestOutput, []uint32{0, 1, 2}, []uint8{0, 1, 1}, nil},
		{"output 3b", 0, uapi.HandleRequestOutput, []uint32{0, 2, 1}, []uint8{0, 1, 0}, nil},
		{"output 4a", 0, uapi.HandleRequestOutput, []uint32{0, 1, 2, 3}, []uint8{0, 1, 1, 1}, nil},
		{"output 4b", 0, uapi.HandleRequestOutput, []uint32{3, 2, 1, 0}, []uint8{1, 1, 0, 1}, nil},
		{"output 8a", 1, uapi.HandleRequestOutput, []uint32{0, 1, 2, 3, 4, 5, 6, 7}, []uint8{0, 1, 1, 1, 1, 1, 0, 0}, nil},
		{"output 8b", 1, uapi.HandleRequestOutput, []uint32{3, 2, 1, 0, 4, 5, 6, 7}, []uint8{1, 1, 0, 1, 1, 1, 0, 1}, nil},
		{"activeLow 8b", 1, uapi.HandleRequestOutput | uapi.HandleRequestActiveLow, []uint32{3, 2, 1, 0, 4, 5, 6, 7}, []uint8{1, 1, 0, 1, 1, 0, 0, 0}, nil},
		// expected failures....
		{"input lo", 0, uapi.HandleRequestInput, []uint32{2}, []uint8{0}, unix.EPERM},
		{"input hi", 0, uapi.HandleRequestInput, []uint32{1}, []uint8{1}, unix.EPERM},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			c, err := mock.Chip(p.cnum)
			require.Nil(t, err)
			require.LessOrEqual(t, len(p.offsets), len(p.val))
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			hr := uapi.HandleRequest{
				Flags: p.handleFlag,
				Lines: uint32(len(p.offsets)),
			}
			copy(hr.Offsets[:], p.offsets)
			err = uapi.GetLineHandle(f.Fd(), &hr)
			require.Nil(t, err)
			var hd uapi.HandleData
			copy(hd[:], p.val)
			err = uapi.SetLineValues(uintptr(hr.Fd), hd)
			assert.Equal(t, p.err, err)
			if p.err == nil {
				// check values from mock
				for i, o := range p.offsets {
					v, err := c.Value(int(o))
					assert.Nil(t, err)
					xv := int(p.val[i])
					if p.handleFlag.IsActiveLow() {
						xv ^= 0x01 // assumes using 1 for high
					}
					assert.Equal(t, xv, v)
				}
			}
			unix.Close(int(hr.Fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	var hd uapi.HandleData
	err := uapi.SetLineValues(0, hd)
	assert.NotNil(t, err)
}

func TestSetLineHandleConfig(t *testing.T) {
	mockupRequired(t)
	patterns := []struct {
		name        string
		cnum        int
		offsets     []uint32
		initialFlag uapi.HandleFlag
		initialVal  []uint8
		configFlag  uapi.HandleFlag
		configVal   []uint8
		err         error
	}{
		{"in to out", 1, []uint32{1, 2, 3},
			uapi.HandleRequestInput, nil,
			uapi.HandleRequestOutput, []uint8{1, 0, 1},
			nil},
		{"out to in", 0, []uint32{2},
			uapi.HandleRequestOutput, nil,
			uapi.HandleRequestInput, nil,
			nil},
		{"low to high", 0, []uint32{1, 2, 3},
			uapi.HandleRequestOutput | uapi.HandleRequestActiveLow, []uint8{1, 0, 1},
			uapi.HandleRequestOutput, []uint8{1, 0, 1},
			nil},
		{"high to low", 0, []uint32{2},
			uapi.HandleRequestOutput, []uint8{1, 0, 1},
			uapi.HandleRequestOutput | uapi.HandleRequestActiveLow, []uint8{1, 0, 1},
			nil},
		// expected errors
		{"input drain", 0, []uint32{2},
			uapi.HandleRequestInput, nil,
			uapi.HandleRequestInput | uapi.HandleRequestOpenDrain, nil,
			unix.EINVAL},
		{"input source", 0, []uint32{2},
			uapi.HandleRequestInput, nil,
			uapi.HandleRequestInput | uapi.HandleRequestOpenSource, nil,
			unix.EINVAL},
		{"as is drain", 0, []uint32{2},
			0, nil,
			uapi.HandleRequestOpenDrain, nil,
			unix.EINVAL},
		{"as is source", 0, []uint32{2},
			0, nil,
			uapi.HandleRequestOpenSource, nil,
			unix.EINVAL},
		{"drain source", 0, []uint32{2},
			uapi.HandleRequestOutput, nil,
			uapi.HandleRequestOutput | uapi.HandleRequestOpenDrain | uapi.HandleRequestOpenSource, nil,
			unix.EINVAL},
		{"low to as-is high", 0, []uint32{1, 2, 3},
			uapi.HandleRequestOutput | uapi.HandleRequestActiveLow, []uint8{1, 0, 1},
			0, []uint8{1, 0, 1},
			unix.EINVAL},
		{"high to as-is low", 0, []uint32{2},
			uapi.HandleRequestOutput, []uint8{1, 0, 1},
			uapi.HandleRequestActiveLow, []uint8{1, 0, 1},
			unix.EINVAL},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			c, err := mock.Chip(p.cnum)
			require.Nil(t, err)
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			hr := uapi.HandleRequest{
				Flags: p.initialFlag,
				Lines: uint32(len(p.offsets)),
			}
			copy(hr.Offsets[:], p.offsets)
			copy(hr.DefaultValues[:], p.initialVal)
			copy(hr.Consumer[:], p.name)
			err = uapi.GetLineHandle(f.Fd(), &hr)
			require.Nil(t, err)
			// apply config change
			hc := uapi.HandleConfig{Flags: p.configFlag}
			copy(hc.DefaultValues[:], p.configVal)
			err = uapi.SetLineConfig(uintptr(hr.Fd), &hc)
			assert.Equal(t, p.err, err)

			if p.err == nil {
				// check line info
				li, err := uapi.GetLineInfo(f.Fd(), int(p.offsets[0]))
				assert.Nil(t, err)
				if p.err != nil {
					assert.False(t, li.Flags.IsRequested())
					return
				}
				xli := uapi.LineInfo{
					Offset: p.offsets[0],
					Flags:  uapi.LineFlagRequested | lineFromHandle(p.configFlag),
				}
				copy(xli.Name[:], li.Name[:]) // don't care about name
				copy(xli.Consumer[:], p.name)
				assert.Equal(t, xli, li)
				if len(p.configVal) != 0 {
					// check values from mock
					require.LessOrEqual(t, len(p.offsets), len(p.configVal))
					for i, o := range p.offsets {
						v, err := c.Value(int(o))
						assert.Nil(t, err)
						xv := int(p.configVal[i])
						if p.configFlag.IsActiveLow() {
							xv ^= 0x01 // assumes using 1 for high
						}
						assert.Equal(t, xv, v)
					}
				}
			}
			unix.Close(int(hr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestSetLineEventConfig(t *testing.T) {
	mockupRequired(t)
	patterns := []struct {
		name        string
		cnum        int
		offset      uint32
		initialFlag uapi.HandleFlag
		configFlag  uapi.HandleFlag
		err         error
	}{
		// expected errors
		{"low to high", 0, 1,
			uapi.HandleRequestInput | uapi.HandleRequestActiveLow,
			0,
			unix.EINVAL},
		{"high to low", 0, 2,
			uapi.HandleRequestInput,
			uapi.HandleRequestInput | uapi.HandleRequestActiveLow,
			unix.EINVAL},
		{"in to out", 1, 2,
			uapi.HandleRequestInput,
			uapi.HandleRequestOutput,
			unix.EINVAL},
		{"drain", 0, 3,
			uapi.HandleRequestInput,
			uapi.HandleRequestInput | uapi.HandleRequestOpenDrain,
			unix.EINVAL},
		{"source", 0, 3,
			uapi.HandleRequestInput,
			uapi.HandleRequestInput | uapi.HandleRequestOpenSource,
			unix.EINVAL},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			c, err := mock.Chip(p.cnum)
			require.Nil(t, err)
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			er := uapi.EventRequest{
				HandleFlags: p.initialFlag,
				EventFlags:  uapi.EventRequestBothEdges,
				Offset:      p.offset,
			}
			copy(er.Consumer[:], p.name)
			err = uapi.GetLineEvent(f.Fd(), &er)
			require.Nil(t, err)
			// apply config change
			hc := uapi.HandleConfig{Flags: p.configFlag}
			err = uapi.SetLineConfig(uintptr(er.Fd), &hc)
			assert.Equal(t, p.err, err)

			if p.err == nil {
				// check line info
				li, err := uapi.GetLineInfo(f.Fd(), int(p.offset))
				assert.Nil(t, err)
				if p.err != nil {
					assert.False(t, li.Flags.IsRequested())
					return
				}
				xli := uapi.LineInfo{
					Offset: p.offset,
					Flags:  uapi.LineFlagRequested | lineFromHandle(p.configFlag),
				}
				copy(xli.Name[:], li.Name[:]) // don't care about name
				copy(xli.Consumer[:], p.name)
				assert.Equal(t, xli, li)
			}
			unix.Close(int(er.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestEventRead(t *testing.T) {
	mockupRequired(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	er := uapi.EventRequest{
		Offset:      1,
		HandleFlags: uapi.HandleRequestInput | uapi.HandleRequestActiveLow,
		EventFlags:  uapi.EventRequestBothEdges,
	}
	err = uapi.GetLineEvent(f.Fd(), &er)
	require.Nil(t, err)

	_, err = readTimeout(er.Fd, time.Second)
	assert.Nil(t, err, "spurious event")

	start := time.Now()
	c.SetValue(1, 1)
	evt, err := readTimeout(er.Fd, time.Second)
	require.Nil(t, err)
	assert.Equal(t, uint32(2), evt.ID) // returns falling edge
	end := time.Now()
	assert.LessOrEqual(t, uint64(start.UnixNano()), evt.Timestamp)
	assert.GreaterOrEqual(t, uint64(end.UnixNano()), evt.Timestamp)

	start = time.Now()
	c.SetValue(1, 0)
	evt, err = readTimeout(er.Fd, time.Second)
	assert.Nil(t, err)
	assert.Equal(t, uint32(1), evt.ID) // returns rising edge
	end = time.Now()
	assert.LessOrEqual(t, uint64(start.UnixNano()), evt.Timestamp)
	assert.GreaterOrEqual(t, uint64(end.UnixNano()), evt.Timestamp)

	unix.Close(int(er.Fd))
}

func readTimeout(fd int32, t time.Duration) (*uapi.EventData, error) {
	pollfd := unix.PollFd{Fd: fd, Events: unix.POLLIN}
	n, err := unix.Poll([]unix.PollFd{pollfd}, int(t.Seconds()))
	if err != nil || n != 1 {
		return nil, err
	}
	evt, err := uapi.ReadEvent(uintptr(fd))
	if err != nil {
		return nil, err
	}
	return &evt, nil
}

func TestBytesToString(t *testing.T) {
	name := "a test string"
	a := [20]byte{}
	copy(a[:], name)

	// empty
	v := uapi.BytesToString(a[:0])
	assert.Equal(t, 0, len(v))

	// normal
	v = uapi.BytesToString(a[:])
	assert.Equal(t, name, v)

	// unterminated
	v = uapi.BytesToString(a[:len(name)])
	assert.Equal(t, name, v)
}
func TestLineFlags(t *testing.T) {
	assert.False(t, uapi.LineFlag(0).IsRequested())
	assert.False(t, uapi.LineFlag(0).IsOut())
	assert.False(t, uapi.LineFlag(0).IsActiveLow())
	assert.False(t, uapi.LineFlag(0).IsOpenDrain())
	assert.False(t, uapi.LineFlag(0).IsOpenSource())
	assert.True(t, uapi.LineFlagRequested.IsRequested())
	assert.True(t, uapi.LineFlagIsOut.IsOut())
	assert.True(t, uapi.LineFlagActiveLow.IsActiveLow())
	assert.True(t, uapi.LineFlagOpenDrain.IsOpenDrain())
	assert.True(t, uapi.LineFlagOpenSource.IsOpenSource())
}

func TestHandleFlags(t *testing.T) {
	assert.False(t, uapi.HandleFlag(0).IsInput())
	assert.False(t, uapi.HandleFlag(0).IsOutput())
	assert.False(t, uapi.HandleFlag(0).IsActiveLow())
	assert.False(t, uapi.HandleFlag(0).IsOpenDrain())
	assert.False(t, uapi.HandleFlag(0).IsOpenSource())
	assert.True(t, uapi.HandleRequestInput.IsInput())
	assert.True(t, uapi.HandleRequestOutput.IsOutput())
	assert.True(t, uapi.HandleRequestActiveLow.IsActiveLow())
	assert.True(t, uapi.HandleRequestOpenDrain.IsOpenDrain())
	assert.True(t, uapi.HandleRequestOpenSource.IsOpenSource())
}

func TestEventFlags(t *testing.T) {
	assert.False(t, uapi.EventRequestFallingEdge.IsBothEdges())
	assert.True(t, uapi.EventRequestFallingEdge.IsFallingEdge())
	assert.False(t, uapi.EventRequestFallingEdge.IsRisingEdge())
	assert.False(t, uapi.EventRequestRisingEdge.IsBothEdges())
	assert.False(t, uapi.EventRequestRisingEdge.IsFallingEdge())
	assert.True(t, uapi.EventRequestRisingEdge.IsRisingEdge())
	assert.True(t, uapi.EventRequestBothEdges.IsBothEdges())
	assert.True(t, uapi.EventRequestBothEdges.IsFallingEdge())
	assert.True(t, uapi.EventRequestBothEdges.IsRisingEdge())
}

func lineFromHandle(hf uapi.HandleFlag) uapi.LineFlag {
	var lf uapi.LineFlag
	if hf.IsOutput() {
		lf |= uapi.LineFlagIsOut
	}
	if hf.IsActiveLow() {
		lf |= uapi.LineFlagActiveLow
	}
	if hf.IsOpenDrain() {
		lf |= uapi.LineFlagOpenDrain | uapi.LineFlagIsOut
	}
	if hf.IsOpenSource() {
		lf |= uapi.LineFlagOpenSource | uapi.LineFlagIsOut
	}
	return lf
}
