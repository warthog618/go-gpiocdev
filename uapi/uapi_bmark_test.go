// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

package uapi_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiosim"
	"golang.org/x/sys/unix"

	"github.com/warthog618/go-gpiocdev/uapi"
)

func BenchmarkChipOpenClose(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	for i := 0; i < b.N; i++ {
		f, _ := os.Open(s.DevPath())
		f.Close()
	}
}

func BenchmarkLineInfo(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	for i := 0; i < b.N; i++ {
		uapi.GetLineInfo(f.Fd(), 0)
	}
}

func BenchmarkGetLine(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{
		Lines: 1,
	}
	for i := 0; i < b.N; i++ {
		uapi.GetLine(f.Fd(), &lr)
		unix.Close(int(lr.Fd))
	}
}

func BenchmarkGetLineWithEdges(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
		},
	}
	for i := 0; i < b.N; i++ {
		uapi.GetLine(f.Fd(), &lr)
		unix.Close(int(lr.Fd))
	}
}

func BenchmarkGetLineValues(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{Lines: 1}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(lr.Fd))
	lv := uapi.LineValues{Mask: 1}
	for i := 0; i < b.N; i++ {
		uapi.GetLineValues(uintptr(lr.Fd), &lv)
	}
}

func BenchmarkSetLineValues(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagOutput,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(lr.Fd))
	lv := uapi.LineValues{Mask: 1}
	for i := 0; i < b.N; i++ {
		uapi.SetLineValues(uintptr(lr.Fd), lv)
	}
}

func BenchmarkSetLineValuesSparse(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{
		Lines:   4,
		Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagOutput,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(lr.Fd))
	lv := uapi.LineValues{Mask: 0x0a}
	for i := 0; i < b.N; i++ {
		uapi.SetLineValues(uintptr(lr.Fd), lv)
	}
}

func BenchmarkSetLineConfig(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{Lines: 1}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(lr.Fd))
	var lc uapi.LineConfig
	for i := 0; i < b.N; i++ {
		uapi.SetLineConfig(uintptr(lr.Fd), &lc)
	}
}

func BenchmarkWatchLineInfo(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	var li uapi.LineInfo
	for i := 0; i < b.N; i++ {
		uapi.WatchLineInfo(f.Fd(), &li)
		uapi.UnwatchLineInfo(f.Fd(), 0)
	}
}
