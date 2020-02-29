// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

package gpiod_test

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod"
	"github.com/warthog618/gpiod/device/rpi"
	"github.com/warthog618/gpiod/mockup"
	"github.com/warthog618/gpiod/uapi"
	"golang.org/x/sys/unix"
)

var platform Platform

func TestMain(m *testing.M) {
	var pname string
	flag.StringVar(&pname, "platform", "mockup", "test platform")
	flag.Parse()
	p, err := newPlatform(pname)
	if err != nil {
		fmt.Println("Platform not supported -", err)
		os.Exit(-1)
	}
	platform = p
	rc := m.Run()
	platform.Close()
	os.Exit(rc)
}

func TestNewChip(t *testing.T) {
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

	// name
	c, err = gpiod.NewChip(platform.Name())
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

func TestChips(t *testing.T) {
	cc := gpiod.Chips()
	require.GreaterOrEqual(t, len(cc), 1)
	assert.Contains(t, cc, platform.Name())
}

func TestFindLine(t *testing.T) {
	cname, n, err := gpiod.FindLine(platform.IntrName())
	assert.Nil(t, err)
	intr := platform.IntrLine()
	// hacky workaround for unnamed lines on RPi
	if len(platform.IntrName()) == 0 {
		intr = 0
	}
	assert.Equal(t, intr, n)
	assert.Equal(t, platform.Name(), cname)

	cname, n, err = gpiod.FindLine("nonexistent")
	assert.Equal(t, gpiod.ErrLineNotFound, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, 0, len(cname))
}

func TestChipClose(t *testing.T) {
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

func TestChipFindLine(t *testing.T) {
	c := getChip(t)
	n, err := c.FindLine(platform.IntrName())
	assert.Nil(t, err)
	intr := platform.IntrLine()
	// hacky workaround for unnamed lines on RPi
	if len(platform.IntrName()) == 0 {
		intr = 0
	}
	assert.Equal(t, intr, n)

	n, err = c.FindLine("nonexistent")
	assert.Equal(t, gpiod.ErrLineNotFound, err)
	assert.Equal(t, 0, n)
}

func TestChipFindLines(t *testing.T) {
	c := getChip(t)
	nn, err := c.FindLines(platform.IntrName(), platform.IntrName())
	assert.Nil(t, err)
	intr := platform.IntrLine()
	// hacky workaround for unnamed lines on RPi
	if len(platform.IntrName()) == 0 {
		intr = 0
	}
	assert.Equal(t, []int{intr, intr}, nn)

	nn, err = c.FindLines(platform.IntrName(), "nonexistent")
	assert.Equal(t, gpiod.ErrLineNotFound, err)
	assert.Equal(t, []int(nil), nn)
}

func TestChipLineInfo(t *testing.T) {
	c := getChip(t)
	xli := gpiod.LineInfo{}

	// out of range
	li, err := c.LineInfo(platform.Lines())
	assert.Equal(t, gpiod.ErrInvalidOffset, err)
	assert.Equal(t, xli, li)

	// valid
	li, err = c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	xli.Offset = platform.IntrLine()
	xli.Name = platform.IntrName()
	assert.Equal(t, xli, li)

	// closed
	c.Close()
	li, err = c.LineInfo(1)
	assert.NotNil(t, err)
	xli = gpiod.LineInfo{}
	assert.Equal(t, xli, li)
}

func TestChipLines(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	lines := c.Lines()
	assert.Equal(t, platform.Lines(), lines)
}

func TestChipRequestLine(t *testing.T) {
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

func TestLineChip(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	cname := l.Chip()
	assert.Equal(t, c.Name, cname)
}

func TestLineClose(t *testing.T) {
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

func TestLineInfo(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	cli, err := c.LineInfo(platform.IntrLine())
	assert.Nil(t, err)
	li, err := l.Info()
	assert.Nil(t, err)
	require.NotNil(t, li)
	assert.Equal(t, cli, li)
}

func TestLineOffset(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLine(platform.IntrLine())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lo := l.Offset()
	assert.Equal(t, platform.IntrLine(), lo)
}

func TestLineValue(t *testing.T) {
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

func TestLinesChip(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lc := l.Chip()
	assert.Equal(t, c.Name, lc)
}

func TestLinesClose(t *testing.T) {
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

func TestLinesInfo(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	li, err := l.Info()
	assert.Nil(t, err)
	for i, o := range platform.FloatingLines() {
		cli, err := c.LineInfo(o)
		assert.Nil(t, err)
		assert.NotNil(t, li[i])
		if li[0] != nil {
			assert.Equal(t, cli, *li[i])
		}
	}
}

func TestLineOffsets(t *testing.T) {
	c := getChip(t)
	defer c.Close()
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	defer l.Close()
	lo := l.Offsets()
	assert.Equal(t, platform.FloatingLines(), lo)
}

func TestLinesValues(t *testing.T) {
	c := getChip(t)
	defer c.Close()

	// input
	platform.TriggerIntr(0)
	lines := append([]int{platform.IntrLine()}, platform.FloatingLines()...)
	l, err := c.RequestLines(lines)
	assert.Nil(t, err)
	require.NotNil(t, l)
	vv := make([]int, len(lines))
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, 0, vv[0])
	platform.TriggerIntr(1)
	err = l.Values(vv)
	assert.Nil(t, err)
	assert.Equal(t, 1, vv[0])

	l.Close()

	// after close
	err = l.Values(vv)
	assert.NotNil(t, err)

	// output
	lines = platform.FloatingLines()
	l, err = c.RequestLines(lines, gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.Values(vv)
	assert.Nil(t, err)
	// actual values are indeterminate

	l.Close()
}

func TestLinesSetValues(t *testing.T) {
	c := getChip(t)
	defer c.Close()

	// input
	l, err := c.RequestLines(platform.FloatingLines())
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValues([]int{0, 1})
	assert.Equal(t, gpiod.ErrPermissionDenied, err)
	l.Close()

	// output
	l, err = c.RequestLines(platform.FloatingLines(),
		gpiod.AsOutput(0))
	assert.Nil(t, err)
	require.NotNil(t, l)
	err = l.SetValues([]int{1, 0})
	assert.Nil(t, err)

	// too many values
	err = l.SetValues([]int{1, 1, 1})
	assert.Equal(t, gpiod.ErrInvalidOffset, err)

	l.Close()
}

func TestIsChip(t *testing.T) {
	// nonexistent
	err := gpiod.IsChip("/dev/nonexistent")
	assert.NotNil(t, err)

	// wrong mode
	err = gpiod.IsChip("/dev/loop0")
	assert.Equal(t, gpiod.ErrNotCharacterDevice, err)

	// no sysfs
	err = gpiod.IsChip("/dev/null")
	assert.Equal(t, gpiod.ErrNotCharacterDevice, err)

	// not sure how to test the remaining conditions...
}

func getChip(t *testing.T) *gpiod.Chip {
	c, err := gpiod.NewChip(platform.Devpath())
	require.Nil(t, err)
	require.NotNil(t, c)
	return c
}

type gpiochip struct {
	name    string
	label   string
	devpath string
	lines   int
	// line triggered by TriggerIntr.
	intro     int
	introName string
	outo      int
	// floating lines - can be harmlessly set to outputs.
	ff []int
}

func (c *gpiochip) Name() string {
	return c.name
}

func (c *gpiochip) Label() string {
	return c.label
}
func (c *gpiochip) Devpath() string {
	return c.devpath
}

func (c *gpiochip) Lines() int {
	return c.lines
}

func (c *gpiochip) IntrLine() int {
	return c.intro
}

func (c *gpiochip) IntrName() string {
	return c.introName
}

func (c *gpiochip) OutLine() int {
	return c.outo
}

func (c *gpiochip) FloatingLines() []int {
	return c.ff
}

// two flavours of chip, raspberry and mockup.
type Platform interface {
	Name() string
	Label() string
	Devpath() string
	Lines() int
	IntrLine() int
	IntrName() string
	OutLine() int
	FloatingLines() []int
	TriggerIntr(int)
	ReadOut() int
	SupportsAsIs() bool
	Close()
}

type RaspberryPi struct {
	gpiochip
	chip  *gpiod.Chip
	wline *gpiod.Line
}

func isPi(path string) error {
	if err := gpiod.IsChip(path); err != nil {
		return err
	}
	f, err := os.OpenFile(path, unix.O_CLOEXEC, unix.O_RDONLY)
	if err != nil {
		return err
	}
	defer f.Close()
	ci, err := uapi.GetChipInfo(f.Fd())
	if err != nil {
		return err
	}
	label := uapi.BytesToString(ci.Label[:])
	if label != "pinctrl-bcm2835" {
		return fmt.Errorf("unsupported gpiochip - %s", label)
	}
	return nil
}

func newPi(path string) (*RaspberryPi, error) {
	if err := isPi(path); err != nil {
		return nil, err
	}
	ch, err := gpiod.NewChip(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			ch.Close()
		}
	}()
	pi := RaspberryPi{
		gpiochip: gpiochip{
			name:      "gpiochip0",
			label:     "pinctrl-bcm2835",
			devpath:   path,
			lines:     int(ch.Lines()),
			intro:     rpi.J8p15,
			introName: "",
			outo:      rpi.J8p16,
			ff:        []int{rpi.J8p11, rpi.J8p12},
		},
		chip: ch,
	}
	// check J8p15 and J8p16 are tied
	w, err := ch.RequestLine(pi.outo, gpiod.AsOutput(1),
		gpiod.WithConsumer("gpiod-test-w"))
	if err != nil {
		return nil, err
	}
	defer w.Close()
	r, err := ch.RequestLine(pi.intro,
		gpiod.WithConsumer("gpiod-test-r"))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	v, _ := r.Value()
	if v != 1 {
		return nil, errors.New("J8p15 and J8p16 must be tied")
	}
	w.SetValue(0)
	v, _ = r.Value()
	if v != 0 {
		return nil, errors.New("J8p15 and J8p16 must be tied")
	}
	return &pi, nil
}

func (c *RaspberryPi) Close() {
	if c.wline != nil {
		c.wline.Close()
		c.wline = nil
	}
	// revert intr trigger line to input
	l, _ := c.chip.RequestLine(c.outo)
	l.Close()
	// revert floating lines to inputs
	ll, _ := c.chip.RequestLines(platform.FloatingLines())
	ll.Close()
	c.chip.Close()
}

func (c *RaspberryPi) OutLine() int {
	if c.wline != nil {
		c.wline.Close()
		c.wline = nil
	}
	return c.outo
}

func (c *RaspberryPi) ReadOut() int {
	r, err := c.chip.RequestLine(c.intro,
		gpiod.WithConsumer("gpiod-test-r"))
	if err != nil {
		return -1
	}
	defer r.Close()
	v, err := r.Value()
	if err != nil {
		return -1
	}
	return v
}

func (c *RaspberryPi) SupportsAsIs() bool {
	return false
}

func (c *RaspberryPi) TriggerIntr(value int) {
	if c.wline != nil {
		c.wline.SetValue(value)
		return
	}
	w, _ := c.chip.RequestLine(c.outo, gpiod.AsOutput(value),
		gpiod.WithConsumer("gpiod-test-w"))
	c.wline = w
}

type Mockup struct {
	gpiochip
	m *mockup.Mockup
	c *mockup.Chip
}

func newMockup() (*Mockup, error) {
	m, err := mockup.New([]int{20}, true)
	if err != nil {
		return nil, err
	}
	c, err := m.Chip(0)
	if err != nil {
		return nil, err
	}
	return &Mockup{
		gpiochip{
			name:      c.Name,
			label:     c.Label,
			devpath:   c.DevPath,
			lines:     20,
			intro:     10,
			introName: "gpio-mockup-A-10",
			outo:      9,
			ff:        []int{11, 12},
		}, m, c}, nil
}

func (c *Mockup) Close() {
	c.m.Close()
}

func (c *Mockup) ReadOut() int {
	v, err := c.c.Value(c.outo)
	if err != nil {
		return -1
	}
	return v
}

func (c *Mockup) SupportsAsIs() bool {
	return true
}

func (c *Mockup) TriggerIntr(value int) {
	c.c.SetValue(c.intro, value)
}

func newPlatform(pname string) (Platform, error) {
	switch pname {
	case "mockup":
		p, err := newMockup()
		if err != nil {
			return nil, fmt.Errorf("error loading gpio-mockup: %w", err)
		}
		return p, nil
	case "rpi":
		return newPi("/dev/gpiochip0")
	default:
		return nil, fmt.Errorf("unknown platform '%s'", pname)
	}
}
