// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

// Package rpi provides convenience mappings from Rasperry Pi pin names to
// offsets.
package bananapi

import (
	"errors"
)

// GPIO aliases to offsets
var (
	GPIO2  = GPIO_TO_OFFSET[2]
	GPIO3  = GPIO_TO_OFFSET[3]
	GPIO4  = GPIO_TO_OFFSET[4]
	GPIO5  = GPIO_TO_OFFSET[5]
	GPIO6  = GPIO_TO_OFFSET[6]
	GPIO7  = GPIO_TO_OFFSET[7]
	GPIO8  = GPIO_TO_OFFSET[8]
	GPIO9  = GPIO_TO_OFFSET[9]
	GPIO10 = GPIO_TO_OFFSET[10]
	GPIO11 = GPIO_TO_OFFSET[11]
	GPIO12 = GPIO_TO_OFFSET[12]
	GPIO13 = GPIO_TO_OFFSET[13]
	GPIO14 = GPIO_TO_OFFSET[14]
	GPIO15 = GPIO_TO_OFFSET[15]
	GPIO16 = GPIO_TO_OFFSET[16]
	GPIO17 = GPIO_TO_OFFSET[17]
	GPIO18 = GPIO_TO_OFFSET[18]
	GPIO19 = GPIO_TO_OFFSET[19]
	GPIO20 = GPIO_TO_OFFSET[20]
	GPIO21 = GPIO_TO_OFFSET[21]
	GPIO22 = GPIO_TO_OFFSET[22]
	GPIO23 = GPIO_TO_OFFSET[23]
	GPIO24 = GPIO_TO_OFFSET[24]
	GPIO25 = GPIO_TO_OFFSET[25]
	GPIO26 = GPIO_TO_OFFSET[26]
	GPIO27 = GPIO_TO_OFFSET[27]
)

var GPIO_TO_OFFSET = map[int]int{
	2:  53,
	3:  52,
	4:  259,
	5:  37,
	6:  38,
	7:  270,
	8:  266,
	9:  269,
	10: 268,
	11: 267,
	12: 38,
	13: 39,
	14: 224,
	15: 225,
	16: 277,
	17: 275,
	18: 226,
	19: 40,
	20: 276,
	21: 45,
	22: 273,
	23: 244,
	24: 245,
	25: 272,
	26: 35,
	27: 274,
}

// ErrInvalid indicates the pin name does not match a known pin.
var ErrInvalid = errors.New("invalid pin number")

func rangeCheck(p int) (int, error) {
	if p < 2 || p >= 27 {
		return 0, ErrInvalid
	}
	return p, nil
}

// Pin maps a pin number to the GPIO offset.
func Pin(s int) (int, error) {
	return rangeCheck(s)
}
