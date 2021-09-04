// SPDX-FileCopyrightText: 2020 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

package uapi_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

func BenchmarkLineInfoV2(b *testing.B) {
	c, err := mock.Chip(0)
	require.Nil(b, err)
	require.NotNil(b, c)
	f, err := os.Open(c.DevPath)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	for i := 0; i < b.N; i++ {
		uapi.GetLineInfoV2(f.Fd(), 0)
	}
}

func BenchmarkGetLine(b *testing.B) {
	c, err := mock.Chip(0)
	require.Nil(b, err)
	require.NotNil(b, c)
	f, err := os.Open(c.DevPath)
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
	c, err := mock.Chip(0)
	require.Nil(b, err)
	require.NotNil(b, c)
	f, err := os.Open(c.DevPath)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Input | uapi.LineFlagV2EdgeBoth,
		},
	}
	for i := 0; i < b.N; i++ {
		uapi.GetLine(f.Fd(), &lr)
		unix.Close(int(lr.Fd))
	}
}

func BenchmarkGetLineValuesV2(b *testing.B) {
	c, err := mock.Chip(0)
	require.Nil(b, err)
	require.NotNil(b, c)
	f, err := os.Open(c.DevPath)
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
		uapi.GetLineValuesV2(uintptr(lr.Fd), &lv)
	}
}

func BenchmarkSetLineValuesV2(b *testing.B) {
	c, err := mock.Chip(0)
	require.Nil(b, err)
	require.NotNil(b, c)
	f, err := os.Open(c.DevPath)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Output,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(lr.Fd))
	lv := uapi.LineValues{Mask: 1}
	for i := 0; i < b.N; i++ {
		uapi.SetLineValuesV2(uintptr(lr.Fd), lv)
	}
}

func BenchmarkSetLineValuesV2Sparse(b *testing.B) {
	c, err := mock.Chip(0)
	require.Nil(b, err)
	require.NotNil(b, c)
	f, err := os.Open(c.DevPath)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	lr := uapi.LineRequest{
		Lines:   4,
		Offsets: [uapi.LinesMax]uint32{0, 1, 2, 3},
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagV2Output,
		},
	}
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(lr.Fd))
	lv := uapi.LineValues{Mask: 0x0a}
	for i := 0; i < b.N; i++ {
		uapi.SetLineValuesV2(uintptr(lr.Fd), lv)
	}
}

func BenchmarkSetLineConfigV2(b *testing.B) {
	c, err := mock.Chip(0)
	require.Nil(b, err)
	require.NotNil(b, c)
	f, err := os.Open(c.DevPath)
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
		uapi.SetLineConfigV2(uintptr(lr.Fd), &lc)
	}
}

func BenchmarkWatchLineInfoV2(b *testing.B) {
	c, err := mock.Chip(0)
	require.Nil(b, err)
	require.NotNil(b, c)
	f, err := os.Open(c.DevPath)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	var li uapi.LineInfoV2
	for i := 0; i < b.N; i++ {
		uapi.WatchLineInfoV2(f.Fd(), &li)
		uapi.UnwatchLineInfo(f.Fd(), 0)
	}
}
