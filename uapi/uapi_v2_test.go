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
					Offset: uint32(l),
					Config: uapi.LineConfig{
						Flags:     uapi.LineFlagV2Direction,
						Direction: uapi.LineDirectionInput,
					},
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
				assert.True(t, li.Config.Flags.IsAvailable())
				unix.Close(int(p.lr.Fd))
				return
			}
			xli := uapi.LineInfoV2{
				Offset: p.lr.Offsets[0],
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Used | uapi.LineFlagV2Direction | p.lr.Config.Flags,
					Direction:     p.lr.Config.Direction,
					Drive:         p.lr.Config.Drive,
					Bias:          p.lr.Config.Bias,
					EdgeDetection: p.lr.Config.EdgeDetection,
					Debounce:      p.lr.Config.Debounce,
				},
			}
			copy(xli.Name[:], li.Name[:]) // don't care about name
			copy(xli.Consumer[:31], p.name)
			if !p.lr.Config.Flags.HasDirection() {
				xli.Config.Direction = li.Config.Direction
			}
			if xli.Config.Direction == uapi.LineDirectionOutput {
				xli.Config.Flags |= uapi.LineFlagV2Drive
			}
			if xli.Config.EdgeDetection == uapi.LineEdgeNone {
				xli.Config.Flags &^= uapi.LineFlagV2EdgeDetection
			}
			assert.Equal(t, xli, li)
			unix.Close(int(p.lr.Fd))
		}
		t.Run(p.name, tf)
	}
}

