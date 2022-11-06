// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// Package adc0832 provides a bit bashed device driver ADC0832s.
package adc0832

import (
	"errors"
	"sync"
	"time"

	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/spi"
)

// ADC0832 reads ADC values from a connected ADC0832.
type ADC0832 struct {
	mu sync.Mutex
	s  *spi.SPI
	// time to allow mux to settle after clocking out ODD/SIGN
	tset time.Duration
}

// New creates a ADC0832.
func New(c *gpiocdev.Chip, clk, csz, di, do int, options ...Option) (*ADC0832, error) {
	s, err := spi.New(c, clk, csz, di, do, spi.WithTclk(2500*time.Nanosecond))
	if err != nil {
		return nil, err
	}
	a := ADC0832{s: s}
	for _, option := range options {
		option(&a)
	}
	return &a, nil
}

// Close releases all resources allocated by the ADC.
func (adc *ADC0832) Close() error {
	adc.mu.Lock()
	defer adc.mu.Unlock()
	if adc.s == nil {
		return ErrClosed
	}
	adc.s.Close()
	adc.s = nil
	return nil
}

// Read returns the value of a single channel read from the ADC.
func (adc *ADC0832) Read(ch int) (uint8, error) {
	return adc.read(ch, 1)
}

// ReadDifferential returns the value of a differential pair read from the ADC.
func (adc *ADC0832) ReadDifferential(ch int) (uint8, error) {
	return adc.read(ch, 0)
}

// ErrClosed indicates the ADC is closed.
var ErrClosed = errors.New("closed")

func (adc *ADC0832) read(ch int, sgl int) (uint8, error) {
	adc.mu.Lock()
	defer adc.mu.Unlock()
	if adc.s == nil {
		return 0, ErrClosed
	}

	err := selectChip(adc.s)
	if err != nil {
		return 0, err
	}

	err = selectChannel(adc.s, sgl, ch)
	if err != nil {
		return 0, err
	}

	// mux settling
	if adc.s.Mosi == adc.s.Miso {
		adc.s.Miso.Reconfigure(gpiocdev.AsInput)
	}
	time.Sleep(adc.tset)
	_, err = adc.s.ClockIn() // sample time - junk
	if err != nil {
		return 0, err
	}

	// MSB first byte
	d, err := adc.clockInData()
	if err != nil {
		return 0, err
	}

	// ignore LSB bits - same as MSB just reversed order
	err = deselectChip(adc.s)
	if err != nil {
		return 0, err
	}
	return d, nil
}

func selectChip(s *spi.SPI) error {
	err := s.Ssz.SetValue(1)
	if err != nil {
		return err
	}
	err = s.Sclk.SetValue(0)
	if err != nil {
		return err
	}
	if s.Mosi == s.Miso {
		err = s.Mosi.Reconfigure(gpiocdev.AsOutput(1))
	} else {
		err = s.Mosi.SetValue(1)
	}
	if err != nil {
		return err
	}
	time.Sleep(s.Tclk)
	return s.Ssz.SetValue(0)
}

func deselectChip(s *spi.SPI) error {
	return s.Ssz.SetValue(1)
}

func selectChannel(s *spi.SPI, sgl, ch int) error {
	odd := 0
	if ch != 0 {
		odd = 1
	}
	bits := []int{1, sgl, odd} // Start, SGL/DIFZ, ODD/Sign
	for _, v := range bits {
		err := s.ClockOut(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (adc *ADC0832) clockInData() (uint8, error) {
	var d uint8
	for i := uint(0); i < 8; i++ {
		v, err := adc.s.ClockIn()
		if err != nil {
			return 0, err
		}
		d = d << 1
		if v != 0 {
			d = d | 0x01
		}
	}
	return d, nil
}

// Option specifies a construction option for the ADC.
type Option func(*ADC0832)

// WithTclk sets the clock period for the ADC.
//
// Note that this is the half-cycle period.
func WithTclk(tclk time.Duration) Option {
	return func(a *ADC0832) {
		a.s.Tclk = tclk
	}
}

// WithTset sets the settling period for the ADC.
func WithTset(tset time.Duration) Option {
	return func(a *ADC0832) {
		a.tset = tset
	}
}
