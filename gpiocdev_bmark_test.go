// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package gpiocdev_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiosim"
)

func BenchmarkChipNewClose(b *testing.B) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(b, err)
	defer s.Close()
	for i := 0; i < b.N; i++ {
		c, _ := gpiocdev.NewChip(s.DevPath())
		c.Close()
	}
}

func BenchmarkLineInfo(b *testing.B) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(b, err)
	defer s.Close()
	c, err := gpiocdev.NewChip(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	for i := 0; i < b.N; i++ {
		c.LineInfo(3)
	}
}

func BenchmarkLineReconfigure(b *testing.B) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(b, err)
	defer s.Close()
	c, err := gpiocdev.NewChip(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLine(3)
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	for i := 0; i < b.N; i++ {
		l.Reconfigure(gpiocdev.AsActiveLow)
	}
}

func BenchmarkLineValue(b *testing.B) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(b, err)
	defer s.Close()
	c, err := gpiocdev.NewChip(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLine(3)
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	for i := 0; i < b.N; i++ {
		l.Value()
	}
}

func BenchmarkLinesValues(b *testing.B) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(b, err)
	defer s.Close()
	c, err := gpiocdev.NewChip(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLines([]int{1, 2, 3})
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	vv := make([]int, len(l.Offsets()))
	for i := 0; i < b.N; i++ {
		l.Values(vv)
	}
}
func BenchmarkLineSetValue(b *testing.B) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(b, err)
	defer s.Close()
	c, err := gpiocdev.NewChip(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLine(3, gpiocdev.AsOutput(0))
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	for i := 0; i < b.N; i++ {
		l.SetValue(1)
	}
}

func BenchmarkLinesSetValues(b *testing.B) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(b, err)
	defer s.Close()
	c, err := gpiocdev.NewChip(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	ll, err := c.RequestLines([]int{1, 2}, gpiocdev.AsOutput(0))
	require.Nil(b, err)
	require.NotNil(b, ll)
	defer ll.Close()
	vv := []int{0, 0}
	for i := 0; i < b.N; i++ {
		vv[0] = i & 1
		ll.SetValues(vv)
	}
}

func BenchmarkInterruptLatency(b *testing.B) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(b, err)
	defer s.Close()
	c, err := gpiocdev.NewChip(s.DevPath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	offset := 2
	s.SetPull(offset, 1)
	ich := make(chan int)
	eh := func(evt gpiocdev.LineEvent) {
		ich <- 1
	}
	r, err := c.RequestLine(offset,
		gpiocdev.WithBothEdges,
		gpiocdev.WithEventHandler(eh))
	require.Nil(b, err)
	require.NotNil(b, r)
	// absorb any pending interrupt
	select {
	case <-ich:
	case <-time.After(time.Millisecond):
	}
	for i := 0; i < b.N; i++ {
		s.SetPull(offset, i&1)
		<-ich
	}
	r.Close()
}
