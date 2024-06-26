// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

package uapi_test

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiocdev/uapi"
	"github.com/warthog618/go-gpiosim"
	"golang.org/x/sys/unix"
)

var (
	setConfigKernel = uapi.Semver{5, 5} // setLineConfig ioctl added
	infoWatchKernel = uapi.Semver{5, 7} // watchLineInfo ioctl added

	// linux kernel timers typically have this granularity, so base timeouts on this...
	clkTick                  = 10 * time.Millisecond
	eventWaitTimeout         = 10 * clkTick
	spuriousEventWaitTimeout = 30 * clkTick
)

func TestGetChipInfo(t *testing.T) {
	s, err := gpiosim.NewSim(
		gpiosim.WithName("gpiosim_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8)),
		gpiosim.WithBank(gpiosim.NewBank("right", 42)),
	)
	require.Nil(t, err)
	defer s.Close()
	for _, c := range s.Chips {
		f := func(t *testing.T) {
			f, err := os.Open(c.DevPath())
			require.Nil(t, err)
			defer f.Close()
			xci := uapi.ChipInfo{
				Lines: uint32(c.Config().NumLines),
			}
			copy(xci.Name[:], c.ChipName())
			copy(xci.Label[:], c.Config().Label)
			ci, err := uapi.GetChipInfo(f.Fd())
			assert.Nil(t, err)
			assert.Equal(t, xci, ci)
		}
		t.Run(c.ChipName(), f)
	}
	// badfd
	f, err := os.CreateTemp("", "uapi_test")
	require.Nil(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	ci, err := uapi.GetChipInfo(f.Fd())
	cix := uapi.ChipInfo{}
	assert.NotNil(t, err)
	assert.Equal(t, cix, ci)
}

func checkLineInfo(t *testing.T, f *os.File, k gpiosim.Bank) {
	for o := 0; o < k.NumLines; o++ {
		xli := uapi.LineInfo{
			Offset: uint32(o),
		}
		if name, ok := k.Names[o]; ok {
			copy(xli.Name[:], name)
		}
		if hog, ok := k.Hogs[o]; ok {
			if hog.Direction != gpiosim.HogDirectionInput {
				xli.Flags = uapi.LineFlagIsOut
			}
			xli.Flags |= uapi.LineFlagUsed
			copy(xli.Consumer[:], []byte(hog.Consumer))
		}
		li, err := uapi.GetLineInfo(f.Fd(), o)
		assert.Nil(t, err)
		assert.Equal(t, xli, li)
	}
	// out of range
	_, err := uapi.GetLineInfo(f.Fd(), k.NumLines)
	assert.Equal(t, unix.EINVAL, err)
}

func TestGetLineInfo(t *testing.T) {
	s, err := gpiosim.NewSim(
		gpiosim.WithName("gpiosim_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8,
			gpiosim.WithNamedLine(3, "LED0"),
			gpiosim.WithNamedLine(5, "BUTTON1"),
			gpiosim.WithHoggedLine(2, "piggy", gpiosim.HogDirectionOutputLow),
		)),
		gpiosim.WithBank(gpiosim.NewBank("right", 42,
			gpiosim.WithNamedLine(3, "BUTTON2"),
			gpiosim.WithNamedLine(4, "LED2"),
			gpiosim.WithHoggedLine(7, "hogster", gpiosim.HogDirectionOutputHigh),
			gpiosim.WithHoggedLine(9, "piggy", gpiosim.HogDirectionInput),
		)),
	)
	require.Nil(t, err)
	defer s.Close()

	for _, c := range s.Chips {
		f, err := os.Open(c.DevPath())
		require.Nil(t, err)
		defer f.Close()
		checkLineInfo(t, f, c.Config())
	}

	// badfd
	f, err := os.CreateTemp("", "uapi_test")
	require.Nil(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	li, err := uapi.GetLineInfo(f.Fd(), 1)
	xli := uapi.LineInfo{}
	assert.NotNil(t, err)
	assert.Equal(t, xli, li)
}

func TestGetLineEvent(t *testing.T) {
	patterns := []struct {
		name       string // unique name for pattern (hf/ef/offsets/xval combo)
		handleFlag uapi.HandleFlag
		eventFlag  uapi.EventFlag
		offset     uint32
		err        error
	}{
		{
			"as-is",
			0,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		{
			"atv-lo",
			uapi.HandleRequestActiveLow,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		{
			"input",
			uapi.HandleRequestInput,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		{
			"input pull-up",
			uapi.HandleRequestInput |
				uapi.HandleRequestPullUp,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		{
			"input pull-down",
			uapi.HandleRequestInput |
				uapi.HandleRequestPullDown,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		{
			"input bias disable",
			uapi.HandleRequestInput |
				uapi.HandleRequestBiasDisable,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		{
			"as-is pull-up",
			uapi.HandleRequestPullUp,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		{
			"as-is pull-down",
			uapi.HandleRequestPullDown,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		{
			"as-is bias disable",
			uapi.HandleRequestBiasDisable,
			uapi.EventRequestBothEdges,
			2,
			nil,
		},
		// expected errors
		{
			"output",
			uapi.HandleRequestOutput,
			uapi.EventRequestBothEdges,
			2,
			unix.EINVAL,
		},
		{
			"oorange",
			uapi.HandleRequestInput,
			uapi.EventRequestBothEdges,
			6,
			unix.EINVAL,
		},
		{
			"input drain",
			uapi.HandleRequestInput |
				uapi.HandleRequestOpenDrain,
			uapi.EventRequestBothEdges,
			2,
			unix.EINVAL,
		},
		{
			"input source",
			uapi.HandleRequestInput |
				uapi.HandleRequestOpenSource,
			uapi.EventRequestBothEdges,
			2,
			unix.EINVAL,
		},
		{
			"as-is drain",
			uapi.HandleRequestOpenDrain,
			uapi.EventRequestBothEdges,
			2,
			unix.EINVAL,
		},
		{
			"as-is source",
			uapi.HandleRequestOpenSource,
			uapi.EventRequestBothEdges,
			2,
			unix.EINVAL,
		},
		{
			"bias disable and pull-up",
			uapi.HandleRequestInput |
				uapi.HandleRequestBiasDisable |
				uapi.HandleRequestPullUp,
			uapi.EventRequestBothEdges,
			2,
			unix.EINVAL,
		},
		{
			"bias disable and pull-down",
			uapi.HandleRequestInput |
				uapi.HandleRequestBiasDisable |
				uapi.HandleRequestPullDown,
			uapi.EventRequestBothEdges,
			2,
			unix.EINVAL,
		},
		{
			"pull-up and pull-down",
			uapi.HandleRequestInput |
				uapi.HandleRequestPullUp |
				uapi.HandleRequestPullDown,
			uapi.EventRequestBothEdges,
			2,
			unix.EINVAL,
		},
	}
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			if p.handleFlag.HasBiasFlag() {
				requireKernel(t, setConfigKernel)
			}
			f, err := os.Open(s.DevPath())
			require.Nil(t, err)
			defer f.Close()
			er := uapi.EventRequest{
				Offset:      p.offset,
				HandleFlags: p.handleFlag,
				EventFlags:  p.eventFlag,
			}
			copy(er.Consumer[:], p.name)
			err = uapi.GetLineEvent(f.Fd(), &er)
			assert.Equal(t, p.err, err)
			if p.offset > uint32(s.Config().NumLines) {
				return
			}
			li, err := uapi.GetLineInfo(f.Fd(), int(p.offset))
			assert.Nil(t, err)
			if p.err != nil {
				assert.False(t, li.Flags.IsUsed())
				unix.Close(int(er.Fd))
				return
			}
			xli := uapi.LineInfo{
				Offset: p.offset,
				Flags:  uapi.LineFlagUsed | lineFromHandle(p.handleFlag),
			}
			copy(xli.Consumer[:31], p.name)
			assert.Equal(t, xli, li)
			unix.Close(int(er.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineHandle(t *testing.T) {
	patterns := []struct {
		name       string // unique name for pattern (hf/ef/offsets/xval combo)
		handleFlag uapi.HandleFlag
		offsets    []uint32
		err        error
	}{
		{
			"as-is",
			0,
			[]uint32{2},
			nil,
		},
		{
			"atv-lo",
			uapi.HandleRequestActiveLow,
			[]uint32{2},
			nil,
		},
		{
			"input",
			uapi.HandleRequestInput,
			[]uint32{2},
			nil,
		},
		{
			"input pull-up",
			uapi.HandleRequestInput |
				uapi.HandleRequestPullUp,
			[]uint32{2},
			nil,
		},
		{
			"input pull-down",
			uapi.HandleRequestInput |
				uapi.HandleRequestPullDown,
			[]uint32{3},
			nil,
		},
		{
			"input bias disable",
			uapi.HandleRequestInput |
				uapi.HandleRequestBiasDisable,
			[]uint32{3},
			nil,
		},
		{
			"output",
			uapi.HandleRequestOutput,
			[]uint32{2},
			nil,
		},
		{
			"output drain",
			uapi.HandleRequestOutput |
				uapi.HandleRequestOpenDrain,
			[]uint32{2},
			nil,
		},
		{
			"output source",
			uapi.HandleRequestOutput |
				uapi.HandleRequestOpenSource,
			[]uint32{3},
			nil,
		},
		{
			"output pull-up",
			uapi.HandleRequestOutput |
				uapi.HandleRequestPullUp,
			[]uint32{1},
			nil,
		},
		{
			"output pull-down",
			uapi.HandleRequestOutput |
				uapi.HandleRequestPullDown,
			[]uint32{2},
			nil,
		},
		{
			"output bias disable",
			uapi.HandleRequestOutput |
				uapi.HandleRequestBiasDisable,
			[]uint32{2},
			nil,
		},
		// expected errors
		{
			"both io",
			uapi.HandleRequestInput |
				uapi.HandleRequestOutput,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"overlength",
			uapi.HandleRequestInput,
			[]uint32{0, 1, 2, 3, 4},
			unix.EINVAL,
		},
		{
			"oorange",
			uapi.HandleRequestInput,
			[]uint32{6},
			unix.EINVAL,
		},
		{
			"input drain",
			uapi.HandleRequestInput |
				uapi.HandleRequestOpenDrain,
			[]uint32{1},
			unix.EINVAL,
		},
		{
			"input source",
			uapi.HandleRequestInput |
				uapi.HandleRequestOpenSource,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"as-is drain",
			uapi.HandleRequestOpenDrain,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"as-is source",
			uapi.HandleRequestOpenSource,
			[]uint32{1},
			unix.EINVAL,
		},
		{
			"drain source",
			uapi.HandleRequestOutput |
				uapi.HandleRequestOpenDrain |
				uapi.HandleRequestOpenSource,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"as-is pull-up",
			uapi.HandleRequestPullUp,
			[]uint32{1},
			unix.EINVAL,
		},
		{
			"as-is pull-down",
			uapi.HandleRequestPullDown,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"as-is bias disable",
			uapi.HandleRequestBiasDisable,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"bias disable and pull-up",
			uapi.HandleRequestInput |
				uapi.HandleRequestBiasDisable |
				uapi.HandleRequestPullUp,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"bias disable and pull-down",
			uapi.HandleRequestInput |
				uapi.HandleRequestBiasDisable |
				uapi.HandleRequestPullDown,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"pull-up and pull-down",
			uapi.HandleRequestInput |
				uapi.HandleRequestPullUp |
				uapi.HandleRequestPullDown,
			[]uint32{2},
			unix.EINVAL,
		},
		{
			"all bias flags",
			uapi.HandleRequestInput |
				uapi.HandleRequestBiasDisable |
				uapi.HandleRequestPullUp |
				uapi.HandleRequestPullDown,
			[]uint32{2},
			unix.EINVAL,
		},
	}
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			if p.handleFlag.HasBiasFlag() {
				requireKernel(t, setConfigKernel)
			}
			f, err := os.Open(s.DevPath())
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
			if p.offsets[0] > uint32(s.Config().NumLines) {
				return
			}
			// check line info
			li, err := uapi.GetLineInfo(f.Fd(), int(p.offsets[0]))
			assert.Nil(t, err)
			if p.err != nil {
				assert.False(t, li.Flags.IsUsed())
				unix.Close(int(hr.Fd))
				return
			}
			xli := uapi.LineInfo{
				Offset: p.offsets[0],
				Flags:  uapi.LineFlagUsed | lineFromHandle(p.handleFlag),
			}
			copy(xli.Consumer[:31], p.name)
			assert.Equal(t, xli, li)
			unix.Close(int(hr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineValues(t *testing.T) {
	patterns := []struct {
		name       string // unique name for pattern (hf/ef/offsets/xval combo)
		handleFlag uapi.HandleFlag
		evtFlag    uapi.EventFlag
		offsets    []uint32
		val        []uint8
	}{
		{
			"as-is atv-lo lo",
			uapi.HandleRequestActiveLow,
			0,
			[]uint32{2},
			[]uint8{0},
		},
		{
			"as-is atv-lo hi",
			uapi.HandleRequestActiveLow,
			0,
			[]uint32{2},
			[]uint8{1},
		},
		{
			"as-is lo",
			0,
			0,
			[]uint32{2},
			[]uint8{0},
		},
		{
			"as-is hi",
			0,
			0,
			[]uint32{1},
			[]uint8{1},
		},
		{
			"input lo",
			uapi.HandleRequestInput,
			0,
			[]uint32{2},
			[]uint8{0},
		},
		{
			"input hi",
			uapi.HandleRequestInput,
			0,
			[]uint32{1},
			[]uint8{1},
		},
		{
			"output lo",
			uapi.HandleRequestOutput,
			0,
			[]uint32{2},
			[]uint8{0},
		},
		{
			"output hi",
			uapi.HandleRequestOutput,
			0,
			[]uint32{1},
			[]uint8{1},
		},
		{
			"both lo",
			0,
			uapi.EventRequestBothEdges,
			[]uint32{2},
			[]uint8{0},
		},
		{
			"both hi",
			0,
			uapi.EventRequestBothEdges,
			[]uint32{1},
			[]uint8{1},
		},
		{
			"falling lo",
			0,
			uapi.EventRequestFallingEdge,
			[]uint32{2},
			[]uint8{0},
		},
		{
			"falling hi",
			0,
			uapi.EventRequestFallingEdge,
			[]uint32{1},
			[]uint8{1},
		},
		{
			"rising lo",
			0,
			uapi.EventRequestRisingEdge,
			[]uint32{2},
			[]uint8{0},
		},
		{
			"rising hi",
			0,
			uapi.EventRequestRisingEdge,
			[]uint32{1},
			[]uint8{1},
		},
		{
			"input 2a",
			uapi.HandleRequestInput,
			0,
			[]uint32{0, 1},
			[]uint8{1, 0},
		},
		{
			"input 2b",
			uapi.HandleRequestInput,
			0,
			[]uint32{2, 1},
			[]uint8{0, 1},
		},
		{
			"input 3a",
			uapi.HandleRequestInput,
			0,
			[]uint32{0, 1, 2},
			[]uint8{0, 1, 1},
		},
		{
			"input 3b",
			uapi.HandleRequestInput,
			0,
			[]uint32{0, 2, 1},
			[]uint8{0, 1, 0},
		},
		{
			"input 4a",
			uapi.HandleRequestInput,
			0,
			[]uint32{0, 1, 2, 3},
			[]uint8{0, 1, 1, 1},
		},
		{
			"input 4b",
			uapi.HandleRequestInput,
			0,
			[]uint32{3, 2, 1, 0},
			[]uint8{1, 1, 0, 1},
		},
		{
			"input 8a",
			uapi.HandleRequestInput,
			0,
			[]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			[]uint8{0, 1, 1, 1, 1, 1, 0, 0},
		},
		{
			"input 8b",
			uapi.HandleRequestInput,
			0,
			[]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			[]uint8{1, 1, 0, 1, 1, 1, 0, 1},
		},
		{
			"atv-lo 8b",
			uapi.HandleRequestInput |
				uapi.HandleRequestActiveLow,
			0,
			[]uint32{3, 2, 1, 0, 4, 6, 7},
			[]uint8{1, 1, 0, 1, 1, 1, 0, 0},
		},
	}
	s, err := gpiosim.NewSimpleton(8)
	require.Nil(t, err)
	defer s.Close()
	for _, p := range patterns {
		// set vals in mock
		require.LessOrEqual(t, len(p.offsets), len(p.val))
		for i, o := range p.offsets {
			v := int(p.val[i])
			if p.handleFlag.IsActiveLow() {
				v ^= 0x01 // assumes using 1 for high
			}
			err := s.SetPull(int(o), v)
			assert.Nil(t, err)
		}
		tf := func(t *testing.T) {
			f, err := os.Open(s.DevPath())
			require.Nil(t, err)
			defer f.Close()
			var fd int32
			xval := p.val
			if p.evtFlag == 0 {
				hr := uapi.HandleRequest{
					Flags: p.handleFlag,
					Lines: uint32(len(p.offsets)),
				}
				copy(hr.Consumer[:31], "test-get-line-values")
				copy(hr.Offsets[:], p.offsets)
				err := uapi.GetLineHandle(f.Fd(), &hr)
				require.Nil(t, err)
				fd = hr.Fd
				if p.handleFlag.IsOutput() {
					// sim is ignored for outputs
					xval = make([]uint8, len(p.val))
				}
			} else {
				assert.Equal(t, 1, len(p.offsets)) // reminder that events are limited to one line
				er := uapi.EventRequest{
					Offset:      p.offsets[0],
					HandleFlags: p.handleFlag,
					EventFlags:  p.evtFlag,
				}
				copy(er.Consumer[:31], "test-get-line-values")
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
	f, err := os.CreateTemp("", "uapi_test")
	require.Nil(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	err = uapi.GetLineValues(f.Fd(), &hd)
	assert.NotNil(t, err)
	assert.Equal(t, hdx, hd)
}

func TestSetLineValues(t *testing.T) {
	patterns := []struct {
		name       string // unique name for pattern (hf/ef/offsets/xval combo)
		handleFlag uapi.HandleFlag
		offsets    []uint32
		val        []uint8
		err        error
	}{
		{
			"output atv-lo lo",
			uapi.HandleRequestOutput |
				uapi.HandleRequestActiveLow,
			[]uint32{2},
			[]uint8{0},
			nil,
		},
		{
			"output atv-lo hi",
			uapi.HandleRequestOutput |
				uapi.HandleRequestActiveLow,
			[]uint32{2},
			[]uint8{1},
			nil,
		},
		{
			"as-is lo",
			0,
			[]uint32{2},
			[]uint8{0},
			nil,
		},
		{
			"as-is hi",
			0,
			[]uint32{2},
			[]uint8{1},
			nil,
		},
		{
			"output lo",
			uapi.HandleRequestOutput,
			[]uint32{2},
			[]uint8{0},
			nil,
		},
		{
			"output hi",
			uapi.HandleRequestOutput,
			[]uint32{1},
			[]uint8{1},
			nil,
		},
		{
			"output 2a",
			uapi.HandleRequestOutput,
			[]uint32{0, 1},
			[]uint8{1, 0},
			nil,
		},
		{
			"output 2b",
			uapi.HandleRequestOutput,
			[]uint32{2, 1},
			[]uint8{0, 1},
			nil,
		},
		{
			"output 3a",
			uapi.HandleRequestOutput,
			[]uint32{0, 1, 2},
			[]uint8{0, 1, 1},
			nil,
		},
		{
			"output 3b",
			uapi.HandleRequestOutput,
			[]uint32{0, 2, 1},
			[]uint8{0, 1, 0},
			nil,
		},
		{
			"output 4a",
			uapi.HandleRequestOutput,
			[]uint32{0, 1, 2, 3},
			[]uint8{0, 1, 1, 1},
			nil,
		},
		{
			"output 4b",
			uapi.HandleRequestOutput,
			[]uint32{3, 2, 1, 0},
			[]uint8{1, 1, 0, 1},
			nil,
		},
		{
			"output 8a",
			uapi.HandleRequestOutput,
			[]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			[]uint8{0, 1, 1, 1, 1, 1, 0, 0},
			nil,
		},
		{
			"output 8b",
			uapi.HandleRequestOutput,
			[]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			[]uint8{1, 1, 0, 1, 1, 1, 0, 1},
			nil,
		},
		{
			"atv-lo 8b",
			uapi.HandleRequestOutput |
				uapi.HandleRequestActiveLow,
			[]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			[]uint8{1, 1, 0, 1, 1, 0, 0, 0},
			nil,
		},
		// expected failures....
		{
			"input lo",
			uapi.HandleRequestInput,
			[]uint32{2},
			[]uint8{0},
			unix.EPERM,
		},
		{
			"input hi",
			uapi.HandleRequestInput,
			[]uint32{1},
			[]uint8{1},
			unix.EPERM,
		},
	}
	s, err := gpiosim.NewSimpleton(8)
	require.Nil(t, err)
	defer s.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			require.LessOrEqual(t, len(p.offsets), len(p.val))
			f, err := os.Open(s.DevPath())
			require.Nil(t, err)
			defer f.Close()
			hr := uapi.HandleRequest{
				Flags: p.handleFlag,
				Lines: uint32(len(p.offsets)),
			}
			copy(hr.Consumer[:31], "test-set-line-values")
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
					v, err := s.Level(int(o))
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
	f, err := os.CreateTemp("", "uapi_test")
	require.Nil(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	err = uapi.SetLineValues(f.Fd(), hd)
	assert.NotNil(t, err)
}

func TestSetLineHandleConfig(t *testing.T) {
	requireKernel(t, setConfigKernel)
	patterns := []struct {
		name        string
		offsets     []uint32
		initialFlag uapi.HandleFlag
		initialVal  []uint8
		configFlag  uapi.HandleFlag
		configVal   []uint8
		err         error
	}{
		{
			"in to out",
			[]uint32{1, 2, 3},
			uapi.HandleRequestInput,
			[]uint8{0, 1, 1},
			uapi.HandleRequestOutput,
			[]uint8{1, 0, 1},
			nil,
		},
		{
			"out to in",
			[]uint32{1, 2, 3},
			uapi.HandleRequestOutput,
			[]uint8{1, 0, 1},
			uapi.HandleRequestInput,
			[]uint8{1, 0, 1},
			nil,
		},
		{
			"as-is atv-hi to as-is atv-lo",
			[]uint32{1, 2, 3},
			0,
			[]uint8{1, 0, 1},
			uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			unix.EINVAL,
		},
		{
			"as-is atv-lo to as-is atv-hi",
			[]uint32{1, 2, 3},
			uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			0,
			[]uint8{1, 0, 1},
			unix.EINVAL,
		},
		{
			"input atv-lo to input atv-hi",
			[]uint32{1, 2, 3},
			uapi.HandleRequestInput |
				uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			uapi.HandleRequestInput,
			[]uint8{1, 0, 1},
			nil,
		},
		{
			"input atv-hi to input atv-lo",
			[]uint32{1, 2, 3},
			uapi.HandleRequestInput,
			[]uint8{1, 0, 1},
			uapi.HandleRequestInput |
				uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			nil,
		},
		{
			"output atv-lo to output atv-hi",
			[]uint32{1, 2, 3},
			uapi.HandleRequestOutput |
				uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			uapi.HandleRequestOutput,
			[]uint8{0, 1, 1},
			nil,
		},
		{
			"output atv-hi to output atv-lo",
			[]uint32{1, 2, 3},
			uapi.HandleRequestOutput,
			[]uint8{1, 0, 1},
			uapi.HandleRequestOutput |
				uapi.HandleRequestActiveLow,
			[]uint8{0, 1, 1},
			nil,
		},
		{
			"input atv-lo to as-is atv-hi",
			[]uint32{1, 2, 3},
			uapi.HandleRequestInput |
				uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			0,
			[]uint8{1, 0, 1},
			unix.EINVAL,
		},
		{
			"input atv-hi to as-is atv-lo",
			[]uint32{1, 2, 3},
			uapi.HandleRequestInput,
			[]uint8{1, 0, 1},
			uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			unix.EINVAL,
		},
		{
			"input pull-up to input pull-down",
			[]uint32{1, 2, 3},
			uapi.HandleRequestInput |
				uapi.HandleRequestPullUp,
			nil,
			uapi.HandleRequestInput |
				uapi.HandleRequestPullDown,
			[]uint8{0, 0, 0},
			nil,
		},
		{
			"input pull-down to input pull-up",
			[]uint32{1, 2, 3},
			uapi.HandleRequestInput |
				uapi.HandleRequestPullDown,
			nil,
			uapi.HandleRequestInput |
				uapi.HandleRequestPullUp,
			[]uint8{1, 1, 1},
			nil,
		},
		{
			"output atv-lo to as-is atv-hi",
			[]uint32{1, 2, 3},
			uapi.HandleRequestOutput |
				uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			0,
			[]uint8{1, 0, 1},
			unix.EINVAL,
		},
		{
			"output atv-hi to as-is atv-lo",
			[]uint32{1, 2, 3},
			uapi.HandleRequestOutput,
			[]uint8{1, 0, 1},
			uapi.HandleRequestActiveLow,
			[]uint8{1, 0, 1},
			unix.EINVAL,
		},
		// expected errors
		{
			"input drain",
			[]uint32{2},
			uapi.HandleRequestInput,
			nil,
			uapi.HandleRequestInput |
				uapi.HandleRequestOpenDrain,
			nil,
			unix.EINVAL,
		},
		{
			"input source",
			[]uint32{2},
			uapi.HandleRequestInput,
			nil,
			uapi.HandleRequestInput |
				uapi.HandleRequestOpenSource,
			nil,
			unix.EINVAL,
		},
		{
			"as-is drain",
			[]uint32{2},
			0,
			nil,
			uapi.HandleRequestOpenDrain,
			nil,
			unix.EINVAL,
		},
		{
			"as-is source",
			[]uint32{2},
			0,
			nil,
			uapi.HandleRequestOpenSource,
			nil,
			unix.EINVAL,
		},
		{
			"drain source",
			[]uint32{2},
			uapi.HandleRequestOutput,
			nil,
			uapi.HandleRequestOutput |
				uapi.HandleRequestOpenDrain |
				uapi.HandleRequestOpenSource,
			nil,
			unix.EINVAL,
		},
		{
			"pull-up and pull-down",
			[]uint32{2},
			uapi.HandleRequestOutput,
			nil,
			uapi.HandleRequestOutput |
				uapi.HandleRequestPullUp |
				uapi.HandleRequestPullDown,
			nil,
			unix.EINVAL,
		},
		{
			"bias disable and pull-up",
			[]uint32{2},
			uapi.HandleRequestOutput,
			nil,
			uapi.HandleRequestOutput |
				uapi.HandleRequestBiasDisable |
				uapi.HandleRequestPullUp,
			nil,
			unix.EINVAL,
		},
		{
			"bias disable and pull-down",
			[]uint32{2},
			uapi.HandleRequestOutput,
			nil,
			uapi.HandleRequestOutput |
				uapi.HandleRequestBiasDisable |
				uapi.HandleRequestPullDown,
			nil,
			unix.EINVAL,
		},
		{
			"all bias flags",
			[]uint32{2},
			uapi.HandleRequestOutput,
			nil,
			uapi.HandleRequestOutput |
				uapi.HandleRequestBiasDisable |
				uapi.HandleRequestPullDown |
				uapi.HandleRequestPullUp,
			nil,
			unix.EINVAL,
		},
	}
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			// setup sim for inputs
			if p.initialVal != nil {
				for i, o := range p.offsets {
					v := int(p.initialVal[i])
					// read is after config, so use config active state
					if p.configFlag.IsActiveLow() {
						v ^= 0x01 // assumes using 1 for high
					}
					err := s.SetPull(int(o), v)
					assert.Nil(t, err)
				}
			}
			f, err := os.Open(s.DevPath())
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
					assert.False(t, li.Flags.IsUsed())
					return
				}
				xli := uapi.LineInfo{
					Offset: p.offsets[0],
					Flags: uapi.LineFlagUsed |
						lineFromConfig(p.initialFlag, p.configFlag),
				}
				copy(xli.Consumer[:31], p.name)
				assert.Equal(t, xli, li)
				if len(p.configVal) != 0 {
					// check values from sim
					require.LessOrEqual(t, len(p.offsets), len(p.configVal))
					for i, o := range p.offsets {
						v, err := s.Level(int(o))
						assert.Nil(t, err)
						xv := int(p.configVal[i])
						if p.configFlag.IsActiveLow() {
							xv ^= 0x01 // assumes using 1 for high
						}
						assert.Equal(t, xv, v, i)
					}
				}
			}
			unix.Close(int(hr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestSetLineEventConfig(t *testing.T) {
	requireKernel(t, setConfigKernel)
	patterns := []struct {
		name        string
		offset      uint32
		initialFlag uapi.HandleFlag
		configFlag  uapi.HandleFlag
		err         error
	}{
		// expected errors
		{
			"low to high",
			1,
			uapi.HandleRequestInput |
				uapi.HandleRequestActiveLow,
			0,
			unix.EINVAL,
		},
		{
			"high to low",
			2,
			uapi.HandleRequestInput,
			uapi.HandleRequestInput |
				uapi.HandleRequestActiveLow,
			unix.EINVAL,
		},
		{
			"in to out",
			2,
			uapi.HandleRequestInput,
			uapi.HandleRequestOutput,
			unix.EINVAL,
		},
		{
			"drain",
			3,
			uapi.HandleRequestInput,
			uapi.HandleRequestInput |
				uapi.HandleRequestOpenDrain,
			unix.EINVAL,
		},
		{
			"source",
			3,
			uapi.HandleRequestInput,
			uapi.HandleRequestInput |
				uapi.HandleRequestOpenSource,
			unix.EINVAL,
		},
	}
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			f, err := os.Open(s.DevPath())
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
					assert.False(t, li.Flags.IsUsed())
					return
				}
				xli := uapi.LineInfo{
					Offset: p.offset,
					Flags: uapi.LineFlagUsed |
						lineFromConfig(p.initialFlag, p.configFlag),
				}
				copy(xli.Consumer[:31], p.name)
				assert.Equal(t, xli, li)
			}
			unix.Close(int(er.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestWatchLineInfo(t *testing.T) {
	// also covers ReadLineInfoChanged

	requireKernel(t, infoWatchKernel)
	s, err := gpiosim.NewSim(
		gpiosim.WithName("gpiosim_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8,
			gpiosim.WithNamedLine(3, "LED0"),
		)),
	)
	require.Nil(t, err)
	defer s.Close()

	c := s.Chips[0]
	f, err := os.Open(c.DevPath())
	require.Nil(t, err)
	defer f.Close()

	offset := uint32(3)

	// unwatched
	hr := uapi.HandleRequest{Lines: 1, Flags: uapi.HandleRequestInput}
	hr.Offsets[0] = offset
	copy(hr.Consumer[:], "testwatch")
	err = uapi.GetLineHandle(f.Fd(), &hr)
	assert.Nil(t, err)
	chg, err := readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")
	unix.Close(int(hr.Fd))

	// out of range
	li := uapi.LineInfo{Offset: uint32(c.Config().NumLines + 1)}
	err = uapi.WatchLineInfo(f.Fd(), &li)
	require.Equal(t, syscall.Errno(0x16), err)

	// set watch
	li = uapi.LineInfo{Offset: offset}
	err = uapi.WatchLineInfo(f.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfo{Offset: offset}
	copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
	assert.Equal(t, xli, li)

	// repeated watch
	err = uapi.WatchLineInfo(f.Fd(), &li)
	assert.Equal(t, unix.EBUSY, err)

	chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// request line
	hr = uapi.HandleRequest{Lines: 1, Flags: uapi.HandleRequestInput}
	hr.Offsets[0] = offset
	copy(hr.Consumer[:], "testwatch")
	err = uapi.GetLineHandle(f.Fd(), &hr)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedRequested, chg.Type)
	xli.Flags |= uapi.LineFlagUsed
	copy(xli.Consumer[:], "testwatch")
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// reconfig line
	hc := uapi.HandleConfig{Flags: uapi.HandleRequestInput | uapi.HandleRequestActiveLow}
	copy(hr.Consumer[:], "testwatch")
	err = uapi.SetLineConfig(uintptr(hr.Fd), &hc)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedConfig, chg.Type)
	xli.Flags |= uapi.LineFlagActiveLow
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// release line
	unix.Close(int(hr.Fd))
	chg, err = readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedReleased, chg.Type)
	xli = uapi.LineInfo{Offset: offset}
	copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")
}

func TestUnwatchLineInfo(t *testing.T) {
	requireKernel(t, infoWatchKernel)
	s, err := gpiosim.NewSim(
		gpiosim.WithName("gpiosim_test"),
		gpiosim.WithBank(gpiosim.NewBank("left", 8,
			gpiosim.WithNamedLine(3, "LED0"),
		)),
	)
	require.Nil(t, err)
	defer s.Close()

	c := s.Chips[0]
	f, err := os.Open(c.DevPath())
	require.Nil(t, err)
	defer f.Close()

	li := uapi.LineInfo{Offset: uint32(c.Config().NumLines + 1)}
	err = uapi.UnwatchLineInfo(f.Fd(), li.Offset)
	require.Equal(t, syscall.Errno(0x16), err)

	offset := uint32(3)
	li = uapi.LineInfo{Offset: offset}
	err = uapi.WatchLineInfo(f.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfo{Offset: offset}
	copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
	assert.Equal(t, xli, li)

	chg, err := readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	err = uapi.UnwatchLineInfo(f.Fd(), li.Offset)
	assert.Nil(t, err)

	// request line
	hr := uapi.HandleRequest{Lines: 1, Flags: uapi.HandleRequestInput}
	hr.Offsets[0] = offset
	err = uapi.GetLineHandle(f.Fd(), &hr)
	assert.Nil(t, err)
	unix.Close(int(hr.Fd))
	chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// repeated unwatch
	err = uapi.UnwatchLineInfo(f.Fd(), offset)
	require.Equal(t, unix.EBUSY, err)

	// repeated watch
	err = uapi.WatchLineInfo(f.Fd(), &li)
	require.Nil(t, err)
}

func TestReadEvent(t *testing.T) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	err = s.SetPull(offset, 0)
	require.Nil(t, err)
	er := uapi.EventRequest{
		Offset: uint32(offset),
		HandleFlags: uapi.HandleRequestInput |
			uapi.HandleRequestActiveLow,
		EventFlags: uapi.EventRequestBothEdges,
	}
	err = uapi.GetLineEvent(f.Fd(), &er)
	require.Nil(t, err)
	defer unix.Close(int(er.Fd))

	evt, err := readEventTimeout(er.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	s.SetPull(offset, 1)
	evt, err = readEventTimeout(er.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.EventRequestFallingEdge, evt.ID)

	s.SetPull(offset, 0)
	evt, err = readEventTimeout(er.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.EventRequestRisingEdge, evt.ID)
}

func readEventTimeout(fd int32, t time.Duration) (*uapi.EventData, error) {
	pollfd := unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
	for {
		n, err := unix.Poll([]unix.PollFd{pollfd}, int(t.Milliseconds()))
		if err == unix.EINTR {
			continue
		}
		if err != nil || n != 1 {
			return nil, err
		}
		break
	}
	evt, err := uapi.ReadEvent(uintptr(fd))
	if err != nil {
		return nil, err
	}
	return &evt, nil
}

func readLineInfoChangedTimeout(fd uintptr,
	t time.Duration) (*uapi.LineInfoChanged, error) {

	pollfd := unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
	for {
		n, err := unix.Poll([]unix.PollFd{pollfd}, int(t.Milliseconds()))
		if err == unix.EINTR {
			continue
		}
		if err != nil || n != 1 {
			return nil, err
		}
		break
	}
	infoChanged, err := uapi.ReadLineInfoChanged(fd)
	if err != nil {
		return nil, err
	}
	return &infoChanged, nil
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
	assert.False(t, uapi.LineFlag(0).IsUsed())
	assert.False(t, uapi.LineFlag(0).IsOut())
	assert.False(t, uapi.LineFlag(0).IsActiveLow())
	assert.False(t, uapi.LineFlag(0).IsOpenDrain())
	assert.False(t, uapi.LineFlag(0).IsOpenSource())
	assert.True(t, uapi.LineFlagUsed.IsUsed())
	assert.True(t, uapi.LineFlagIsOut.IsOut())
	assert.True(t, uapi.LineFlagActiveLow.IsActiveLow())
	assert.True(t, uapi.LineFlagOpenDrain.IsOpenDrain())
	assert.False(t, uapi.LineFlagOpenDrain.IsOpenSource())
	assert.True(t, uapi.LineFlagOpenSource.IsOpenSource())
	assert.False(t, uapi.LineFlagOpenSource.IsOpenDrain())
	assert.True(t, uapi.LineFlagPullUp.IsPullUp())
	assert.False(t, uapi.LineFlagPullUp.IsPullDown())
	assert.False(t, uapi.LineFlagPullUp.IsBiasDisable())
	assert.True(t, uapi.LineFlagPullDown.IsPullDown())
	assert.False(t, uapi.LineFlagPullDown.IsBiasDisable())
	assert.False(t, uapi.LineFlagPullDown.IsPullUp())
	assert.True(t, uapi.LineFlagBiasDisabled.IsBiasDisable())
	assert.False(t, uapi.LineFlagBiasDisabled.IsPullUp())
	assert.False(t, uapi.LineFlagBiasDisabled.IsPullDown())
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
	assert.False(t, uapi.HandleRequestOpenDrain.IsOpenSource())
	assert.True(t, uapi.HandleRequestOpenSource.IsOpenSource())
	assert.False(t, uapi.HandleRequestOpenSource.IsOpenDrain())
	assert.True(t, uapi.HandleRequestPullUp.IsPullUp())
	assert.False(t, uapi.HandleRequestPullUp.IsPullDown())
	assert.False(t, uapi.HandleRequestPullUp.IsBiasDisable())
	assert.True(t, uapi.HandleRequestPullDown.IsPullDown())
	assert.False(t, uapi.HandleRequestPullDown.IsBiasDisable())
	assert.False(t, uapi.HandleRequestPullDown.IsPullUp())
	assert.True(t, uapi.HandleRequestBiasDisable.IsBiasDisable())
	assert.False(t, uapi.HandleRequestBiasDisable.IsPullUp())
	assert.False(t, uapi.HandleRequestBiasDisable.IsPullDown())
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

func lineFromConfig(of, cf uapi.HandleFlag) uapi.LineFlag {
	lf := lineFromHandle(cf)
	if !(cf.IsInput() || cf.IsOutput()) {
		if of.IsOutput() {
			lf |= uapi.LineFlagIsOut
		}
	}
	return lf
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
		lf |= uapi.LineFlagOpenDrain
	}
	if hf.IsOpenSource() {
		lf |= uapi.LineFlagOpenSource
	}
	if hf.IsPullUp() {
		lf |= uapi.LineFlagPullUp
	}
	if hf.IsPullDown() {
		lf |= uapi.LineFlagPullDown
	}
	if hf.IsBiasDisable() {
		lf |= uapi.LineFlagBiasDisabled
	}
	return lf
}

func requireKernel(t *testing.T, min uapi.Semver) {
	if err := uapi.CheckKernelVersion(min); err != nil {
		t.Skip(err)
	}
}
