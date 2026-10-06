// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

package uapi_test

import (
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

func TestUnwatchLineInfo(t *testing.T) {
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

	li := uapi.LineInfoV2{Offset: uint32(c.Config().NumLines + 1)}
	err = uapi.UnwatchLineInfo(f.Fd(), li.Offset)
	require.Equal(t, syscall.Errno(0x16), err)

	offset := uint32(3)
	li = uapi.LineInfoV2{Offset: offset}
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfoV2{Offset: offset, Flags: uapi.LineFlagV2Input}
	copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
	assert.Equal(t, xli, li)

	chg, err := readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	err = uapi.UnwatchLineInfo(f.Fd(), li.Offset)
	assert.Nil(t, err)

	// request line
	lr := uapi.LineRequest{
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input,
		},
		Lines: 1,
	}
	lr.Offsets[0] = offset
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Nil(t, err)
	unix.Close(int(lr.Fd))
	chg, err = readLineInfoChangedV2Timeout(f.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change")

	// repeated unwatch
	err = uapi.UnwatchLineInfo(f.Fd(), offset)
	require.Equal(t, unix.EBUSY, err)

	// repeated watch
	err = uapi.WatchLineInfoV2(f.Fd(), &li)
	require.Nil(t, err)
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
