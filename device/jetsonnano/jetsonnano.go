// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
// SPDX-FileCopyrightText: 2023 Alex Bucknall <alex.bucknall@gmail.com>
//
// SPDX-License-Identifier: MIT

// Package jetsonnano provides convenience mappings from Jetson Nano pin names to
// offsets.
package jetsonnano

import (
	"errors"
	"strconv"
	"strings"
)

// Convenience mapping from J41 pinouts to BCM pinouts.
const (
	J41p27 = iota
	J41p28
	J41p3
	J41p5
	J41p7
	J41p29
	J41p31
	J41p26
	J41p24
	J41p21
	J41p19
	J41p23
	J41p32
	J41p33
	J41p8
	J41p10
	J41p36
	J41p11
	J41p12
	J41p35
	J41p38
	J41p40
	J41p15
	J41p16
	J41p18
	J41p22
	J41p37
	J41p13
)

// GPIO aliases to J41 pins
const (
	_ = iota
	_
	GPIO2
	GPIO3
	GPIO4
	GPIO5
	GPIO6
	GPIO7
	GPIO8
	GPIO9
	GPIO10
	GPIO11
	GPIO12
	GPIO13
	GPIO14
	GPIO15
	GPIO16
	GPIO17
	GPIO18
	GPIO19
	GPIO20
	GPIO21
	GPIO22
	GPIO23
	GPIO24
	GPIO25
	GPIO26
	GPIO27
	MaxGPIOPin
)

var j41Names = map[string]int{
	"3":  J41p3,
	"5":  J41p5,
	"7":  J41p7,
	"8":  J41p8,
	"10": J41p10,
	"11": J41p11,
	"12": J41p12,
	"13": J41p13,
	"15": J41p15,
	"16": J41p16,
	"18": J41p18,
	"19": J41p19,
	"21": J41p21,
	"22": J41p22,
	"23": J41p23,
	"24": J41p24,
	"26": J41p26,
	"27": J41p27,
	"28": J41p28,
	"29": J41p29,
	"31": J41p31,
	"32": J41p32,
	"33": J41p33,
	"35": J41p35,
	"36": J41p36,
	"37": J41p37,
	"38": J41p38,
	"40": J41p40,
}

// ErrInvalid indicates the pin name does not match a known pin.
var ErrInvalid = errors.New("invalid pin name")

func rangeCheck(p int) (int, error) {
	if p < GPIO2 || p >= MaxGPIOPin {
		return 0, ErrInvalid
	}
	return p, nil
}

// Pin maps a pin string name to a pin number.
//
// Pin names are case insensitive and may be of the form J41pX, GPIOX, or X.
func Pin(s string) (int, error) {
	s = strings.ToLower(s)
	switch {
	case strings.HasPrefix(s, "j41p"):
		v, ok := j41Names[s[3:]]
		if !ok {
			return 0, ErrInvalid
		}
		return v, nil
	case strings.HasPrefix(s, "gpio"):
		v, err := strconv.ParseInt(s[4:], 10, 8)
		if err != nil {
			return 0, err
		}
		return rangeCheck(int(v))
	default:
		v, err := strconv.ParseInt(s, 10, 8)
		if err != nil {
			return 0, err
		}
		return rangeCheck(int(v))
	}
}

// MustPin converts the string to the corresponding pin number or panics if that
// is not possible.
func MustPin(s string) int {
	v, err := Pin(s)
	if err != nil {
		panic(err)
	}
	return v
}
