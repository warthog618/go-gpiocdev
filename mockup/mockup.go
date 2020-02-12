// SPDX-License-Identifier: MIT
//
// Copyright © 2019 Kent Gibson <warthog618@gmail.com>.

// +build linux

// Package mockup provides GPIO mockups using the Linux gpio-mockup kernel
// module.
// This is intended for GPIO testing of gpiod, but could also be used for
// testing by users of their own code that uses gpiod.
package mockup

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pilebones/go-udev/netlink"
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

	um, err := newUdevMonitor()
	if err != nil {
		return nil, fmt.Errorf("failed to start udev monitor: %s", err)
	}
	defer um.close()

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to load gpio-mockup: %s", err)
	}
	// check debug path exists
	err = unix.Access("/sys/kernel/debug/gpio-mockup", unix.R_OK|unix.W_OK)
	if err != nil {
		return nil, err
	}
	evts := make([]netlink.UEvent, len(lines))
	for i := range evts {
		select {
		case evts[i] = <-um.queue:
		case <-time.After(100 * time.Millisecond):
			return nil, errors.New("timeout waiting for udev events")
		}
	}
	sort.Slice(evts, func(i, j int) bool {
		return evts[i].Env["DEVNAME"] < evts[j].Env["DEVNAME"]
	})
	// apply udev events to chips
	cc := make([]Chip, len(lines))
	for i, l := range lines {
		devpath := evts[i].Env["DEVNAME"]
		name := devpath[len("/dev/"):]
		var num int
		_, err = fmt.Sscanf(name, "gpiochip%d", &num)
		if err != nil {
			return nil, fmt.Errorf("failed to parse chip num: %s", err)
		}
		cc[i] = Chip{
			Name:      name,
			Label:     fmt.Sprintf("gpio-mockup-%c", 'A'+i),
			Lines:     l,
			DevPath:   devpath,
			DbgfsPath: fmt.Sprintf("/sys/kernel/debug/gpio-mockup/gpiochip%d/", num)}
	}
	m := Mockup{cc: cc}
	return &m, nil
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
	return CheckKernelVersion(version{5, 1, 0})
}

// KernelVersion returns the running kernel version.
func KernelVersion() ([]byte, error) {
	cmd := exec.Command("uname", "-r")
	release, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	r := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	vers := r.FindStringSubmatch(string(release))
	if len(vers) != 4 {
		return nil, fmt.Errorf("can't parse uname: %s", release)
	}
	v := []byte{0, 0, 0}
	for i, vf := range vers[1:] {
		vfi, err := strconv.ParseUint(vf, 10, 64)
		if err != nil {
			return nil, err
		}
		v[i] = byte(vfi)
	}
	return v, nil
}

// CheckKernelVersion returns an error if the kernel verion is less than the
// min.
func CheckKernelVersion(min version) error {
	kv, err := KernelVersion()
	if err != nil {
		return err
	}
	n := bytes.Compare(kv, min[:])
	if n < 1 {
		return ErrorBadVersion{Need: min, Have: kv}
	}
	return nil
}

// 3 part version, Major, Minor, Patch.
type version []byte

func (v version) String() string {
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
	Need version
	Have version
}

func (e ErrorBadVersion) Error() string {
	return fmt.Sprintf("require kernel %s or later, but running %s", e.Need, e.Have)
}
