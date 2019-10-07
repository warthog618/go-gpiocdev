// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

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
	Tclk time.Duration
	Sclk *gpiod.Line
	Ssz  *gpiod.Line
	Mosi *gpiod.Line
	Miso *gpiod.Line
	cpol int
	cpha int
}

// New creates a SPI.
func New(c *gpiod.Chip, sclk, ssz, mosi, miso int, options ...Option) (*SPI, error) {
	s := SPI{}
	for _, option := range options {
		option(&s)
	}
	if s.Tclk == 0 {
		// default to 1MHz full cycle.
		s.Tclk = 500 * time.Nanosecond
	}
	var err error
	var l *gpiod.Line
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	// hold SPI reset until needed...
	l, err = c.RequestLine(ssz, gpiod.AsOutput(1))
	if err != nil {
		return nil, err
	}
	s.Ssz = l
	clkOpts := []gpiod.LineOption{gpiod.AsOutput(0)}
	if s.cpol != 0 {
		clkOpts = append(clkOpts, gpiod.AsActiveLow)
	}
	l, err = c.RequestLine(sclk, clkOpts...)
	if err != nil {
		return nil, err
	}
	s.Sclk = l
	l, err = c.RequestLine(miso, gpiod.AsInput)
	if err != nil {
		return nil, err
	}
	s.Miso = l
	if miso == mosi {
		s.Mosi = s.Miso
		panic("setting line direction not currently supported")
	} else {
		l, err := c.RequestLine(mosi, gpiod.AsOutput(0))
		if err != nil {
			return nil, err
		}
		s.Mosi = l
	}
	return &s, nil
}

// Close releases allocated resources.
func (s *SPI) Close() {
	if s.Sclk != nil {
		s.Sclk.Close()
	}
	if s.Miso != nil {
		s.Miso.Close()
	}
	if s.Mosi != nil && s.Mosi != s.Miso {
		s.Mosi.Close()
	}
	if s.Ssz != nil {
		s.Ssz.Close()
	}
}

// ClockIn clocks in a data bit from the SPI device on Miso.
//
// Starts and ends just after the falling edge of the clock.
func (s *SPI) ClockIn() (int, error) {
	time.Sleep(s.Tclk)
	err := s.Sclk.SetValue(1)
	if err != nil {
		return 0, err
	}
	if s.cpha == 1 {
		time.Sleep(s.Tclk)
	}
	v, err := s.Miso.Value()
	if err != nil {
		return 0, err
	}
	if s.cpha == 0 {
		time.Sleep(s.Tclk)
	}
	err = s.Sclk.SetValue(0)
	if err != nil {
		return 0, err
	}
	return v, err
}

// ClockOut clocks out a data bit to the SPI device on Mosi.
//
// Starts and ends just after the falling edge of the clock.
func (s *SPI) ClockOut(v int) error {
	if s.cpha == 1 {
		time.Sleep(s.Tclk)
	}
	err := s.Mosi.SetValue(v)
	if err != nil {
		return err
	}
	if s.cpha == 0 {
		time.Sleep(s.Tclk)
	}
	err = s.Sclk.SetValue(1)
	if err != nil {
		return err
	}
	time.Sleep(s.Tclk)
	return s.Sclk.SetValue(0)
}

// Option specifies a construction option for the SPI.
type Option func(*SPI)

// WithCPOL sets the cpol for the SPI.
func WithCPOL(cpol int) Option {
	return func(s *SPI) {
		s.cpol = cpol
	}
}

// WithCPHA sets the cpol for the SPI.
func WithCPHA(cpha int) Option {
	return func(s *SPI) {
		s.cpha = cpha
	}
}

// WithTclk sets the clock period for the SPI.
//
// Note that this is the half-cycle period.
func WithTclk(tclk time.Duration) Option {
	return func(s *SPI) {
		s.Tclk = tclk
	}
}
