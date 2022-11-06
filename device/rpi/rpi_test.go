// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

package rpi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/warthog618/go-gpiocdev/device/rpi"
)

var patterns = []struct {
	name string
	val  int
	err  error
}{
	{"gpio0", 0, rpi.ErrInvalid},
	{"gpio1", 0, rpi.ErrInvalid},
	{"gpio2", 2, nil},
	{"gpio02", 2, nil},
	{"GPIO2", 2, nil},
	{"Gpio2", 2, nil},
	{"gpio27", 27, nil},
	{"gpio28", 0, rpi.ErrInvalid},
	{"J8p0", 0, rpi.ErrInvalid},
	{"J8p1", 0, rpi.ErrInvalid},
	{"J8p2", 0, rpi.ErrInvalid},
	{"j8p3", rpi.J8p3, nil},
	{"J8P3", rpi.J8p3, nil},
	{"J8p3", rpi.J8p3, nil},
	{"J8p4", 0, rpi.ErrInvalid},
	{"J8p5", rpi.J8p5, nil},
	{"J8p6", 0, rpi.ErrInvalid},
	{"J8p7", rpi.J8p7, nil},
	{"J8p8", rpi.J8p8, nil},
	{"J8p9", 0, rpi.ErrInvalid},
	{"J8p10", rpi.J8p10, nil},
	{"J8p11", rpi.J8p11, nil},
	{"J8p12", rpi.J8p12, nil},
	{"J8p13", rpi.J8p13, nil},
	{"J8p14", 0, rpi.ErrInvalid},
	{"J8p15", rpi.J8p15, nil},
	{"J8p16", rpi.J8p16, nil},
	{"J8p17", 0, rpi.ErrInvalid},
	{"J8p18", rpi.J8p18, nil},
	{"J8p19", rpi.J8p19, nil},
	{"J8p20", 0, rpi.ErrInvalid},
	{"J8p21", rpi.J8p21, nil},
	{"J8p22", rpi.J8p22, nil},
	{"J8p23", rpi.J8p23, nil},
	{"J8p24", rpi.J8p24, nil},
	{"J8p25", 0, rpi.ErrInvalid},
	{"J8p26", rpi.J8p26, nil},
	{"J8p27", rpi.J8p27, nil},
	{"J8p28", rpi.J8p28, nil},
	{"J8p29", rpi.J8p29, nil},
	{"J8p30", 0, rpi.ErrInvalid},
	{"J8p31", rpi.J8p31, nil},
	{"J8p32", rpi.J8p32, nil},
	{"J8p33", rpi.J8p33, nil},
	{"J8p34", 0, rpi.ErrInvalid},
	{"J8p35", rpi.J8p35, nil},
	{"J8p36", rpi.J8p36, nil},
	{"J8p37", rpi.J8p37, nil},
	{"J8p38", rpi.J8p38, nil},
	{"J8p39", 0, rpi.ErrInvalid},
	{"J8p40", rpi.J8p40, nil},
	{"0", 0, rpi.ErrInvalid},
	{"02", 2, nil},
	{"2", 2, nil},
	{"27", 27, nil},
	{"40", 0, rpi.ErrInvalid},
}

func TestPin(t *testing.T) {
	for _, p := range patterns {
		tf := func(t *testing.T) {
			val, err := rpi.Pin(p.name)
			assert.Equal(t, p.err, err)
			assert.Equal(t, p.val, val)
		}
		t.Run(p.name, tf)
	}
}

func TestMustPin(t *testing.T) {
	for _, p := range patterns {
		tf := func(t *testing.T) {
			if p.err != nil {
				assert.Panics(t, func() {
					rpi.MustPin(p.name)
				})
			} else {
				val := rpi.MustPin(p.name)
				assert.Equal(t, p.val, val)
			}
		}
		t.Run(p.name, tf)
	}
}
