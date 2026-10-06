// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

package uapi_test

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiosim"
	"golang.org/x/sys/unix"

	"github.com/warthog618/go-gpiocdev/uapi"
)

var (
	debouncePeriod = 5 * clkTick
)

type AttributeEncoder interface {
	Encode() uapi.LineAttribute
}

func checkLineInfo(t *testing.T, f *os.File, k gpiosim.Bank) {
	for o := 0; o < k.NumLines; o++ {
		xli := uapi.LineInfo{
			Offset: uint32(o),
			Flags:  uapi.LineFlagInput,
		}
		if name, ok := k.Names[o]; ok {
			copy(xli.Name[:], name)
		}
		if hog, ok := k.Hogs[o]; ok {
			if hog.Direction != gpiosim.HogDirectionInput {
				xli.Flags = uapi.LineFlagOutput
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

func TestGetLine(t *testing.T) {
	patterns := []struct {
		name string // unique name for pattern (hf/ef/offsets/xval combo)
		lr   uapi.LineRequest
		err  error
	}{
		{
			"as-is",
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"atv-lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input pull-up",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagBiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input pull-down",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagBiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input bias disable",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagBiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input edge rising",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input edge falling",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{1, 3},
			},
			nil,
		},
		{
			"input edge both",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
				},
				Lines:           1,
				Offsets:         [uapi.LinesMax]uint32{3},
				EventBufferSize: 42,
			},
			nil,
		},
		{
			"output",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagOpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagOpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output pull-up",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagBiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			nil,
		},
		{
			"output pull-down",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagBiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output bias disabled",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagBiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			nil,
		},
		// expected errors
		{
			"overlength",
			uapi.LineRequest{
				Lines:   5,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4},
			},
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
			copy(p.lr.Consumer[:], p.name)
			err = uapi.GetLine(f.Fd(), &p.lr)
			assert.Equal(t, p.err, err)
			if p.lr.Offsets[0] > uint32(s.Config().NumLines) {
				return
			}
			// check line info
			li, err := uapi.GetLineInfo(f.Fd(), int(p.lr.Offsets[0]))
			assert.Nil(t, err)
			if p.err != nil {
				assert.True(t, li.Flags.IsAvailable())
				unix.Close(int(p.lr.Fd))
				return
			}
			xli := uapi.LineInfo{
				Offset: p.lr.Offsets[0],
				Flags:  uapi.LineFlagUsed | p.lr.Config.Flags,
			}
			copy(xli.Consumer[:31], p.name)
			if xli.Flags&uapi.LineFlagDirectionMask == 0 {
				xli.Flags |= uapi.LineFlagInput
			}
			assert.Equal(t, xli, li)
			unix.Close(int(p.lr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineValidation(t *testing.T) {
	patterns := []struct {
		name string
		lr   uapi.LineRequest
	}{
		{
			"oorange offset",
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{6},
			},
		},
		{
			"input drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagOpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"input source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagOpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"as-is pull-up",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagBiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"as-is pull-down",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagBiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is bias disabled",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagBiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"output edge rising",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"output edge falling",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagEdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"output edge both",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is edge rising",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is edge falling",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagEdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is edge both",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"non-zero padding",
			uapi.LineRequest{
				Config:  uapi.LineConfig{Padding: [5]uint32{1}},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
	}
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			copy(p.lr.Consumer[:31], "test-get-line-validation")
			err = uapi.GetLine(f.Fd(), &p.lr)
			assert.Equal(t, unix.EINVAL, err)
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineValues(t *testing.T) {
	patterns := []struct {
		name   string
		lr     uapi.LineRequest
		active []int
		mask   []int
		err    error
	}{
		{
			"as-is atv-lo lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"as-is atv-lo hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"as-is lo",
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"as-is hi",
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"input lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"input hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"output lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"output hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"both lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"both hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"falling lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"falling hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"rising lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"rising hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"input 2a",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{0, 1},
			},
			[]int{0},
			[]int{0, 1},
			nil,
		},
		{
			"input 2b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{2, 1},
			},
			[]int{1},
			[]int{0, 1},
			nil,
		},
		{
			"input 3a",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2},
			},
			[]int{1, 2},
			[]int{0, 1, 2},
			nil,
		},
		{
			"input 3b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 2, 1},
			},
			[]int{1},
			[]int{0, 1, 2},
			nil,
		},
		{
			"input 4a",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
			},
			[]int{1, 2, 3},
			[]int{0, 1, 2, 3},
			nil,
		},
		{
			"input 4b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0},
			},
			[]int{0, 1, 3},
			[]int{0, 1, 2, 3},
			nil,
		},
		{
			"input 8a",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"input 8b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5, 7},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"atv-lo 8b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"sparse atv-lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{0, 2, 3, 5, 7},
			nil,
		},
		{
			"sparse atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{0, 2, 3, 5, 7},
			nil,
		},
		{
			"sparse one lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{2},
			nil,
		},
		{
			"sparse one hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{3},
			nil,
		},
		{
			"overwide sparse atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5},
			[]int{0, 2, 3, 5, 7, 8, 9, 10, 11},
			nil,
		},
		{
			"edge detection lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"edge detection hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"zero mask",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
			},
			[]int{0, 2, 3},
			[]int{},
			unix.EINVAL,
		},
	}
	s, err := gpiosim.NewSimpleton(8)
	require.Nil(t, err)
	defer s.Close()
	for _, p := range patterns {
		// set vals in sim
		vals := uapi.NewLineBits(p.active...)
		for i := 0; i < int(p.lr.Lines); i++ {
			v := vals.Get(i)
			o := int(p.lr.Offsets[i])
			if p.lr.Config.Flags.IsActiveLow() {
				v ^= 0x01
			}
			err := s.SetPull(o, v)
			assert.Nil(t, err)
		}
		tf := func(t *testing.T) {
			f, err := os.Open(s.DevPath())
			require.Nil(t, err)
			defer f.Close()
			var fd int32
			mask := uapi.NewLineBits(p.mask...)
			xval := vals & mask
			copy(p.lr.Consumer[:31], "test-get-line-values-")
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			fd = p.lr.Fd
			if p.lr.Config.Flags.IsOutput() {
				// sim pull is ignored for outputs
				xval = 0
			}
			lvx := uapi.LineValues{
				Mask: mask,
				Bits: xval,
			}
			lv := uapi.LineValues{
				Mask: mask,
				Bits: uapi.NewLineBits(p.active...),
			}
			err = uapi.GetLineValues(uintptr(fd), &lv)
			assert.Equal(t, p.err, err)
			if p.err == nil {
				assert.Equal(t, lvx, lv)
			}
			unix.Close(int(fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	lvx := uapi.LineValues{
		Mask: uapi.NewLineBitMask(3),
	}
	lv := uapi.LineValues{
		Mask: uapi.NewLineBitMask(3),
	}
	f, err := os.CreateTemp("", "uapi_test")
	require.Nil(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	err = uapi.GetLineValues(f.Fd(), &lv)
	assert.NotNil(t, err)
	assert.Equal(t, lvx, lv)
}

func TestSetLineValues(t *testing.T) {
	patterns := []struct {
		name   string
		lr     uapi.LineRequest
		active []int
		mask   []int
		err    error
	}{
		{
			"output atv-lo lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow | uapi.LineFlagOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"output atv-lo hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow | uapi.LineFlagOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"as-is lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"as-is hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"output lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			nil,
		},
		{
			"output hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			nil,
		},
		{
			"output 2a",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{0, 1},
			},
			[]int{0},
			[]int{0, 1},
			nil,
		},
		{
			"output 2b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{2, 1},
			},
			[]int{1},
			[]int{0, 1},
			nil,
		},
		{
			"output 3a",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2},
			},
			[]int{1, 2},
			[]int{0, 1, 2},
			nil,
		},
		{
			"output 3b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 2, 1},
			},
			[]int{1},
			[]int{0, 1, 2},
			nil,
		},
		{
			"output 4a",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
			},
			[]int{1, 2, 3},
			[]int{0, 1, 2, 3},
			nil,
		},
		{
			"output 4b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0},
			},
			[]int{0, 1, 3},
			[]int{0, 1, 2, 3},
			nil,
		},
		{
			"output 8a",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"output 8b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5, 7},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"atv-lo 8b",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4},
			[]int{0, 1, 2, 3, 4, 5, 6, 7},
			nil,
		},
		{
			"sparse atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 2, 4, 5, 7},
			nil,
		},
		{
			"sparse one lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 5},
			[]int{4},
			nil,
		},
		{
			"sparse one hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{4},
			nil,
		},
		{
			"overwide sparse atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 2, 4, 5, 7, 8, 9, 10, 11},
			nil,
		},
		{
			"sparse atv-lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{1, 2, 3, 4, 5},
			[]int{0, 2, 4, 5, 7},
			nil,
		},
		// expected failures....
		{
			"input lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			unix.EPERM,
		},
		{
			"input hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{0},
			[]int{0},
			unix.EPERM,
		},
		{
			"edge detection",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{},
			[]int{0},
			unix.EPERM,
		},
		{
			"zero mask",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]int{0, 1, 3, 4, 5, 7},
			[]int{},
			unix.EINVAL,
		},
	}
	s, err := gpiosim.NewSimpleton(8)
	require.Nil(t, err)
	defer s.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			f, err := os.Open(s.DevPath())
			require.Nil(t, err)
			defer f.Close()
			copy(p.lr.Consumer[:31], "test-set-line-values-")
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			lv := uapi.NewLineBits(p.active...)
			mask := uapi.NewLineBits(p.mask...)
			xlv := lv & mask
			lsv := uapi.LineValues{
				Mask: mask,
				Bits: lv,
			}
			err = uapi.SetLineValues(uintptr(p.lr.Fd), lsv)
			assert.Equal(t, p.err, err)
			if p.err == nil {
				// check values from sim
				for i := 0; i < int(p.lr.Lines); i++ {
					o := int(p.lr.Offsets[i])
					v, err := s.Level(int(o))
					assert.Nil(t, err)
					xv := xlv.Get(i)
					if p.lr.Config.Flags.IsActiveLow() {
						xv ^= 0x01
					}
					assert.Equal(t, xv, v)
				}
			}
			unix.Close(int(p.lr.Fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	f, err := os.CreateTemp("", "uapi_test")
	require.Nil(t, err)
	defer os.Remove(f.Name())
	defer f.Close()
	err = uapi.SetLineValues(f.Fd(),
		uapi.LineValues{
			Mask: 1,
			Bits: 1,
		})
	assert.NotNil(t, err)
}

func zeroed(data []byte) bool {
	for _, d := range data {
		if d != 0 {
			return false
		}
	}
	return true
}

func TestSetLineConfig(t *testing.T) {
	patterns := []struct {
		name   string
		lr     uapi.LineRequest
		ra     []AttributeEncoder
		config uapi.LineConfig
		ca     []AttributeEncoder
		err    error
	}{
		{
			"in to out",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagOutput,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"out to in",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			uapi.LineConfig{
				Flags: uapi.LineFlagInput,
			},
			nil,
			nil,
		},
		{
			"as-is atv-hi to as-is atv-lo",
			uapi.LineRequest{
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagActiveLow,
			},
			nil,
			nil,
		},
		{
			"as-is atv-lo to as-is atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{},
			nil,
			nil,
		},
		{
			"input atv-lo to input atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput,
			},
			nil,
			nil,
		},
		{
			"input atv-hi to input atv-lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagActiveLow,
			},
			nil,
			nil,
		},
		{
			"output atv-lo to output atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 0, 1))},
			uapi.LineConfig{
				Flags: uapi.LineFlagOutput,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"output atv-hi to output atv-lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 0, 1))},
			uapi.LineConfig{
				Flags: uapi.LineFlagOutput | uapi.LineFlagActiveLow,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"input atv-lo to as-is atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{},
			nil,
			nil,
		},
		{
			"input atv-hi to as-is atv-lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagActiveLow,
			},
			nil,
			nil,
		},
		{
			"input pull-up to input pull-down",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagBiasPullUp,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagBiasPullDown,
			},
			nil,
			nil,
		},
		{
			"input pull-down to input pull-up",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagBiasPullDown,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagBiasPullUp,
			},
			nil,
			nil,
		},
		{
			"output atv-lo to as-is atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput | uapi.LineFlagActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			uapi.LineConfig{},
			nil,
			nil,
		},
		{
			"output atv-hi to as-is atv-lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			uapi.LineConfig{
				Flags: uapi.LineFlagActiveLow,
			},
			nil,
			nil,
		},
		{
			"edge to biased",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling | uapi.LineFlagBiasPullUp,
			},
			nil,
			nil,
		},
		{
			"in to debounced",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput,
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			nil,
		},
		{
			"debounced to undebounced",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			uapi.LineConfig{
				Flags: uapi.LineFlagInput,
			},
			[]AttributeEncoder{uapi.DebouncePeriod(0)},
			nil,
		},
		{
			"debounce changed",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			uapi.LineConfig{
				Flags: uapi.LineFlagInput,
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			nil,
		},
		{
			"out to debounced in",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			uapi.LineConfig{
				Flags: uapi.LineFlagInput,
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			nil,
		},
		{
			"debounced in to out",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			[]AttributeEncoder{uapi.DebouncePeriod(20 * time.Microsecond)},
			uapi.LineConfig{
				Flags: uapi.LineFlagOutput,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(2))},
			nil,
		},
		{
			"edge to no edge",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput,
			},
			nil,
			nil,
		},
		{
			"edge to none",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput,
			},
			nil,
			nil,
		},
		{
			"rising edge to falling edge",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagEdgeRising,
			},
			nil,
			nil,
		},
		{
			"output to edge",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"edge to output",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagOutput,
			},
			[]AttributeEncoder{uapi.OutputValues(uapi.NewLineBits(0, 2))},
			nil,
		},
		{
			"edge to atv-lo",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling | uapi.LineFlagActiveLow,
			},
			nil,
			nil,
		},
		{
			"edge atv-lo to atv-hi",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling | uapi.LineFlagActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagEdgeFalling,
			},
			nil,
			nil,
		},
		// expected errors
		{
			"input drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagOpenDrain,
			},
			nil,
			unix.EINVAL,
		},
		{
			"input source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagOpenSource,
			},
			nil,
			unix.EINVAL,
		},
		{
			"as-is drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagOpenDrain,
			},
			nil,
			unix.EINVAL,
		},
		{
			"as-is source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
			uapi.LineConfig{
				Flags: uapi.LineFlagOpenSource,
			},
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
			if p.lr.Config.Flags.IsOutput() {
				for i := uint(0); i < uint(p.lr.Lines); i++ {
					v := p.lr.Config.Attrs[0].Attr.Value64() >> i & 1
					// read is after config, so use config active state
					if p.config.Flags.IsActiveLow() {
						v ^= 0x01 // assumes using 1 for high
					}
					err := s.SetPull(int(p.lr.Offsets[i]), int(v))
					assert.Nil(t, err)
				}
			}
			f, err := os.Open(s.DevPath())
			require.Nil(t, err)
			defer f.Close()
			copy(p.lr.Consumer[:31], p.name)
			p.lr.Config.NumAttrs = uint32(len(p.ra))
			for i, a := range p.ra {
				p.lr.Config.Attrs[i].Mask = 7 // for 3 lines
				p.lr.Config.Attrs[i].Attr = a.Encode()
			}
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			defer unix.Close(int(p.lr.Fd))
			// apply config change
			p.config.NumAttrs = uint32(len(p.ca))
			for i, a := range p.ca {
				p.config.Attrs[i].Mask = 7 // for 3 lines
				p.config.Attrs[i].Attr = a.Encode()
			}
			err = uapi.SetLineConfig(uintptr(p.lr.Fd), &p.config)
			require.Equal(t, p.err, err)

			if p.err == nil {
				// check line info
				li, err := uapi.GetLineInfo(f.Fd(), int(p.lr.Offsets[0]))
				assert.Nil(t, err)
				if p.err != nil {
					assert.False(t, li.Flags.IsUsed())
					return
				}
				xli := uapi.LineInfo{
					Offset: p.lr.Offsets[0],
					Flags:  uapi.LineFlagUsed | p.config.Flags,
				}
				if p.config.Flags&uapi.LineFlagDirectionMask == 0 {
					xli.Flags = uapi.LineFlagUsed | p.lr.Config.Flags
				}
				if xli.Flags&uapi.LineFlagDirectionMask == 0 {
					li.Flags &^= uapi.LineFlagDirectionMask
				}
				for _, a := range p.ca {
					enc := a.Encode()
					if enc.ID == uapi.LineAttributeIDDebounce && !zeroed(enc.Value[:]) {
						xli.Attrs[xli.NumAttrs] = enc
						xli.NumAttrs++
					}
				}
				copy(xli.Consumer[:31], p.name)
				assert.Equal(t, xli, li)
				// check values from sim
				if p.config.Flags.IsOutput() {
					for i := uint(0); i < uint(p.lr.Lines); i++ {
						v, err := s.Level(int(p.lr.Offsets[i]))
						assert.Nil(t, err)
						xv := int(p.config.Attrs[0].Attr.Value64()>>i) & 1
						if p.config.Flags.IsActiveLow() {
							xv ^= 0x01 // assumes using 1 for high
						}
						assert.Equal(t, xv, v, i)
					}
				}
			}
		}
		t.Run(p.name, tf)
	}
}

