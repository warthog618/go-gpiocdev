// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux
// +build linux

package mockup_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/gpiod/mockup"
	"golang.org/x/sys/unix"
)

func TestNew(t *testing.T) {
	mockup.New([]int{4, 8}, true)
	patterns := []struct {
		name  string
		lines []int
		named bool
	}{
		{"one", []int{3}, false},
		{"two", []int{3, 4}, false},
		{"three", []int{3, 2, 1}, false},
		{"named", []int{3}, true},
	}
	for _, p := range patterns {
		m, err := mockup.New(p.lines, p.named)
		require.Nil(t, err)
		require.NotNil(t, m)
		defer m.Close()
		tf := func(t *testing.T) {
			assert.Equal(t, len(p.lines), m.Chips())
			for i := 0; i < m.Chips(); i++ {
				c, err := m.Chip(i)
				assert.Nil(t, err)
				checkChipExists(t, c)
			}
		}
		t.Run(p.name, tf)
	}
	_, err := mockup.New([]int{}, false)
	assert.Equal(t, unix.EINVAL, err)
}

func TestChip(t *testing.T) {
	patterns := []struct {
		name string
		cnum int
		err  error
	}{
		{"one", 0, nil},
		{"two", 1, nil},
		{"three", 2, nil},
		{"negative", -2, mockup.ErrorIndexRange{-2, 3}},
		{"oorange", 4, mockup.ErrorIndexRange{4, 3}},
	}
	m, err := mockup.New([]int{4, 8, 8}, false)
	require.Nil(t, err)
	defer m.Close()
	for _, p := range patterns {
		tf := func(t *testing.T) {
			c, err := m.Chip(p.cnum)
			assert.Equal(t, p.err, err)
			if p.err == nil {
				checkChipExists(t, c)
			}
		}
		t.Run(p.name, tf)
	}
}

func checkChipExists(t *testing.T, c *mockup.Chip) {
	require.NotNil(t, c)
	err := unix.Access(c.DevPath, unix.R_OK|unix.W_OK)
	assert.Nil(t, err)
	v, err := c.Value(0)
	assert.Nil(t, err)
	assert.Equal(t, 0, v)
}

func TestClose(t *testing.T) {
	patterns := []struct {
		name string
		ll   []int
	}{
		{"one", []int{4}},
		{"two", []int{4, 8}},
		{"three", []int{4, 8, 8}},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			m, err := mockup.New([]int{4, 8}, true)
			require.Nil(t, err)
			cc := []*mockup.Chip{}
			for i := 0; i < m.Chips(); i++ {
				c, err := m.Chip(i)
				require.Nil(t, err)
				require.NotNil(t, c)
				cc = append(cc, c)
			}
			m.Close()
			for _, c := range cc {
				// confirm chip dev removed
				_, err := os.Stat(c.DevPath)
				assert.True(t, os.IsNotExist(err))
				// confirm chip debugfs removed
				_, err = c.Value(0)
				assert.IsType(t, &os.PathError{}, err)
			}
		}
		t.Run(p.name, tf)
	}
}

func TestChipValue(t *testing.T) {
	patterns := []struct {
		name string
		line int
		err  error
	}{
		{"negative", -2, mockup.ErrorIndexRange{-2, 3}},
		{"oorange", 4, mockup.ErrorIndexRange{4, 3}},
	}
	m, err := mockup.New([]int{3}, true)
	require.Nil(t, err)
	require.NotNil(t, m)
	defer m.Close()
	c, err := m.Chip(0)
	require.Nil(t, err)
	require.NotNil(t, c)
	for _, p := range patterns {
		tf := func(t *testing.T) {
			v, err := c.Value(p.line)
			assert.Equal(t, p.err, err)
			assert.Equal(t, 0, v)
		}
		t.Run(p.name, tf)
	}
}

func TestChipSetValue(t *testing.T) {
	patterns := []struct {
		name string
		ll   []int
	}{
		{"one", []int{4}},
		{"two", []int{4, 8}},
		{"three", []int{4, 8, 8}},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			m, err := mockup.New(p.ll, true)
			require.Nil(t, err)
			defer m.Close()
			for i := 0; i < m.Chips(); i++ {
				c, err := m.Chip(i)
				require.Nil(t, err)
				require.NotNil(t, c)
				for l := 0; l < c.Lines; l++ {
					v, err := c.Value(l)
					assert.Nil(t, err)
					assert.Equal(t, 0, v)
					vv := []int{1, 0, 2, 1, 0}
					for _, v = range vv {
						err = c.SetValue(l, v)
						assert.Nil(t, err)
						rv, err := c.Value(l)
						assert.Nil(t, err)
						xv := v
						if xv > 1 {
							xv = 1
						}
						assert.Equal(t, xv, rv)
					}
				}
			}
		}
		t.Run(p.name, tf)
	}
	epatterns := []struct {
		name string
		line int
		err  error
	}{
		{"negative", -2, mockup.ErrorIndexRange{-2, 3}},
		{"oorange", 4, mockup.ErrorIndexRange{4, 3}},
	}
	m, err := mockup.New([]int{3}, true)
	require.Nil(t, err)
	require.NotNil(t, m)
	defer m.Close()
	c, err := m.Chip(0)
	require.Nil(t, err)
	require.NotNil(t, c)
	for _, p := range epatterns {
		tf := func(t *testing.T) {
			err := c.SetValue(p.line, 1)
			assert.Equal(t, p.err, err)
		}
		t.Run(p.name, tf)
	}
}
