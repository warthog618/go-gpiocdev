// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

// +build linux

package uapi_test

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod/mockup"
	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

var uapiV2Kernel = mockup.Semver{5, 7} // uapi v2 added

func TestGetLineInfoV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	reloadMockup() // test assumes clean mockups
	requireMockup(t)
	for n := 0; n < mock.Chips(); n++ {
		c, err := mock.Chip(n)
		require.Nil(t, err)
		for l := 0; l < c.Lines; l++ {
			f := func(t *testing.T) {
				f, err := os.Open(c.DevPath)
				require.Nil(t, err)
				defer f.Close()
				xli := uapi.LineInfoV2{
					Offset:    uint32(l),
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				}
				copy(xli.Name[:], fmt.Sprintf("%s-%d", c.Label, l))
				copy(xli.Consumer[:], "")
				li, err := uapi.GetLineInfoV2(f.Fd(), l)
				assert.Nil(t, err)
				assert.Equal(t, xli, li)
			}
			t.Run(fmt.Sprintf("%s-%d", c.Name, l), f)
		}
	}
	// badfd
	li, err := uapi.GetLineInfoV2(0, 1)
	lix := uapi.LineInfoV2{}
	assert.NotNil(t, err)
	assert.Equal(t, lix, li)
}

func TestGetLine(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	reloadMockup()
	requireMockup(t)
	patterns := []struct {
		name string // unique name for pattern (hf/ef/offsets/xval combo)
		cnum int
		lr   uapi.LineRequest
		err  error
	}{
		{
			"as-is",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"atv-lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input pull-up",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionInput,
					Bias:      uapi.LineBiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"input pull-down",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionInput,
					Bias:      uapi.LineBiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input bias disable",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionInput,
					Bias:      uapi.LineBiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input edge rising",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input edge falling",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeRising,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{1, 3},
			},
			nil,
		},
		{
			"input edge both",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"input edge nonr",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeNone,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{3},
			},
			nil,
		},
		{
			"output",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output drain",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
					Direction: uapi.LineDirectionOutput,
					Drive:     uapi.LineDriveOpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output source",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
					Direction: uapi.LineDirectionOutput,
					Drive:     uapi.LineDriveOpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output pull-up",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionOutput,
					Bias:      uapi.LineBiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			nil,
		},
		{
			"output pull-down",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionOutput,
					Bias:      uapi.LineBiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			nil,
		},
		{
			"output bias disabled",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionOutput,
					Bias:      uapi.LineBiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			nil,
		},
		// expected errors
		{
			"overlength",
			0,
			uapi.LineRequest{
				Lines:   5,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4},
			},
			unix.EINVAL,
		},
		{
			"oorange offset",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{6},
			},
			unix.EINVAL,
		},
		{
			"oorange direction",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput + 1,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"oorange drive",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
					Direction: uapi.LineDirectionOutput,
					Drive:     uapi.LineDriveOpenSource + 1,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"oorange bias",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionInput,
					Bias:      uapi.LineBiasPullDown + 1,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"oorange edge",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeBoth + 1,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"input drain",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
					Direction: uapi.LineDirectionInput,
					Drive:     uapi.LineDriveOpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			unix.EINVAL,
		},
		{
			"input source",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
					Direction: uapi.LineDirectionInput,
					Drive:     uapi.LineDriveOpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"as-is drain",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Drive,
					Drive: uapi.LineDriveOpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"as-is source",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Drive,
					Drive: uapi.LineDriveOpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			unix.EINVAL,
		},
		{
			"as-is pull-up",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Bias,
					Bias:  uapi.LineBiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			unix.EINVAL,
		},
		{
			"as-is pull-down",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Bias,
					Bias:  uapi.LineBiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"as-is bias disabled",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Bias,
					Bias:  uapi.LineBiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"output edge",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionOutput,
					EdgeDetection: uapi.LineEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
		{
			"as-is edge",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2EdgeDetection,
					EdgeDetection: uapi.LineEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			unix.EINVAL,
		},
	}
	for _, p := range patterns {
		c, err := mock.Chip(p.cnum)
		require.Nil(t, err)
		tf := func(t *testing.T) {
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			copy(p.lr.Consumer[:], p.name)
			err = uapi.GetLine(f.Fd(), &p.lr)
			assert.Equal(t, p.err, err)
			if p.lr.Offsets[0] > uint32(c.Lines) {
				return
			}
			// check line info
			li, err := uapi.GetLineInfoV2(f.Fd(), int(p.lr.Offsets[0]))
			assert.Nil(t, err)
			if p.err != nil {
				assert.True(t, li.Flags.IsAvailable())
				unix.Close(int(p.lr.Fd))
				return
			}
			xli := uapi.LineInfoV2{
				Offset:        p.lr.Offsets[0],
				Flags:         uapi.LineFlagV2Unavailable | uapi.LineFlagV2Direction | p.lr.Config.Flags,
				Direction:     p.lr.Config.Direction,
				Drive:         p.lr.Config.Drive,
				Bias:          p.lr.Config.Bias,
				EdgeDetection: p.lr.Config.EdgeDetection,
				Debounce:      p.lr.Config.Debounce,
			}
			copy(xli.Name[:], li.Name[:]) // don't care about name
			copy(xli.Consumer[:31], p.name)
			if !p.lr.Config.Flags.HasDirection() {
				xli.Direction = li.Direction
			}
			if xli.Direction == uapi.LineDirectionOutput {
				xli.Flags |= uapi.LineFlagV2Drive
			}
			if xli.EdgeDetection == uapi.LineEdgeNone {
				xli.Flags &^= uapi.LineFlagV2EdgeDetection
			}
			assert.Equal(t, xli, li)
			unix.Close(int(p.lr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineValuesV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name string
		cnum int
		lr   uapi.LineRequest
		val  []uint8
	}{
		{
			"as-is atv-lo lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
		},
		{
			"as-is atv-lo hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{1},
		},
		{
			"as-is lo",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
		},
		{
			"as-is hi",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
		},
		{
			"input lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0}},
		{
			"input hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
		},
		{
			"output lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
		},
		{
			"output hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
		},
		{
			"both lo",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
		},
		{
			"both hi",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
		},
		{
			"falling lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
		},
		{
			"falling hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
		},
		{
			"rising lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
		},
		{
			"rising hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
		},
		{
			"input 2a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{0, 1},
			},
			[]uint8{1, 0},
		},
		{
			"input 2b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{2, 1},
			},
			[]uint8{0, 1},
		},
		{
			"input 3a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2},
			},
			[]uint8{0, 1, 1},
		},
		{
			"input 3b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 2, 1},
			},
			[]uint8{0, 1, 0},
		},
		{
			"input 4a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
			},
			[]uint8{0, 1, 1, 1},
		},
		{
			"input 4b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0},
			},
			[]uint8{1, 1, 0, 1},
		},
		{
			"input 8a",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			},
			[]uint8{0, 1, 1, 1, 1, 1, 0, 0},
		},
		{
			"input 8b",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]uint8{1, 1, 0, 1, 1, 1, 0, 1},
		},
		{
			"atv-lo 8b",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]uint8{1, 1, 0, 1, 1, 1, 0, 0},
		},
		{
			"edge detection lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0}},
		{
			"edge detection hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
		},
	}
	for _, p := range patterns {
		c, err := mock.Chip(p.cnum)
		require.Nil(t, err)
		// set vals in mock
		require.Equal(t, int(p.lr.Lines), len(p.val))
		for i := 0; i < int(p.lr.Lines); i++ {
			v := int(p.val[i])
			o := int(p.lr.Offsets[i])
			if p.lr.Config.Flags.IsActiveLow() {
				v ^= 0x01 // assumes using 1 for high
			}
			err := c.SetValue(o, v)
			assert.Nil(t, err)
		}
		tf := func(t *testing.T) {
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			var fd int32
			xval := p.val
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			fd = p.lr.Fd
			if p.lr.Config.Flags.HasDirection() && p.lr.Config.Direction == uapi.LineDirectionOutput {
				// mock is ignored for outputs
				xval = make([]uint8, len(p.val))
			}
			var lvx uapi.LineValues
			copy(lvx[:], xval)
			var lv uapi.LineValues
			err = uapi.GetLineValuesV2(uintptr(fd), &lv)
			assert.Nil(t, err)
			assert.Equal(t, lvx, lv)
			unix.Close(int(fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	var hdx uapi.LineValues
	var hd uapi.LineValues
	err := uapi.GetLineValuesV2(0, &hd)
	assert.NotNil(t, err)
	assert.Equal(t, hdx, hd)
}

func TestSetLineValuesV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name string
		cnum int
		lr   uapi.LineRequest
		val  []uint8
		err  error
	}{
		{
			"output atv-lo lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2ActiveLow | uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
			nil,
		},
		{
			"output atv-lo hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2ActiveLow | uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{1},
			nil,
		},
		{
			"as-is lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
			nil,
		},
		{
			"as-is hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{1},
			nil,
		},
		{
			"output lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
			nil,
		},
		{
			"output hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
			nil,
		},
		{
			"output 2a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{0, 1},
			},
			[]uint8{1, 0},
			nil,
		},
		{
			"output 2b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   2,
				Offsets: [uapi.LinesMax]uint32{2, 1},
			},
			[]uint8{0, 1},
			nil,
		},
		{
			"output 3a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2},
			},
			[]uint8{0, 1, 1},
			nil,
		},
		{
			"output 3b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{0, 2, 1},
			},
			[]uint8{0, 1, 0},
			nil,
		},
		{
			"output 4a",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
			},
			[]uint8{0, 1, 1, 1},
			nil,
		},
		{
			"output 4b",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   4,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0},
			},
			[]uint8{1, 1, 0, 1},
			nil,
		},
		{
			"output 8a",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3, 4, 5, 6, 7},
			},
			[]uint8{0, 1, 1, 1, 1, 1, 0, 0},
			nil,
		},
		{
			"output 8b",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]uint8{1, 1, 0, 1, 1, 1, 0, 1},
			nil,
		},
		{
			"atv-lo 8b",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   8,
				Offsets: [uapi.LinesMax]uint32{3, 2, 1, 0, 4, 5, 6, 7},
			},
			[]uint8{1, 1, 0, 1, 1, 0, 0, 0},
			nil,
		},
		// expected failures....
		{
			"input lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
			unix.EPERM,
		},
		{
			"input hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]uint8{1},
			unix.EPERM,
		},
		{
			"edge detection",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeRising,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]uint8{0},
			unix.EPERM,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			c, err := mock.Chip(p.cnum)
			require.Nil(t, err)
			require.Equal(t, int(p.lr.Lines), len(p.val))
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			var lv uapi.LineValues
			copy(lv[:], p.val)
			err = uapi.SetLineValuesV2(uintptr(p.lr.Fd), lv)
			assert.Equal(t, p.err, err)
			if p.err == nil {
				// check values from mock
				for i := 0; i < int(p.lr.Lines); i++ {
					o := int(p.lr.Offsets[i])
					v, err := c.Value(int(o))
					assert.Nil(t, err)
					xv := int(p.val[i])
					if p.lr.Config.Flags.IsActiveLow() {
						xv ^= 0x01 // assumes using 1 for high
					}
					assert.Equal(t, xv, v)
				}
			}
			unix.Close(int(p.lr.Fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	var lv uapi.LineValues
	err := uapi.SetLineValuesV2(0, lv)
	assert.NotNil(t, err)
}
func TestSetLineConfigV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name   string
		cnum   int
		lr     uapi.LineRequest
		config uapi.LineConfig
		err    error
	}{
		{
			"in to out",
			1,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction,
				Direction: uapi.LineDirectionOutput,
				Values:    [uapi.LinesMax]uint8{1, 0, 1},
			},
			nil,
		},
		{
			"out to in",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
					Values:    [uapi.LinesMax]uint8{1, 0, 1},
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction,
				Direction: uapi.LineDirectionInput,
			},
			nil,
		},
		{
			"as-is atv-hi to as-is atv-lo",
			0,
			uapi.LineRequest{
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2ActiveLow,
			},
			nil,
		},
		{
			"as-is atv-lo to as-is atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2ActiveLow,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{},
			nil,
		},
		{
			"input atv-lo to input atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2ActiveLow,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction,
				Direction: uapi.LineDirectionInput,
			},
			nil,
		},
		{
			"input atv-hi to input atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2ActiveLow,
				Direction: uapi.LineDirectionInput,
			},
			nil,
		},
		{
			"output atv-lo to output atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2ActiveLow,
					Direction: uapi.LineDirectionOutput,
					Values:    [uapi.LinesMax]uint8{0, 0, 1},
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction,
				Direction: uapi.LineDirectionOutput,
				Values:    [uapi.LinesMax]uint8{1, 0, 1},
			},
			nil,
		},
		{
			"output atv-hi to output atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
					Values:    [uapi.LinesMax]uint8{0, 0, 1},
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2ActiveLow,
				Direction: uapi.LineDirectionOutput,
				Values:    [uapi.LinesMax]uint8{1, 0, 1},
			},
			nil,
		},
		{
			"input atv-lo to as-is atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2ActiveLow,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{},
			nil,
		},
		{
			"input atv-hi to as-is atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2ActiveLow,
			},
			nil,
		},
		{
			"input pull-up to input pull-down",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionInput,
					Bias:      uapi.LineBiasPullUp,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
				Direction: uapi.LineDirectionInput,
				Bias:      uapi.LineBiasPullDown,
			},
			nil,
		},
		{
			"input pull-down to input pull-up",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionInput,
					Bias:      uapi.LineBiasPullDown,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
				Direction: uapi.LineDirectionInput,
				Bias:      uapi.LineBiasPullUp,
			},
			nil,
		},
		{
			"output atv-lo to as-is atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2ActiveLow,
					Direction: uapi.LineDirectionOutput,
					Values:    [uapi.LinesMax]uint8{1, 0, 1},
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{},
			nil,
		},
		{
			"output atv-hi to as-is atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput,
					Values:    [uapi.LinesMax]uint8{1, 0, 1},
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2ActiveLow,
			},
			nil,
		},
		{
			"edge to biased",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection | uapi.LineFlagV2Bias,
				Direction:     uapi.LineDirectionInput,
				EdgeDetection: uapi.LineEdgeFalling,
				Bias:          uapi.LineBiasPullUp,
			},
			nil,
		},
		// expected errors
		{
			"input drain",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
				Direction: uapi.LineDirectionInput,
				Drive:     uapi.LineDriveOpenDrain,
			},
			unix.EINVAL,
		},
		{
			"input source",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
				Direction: uapi.LineDirectionInput,
				Drive:     uapi.LineDriveOpenSource,
			},
			unix.EINVAL,
		},
		{
			"as-is drain",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Drive,
				Drive: uapi.LineDriveOpenDrain,
			},
			unix.EINVAL,
		},
		{
			"as-is source",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionInput,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Drive,
				Drive: uapi.LineDriveOpenSource,
			},
			unix.EINVAL,
		},
		{
			"edge to no edge",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction,
				Direction: uapi.LineDirectionInput,
			},
			unix.EINVAL,
		},
		{
			"edge to none",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
				Direction:     uapi.LineDirectionInput,
				EdgeDetection: uapi.LineEdgeNone,
			},
			unix.EINVAL,
		},
		{
			"rising edge to falling edge",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
				Direction:     uapi.LineDirectionInput,
				EdgeDetection: uapi.LineEdgeRising,
			},
			unix.EINVAL,
		},
		{
			"edge to output",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction,
				Direction: uapi.LineDirectionOutput,
				Values:    [uapi.LinesMax]uint8{1, 0, 1},
			},
			unix.EINVAL,
		},
		{
			"edge to atv-lo",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection | uapi.LineFlagV2ActiveLow,
				Direction:     uapi.LineDirectionInput,
				EdgeDetection: uapi.LineEdgeFalling,
			},
			unix.EINVAL,
		},
		{
			"edge atv-lo to atv-hi",
			0,
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection | uapi.LineFlagV2ActiveLow,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeFalling,
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
				Direction:     uapi.LineDirectionInput,
				EdgeDetection: uapi.LineEdgeFalling,
			},
			unix.EINVAL,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			c, err := mock.Chip(p.cnum)
			require.Nil(t, err)
			// setup mockup for inputs
			if p.config.Flags.HasDirection() && p.config.Direction == uapi.LineDirectionOutput {
				for i, o := range p.lr.Offsets {
					v := int(p.lr.Config.Values[i])
					// read is after config, so use config active state
					if p.config.Flags.IsActiveLow() {
						v ^= 0x01 // assumes using 1 for high
					}
					err := c.SetValue(int(o), v)
					assert.Nil(t, err)
				}
			}
			f, err := os.Open(c.DevPath)
			require.Nil(t, err)
			defer f.Close()
			copy(p.lr.Consumer[:31], p.name)
			err = uapi.GetLine(f.Fd(), &p.lr)
			require.Nil(t, err)
			// apply config change
			err = uapi.SetLineConfigV2(uintptr(p.lr.Fd), &p.config)
			assert.Equal(t, p.err, err)

			if p.err == nil {
				// check line info
				li, err := uapi.GetLineInfoV2(f.Fd(), int(p.lr.Offsets[0]))
				assert.Nil(t, err)
				if p.err != nil {
					assert.True(t, li.Flags.IsAvailable())
					return
				}
				xli := uapi.LineInfoV2{
					Offset: p.lr.Offsets[0],
					Flags: uapi.LineFlagV2Unavailable |
						uapi.LineFlagV2Direction |
						p.lr.Config.Flags |
						p.config.Flags,
					Direction:     p.config.Direction,
					Drive:         p.config.Drive,
					Bias:          p.config.Bias,
					EdgeDetection: p.config.EdgeDetection,
					Debounce:      p.config.Debounce,
				}
				if !p.config.Flags.IsActiveLow() {
					xli.Flags &^= uapi.LineFlagV2ActiveLow
				}
				if !p.config.Flags.HasDirection() {
					xli.Direction = li.Direction
				}
				if xli.Direction == uapi.LineDirectionOutput {
					xli.Flags |= uapi.LineFlagV2Drive
				}
				copy(xli.Name[:], li.Name[:]) // don't care about name
				copy(xli.Consumer[:31], p.name)
				assert.Equal(t, xli, li)
				// check values from mock
				if p.config.Flags.HasDirection() && p.config.Direction == uapi.LineDirectionOutput {
					for i, o := range p.lr.Offsets {
						v, err := c.Value(int(o))
						assert.Nil(t, err)
						xv := int(p.config.Values[i])
						if p.config.Flags.IsActiveLow() {
							xv ^= 0x01 // assumes using 1 for high
						}
						assert.Equal(t, xv, v, i)
					}
				}
			}
			unix.Close(int(p.lr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestWatchLineInfoV2(t *testing.T) {
	// also covers ReadLineInfoChangedV2

	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)

	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()

	// unwatched
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags:     uapi.LineFlagV2Direction,
			Direction: uapi.LineDirectionInput,
		},
		Offsets: [uapi.LinesMax]uint32{3},
	}
	copy(lr.Consumer[:], "testwatch")
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Nil(t, err)
	chg, err := readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")
	unix.Close(int(lr.Fd))

	// set watch
	li := uapi.LineInfoV2{Offset: uint32(c.Lines + 1)}
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Equal(t, syscall.Errno(0x16), err)

	li = uapi.LineInfoV2{Offset: 3}
	lname := c.Label + "-3"
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfoV2{
		Offset:    3,
		Flags:     uapi.LineFlagV2Direction,
		Direction: uapi.LineDirectionInput,
	}
	copy(xli.Name[:], lname)
	assert.Equal(t, xli, li)

	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// request line
	lr = uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags:     uapi.LineFlagV2Direction,
			Direction: uapi.LineDirectionInput,
		},
		Offsets: [uapi.LinesMax]uint32{3},
	}
	copy(lr.Consumer[:], "testwatch")
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedV2Timeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedRequested, chg.Type)
	xli.Flags |= uapi.LineFlagV2Unavailable
	copy(xli.Consumer[:], "testwatch")
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// reconfig line
	lc := uapi.LineConfig{Flags: uapi.LineFlagV2ActiveLow}
	err = uapi.SetLineConfigV2(uintptr(lr.Fd), &lc)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedV2Timeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedConfig, chg.Type)
	xli.Flags |= uapi.LineFlagV2ActiveLow
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// release line
	unix.Close(int(lr.Fd))
	chg, err = readLineInfoChangedV2Timeout(f.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedReleased, chg.Type)
	xli = uapi.LineInfoV2{
		Offset:    3,
		Flags:     uapi.LineFlagV2Direction,
		Direction: uapi.LineDirectionInput,
	}
	copy(xli.Name[:], lname)
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")
}

