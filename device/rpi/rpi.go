// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package rpi

// Convenience mapping from J8 pinouts to BCM pinouts.
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

// GPIO aliases to J8 pins
const (
	GPIO2  = J8p3
	GPIO3  = J8p5
	GPIO4  = J8p7
	GPIO5  = J8p29
	GPIO6  = J8p31
	GPIO7  = J8p26
	GPIO8  = J8p24
	GPIO9  = J8p21
	GPIO10 = J8p19
	GPIO11 = J8p23
	GPIO12 = J8p32
	GPIO13 = J8p33
	GPIO14 = J8p8
	GPIO15 = J8p10
	GPIO16 = J8p36
	GPIO17 = J8p11
	GPIO18 = J8p12
	GPIO19 = J8p35
	GPIO20 = J8p38
	GPIO21 = J8p40
	GPIO22 = J8p15
	GPIO23 = J8p16
	GPIO24 = J8p18
	GPIO25 = J8p22
	GPIO26 = J8p37
	GPIO27 = J8p13
)
