// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// Package adc0832 provides a bit bashed device driver ADC0832s.
package adc0832

import (
	"errors"
	"sync"
	"time"

	"github.com/warthog618/gpiod/spi"
)

// ADC0832 reads ADC values from a connected ADC0832.
type ADC0832 struct {
	mu sync.Mutex
	s  *spi.SPI
	// time to allow mux to settle after clocking out ODD/SIGN
	tset time.Duration
}

// New creates a ADC0832.
func New(tclk, tset time.Duration, sclk, ssz, mosi, miso int) (*ADC0832, error) {
	s, err := spi.New(tclk, sclk, ssz, mosi, miso, nil)
	if err != nil {
		return nil, err
	}
	return &ADC0832{s: s, tset: tset}, nil
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
	s := adc.s
	err := s.Ssz.SetValue(1)
	if err != nil {
		return 0, err
	}
	err = s.Sclk.SetValue(0)
	if err != nil {
		return 0, err
	}
	err = s.Mosi.SetValue(1)
	if err != nil {
		return 0, err
	}
	time.Sleep(adc.s.Tclk)
	err = s.Ssz.SetValue(0)
	if err != nil {
		return 0, err
	}

	odd := 0
	if ch != 0 {
		odd = 1
	}
	err = s.ClockOut(1) // Start
	if err != nil {
		return 0, err
	}
	err = s.ClockOut(sgl) // SGL/DIFZ
	if err != nil {
		return 0, err
	}
	err = s.ClockOut(odd) // ODD/Sign
	if err != nil {
		return 0, err
	}
	// mux settling
	time.Sleep(adc.tset)
	err = s.Sclk.SetValue(1)
	if err != nil {
		return 0, err
	}
	// MSB first byte
	var d uint8
	for i := uint(0); i < 8; i++ {
		v, err := s.ClockIn()
		if err != nil {
			return 0, err
		}
		d = d << 1
		if v != 0 {
			d = d | 0x01
		}
	}
	// ignore LSB bits - same as MSB just reversed order
	err = s.Ssz.SetValue(1)
	if err != nil {
		return 0, err
	}
	return d, nil
}
