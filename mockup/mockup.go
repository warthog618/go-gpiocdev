// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

// Package mockup provides GPIO mockups using the Linux gpio-mockup kernel
// module.
// This is intended for GPIO testing of gpiod, but could also be used for
// testing by users of their own code that uses gpiod.
package mockup

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"

	"github.com/warthog618/gpiod"
	"golang.org/x/sys/unix"
)

// Mockup represents a number of GPIO chips being mocked.
type Mockup struct {
	mu sync.Mutex
	cc []Chip
}

// Chip represents a single mocked GPIO chip.
type Chip struct {
	Name      string
	Label     string
	Lines     int
	DevPath   string
	DbgfsPath string
}

// New creates a new Mockup.
// A number of GPIO chips can be mocked, with the number of lines on each
// specified in lines. e.g. []int{4,6} would create two chips, the first with 4
// lines and the second with 6.
// Requires the gpio-mockup kernel module and Linux 5.1.0 or later. Note
// that only one Mockup can be present on a system at any time and that this
// function unloads the gpio-mockup module if it is already loaded.
func New(lines []int, namedLines bool) (*Mockup, error) {
	if len(lines) == 0 {
		return nil, unix.EINVAL
	}
	err := IsSupported()
	if err != nil {
		return nil, err
	}
	args := []string{"gpio-mockup"}
	// remove any existing mockup setup
	cmd := exec.Command("rmmod", args...)
	cmd.Run()

	if namedLines {
		args = append(args, "gpio_mockup_named_lines")
	}
	rangesArg := "gpio_mockup_ranges="
	for _, l := range lines {
		rangesArg += fmt.Sprintf("-1,%d,", l)
	}
	rangesArg = rangesArg[:len(rangesArg)-1]
	args = append(args, rangesArg)
	cmd = exec.Command("modprobe", args...)

	mm, err := newModprobeMonitor()
	if err != nil {
		return nil, fmt.Errorf("failed to start modprobe monitor: %s", err)
	}
	defer mm.Close()

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to load gpio-mockup: %s", err)
	}
	// check debug path exists
	err = unix.Access("/sys/kernel/debug/gpio-mockup", unix.R_OK|unix.W_OK)
	if err != nil {
		return nil, fmt.Errorf("can't access /sys/kernel/debug/gpio-mockup: %s", err)
	}
	cc, err := mm.Chips(lines)
	if err != nil {
		return nil, err
	}
	m := Mockup{cc: cc}
	return &m, nil
}

// ModprobeMonitor finds the details of gpio-mockup based gpiochips loaded via
// modprobe.
type ModprobeMonitor interface {
	Chips([]int) ([]Chip, error)
	Close()
}

func newModprobeMonitor() (ModprobeMonitor, error) {
	if len(gpiod.Chips()) != 0 {
		// need udev monitor to determine chip details...
		return newUdevMonitor()
	}
	// all gpiochips are gpio-mockups
	return &SimpleMonitor{}, nil
}

// SimpleMonitor assumes an empty platform so any added gpiochips will be the
// first and only gpiochips.
type SimpleMonitor struct{}

// Chips returns the chips corresponding to the requested number of lines per
// chip.
func (m *SimpleMonitor) Chips(lines []int) ([]Chip, error) {
	// make chips from lines
	cc := make([]Chip, len(lines))
	for i, l := range lines {
		devpath := fmt.Sprintf("/dev/gpiochip%d", i)
		name := devpath[len("/dev/"):]
		cc[i] = Chip{
			Name:      name,
			Label:     fmt.Sprintf("gpio-mockup-%c", 'A'+i),
			Lines:     l,
			DevPath:   devpath,
			DbgfsPath: fmt.Sprintf("/sys/kernel/debug/gpio-mockup/gpiochip%d/", i)}
	}
	return cc, nil
}

// Close is just a stub to fulfil the ModprobeMonitor interface.
func (m *SimpleMonitor) Close() {
}

// Chip returns the mocked chip indicated by num.
func (m *Mockup) Chip(num int) (*Chip, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if num < 0 || num >= len(m.cc) {
		return nil, ErrorIndexRange{num, len(m.cc)}
	}
	return &m.cc[num], nil
}

// Chips returns the number of chips mocked.
func (m *Mockup) Chips() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.cc)
}

// Close releases all resources held by the Mockup and unloads the gpio-mockup
// module.
func (m *Mockup) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cc = []Chip{}
	cmd := exec.Command("rmmod", "gpio-mockup")
	return cmd.Run()
}

// Value returns the value of the line.
func (c *Chip) Value(line int) (int, error) {
	if line < 0 || line >= c.Lines {
		return 0, ErrorIndexRange{line, c.Lines}
	}
	path := fmt.Sprintf("%s%d", c.DbgfsPath, line)
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	v := []byte{0}
	_, err = f.Read(v)
	if err != nil {
		return 0, err
	}
	if v[0] == '1' {
		return 1, nil
	}
	return 0, nil
}

// SetValue sets the pull value of the line.
func (c *Chip) SetValue(line int, value int) error {
	if line < 0 || line >= c.Lines {
		return ErrorIndexRange{line, c.Lines}
	}
	path := fmt.Sprintf("%s%d", c.DbgfsPath, line)
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	v := []byte{'0'}
	if value != 0 {
		v[0] = '1'
	}
	_, err = f.Write(v)
	return err
}

// IsSupported returns an error if this package cannot run on this platform.
func IsSupported() error {
	return CheckKernelVersion(Semver{5, 1})
}

// KernelVersion returns the running kernel version.
func KernelVersion() (semver []byte, err error) {
	uname := unix.Utsname{}
	err = unix.Uname(&uname)
	if err != nil {
		return
	}
	release := string(uname.Release[:])
	r := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	vers := r.FindStringSubmatch(release)
	if len(vers) != 4 {
		err = fmt.Errorf("can't parse uname: %s", release)
		return
	}
	v := []byte{0, 0, 0}
	for i, vf := range vers[1:] {
		var vfi uint64
		vfi, err = strconv.ParseUint(vf, 10, 64)
		if err != nil {
			return
		}
		v[i] = byte(vfi)
	}
	semver = v
	return
}

// CheckKernelVersion returns an error if the kernel version is less than the
// min.
func CheckKernelVersion(min Semver) error {
	kv, err := KernelVersion()
	if err != nil {
		return err
	}
	n := bytes.Compare(kv, min[:])
	if n < 0 {
		return ErrorBadVersion{Need: min, Have: kv}
	}
	return nil
}

// Semver is 3 part version, Major, Minor, Patch.
type Semver []byte

func (v Semver) String() string {
	if len(v) == 0 {
		return ""
	}
	vstr := fmt.Sprintf("%d", v[0])
	for i := 1; i < len(v); i++ {
		vstr += fmt.Sprintf(".%d", v[i])
	}
	return vstr
}

// ErrorIndexRange indicates the requested index is beyond the limit of the array.
type ErrorIndexRange struct {
	Req   int
	Limit int
}

func (e ErrorIndexRange) Error() string {
	return fmt.Sprintf("index out of range - got %d, limit is %d.", e.Req, e.Limit)
}

// ErrorBadVersion indicates the kernel version is insufficient.
type ErrorBadVersion struct {
	Need Semver
	Have Semver
}

func (e ErrorBadVersion) Error() string {
	return fmt.Sprintf("require kernel %s or later, but running %s", e.Need, e.Have)
}