func TestGetLineValidation(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
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
			"oorange direction",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction,
					Direction: uapi.LineDirectionOutput + 1,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"oorange drive",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
					Direction: uapi.LineDirectionOutput,
					Drive:     uapi.LineDriveOpenSource + 1,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"oorange bias",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
					Direction: uapi.LineDirectionInput,
					Bias:      uapi.LineBiasPullDown + 1,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"oorange edge",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionInput,
					EdgeDetection: uapi.LineEdgeBoth + 1,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"input drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
					Direction: uapi.LineDirectionInput,
					Drive:     uapi.LineDriveOpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"input source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
					Direction: uapi.LineDirectionInput,
					Drive:     uapi.LineDriveOpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is drain",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Drive,
					Drive: uapi.LineDriveOpenDrain,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is source",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Drive,
					Drive: uapi.LineDriveOpenSource,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"as-is pull-up",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Bias,
					Bias:  uapi.LineBiasPullUp,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"as-is pull-down",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Bias,
					Bias:  uapi.LineBiasPullDown,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is bias disabled",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagV2Bias,
					Bias:  uapi.LineBiasDisabled,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"output edge",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
					Direction:     uapi.LineDirectionOutput,
					EdgeDetection: uapi.LineEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"as-is edge",
			uapi.LineRequest{
				Config: uapi.LineConfig{
					Flags:         uapi.LineFlagV2EdgeDetection,
					EdgeDetection: uapi.LineEdgeBoth,
				},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
		},
		{
			"non-zero direction",
			uapi.LineRequest{
				Config:  uapi.LineConfig{Direction: 1},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"non-zero drive",
			uapi.LineRequest{
				Config:  uapi.LineConfig{Drive: 1},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"non-zero bias",
			uapi.LineRequest{
				Config:  uapi.LineConfig{Bias: 1},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"non-zero edge_detection",
			uapi.LineRequest{
				Config:  uapi.LineConfig{EdgeDetection: 1},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"non-zero debounce",
			uapi.LineRequest{
				Config:  uapi.LineConfig{Debounce: 1},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
		{
			"non-zero padding",
			uapi.LineRequest{
				Config:  uapi.LineConfig{Padding: [7]uint32{1}},
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
		},
	}
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			err = uapi.GetLine(f.Fd(), &p.lr)
			assert.Equal(t, unix.EINVAL, err)
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
		val  []int
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
			[]int{0},
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
			[]int{1},
		},
		{
			"as-is lo",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{2},
			},
			[]int{0},
		},
		{
			"as-is hi",
			0,
			uapi.LineRequest{
				Lines:   1,
				Offsets: [uapi.LinesMax]uint32{1},
			},
			[]int{1},
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
			[]int{0}},
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
			[]int{1},
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
			[]int{0},
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
			[]int{1},
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
			[]int{0},
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
			[]int{1},
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
			[]int{0},
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
			[]int{1},
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
			[]int{0},
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
			[]int{1},
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
			[]int{1, 0},
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
			[]int{0, 1},
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
			[]int{0, 1, 1},
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
			[]int{0, 1, 0},
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
			[]int{0, 1, 1, 1},
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
			[]int{1, 1, 0, 1},
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
			[]int{0, 1, 1, 1, 1, 1, 0, 0},
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
			[]int{1, 1, 0, 1, 1, 1, 0, 1},
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
			[]int{1, 1, 0, 1, 1, 1, 0, 0},
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
			[]int{0}},
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
			[]int{1},
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
			if p.lr.Config.Direction == uapi.LineDirectionOutput {
				// mock is ignored for outputs
				xval = make([]int, len(p.val))
			}
			lvx := uapi.NewLineValues(xval...)
			lv := uapi.LineValues{}
			err = uapi.GetLineValuesV2(uintptr(fd), &lv)
			assert.Nil(t, err)
			assert.Equal(t, lvx, lv)
			unix.Close(int(fd))
		}
		t.Run(p.name, tf)
	}
	// badfd
	lvx := uapi.NewLineValues(0, 0, 0)
	lv := uapi.NewLineValues(0, 0, 0)
	err := uapi.GetLineValuesV2(0, &lv)
	assert.NotNil(t, err)
	assert.Equal(t, lvx, lv)
}

func TestSetLineValuesV2(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name string
		cnum int
		lr   uapi.LineRequest
		val  []int
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
			[]int{0},
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
			[]int{1},
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
			[]int{0},
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
			[]int{1},
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
			[]int{0},
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
			[]int{1},
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
			[]int{1, 0},
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
			[]int{0, 1},
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
			[]int{0, 1, 1},
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
			[]int{0, 1, 0},
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
			[]int{0, 1, 1, 1},
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
			[]int{1, 1, 0, 1},
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
			[]int{0, 1, 1, 1, 1, 1, 0, 0},
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
			[]int{1, 1, 0, 1, 1, 1, 0, 1},
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
			[]int{1, 1, 0, 1, 1, 0, 0, 0},
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
			[]int{0},
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
			[]int{1},
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
			[]int{0},
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
			lv := uapi.NewLineValues(p.val...)
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
	err := uapi.SetLineValuesV2(0, uapi.NewLineValues(1))
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
				Values:    uapi.NewLineValues(1, 0, 1),
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
					Values:    uapi.NewLineValues(1, 0, 1),
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
					Values:    uapi.NewLineValues(0, 0, 1),
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction,
				Direction: uapi.LineDirectionOutput,
				Values:    uapi.NewLineValues(1, 0, 1),
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
					Values:    uapi.NewLineValues(0, 0, 1),
				},
				Lines:   3,
				Offsets: [uapi.LinesMax]uint32{1, 2, 3},
			},
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2ActiveLow,
				Direction: uapi.LineDirectionOutput,
				Values:    uapi.NewLineValues(1, 0, 1),
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
					Values:    uapi.NewLineValues(1, 0, 1),
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
					Values:    uapi.NewLineValues(1, 0, 1),
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
				Values:    uapi.NewLineValues(1, 0, 1),
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
					v := p.lr.Config.Values.Get(i)
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
					assert.True(t, li.Config.Flags.IsAvailable())
					return
				}
				xli := uapi.LineInfoV2{
					Offset: p.lr.Offsets[0],
					Config: uapi.LineConfig{
						Flags: uapi.LineFlagV2Used |
							uapi.LineFlagV2Direction |
							p.lr.Config.Flags |
							p.config.Flags,
						Direction:     p.config.Direction,
						Drive:         p.config.Drive,
						Bias:          p.config.Bias,
						EdgeDetection: p.config.EdgeDetection,
						Debounce:      p.config.Debounce,
					},
				}
				if !p.config.Flags.IsActiveLow() {
					xli.Config.Flags &^= uapi.LineFlagV2ActiveLow
				}
				if !p.config.Flags.HasDirection() {
					xli.Config.Direction = li.Config.Direction
				}
				if xli.Config.Direction == uapi.LineDirectionOutput {
					xli.Config.Flags |= uapi.LineFlagV2Drive
				}
				copy(xli.Name[:], li.Name[:]) // don't care about name
				copy(xli.Consumer[:31], p.name)
				assert.Equal(t, xli, li)
				// check values from mock
				if p.config.Flags.HasDirection() && p.config.Direction == uapi.LineDirectionOutput {
					for i, o := range p.lr.Offsets {
						v, err := c.Value(int(o))
						assert.Nil(t, err)
						xv := p.config.Values.Get(i)
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

func TestSetLineV2Validation(t *testing.T) {
	requireKernel(t, uapiV2Kernel)
	requireMockup(t)
	patterns := []struct {
		name string
		lc   uapi.LineConfig
	}{
		{
			"oorange direction",
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction,
				Direction: uapi.LineDirectionOutput + 1,
			},
		},
		{
			"oorange drive",
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
				Direction: uapi.LineDirectionOutput,
				Drive:     uapi.LineDriveOpenSource + 1,
			},
		},
		{
			"oorange bias",
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Bias,
				Direction: uapi.LineDirectionInput,
				Bias:      uapi.LineBiasPullDown + 1,
			},
		},
		{
			"oorange edge",
			uapi.LineConfig{
				Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
				Direction:     uapi.LineDirectionInput,
				EdgeDetection: uapi.LineEdgeBoth + 1,
			},
		},
		{
			"input drain",
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
				Direction: uapi.LineDirectionInput,
				Drive:     uapi.LineDriveOpenDrain,
			},
		},
		{
			"input source",
			uapi.LineConfig{
				Flags:     uapi.LineFlagV2Direction | uapi.LineFlagV2Drive,
				Direction: uapi.LineDirectionInput,
				Drive:     uapi.LineDriveOpenSource,
			},
		},
		{
			"as-is drain",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Drive,
				Drive: uapi.LineDriveOpenDrain,
			},
		},
		{
			"as-is source",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Drive,
				Drive: uapi.LineDriveOpenSource,
			},
		},
		{
			"as-is pull-up",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Bias,
				Bias:  uapi.LineBiasPullUp,
			},
		},
		{
			"as-is pull-down",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Bias,
				Bias:  uapi.LineBiasPullDown,
			},
		},
		{
			"as-is bias disabled",
			uapi.LineConfig{
				Flags: uapi.LineFlagV2Bias,
				Bias:  uapi.LineBiasDisabled,
			},
		},
		{
			"output edge",
			uapi.LineConfig{
				Flags:         uapi.LineFlagV2Direction | uapi.LineFlagV2EdgeDetection,
				Direction:     uapi.LineDirectionOutput,
				EdgeDetection: uapi.LineEdgeBoth,
			},
		},
		{
			"as-is edge",
			uapi.LineConfig{
				Flags:         uapi.LineFlagV2EdgeDetection,
				EdgeDetection: uapi.LineEdgeBoth,
			},
		},
		{
			"non-zero direction",
			uapi.LineConfig{Direction: 1},
		},
		{
			"non-zero drive",
			uapi.LineConfig{Drive: 1},
		},
		{
			"non-zero bias",
			uapi.LineConfig{Bias: 1},
		},
		{
			"non-zero edge_detection",
			uapi.LineConfig{EdgeDetection: 1},
		},
		{
			"non-zero debounce",
			uapi.LineConfig{Debounce: 1},
		},
		{
			"non-zero padding",
			uapi.LineConfig{Padding: [7]uint32{1}},
		},
	}
	c, err := mock.Chip(0)
	require.Nil(t, err)
	f, err := os.Open(c.DevPath)
	require.Nil(t, err)
	defer f.Close()
	lr := uapi.LineRequest{
		Config: uapi.LineConfig{
			Flags:     uapi.LineFlagV2Direction,
			Direction: uapi.LineDirectionInput,
		},
		Lines:   1,
		Offsets: [uapi.LinesMax]uint32{2},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	for _, p := range patterns {
		tf := func(t *testing.T) {
			err = uapi.SetLineConfigV2(uintptr(lr.Fd), &p.lc)
			assert.Equal(t, unix.EINVAL, err)
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

	// out of range
	li := uapi.LineInfoV2{Offset: uint32(c.Lines + 1)}
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Equal(t, syscall.Errno(0x16), err)

	// non-zero pad
	li = uapi.LineInfoV2{
		Offset:  3,
		Padding: [5]uint32{1},
	}
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Equal(t, syscall.Errno(0x16), err)

	// set watch
	li = uapi.LineInfoV2{Offset: 3}
	lname := c.Label + "-3"
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfoV2{
		Offset: 3,
		Config: uapi.LineConfig{
			Flags:     uapi.LineFlagV2Direction,
			Direction: uapi.LineDirectionInput,
		},
	}
	copy(xli.Name[:], lname)
	assert.Equal(t, xli, li)

	// repeated watch
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	assert.Equal(t, unix.EBUSY, err)

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
	xli.Config.Flags |= uapi.LineFlagV2Used
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
	xli.Config.Flags |= uapi.LineFlagV2ActiveLow
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
		Offset: 3,
		Config: uapi.LineConfig{
			Flags:     uapi.LineFlagV2Direction,
			Direction: uapi.LineDirectionInput,
		},
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

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
	assert.Equal(t, uint32(1), evt.Offset)

	c.SetValue(2, 0)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	assert.Equal(t, uint32(2), evt.Offset)

	c.SetValue(2, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
	assert.Equal(t, uint32(2), evt.Offset)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	assert.Equal(t, uint32(1), evt.Offset)

	unix.Close(int(lr.Fd))

	lr.Config.EdgeDetection = uapi.LineEdgeFalling
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
	assert.Equal(t, uint32(1), evt.Offset)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))

	lr.Config.Flags &^= uapi.LineFlagV2ActiveLow
	lr.Config.EdgeDetection = uapi.LineEdgeRising
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	c.SetValue(1, 1)
	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	require.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	assert.Equal(t, uint32(1), evt.Offset)

	c.SetValue(1, 0)
	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))
}

