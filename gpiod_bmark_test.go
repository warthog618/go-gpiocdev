// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package gpiod_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod"
)

func BenchmarkChipNewClose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c, _ := gpiod.NewChip(platform.Devpath())
		c.Close()
	}
}

func BenchmarkLineInfo(b *testing.B) {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	for i := 0; i < b.N; i++ {
		c.LineInfo(platform.IntrLine())
	}
}

func BenchmarkLineReconfigure(b *testing.B) {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	for i := 0; i < b.N; i++ {
		l.Reconfigure(gpiod.AsActiveLow)
	}
}

func BenchmarkLineValue(b *testing.B) {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	for i := 0; i < b.N; i++ {
		l.Value()
	}
}

func BenchmarkLinesValues(b *testing.B) {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	vv := make([]int, c.Lines())
	for i := 0; i < b.N; i++ {
		l.Values(vv)
	}
}
func BenchmarkLineSetValue(b *testing.B) {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput(0))
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	for i := 0; i < b.N; i++ {
		l.SetValue(1)
	}
}

func BenchmarkLinesSetValues(b *testing.B) {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	ll, err := c.RequestLines(platform.FloatingLines(), gpiod.AsOutput(0))
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
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	platform.TriggerIntr(1)
	ich := make(chan int)
	r, err := c.RequestLine(platform.IntrLine(),
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- 1
		}))
	require.Nil(b, err)
	require.NotNil(b, r)
	// absorb any pending interrupt
	select {
	case <-ich:
	case <-time.After(time.Millisecond):
	}
	for i := 0; i < b.N; i++ {
		platform.TriggerIntr(i & 1)
		<-ich
	}
	r.Close()
}