func TestSetLineConfigValidation(t *testing.T) {
	patterns := []struct {
		name string
		lc   uapi.LineConfig
	}{
		{
			"input drain",
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagOpenDrain,
			},
		},
		{
			"input source",
			uapi.LineConfig{
				Flags: uapi.LineFlagInput | uapi.LineFlagOpenSource,
			},
		},
		{
			"as-is drain",
			uapi.LineConfig{
				Flags: uapi.LineFlagOpenDrain,
			},
		},
		{
			"as-is source",
			uapi.LineConfig{
				Flags: uapi.LineFlagOpenSource,
			},
		},
		{
			"as-is pull-up",
			uapi.LineConfig{
				Flags: uapi.LineFlagBiasPullUp,
			},
		},
		{
			"as-is pull-down",
			uapi.LineConfig{
				Flags: uapi.LineFlagBiasPullDown,
			},
		},
		{
			"as-is bias disabled",
			uapi.LineConfig{
				Flags: uapi.LineFlagBiasDisabled,
			},
		},
		{
			"output edge",
			uapi.LineConfig{
				Flags: uapi.LineFlagOutput | uapi.LineFlagEdgeBoth,
			},
		},
		{
			"as-is edge",
			uapi.LineConfig{
				Flags: uapi.LineFlagEdgeBoth,
			},
		},
		{
			"non-zero padding",
			uapi.LineConfig{Padding: [5]uint32{1}},
		},
	}
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	lr := uapi.LineRequest{
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput,
		},
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{2},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	for _, p := range patterns {
		tf := func(t *testing.T) {
			err = uapi.SetLineConfig(uintptr(lr.Fd), &p.lc)
			assert.Equal(t, unix.EINVAL, err)
		}
		t.Run(p.name, tf)
	}
}