func TestReadLineEvent(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	err = c.SetValue(1, 0)
	require.Nil(t, err)
	err = c.SetValue(2, 1)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines:   2,
		Offsets: [uapi.LinesMax]uint32{1, 2},
		Config: uapi.LineConfig{
			Flags:         uapi.LineFlagV2ActiveLow | uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
			Direction:     uapi.LineDirectionInput,
			EdgeDetection: uapi.LineEdgeBoth,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(uintptr(lr.Fd), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(uintptr(lr.Fd), eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
	assert.Equal(t, uint32(1), evt.Offset)

	c.SetValue(2, 0)
	evt, err = readLineEventTimeout(uintptr(lr.Fd), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	assert.Equal(t, uint32(2), evt.Offset)

	c.SetValue(2, 1)
	evt, err = readLineEventTimeout(uintptr(lr.Fd), eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
	assert.Equal(t, uint32(2), evt.Offset)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(uintptr(lr.Fd), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	assert.Equal(t, uint32(1), evt.Offset)

	unix.Close(int(lr.Fd))

	lr.Config.EdgeDetection = uapi.LineEdgeFalling
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(uintptr(lr.Fd), eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
	assert.Equal(t, uint32(1), evt.Offset)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(uintptr(lr.Fd), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))

	lr.Config.Flags &^= uapi.LineFlagV2ActiveLow
	lr.Config.EdgeDetection = uapi.LineEdgeRising
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(uintptr(lr.Fd), eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	assert.Equal(t, uint32(1), evt.Offset)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(uintptr(lr.Fd), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))
}

func readLineEventTimeout(fd uintptr, t time.Duration) (*uapi.LineEvent, error) {
	pollfd := unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
	n, err := unix.Poll([]unix.PollFd{pollfd}, int(t.Milliseconds()))
	if err != nil || n != 1 {
		return nil, err
	}
	evt, err := uapi.ReadLineEvent(fd)
	if err != nil {
		return nil, err
	}
	return &evt, nil
}

func readLineInfoChangedV2Timeout(fd uintptr,
	t time.Duration) (*uapi.LineInfoChangedV2, error) {

	pollfd := unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
	n, err := unix.Poll([]unix.PollFd{pollfd}, int(t.Milliseconds()))
	if err != nil || n != 1 {
		return nil, err
	}
	infoChanged, err := uapi.ReadLineInfoChangedV2(fd)
	if err != nil {
		return nil, err
	}
	return &infoChanged, nil
}

func TestLineFlagsV2(t *testing.T) {
	assert.True(t, uapi.LineFlagV2(0).IsAvailable())
	assert.False(t, uapi.LineFlagV2(0).IsActiveLow())
	assert.False(t, uapi.LineFlagV2(0).HasDirection())
	assert.False(t, uapi.LineFlagV2(0).HasDrive())
	assert.False(t, uapi.LineFlagV2(0).HasBias())
	assert.False(t, uapi.LineFlagV2(0).HasEdgeDetection())
	assert.False(t, uapi.LineFlagV2(0).HasDebounce())
	assert.False(t, uapi.LineFlagV2Unavailable.IsAvailable())
	assert.True(t, uapi.LineFlagV2ActiveLow.IsActiveLow())
	assert.True(t, uapi.LineFlagV2Direction.HasDirection())
	assert.True(t, uapi.LineFlagV2Drive.HasDrive())
	assert.True(t, uapi.LineFlagV2Bias.HasBias())
	assert.True(t, uapi.LineFlagV2EdgeDetection.HasEdgeDetection())
	assert.True(t, uapi.LineFlagV2Debounce.HasDebounce())
}
