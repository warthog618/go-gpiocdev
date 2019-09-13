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

	// non-existent
	c, err := gpiod.NewChip(platform.Devpath() + "not")
	assert.NotNil(t, err)
	assert.Nil(t, c)

	// success
	c, err = gpiod.NewChip(platform.Devpath())
	assert.Nil(t, err)
	require.NotNil(t, c)
	err = c.Close()
	assert.Nil(t, err)

	// option
	c, err = gpiod.NewChip(platform.Devpath(),
		gpiod.WithConsumer("gpiod_test"))
	assert.Nil(t, err)
	require.NotNil(t, c)
	assert.Equal(t, platform.Name(), c.Name)
	assert.Equal(t, platform.Label(), c.Label)
	err = c.Close()
	assert.Nil(t, err)
}

func TestChipClose(t *testing.T) {
	requirePlatform(t)
	// without lines
	c := getChip(t)
	err := c.Close()
	assert.Nil(t, err)

	// closed
	err = c.Close()
	assert.Equal(t, gpiod.ErrClosed, err)

	// with lines
	c = getChip(t)
	require.NotNil(t, c)
	ll, err := c.RequestLines(platform.FloatingLines(),
		gpiod.WithBothEdges(func(gpiod.LineEvent) {}))
	assert.Nil(t, err)
	err = c.Close()
	assert.Nil(t, err)
	require.NotNil(t, ll)
	err = ll.Close()
	assert.Nil(t, err)

	// after lines closed
	c = getChip(t)
	require.NotNil(t, c)
	ll, err = c.RequestLines(platform.FloatingLines(),
		gpiod.WithBothEdges(func(gpiod.LineEvent) {}))
	assert.Nil(t, err)
	require.NotNil(t, ll)
	err = ll.Close()
	assert.Nil(t, err)
	err = c.Close()
	assert.Nil(t, err)
}

func TestChipLineInfo(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()
	xli := gpiod.LineInfo{}

	// out of range
	li, err := c.LineInfo(platform.Lines())
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	assert.Equal(t, xli, li)

	// valid
	li, err = c.LineInfo(1)
	assert.Nil(t, err)
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
	c := getChip(t)
	defer c.Close()

	// negative
	l, err := c.RequestLine(-1)
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, l)

	// out of range
	l, err = c.RequestLine(c.Lines())
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, l)

	// success - input
	l, err = c.RequestLine(platform.FloatingLines()[0])
	assert.Nil(t, err)
	require.NotNil(t, l)

	// already requested input
	l2, err := c.RequestLine(platform.FloatingLines()[0])
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output
	l2, err = c.RequestLine(platform.FloatingLines()[0], gpiod.AsOutput(0))
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	// already requested output as event
	l2, err = c.RequestLine(platform.FloatingLines()[0],
		gpiod.WithBothEdges(func(gpiod.LineEvent) {}))
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, l2)

	err = l.Close()
	assert.Nil(t, err)
}

func TestChipRequestLines(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	// negative
	ll, err := c.RequestLines([]int{platform.IntrLine(), -1})
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// out of range
	ll, err = c.RequestLines([]int{platform.IntrLine(), c.Lines()})
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	require.Nil(t, ll)

	// success - output
	ll, err = c.RequestLines(platform.FloatingLines(), gpiod.AsOutput())
	assert.Nil(t, err)
	require.NotNil(t, ll)

	// already requested input
	ll2, err := c.RequestLines(platform.FloatingLines())
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output
	ll2, err = c.RequestLines(platform.FloatingLines(), gpiod.AsOutput())
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	// already requested output as event
	ll2, err = c.RequestLines(platform.FloatingLines(),
		gpiod.WithBothEdges(func(gpiod.LineEvent) {}))
	assert.Equal(t, unix.EBUSY, err)
	require.Nil(t, ll2)

	err = ll.Close()
	assert.Nil(t, err)
}

func TestLineClose(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)

	err = l.Close()
	assert.Equal(t, gpiod.ErrClosed, err)
}