func readLineEventTimeout(fd int32, t time.Duration) (*uapi.LineEvent, error) {
	pollfd := unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
	n, err := unix.Poll([]unix.PollFd{pollfd}, int(t.Milliseconds()))
	if err != nil || n != 1 {
		return nil, err
	}
	evt, err := uapi.ReadLineEvent(uintptr(fd))
	if err != nil {
		return nil, err
	}
	return &evt, nil
}

func TestDebounce(t *testing.T) {
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
			Flags: uapi.LineFlagV2Direction |
				uapi.LineFlagV2EdgeDetection |
				uapi.LineFlagV2Debounce,
			Direction:     uapi.LineDirectionInput,
			EdgeDetection: uapi.LineEdgeBoth,
			Debounce:      10000, // 10msec
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	// toggle faster than the debounce period - should be filtered
	for i := 0; i < 10; i++ {
		c.SetValue(1, 1)
		time.Sleep(time.Millisecond)
		c.SetValue(1, 0)
		time.Sleep(time.Millisecond)
	}
	// but this change will persist and get through...
	c.SetValue(1, 1)

	evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, evt)
	assert.Equal(t, uint32(1), evt.Offset)
	assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
	lastTime := evt.Timestamp

	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	// toggle slower than the debounce period - should all get through
	for i := 0; i < 2; i++ {
		c.SetValue(1, 0)
		time.Sleep(20 * time.Millisecond)
		c.SetValue(1, 1)
		time.Sleep(20 * time.Millisecond)
	}
	c.SetValue(1, 0)
	time.Sleep(20 * time.Millisecond)

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
	assert.False(t, uapi.LineFlagV2(0).IsUsed())
	assert.False(t, uapi.LineFlagV2(0).IsActiveLow())
	assert.False(t, uapi.LineFlagV2(0).HasDirection())
	assert.False(t, uapi.LineFlagV2(0).HasDrive())
	assert.False(t, uapi.LineFlagV2(0).HasBias())
	assert.False(t, uapi.LineFlagV2(0).HasEdgeDetection())
	assert.False(t, uapi.LineFlagV2(0).HasDebounce())
	assert.False(t, uapi.LineFlagV2Used.IsAvailable())
	assert.True(t, uapi.LineFlagV2Used.IsUsed())
	assert.True(t, uapi.LineFlagV2ActiveLow.IsActiveLow())
	assert.True(t, uapi.LineFlagV2Direction.HasDirection())
	assert.True(t, uapi.LineFlagV2Drive.HasDrive())
	assert.True(t, uapi.LineFlagV2Bias.HasBias())
	assert.True(t, uapi.LineFlagV2EdgeDetection.HasEdgeDetection())
	assert.True(t, uapi.LineFlagV2Debounce.HasDebounce())
}
