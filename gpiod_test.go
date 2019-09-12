// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package gpiod_test

import (
	"bytes"
	"fmt"
	"gpiod"
	"gpiod/mockup"
	"gpiod/uapi"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var platform Platform
var setupError error

func TestMain(m *testing.M) {
	detectPlatform()
	if platform == nil {
		fmt.Println("Unsupported platform -", setupError)
		os.Exit(-1)
	}
	rc := m.Run()
	platform.Close()
	os.Exit(rc)
}

func TestNewChip(t *testing.T) {
	requirePlatform(t)

	// not chardev

	// success

	// remainder is hard to test.
}

func TestChipClose(t *testing.T) {
	requirePlatform(t)
	// without lines
	c := getChip(t)
	err := c.Close()
	require.Nil(t, err)
	err = c.Close()
	require.Equal(t, gpiod.ErrClosed, err)

	// with lines
	c = getChip(t)
	_, err = c.RequestLines([]int{1, 2, 3}, gpiod.WithBothEdges(nil))
	require.Nil(t, err)
	err = c.Close()
	require.Nil(t, err)
	err = c.Close()
	require.Equal(t, gpiod.ErrClosed, err)
}

func TestChipLineInfo(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()
	xli := gpiod.LineInfo{}

	// out of range
	li, err := c.LineInfo(platform.Lines())
	assert.Equal(t, unix.EINVAL, err)
	assert.Equal(t, xli, li)

	// valid
	li, err = c.LineInfo(1)
	require.Nil(t, err)
	xli.Offset = 1
	assert.Equal(t, xli, li)
}

func TestChipLines(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()
	lines := c.Lines()
	assert.Equal(t, platform.Lines(), lines)
}

func TestChipRequestLine(t *testing.T) {
	requirePlatform(t)
}

func TestChipRequestLines(t *testing.T) {
	requirePlatform(t)
}

func TestLineClose(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()
}

func TestLineValue(t *testing.T) {
}

func TestLineSetValue(t *testing.T) {
}

func TestLineValues(t *testing.T) {
}

func TestLineSetValues(t *testing.T) {
}

func getChip(t *testing.T) *gpiod.Chip {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(t, err)
	require.NotNil(t, c)
	return c
}

type chip struct {
	devpath string
	lines   int
	intr    int
	ff      []int
}

func (c *chip) Devpath() string {
	return c.devpath
}

func (c *chip) Lines() int {
	return c.lines
}

func (c *chip) IntrLine() int {
	return c.intr
}

func (c *chip) FloatingLines() []int {
	return c.ff
}

// two flavours of chip, raspberry and mockup.
type Platform interface {
	Name() string
	Devpath() string
	Lines() int
	IntrLine() int
	FloatingLines() []int
	TriggerIntr(int)
	Close()
}

func requirePlatform(t *testing.T) {
	t.Helper()
	if platform == nil {
		t.Skip("platform not supported -", setupError)
	}
}

type RaspberryPi struct {
	chip
	c     *gpiod.Chip
	wline *gpiod.Line
}

func newPi(path string) (*RaspberryPi, error) {
	if err := gpiod.IsCharDev(path); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, unix.O_CLOEXEC, unix.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ci, err := uapi.GetChipInfo(f.Fd())
	if err != nil {
		return nil, err
	}
	label := bytesToString(ci.Label[:])
	if label != "pinctrl-bcm2835" {
		return nil, fmt.Errorf("unsupported gpiochip - %s", label)
	}
	ch, err := gpiod.NewChip(path)
	if err != nil {
		return nil, err
	}
	w, err := ch.RequestLine(J8p16, gpiod.AsOutput(0),
		gpiod.WithConsumer("gpiod-test-w"))
	pi := RaspberryPi{chip{path, int(ci.Lines), J8p15, []int{J8p11, J8p12}}, ch, w}
	// !!! check J8p15 and J8p16 are looped
	return &pi, nil
}

func bytesToString(a []byte) string {
	n := bytes.IndexByte(a, 0)
	if n == -1 {
		return string(a)
	}
	return string(a[:n])
}

func (c *RaspberryPi) Name() string {
	return "raspberrypi"
}

func (c *RaspberryPi) TriggerIntr(value int) {
	c.wline.SetValue(value)
}

func (c *RaspberryPi) Close() {
	c.wline.Close()
	c.c.Close()
}

type Mockup struct {
	chip
	m *mockup.Mockup
	c *mockup.Chip
}

func newMockup() (*Mockup, error) {
	m, err := mockup.New([]int{20}, false)
	if err != nil {
		return nil, err
	}
	c, err := m.Chip(0)
	if err != nil {
		return nil, err
	}
	return &Mockup{chip{c.DevPath, 20, 10, []int{11, 12}}, m, c}, nil
}

func (c *Mockup) Name() string {
	return "gpio-mockup"
}

func (c *Mockup) TriggerIntr(value int) {
	c.c.SetValue(c.intr, value)
}

func (c *Mockup) Close() {
	c.m.Close()
}

func detectPlatform() {
	if pi, err := newPi("/dev/gpiochip0"); err == nil {
		platform = pi
		return
	}
	mock, err := newMockup()
	if err != nil {
		setupError = err
		return
	}
	platform = mock
}

// Raspberry Pi BCM GPIO pins
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