func TestWatchLineInfo(t *testing.T) {
	// also covers ReadLineInfoChanged

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
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput,
		},
		Offsets: [uapi.LinesMax]uint32{3},
	}
	copy(lr.Consumer[:], "testwatch")
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Nil(t, err)
	chg, err := readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")
	unix.Close(int(lr.Fd))

	// out of range
	li := uapi.LineInfo{Offset: uint32(c.Config().NumLines + 1)}
	err = uapi.WatchLineInfo(f.Fd(), &li)
	require.Equal(t, syscall.Errno(0x16), err)

	// non-zero pad
	li = uapi.LineInfo{
		Offset:  offset,
		Padding: [4]uint32{1},
	}
	err = uapi.WatchLineInfo(f.Fd(), &li)
	require.Equal(t, syscall.Errno(0x16), err)

	// set watch
	li = uapi.LineInfo{Offset: offset}
	err = uapi.WatchLineInfo(f.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfo{
		Offset: offset,
		Flags:  uapi.LineFlagInput,
	}
	copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
	assert.Equal(t, xli, li)

	// repeated watch
	err = uapi.WatchLineInfo(f.Fd(), &li)
	assert.Equal(t, unix.EBUSY, err)

	chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// request line
	lr = uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput,
		},
	}
	lr.Offsets[0] = offset
	copy(lr.Consumer[:], "testwatch")
	err = uapi.GetLine(f.Fd(), &lr)
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
	lc := uapi.LineConfig{Flags: uapi.LineFlagInput | uapi.LineFlagActiveLow}
	err = uapi.SetLineConfig(uintptr(lr.Fd), &lc)
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
	unix.Close(int(lr.Fd))
	chg, err = readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedReleased, chg.Type)
	xli = uapi.LineInfo{
		Offset: 3,
		Flags:  uapi.LineFlagInput,
	}
	copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")
}

