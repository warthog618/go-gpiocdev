// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

//go:build linux
// +build linux

package uapi_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiocdev/uapi"
	"github.com/warthog618/go-gpiosim"
	"golang.org/x/sys/unix"
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

func BenchmarkGetLineHandle(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	hr := uapi.HandleRequest{Lines: 1}
	for i := 0; i < b.N; i++ {
		uapi.GetLineHandle(f.Fd(), &hr)
		unix.Close(int(hr.Fd))
	}
}

func BenchmarkGetLineEvent(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, f)
	defer f.Close()
	er := uapi.EventRequest{
		HandleFlags: uapi.HandleRequestInput,
		EventFlags:  uapi.EventRequestBothEdges,
	}
	for i := 0; i < b.N; i++ {
		uapi.GetLineEvent(f.Fd(), &er)
		unix.Close(int(er.Fd))
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
	hr := uapi.HandleRequest{Lines: 1}
	err = uapi.GetLineHandle(f.Fd(), &hr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(hr.Fd))
	var hd uapi.HandleData
	for i := 0; i < b.N; i++ {
		uapi.GetLineValues(uintptr(hr.Fd), &hd)
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
	hr := uapi.HandleRequest{Lines: 1, Flags: uapi.HandleRequestOutput}
	err = uapi.GetLineHandle(f.Fd(), &hr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(hr.Fd))
	var hd uapi.HandleData
	for i := 0; i < b.N; i++ {
		uapi.SetLineValues(uintptr(hr.Fd), hd)
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
	hr := uapi.HandleRequest{Lines: 1}
	err = uapi.GetLineHandle(f.Fd(), &hr)
	require.Nil(b, err)
	require.NotNil(b, f)
	defer unix.Close(int(hr.Fd))
	var hc uapi.HandleConfig
	for i := 0; i < b.N; i++ {
		uapi.SetLineConfig(uintptr(hr.Fd), &hc)
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