func TestLineValue(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	platform.TriggerIntr(0)
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	v, err := l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
	platform.TriggerIntr(1)
	v, err = l.Value()
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	l.Close()
}

func TestLineSetValue(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	// input
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValue(1)
	assert.Equal(t, gpiod.ErrPermissionDenied, err)
	l.Close()

	// output
	l, err = c.RequestLine(platform.FloatingLines()[0],
		gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValue(1)
	assert.Nil(t, err)
	l.Close()
}

func TestLinesClose(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Close()
	assert.Nil(t, err)

	err = l.Close()
	assert.Equal(t, gpiod.ErrClosed, err)
}
func TestLinesValues(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	// input
	platform.TriggerIntr(0)
	lines := append([]int{platform.IntrLine()}, platform.FloatingLines()...)
	l, err := c.RequestLines(lines)
	assert.Nil(t, err)
	require.NotNil(t, l)
	vv, err := l.Values()
	assert.Nil(t, err)
	assert.Equal(t, len(lines), len(vv))
	assert.Equal(t, 0, vv[0])
	platform.TriggerIntr(1)
	vv, err = l.Values()
	assert.Nil(t, err)
	assert.Equal(t, len(lines), len(vv))
	assert.Equal(t, 1, vv[0])

	l.Close()

	// output
	lines = platform.FloatingLines()
	l, err = c.RequestLines(lines, gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	vv, err = l.Values()
	assert.Nil(t, err)
	assert.Equal(t, len(lines), len(vv))
	// actual values are indeterminate

	l.Close()
}

func TestLinesSetValues(t *testing.T) {
	requirePlatform(t)
	c := getChip(t)
	defer c.Close()

	// input
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValues(0, 1)
	assert.Equal(t, gpiod.ErrPermissionDenied, err)
	l.Close()

	// output
	l, err = c.RequestLines(platform.FloatingLines(),
		gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValues(1, 0)
	assert.Nil(t, err)

	// too many values
	err = l.SetValues(1, 1, 1)
	assert.Equal(t, gpiod.ErrInvalidOffset, err)

	l.Close()
}

func TestIsChip(t *testing.T) {
	// nonexistent
	err := gpiod.IsChip("/dev/nosuch")
	assert.NotNil(t, err)

	// wrong mode
	err = gpiod.IsChip("/dev/loop0")
	assert.Equal(t, gpiod.ErrNotCharacterDevice, err)

	// no sysfs
	err = gpiod.IsChip("/dev/null")
	assert.Equal(t, gpiod.ErrNotCharacterDevice, err)

	// not sure how to test the remaining condtions...
}

func getChip(t *testing.T) *gpiod.Chip {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(t, err)
	require.NotNil(t, c)
	return c
}

type chip struct {
	name    string
	label   string
	devpath string
	lines   int
	intr    int
	ff      []int
}

func (c *chip) Name() string {
	return c.name
}

func (c *chip) Label() string {
	return c.label
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
	Label() string
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
	if err := gpiod.IsChip(path); err != nil {
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
	pi := RaspberryPi{
		chip{
			name:    "gpiochip0",
			label:   "pinctrl-bcm2835",
			devpath: path,
			lines:   int(ci.Lines),
			intr:    J8p15,
			ff:      []int{J8p11, J8p12},
		}, ch, w}
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

func (c *RaspberryPi) TriggerIntr(value int) {
	c.wline.SetValue(value)
}

func (c *RaspberryPi) Close() {
	c.wline.Close()
	// revert intr trigger line to input
	l, _ := c.c.RequestLine(J8p16)
	l.Close()
	// revert floating lines to inputs
	ll, _ := c.c.RequestLines(platform.FloatingLines())
	ll.Close()
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
	return &Mockup{
		chip{
			name:    c.Name,
			label:   c.Label,
			devpath: c.DevPath,
			lines:   20,
			intr:    10,
			ff:      []int{11, 12},
		}, m, c}, nil
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
