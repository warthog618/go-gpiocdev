// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package spi

import (
	"time"

	"github.com/warthog618/gpiod"
)

// SPI represents a device connected an SPI bus using 4 GPIO lines.
//
// This is the basis for bit bashed SPI interfaces using GPIO pins. It is not
// related to the SPI device drivers provided by Linux.
type SPI struct {
	// time between clock edges (i.e. half the cycle time)
	Tclk  time.Duration
	Sclk  *gpiod.Line
	Ssz   *gpiod.Line
	Mosi  *gpiod.Line
	Miso  *gpiod.Line
	lines [4]*gpiod.Line
}

// New creates a SPI.
func New(tclk time.Duration, sclk, ssz, mosi, miso int, c *gpiod.Chip) (*SPI, error) {
	var err error
	if c == nil {
		c, err = gpiod.NewChip("gpiochip0")
		if err != nil {
			return nil, err
		}
		defer c.Close()
	}
	lines := []struct {
		offset int
		option gpiod.LineOption
	}{
		// hold SPI reset until needed...
		{ssz, gpiod.AsOutput(1)},
		{sclk, gpiod.AsOutput(0)},
		{mosi, gpiod.AsOutput(0)},
		{miso, gpiod.AsInput},
	}
	s := SPI{
		Tclk: tclk,
	}
	for i, l := range lines {
		r, err := c.RequestLine(l.offset, l.option)
		if err != nil {
			return nil, err
		}
		s.lines[i] = r
	}
	s.Ssz = s.lines[0]
	s.Sclk = s.lines[1]
	s.Mosi = s.lines[2]
	s.Miso = s.lines[3]
	return &s, nil
}

// Close releases allocated resources.
func (s *SPI) Close() {
	for _, l := range s.lines {
		if l != nil {
			l.Close()
		}
	}
}

// ClockIn clocks in a data bit from the SPI device on Miso.
//
// Assumes clock starts high and ends with the rising edge of the next clock.
func (s *SPI) ClockIn() (int, error) {
	time.Sleep(s.Tclk)
	err := s.Sclk.SetValue(0) // SPI device writes on the falling edge
	if err != nil {
		return 0, err
	}
	time.Sleep(s.Tclk)
	v, err := s.Miso.Value()
	if err != nil {
		return 0, err
	}
	err = s.Sclk.SetValue(1)
	return v, err
}

// ClockOut clocks out a data bit to the SPI device on Mosi.
//
// Assumes clock starts low and ends with the falling edge of the next clock.
func (s *SPI) ClockOut(v int) error {
	err := s.Mosi.SetValue(v)
	if err != nil {
		return err
	}
	time.Sleep(s.Tclk)
	err = s.Sclk.SetValue(1) // SPI device reads on the rising edge
	if err != nil {
		return err
	}
	time.Sleep(s.Tclk)
	return s.Sclk.SetValue(0)
}
