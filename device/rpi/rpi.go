// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

package rpi

import (
	"errors"
	"strconv"
	"strings"
)

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
)

// GPIO aliases to J8 pins
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

var j8Names = map[string]int{
	"3":  J8p3,
	"5":  J8p5,
	"7":  J8p7,
	"8":  J8p8,
	"10": J8p10,
	"11": J8p11,
	"12": J8p12,
	"13": J8p13,
	"15": J8p15,
	"16": J8p16,
	"18": J8p18,
	"19": J8p19,
	"21": J8p21,
	"22": J8p22,
	"23": J8p23,
	"24": J8p24,
	"26": J8p26,
	"27": J8p27,
	"28": J8p28,
	"29": J8p29,
	"31": J8p31,
	"32": J8p32,
	"33": J8p33,
	"35": J8p35,
	"36": J8p36,
	"37": J8p37,
	"38": J8p38,
	"40": J8p40,
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
// Pin names are case insensitive and may be of the form J8pX, GPIOX, or X.
func Pin(s string) (int, error) {
	s = strings.ToLower(s)
	switch {
	case strings.HasPrefix(s, "j8p"):
		v, ok := j8Names[s[3:]]
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