func TestReadLineEvent(t *testing.T) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	err = s.SetPull(1, 0)
	require.Nil(t, err)
	err = s.SetPull(2, 1)
	require.Nil(t, err)

	// active low, both edges
	lr := uapi.LineRequest{
		Lines:   2,
		Offsets: [uapi.LinesMax]uint32{1, 2},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagActiveLow | uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
	}

	s.SetPull(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventFallingEdge
	xevt.Offset = 1
	assert.Equal(t, xevt, *evt)

	s.SetPull(2, 0)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	xevt.Offset = 2
	xevt.Seqno++
	assert.Equal(t, xevt, *evt)

	s.SetPull(2, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventFallingEdge
	xevt.Seqno++
	xevt.LineSeqno++
	assert.Equal(t, xevt, *evt)

	s.SetPull(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	xevt.Seqno++
	xevt.Offset = 1
	assert.Equal(t, xevt, *evt)

	unix.Close(int(lr.Fd))

	// falling edge
	lr.Config.Flags &^= uapi.LineFlagEdgeRising
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	s.SetPull(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventFallingEdge
	xevt.Seqno = 1
	xevt.LineSeqno = 1
	xevt.Offset = 1
	assert.Equal(t, xevt, *evt)

	s.SetPull(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))

	// active hi, rising edge
	lr.Lines = 1
	lr.Config.Flags &^= uapi.LineFlagActiveLow | uapi.LineFlagEdgeMask
	lr.Config.Flags |= uapi.LineFlagEdgeRising
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	s.SetPull(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	assert.Equal(t, xevt, *evt)

	s.SetPull(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	// test single line seqno paths
	s.SetPull(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	xevt.Seqno++
	xevt.LineSeqno++
	assert.Equal(t, xevt, *evt)

	unix.Close(int(lr.Fd))

	// realtime timestamp
	lr.Lines = 1
	lr.Config.Flags |= uapi.LineFlagEventClockRealtime | uapi.LineFlagEdgeBoth
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	start := time.Now()
	s.SetPull(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	end := time.Now()
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.LessOrEqual(t, uint64(start.UnixNano()), evt.Timestamp)
	assert.GreaterOrEqual(t, uint64(end.UnixNano()), evt.Timestamp)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventFallingEdge
	xevt.Seqno = 1
	xevt.LineSeqno = 1
	assert.Equal(t, xevt, *evt)

	start = time.Now()
	s.SetPull(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	end = time.Now()
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.LessOrEqual(t, uint64(start.UnixNano()), evt.Timestamp)
	assert.GreaterOrEqual(t, uint64(end.UnixNano()), evt.Timestamp)
	evt.Timestamp = 0
	xevt.ID = uapi.LineEventRisingEdge
	xevt.Seqno++
	xevt.LineSeqno++
	assert.Equal(t, xevt, *evt)

	unix.Close(int(lr.Fd))
}

func readLineEventTimeout(fd int32, t time.Duration) (*uapi.LineEvent, error) {
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
	evt, err := uapi.ReadLineEvent(uintptr(fd))
	if err != nil {
		return nil, err
	}
	return &evt, nil
}

func TestLineAttribute(t *testing.T) {
	var la uapi.LineAttribute

	la.Encode32(1, 1000000)
	assert.Equal(t, uapi.LineAttributeID(1), la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint32(1000000), la.Value32())

	la.Encode64(2, 200000000000)
	assert.Equal(t, uapi.LineAttributeID(2), la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(200000000000), la.Value64())
}

func TestLineFlag(t *testing.T) {
	var f uapi.LineFlag

	la := f.Encode()
	assert.Equal(t, uapi.LineAttributeIDFlags, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(0), la.Value64())
	la.Encode64(uapi.LineAttributeIDFlags, 42000)
	f.Decode(la)
	assert.Equal(t, uapi.LineFlag(42000), f)

	f = uapi.LineFlag(1234567)
	la = f.Encode()
	assert.Equal(t, uapi.LineAttributeIDFlags, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(1234567), la.Value64())
	f = 0
	f.Decode(la)
	assert.Equal(t, uapi.LineFlag(1234567), f)
}

func TestDebouncePeriod(t *testing.T) {
	var dp uapi.DebouncePeriod

	la := dp.Encode()
	assert.Equal(t, uapi.LineAttributeIDDebounce, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint32(0), la.Value32())
	la.Encode32(uapi.LineAttributeIDDebounce, 42000)
	dp.Decode(la)
	assert.Equal(t, uapi.DebouncePeriod(42*time.Millisecond), dp)

	dp = uapi.DebouncePeriod(1234567 * time.Nanosecond)
	la = dp.Encode()
	assert.Equal(t, uapi.LineAttributeIDDebounce, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint32(1234), la.Value32())
	dp = 0
	dp.Decode(la)
	assert.Equal(t, uapi.DebouncePeriod(1234*time.Microsecond), dp)
}

func TestOutputValues(t *testing.T) {
	var bits uapi.OutputValues

	la := bits.Encode()
	assert.Equal(t, uapi.LineAttributeIDOutputValues, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(0), la.Value64())
	la.Encode64(uapi.LineAttributeIDOutputValues, 42234)
	bits.Decode(la)
	assert.Equal(t, uapi.OutputValues(42234), bits)

	bits = uapi.OutputValues(0x123456789)
	la = bits.Encode()
	assert.Equal(t, uapi.LineAttributeIDOutputValues, la.ID)
	assert.Zero(t, la.Padding)
	assert.Equal(t, uint64(0x123456789), la.Value64())
	bits = 0
	bits.Decode(la)
	assert.Equal(t, uapi.OutputValues(0x123456789), bits)
}

func TestNewLineBits(t *testing.T) {
	patterns := []struct {
		name string
		bits []int
		mask uapi.LineBitmap
	}{
		{
			"zero",
			[]int{0},
			1,
		},
		{
			"one",
			[]int{1},
			2,
		},
		{
			"three",
			[]int{3},
			8,
		},
		{
			"seven",
			[]int{0, 1, 2},
			7,
		},
		{
			"top",
			[]int{63},
			0x8000000000000000,
		},
		{
			"ends",
			[]int{0, 63},
			0x8000000000000001,
		},
		{
			"overflow",
			[]int{64},
			0,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			mask := uapi.NewLineBits(p.bits...)
			assert.Equal(t, p.mask, mask)
		}
		t.Run(p.name, tf)
	}
}

func TestNewLineBitmap(t *testing.T) {
	patterns := []struct {
		name string
		bits []int
		mask uapi.LineBitmap
	}{
		{
			"zero",
			[]int{0},
			0,
		},
		{
			"one",
			[]int{1},
			1,
		},
		{
			"three",
			[]int{1, 1},
			3,
		},
		{
			"seven",
			[]int{1, 1, 1},
			7,
		},
		{
			"max",
			[]int{
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
			},
			0xffffffffffffffff,
		},
		{
			"overflow",
			[]int{
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
			},
			0xffffffffffffffff,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			mask := uapi.NewLineBitmap(p.bits...)
			assert.Equal(t, p.mask, mask)
		}
		t.Run(p.name, tf)
	}
}

func TestNewLineBitMask(t *testing.T) {
	patterns := []struct {
		name string
		bits int
		mask uapi.LineBitmap
	}{
		{
			"zero",
			0,
			0,
		},
		{
			"one",
			1,
			1,
		},
		{
			"two",
			2,
			3,
		},
		{
			"three",
			3,
			7,
		},
		{
			"max",
			64,
			0xffffffffffffffff,
		},
		{
			"overflow",
			65,
			0xffffffffffffffff,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			mask := uapi.NewLineBitMask(p.bits)
			assert.Equal(t, p.mask, mask)
		}
		t.Run(p.name, tf)
	}
}

func TestLineBitmap(t *testing.T) {
	lb := uapi.LineBitmap(0)

	assert.Zero(t, lb.Get(0))
	lb = lb.Set(2, 1)
	assert.Equal(t, uapi.LineBitmap(4), lb)
	assert.Equal(t, 0, lb.Get(0))
	assert.Equal(t, 0, lb.Get(1))
	assert.Equal(t, 1, lb.Get(2))

	lb = lb.Set(0, 1)
	assert.Equal(t, uapi.LineBitmap(5), lb)
	assert.Equal(t, 1, lb.Get(0))
	assert.Equal(t, 0, lb.Get(1))
	assert.Equal(t, 1, lb.Get(2))

	lb = lb.Set(0, 0)
	assert.Equal(t, uapi.LineBitmap(4), lb)
	assert.Equal(t, 0, lb.Get(0))
	assert.Equal(t, 0, lb.Get(1))
	assert.Equal(t, 1, lb.Get(2))

	lb = lb.Set(2, 0)
	assert.Zero(t, lb.Get(0))
}

func TestLineConfig(t *testing.T) {
	var lc uapi.LineConfig

	// remove when empty
	assert.Zero(t, lc.NumAttrs)
	lc.RemoveAttributeID(1)
	assert.Zero(t, lc.NumAttrs)
	lc.RemoveAttributeID(0)
	assert.Zero(t, lc.NumAttrs)

	// add
	lca := uapi.LineConfigAttribute{
		Attr: uapi.LineAttribute{
			ID: 56,
		},
		Mask: uapi.NewLineBitMask(63),
	}
	lca2 := uapi.LineConfigAttribute{
		Attr: uapi.LineAttribute{
			ID: 23,
		},
		Mask: uapi.NewLineBitMask(64),
	}

	lc.AddAttribute(lca)
	assert.Equal(t, uint32(1), lc.NumAttrs)
	lc.AddAttribute(lca2)
	assert.Equal(t, uint32(2), lc.NumAttrs)
	lc.AddAttribute(lca)
	assert.Equal(t, uint32(3), lc.NumAttrs)
	assert.Equal(t, lca, lc.Attrs[0])
	assert.Equal(t, lca2, lc.Attrs[1])
	assert.Equal(t, lca, lc.Attrs[2])

	// remove by id
	lc.RemoveAttributeID(42)
	assert.Equal(t, uint32(3), lc.NumAttrs)
	lc.RemoveAttributeID(56)
	assert.Equal(t, uint32(1), lc.NumAttrs)

	// remove by value
	lc.AddAttribute(lca)
	lc.AddAttribute(lca)
	assert.Equal(t, uint32(3), lc.NumAttrs)
	lc.RemoveAttribute(lca2)
	assert.Equal(t, uint32(2), lc.NumAttrs)
	lc.RemoveAttribute(lca)
	assert.Zero(t, lc.NumAttrs)
}

func TestDebounce(t *testing.T) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	err = s.SetPull(offset, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput |
				uapi.LineFlagEdgeBoth,
			NumAttrs: 1,
		},
	}
	lr.Offsets[0] = uint32(offset)
	lr.Config.Attrs[0].Mask = 1
	lr.Config.Attrs[0].Attr = uapi.DebouncePeriod(debouncePeriod).Encode()
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	// toggle faster than the debounce period - should be filtered
	for i := 0; i < 10; i++ {
		s.SetPull(offset, 1)
		time.Sleep(clkTick)
		checkLineValue(t, lr, 0)
		s.SetPull(offset, 0)
		time.Sleep(clkTick)
		checkLineValue(t, lr, 0)
	}
	// but this change will persist and get through...
	s.SetPull(offset, 1)

	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uint32(offset), evt.Offset)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	lastTime := evt.Timestamp

	checkLineValue(t, lr, 1)

	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	// toggle slower than the debounce period - should all get through
	for i := 0; i < 2; i++ {
		s.SetPull(offset, 0)
		time.Sleep(2 * debouncePeriod)
		checkLineValue(t, lr, 0)
		s.SetPull(offset, 1)
		time.Sleep(2 * debouncePeriod)
		checkLineValue(t, lr, 1)
	}
	s.SetPull(offset, 0)
	time.Sleep(2 * debouncePeriod)
	checkLineValue(t, lr, 0)
	for i := 0; i < 2; i++ {
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		assert.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uint32(1), evt.Offset)
		assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
		assert.GreaterOrEqual(t, evt.Timestamp-lastTime, uint64(10000000))
		lastTime = evt.Timestamp

		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		assert.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uint32(1), evt.Offset)
		assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
		assert.GreaterOrEqual(t, evt.Timestamp-lastTime, uint64(10000000))
		lastTime = evt.Timestamp
	}
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uint32(1), evt.Offset)
	assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
	assert.GreaterOrEqual(t, evt.Timestamp-lastTime, uint64(10000000))
	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))
}

