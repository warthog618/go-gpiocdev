// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// Package spi provides an example bit-bashed spi driver.
//
// The intent of the package is to demonstrate using GPIO pins to implement
// bit-bashed interfaces for prototyping.
//
// The package is not related to the SPI device drivers provided by the Linux
// kernel.  Kernel device drivers are the preferred solution for production
// applications.
package spi

import (
	"time"

	"github.com/warthog618/go-gpiocdev"
)

// SPI represents a device connected an SPI bus using 4 GPIO lines.
//
// This is the basis for bit bashed SPI interfaces using GPIO pins.
// It is not related to the SPI device drivers provided by the Linux kernel.
type SPI struct {
	// time between clock edges (i.e. half the cycle time)
	Tclk time.Duration

	// Clock line
	Sclk *gpiocdev.Line

	// Slave select - active low
	Ssz *gpiocdev.Line

	// Master-out slave-in
	Mosi *gpiocdev.Line

	// Master-in slave-out
	Miso *gpiocdev.Line

	// Polarity of idle state
	cpol int

	// Phase - clock edge used for reads and writes
	//
	// 0 => out side writes on trailing clock edge
	// 1 => out side writes on leading clock edge
	//
	// In side reads on the opposite clock edge.
	cpha int
}

// New creates a SPI.
func New(c *gpiocdev.Chip, sclk, ssz, mosi, miso int, options ...Option) (*SPI, error) {
	s := SPI{}
	for _, option := range options {
		option(&s)
	}
	if s.Tclk == 0 {
		// default to 1MHz full cycle.
		s.Tclk = 500 * time.Nanosecond
	}
	var err error
	var l *gpiocdev.Line
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	// hold SPI reset until needed...
	l, err = c.RequestLine(ssz, gpiocdev.AsOutput(1))
	if err != nil {
		return nil, err
	}
	s.Ssz = l
	clkOpts := []gpiocdev.LineReqOption{gpiocdev.AsOutput(0)}
	if s.cpol != 0 {
		clkOpts = append(clkOpts, gpiocdev.AsActiveLow)
	}
	l, err = c.RequestLine(sclk, clkOpts...)
	if err != nil {
		return nil, err
	}
	s.Sclk = l
	l, err = c.RequestLine(miso, gpiocdev.AsInput)
	if err != nil {
		return nil, err
	}
	s.Miso = l
	if miso == mosi {
		s.Mosi = s.Miso
	} else {
		l, err := c.RequestLine(mosi, gpiocdev.AsOutput(0))
		if err != nil {
			return nil, err
		}
		s.Mosi = l
	}
	return &s, nil
}

// Close releases allocated resources and reverts all output lines to inputs.
func (s *SPI) Close() {
	if s.Sclk != nil {
		s.Sclk.Reconfigure(gpiocdev.AsInput)
		s.Sclk.Close()
	}
	if s.Mosi != nil {
		s.Mosi.Reconfigure(gpiocdev.AsInput)
		s.Mosi.Close()
	}
	if s.Miso != nil && s.Mosi != s.Miso {
		s.Miso.Close()
	}
	if s.Ssz != nil {
		s.Ssz.Reconfigure(gpiocdev.AsInput)
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

// WithCPHA sets the cpha for the SPI.
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
