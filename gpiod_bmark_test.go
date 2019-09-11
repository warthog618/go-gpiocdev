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

func BenchmarkRead(b *testing.B) {
	c, err := gpiod.NewChip("/dev/gpiochip0")
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLine(J8p15)
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	for i := 0; i < b.N; i++ {
		l.Value()
	}
}

func BenchmarkWrite(b *testing.B) {
	c, err := gpiod.NewChip("/dev/gpiochip0")
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	l, err := c.RequestLine(J8p16, gpiod.AsOutput(0))
	require.Nil(b, err)
	require.NotNil(b, l)
	defer l.Close()
	for i := 0; i < b.N; i++ {
		l.SetValue(1)
	}
}

// Raspberry Pi BCM GPIO pins
const (
	J8p27 = iota
	J8p28
	J8p3
	J8p5
	J8p7
	J8p29
	J8p31
	J8p26
	J8p24
	J8p21
	J8p19
	J8p23
	J8p32
	J8p33
	J8p8
	J8p10
	J8p36
	J8p11
	J8p12
	J8p35
	J8p38
	J8p40
	J8p15
	J8p16
	J8p18
	J8p22
	J8p37
	J8p13
	MaxGPIOPin
)

func BenchmarkInterruptLatency(b *testing.B) {
	c, err := gpiod.NewChip("/dev/gpiochip0")
	require.Nil(b, err)
	require.NotNil(b, c)
	defer c.Close()
	w, err := c.RequestLine(J8p16, gpiod.AsOutput(1))
	require.Nil(b, err)
	require.NotNil(b, w)
	defer w.Close()
	ich := make(chan int)
	r, err := c.RequestLine(J8p15,
		gpiod.WithBothEdges(func(evt gpiod.LineEvent) {
			ich <- 1
		}))
	require.Nil(b, err)
	require.NotNil(b, r)
	defer r.Close()
	for i := 0; i < b.N; i++ {
		w.SetValue(i & 1)
		<-ich
	}
}
