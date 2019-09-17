// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package gpiod_test

import (
	"gpiod"
	"testing"

	"github.com/stretchr/testify/require"
)

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
	for i := 0; i < b.N; i++ {
		l.Values()
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
	defer r.Close()
	for i := 0; i < b.N; i++ {
		platform.TriggerIntr(i & 1)
		<-ich
	}
}