func TestReleaseWakesPoll(t *testing.T) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(t, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	err = s.SetPull(offset, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput |
				uapi.LineFlagEdgeBoth,
			NumAttrs: 1,
		},
	}

	errChan := make(chan error)

	monitor := func() {
		evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
		if evt != nil {
			errChan <- errors.New("spurious event")
		} else {
			errChan <- err
		}
	}

	go monitor()
	unix.Close(int(lr.Fd))
	err = <-errChan
	assert.Equal(t, unix.EBADF, err)
}

func checkLineValue(t *testing.T, lr uapi.LineRequest, v int) {
	t.Helper()
	lv := uapi.LineValues{Mask: 1}
	err := uapi.GetLineValues(uintptr(lr.Fd), &lv)
	assert.Nil(t, err)
	assert.Equal(t, v, lv.Get(0))
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

func TestLineFlags(t *testing.T) {
	assert.True(t, uapi.LineFlag(0).IsAvailable())
	assert.False(t, uapi.LineFlag(0).IsUsed())
	assert.False(t, uapi.LineFlag(0).IsActiveLow())
	assert.False(t, uapi.LineFlag(0).IsInput())
	assert.False(t, uapi.LineFlag(0).IsOutput())
	assert.False(t, uapi.LineFlag(0).IsRisingEdge())
	assert.False(t, uapi.LineFlag(0).IsFallingEdge())
	assert.False(t, uapi.LineFlag(0).IsBothEdges())
	assert.False(t, uapi.LineFlag(0).IsOpenDrain())
	assert.False(t, uapi.LineFlag(0).IsOpenSource())
	assert.False(t, uapi.LineFlag(0).IsBiasDisabled())
	assert.False(t, uapi.LineFlag(0).IsBiasPullUp())
	assert.False(t, uapi.LineFlag(0).IsBiasPullDown())
	assert.False(t, uapi.LineFlag(0).IsRealtimeEventClock())
	assert.False(t, uapi.LineFlagUsed.IsAvailable())
	assert.True(t, uapi.LineFlagUsed.IsUsed())
	assert.True(t, uapi.LineFlagActiveLow.IsActiveLow())
	assert.True(t, uapi.LineFlagInput.IsInput())
	assert.True(t, uapi.LineFlagOutput.IsOutput())
	assert.True(t, uapi.LineFlagEdgeRising.IsRisingEdge())
	assert.True(t, uapi.LineFlagEdgeFalling.IsFallingEdge())
	assert.True(t, uapi.LineFlagEdgeBoth.IsBothEdges())
	assert.False(t, uapi.LineFlagEdgeRising.IsBothEdges())
	assert.False(t, uapi.LineFlagEdgeFalling.IsBothEdges())
	assert.True(t, uapi.LineFlagBiasDisabled.IsBiasDisabled())
	assert.True(t, uapi.LineFlagBiasPullUp.IsBiasPullUp())
	assert.True(t, uapi.LineFlagBiasPullDown.IsBiasPullDown())
	assert.True(t, uapi.LineFlagEventClockRealtime.IsRealtimeEventClock())
}
