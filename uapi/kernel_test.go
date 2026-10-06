// SPDX-FileCopyrightText: 2019 Kent Gibson <warthog618@gmail.com>
//
// SPDX-License-Identifier: MIT

//go:build linux

package uapi_test

import (
	"fmt"
	"os"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiosim"
	"golang.org/x/sys/unix"

	"github.com/warthog618/go-gpiocdev/uapi"
)

func TestRepeatedGetLine(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()

	lr := uapi.LineRequest{
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput,
		},
		Lines:   2,
		Offsets: [uapi.LinesMax]uint32{1, 3},
	}
	copy(lr.Consumer[:31], "test-repeated-get-line")

	// input
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	// busy
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Equal(t, unix.EBUSY, err)

	// output
	lr.Config.Flags = uapi.LineFlagOutput
	err = uapi.GetLine(f.Fd(), &lr)
	assert.Equal(t, unix.EBUSY, err)

	unix.Close(int(lr.Fd))
}

func TestWatchIsolation(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f1, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f1.Close()

	f2, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f2.Close()

	offset := uint32(3)
	// set watch
	li := uapi.LineInfo{Offset: offset}
	err = uapi.WatchLineInfo(f1.Fd(), &li)
	require.Nil(t, err)
	xli := uapi.LineInfo{Offset: offset, Flags: uapi.LineFlagInput}
	assert.Equal(t, xli, li)

	chg, err := readLineInfoChangedTimeout(f1.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f1")

	chg, err = readLineInfoChangedTimeout(f2.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f2")

	// request line
	lr := uapi.LineRequest{
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput,
		},
		Lines: 1,
	}
	lr.Offsets[0] = offset
	copy(lr.Consumer[:], "test-watch-isolation")
	err = uapi.GetLine(f2.Fd(), &lr)
	assert.Nil(t, err)
	chg, err = readLineInfoChangedTimeout(f1.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedRequested, chg.Type)
	xli.Flags |= uapi.LineFlagUsed
	copy(xli.Consumer[:], "test-watch-isolation")
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f2.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f2")

	err = uapi.WatchLineInfo(f2.Fd(), &li)
	require.Nil(t, err)
	err = uapi.UnwatchLineInfo(f1.Fd(), li.Offset)
	require.Nil(t, err)
	unix.Close(int(lr.Fd))

	unix.Close(int(lr.Fd))
	chg, err = readLineInfoChangedTimeout(f2.Fd(), eventWaitTimeout)
	assert.Nil(t, err)
	require.NotNil(t, chg)
	assert.Equal(t, uapi.LineChangedReleased, chg.Type)
	xli = uapi.LineInfo{Offset: offset, Flags: uapi.LineFlagInput}
	assert.Equal(t, xli, chg.Info)

	chg, err = readLineInfoChangedTimeout(f1.Fd(), spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, chg, "spurious change on f1")
}

func TestBulkEventRead(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	err = s.SetPull(offset, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
		},
	}
	lr.Offsets[0] = uint32(offset)
	copy(lr.Consumer[:31], "test-bulk-event-read-")
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	s.SetPull(offset, 1)
	time.Sleep(clkTick)
	s.SetPull(offset, 0)
	time.Sleep(clkTick)
	s.SetPull(offset, 1)
	time.Sleep(clkTick)
	s.SetPull(offset, 0)
	time.Sleep(clkTick)

	var ed uapi.LineEvent
	b := make([]byte, unsafe.Sizeof(ed)*3)
	n, err := unix.Read(int(lr.Fd), b[:])
	assert.Nil(t, err)
	assert.Equal(t, len(b), n)

	unix.Close(int(lr.Fd))
}

func TestWatchLineInfoRequested(t *testing.T) {
	patterns := []struct {
		name   string
		flags  uapi.LineFlag
		period int
	}{
		{"input", uapi.LineFlagInput, 0},
		{"active_low", uapi.LineFlagInput, 0},
		{"debounced", uapi.LineFlagInput, 20000},
		{"output", uapi.LineFlagOutput, 0},
		{"open_drain", uapi.LineFlagOutput | uapi.LineFlagOpenDrain, 0},
		{"open_source", uapi.LineFlagOutput | uapi.LineFlagOpenSource, 0},
		{"rising", uapi.LineFlagInput | uapi.LineFlagEdgeRising, 0},
		{"falling", uapi.LineFlagInput | uapi.LineFlagEdgeFalling, 0},
		{"both_edges", uapi.LineFlagInput | uapi.LineFlagEdgeBoth, 0},
		{"rising_debounced", uapi.LineFlagInput | uapi.LineFlagEdgeRising, 13000},
		{"falling_debounced", uapi.LineFlagInput | uapi.LineFlagEdgeFalling, 15000},
		{"both_edges_debounced", uapi.LineFlagInput | uapi.LineFlagEdgeBoth, 17000},
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			s, err := gpiosim.NewSim(
				gpiosim.WithName("gpiosim_test"),
				gpiosim.WithBank(gpiosim.NewBank("left", 8,
					gpiosim.WithNamedLine(3, "LED0"),
				)),
			)
			require.Nil(t, err)
			defer s.Close()

			c := s.Chips[0]
			f, err := os.Open(c.DevPath())
			require.Nil(t, err)
			defer f.Close()

			offset := uint32(3)

			// set watch
			li := uapi.LineInfo{Offset: offset}
			err = uapi.WatchLineInfo(f.Fd(), &li)
			require.Nil(t, err)
			xli := uapi.LineInfo{
				Offset: offset,
				Flags:  uapi.LineFlagInput,
			}
			copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
			assert.Equal(t, xli, li)

			// request line
			lr := uapi.LineRequest{
				Lines: 1,
				Config: uapi.LineConfig{
					Flags: p.flags,
				},
			}
			lr.Offsets[0] = offset
			copy(lr.Consumer[:], "testwatchrequested")
			if p.period > 0 {
				lr.Config.NumAttrs = 1
				lr.Config.Attrs[0].Mask = 1
				lr.Config.Attrs[0].Attr = uapi.DebouncePeriod(p.period).Encode()
			}
			err = uapi.GetLine(f.Fd(), &lr)
			assert.Nil(t, err)
			chg, err := readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
			assert.Nil(t, err)
			require.NotNil(t, chg)
			assert.Equal(t, uapi.LineChangedRequested, chg.Type)
			xli.Flags = lr.Config.Flags | uapi.LineFlagUsed
			copy(xli.Consumer[:], "testwatchrequested")
			if p.period > 0 {
				xli.NumAttrs = 1
				xli.Attrs[0] = uapi.DebouncePeriod(p.period).Encode()
			}
			assert.Equal(t, xli, chg.Info)

			chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, chg, "spurious change")

			// release line
			unix.Close(int(lr.Fd))
			chg, err = readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
			assert.Nil(t, err)
			require.NotNil(t, chg)
			assert.Equal(t, uapi.LineChangedReleased, chg.Type)
			xli = uapi.LineInfo{
				Offset: 3,
				Flags:  p.flags & (uapi.LineFlagInput | uapi.LineFlagOutput),
			}
			copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
			assert.Equal(t, xli, chg.Info)

			chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, chg, "spurious change")
		}
		t.Run(fmt.Sprintf("%s", p.name), tf)
	}
}

func TestWatchLineInfoConfig(t *testing.T) {
	patterns := []struct {
		name   string
		flags  uapi.LineFlag
		period int
	}{
		{"input", uapi.LineFlagInput, 0},
		{"active_low", uapi.LineFlagInput, 0},
		{"debounced", uapi.LineFlagInput, 20000},
		{"output", uapi.LineFlagOutput, 0},
		{"open_drain", uapi.LineFlagOutput | uapi.LineFlagOpenDrain, 0},
		{"open_source", uapi.LineFlagOutput | uapi.LineFlagOpenSource, 0},
		{"rising", uapi.LineFlagInput | uapi.LineFlagEdgeRising, 0},
		{"falling", uapi.LineFlagInput | uapi.LineFlagEdgeFalling, 0},
		{"both_edges", uapi.LineFlagInput | uapi.LineFlagEdgeBoth, 0},
		{"rising_debounced", uapi.LineFlagInput | uapi.LineFlagEdgeRising, 13000},
		{"falling_debounced", uapi.LineFlagInput | uapi.LineFlagEdgeFalling, 15000},
		{"both_edges_debounced", uapi.LineFlagInput | uapi.LineFlagEdgeBoth, 17000},
	}

	for _, p := range patterns {
		tf := func(t *testing.T) {
			s, err := gpiosim.NewSim(
				gpiosim.WithName("gpiosim_test"),
				gpiosim.WithBank(gpiosim.NewBank("left", 8,
					gpiosim.WithNamedLine(3, "LED0"),
				)),
			)
			require.Nil(t, err)
			defer s.Close()

			c := s.Chips[0]
			f, err := os.Open(c.DevPath())
			require.Nil(t, err)
			defer f.Close()

			offset := uint32(3)

			// set watch
			li := uapi.LineInfo{Offset: offset}
			err = uapi.WatchLineInfo(f.Fd(), &li)
			require.Nil(t, err)
			xli := uapi.LineInfo{
				Offset: offset,
				Flags:  uapi.LineFlagInput,
			}
			copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
			assert.Equal(t, xli, li)

			// request line
			lr := uapi.LineRequest{
				Lines: 1,
			}
			lr.Offsets[0] = offset
			copy(lr.Consumer[:], "testwatchconfig")
			err = uapi.GetLine(f.Fd(), &lr)
			assert.Nil(t, err)
			chg, err := readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
			assert.Nil(t, err)
			require.NotNil(t, chg)
			assert.Equal(t, uapi.LineChangedRequested, chg.Type)
			xli.Flags = uapi.LineFlagUsed | uapi.LineFlagInput
			copy(xli.Consumer[:], "testwatchconfig")
			assert.Equal(t, xli, chg.Info)

			chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, chg, "spurious change")

			// reconfig line
			lc := uapi.LineConfig{Flags: p.flags}
			if p.period > 0 {
				lc.NumAttrs = 1
				lc.Attrs[0].Mask = 1
				lc.Attrs[0].Attr = uapi.DebouncePeriod(p.period).Encode()
			}
			err = uapi.SetLineConfig(uintptr(lr.Fd), &lc)
			assert.Nil(t, err)
			chg, err = readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
			assert.Nil(t, err)
			require.NotNil(t, chg)
			assert.Equal(t, uapi.LineChangedConfig, chg.Type)
			xli.Flags = p.flags | uapi.LineFlagUsed
			if p.period > 0 {
				xli.NumAttrs = 1
				xli.Attrs[0] = uapi.DebouncePeriod(p.period).Encode()
			}
			assert.Equal(t, xli, chg.Info)

			chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, chg, "spurious change")

			// release line
			unix.Close(int(lr.Fd))
			chg, err = readLineInfoChangedTimeout(f.Fd(), eventWaitTimeout)
			assert.Nil(t, err)
			require.NotNil(t, chg)
			assert.Equal(t, uapi.LineChangedReleased, chg.Type)
			xli = uapi.LineInfo{
				Offset: 3,
				Flags:  p.flags & (uapi.LineFlagInput | uapi.LineFlagOutput),
			}
			copy(xli.Name[:], []byte(c.Config().Names[int(offset)]))
			assert.Equal(t, xli, chg.Info)

			chg, err = readLineInfoChangedTimeout(f.Fd(), spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, chg, "spurious change")
		}
		t.Run(fmt.Sprintf("%s", p.name), tf)
	}
}

func TestSetConfigEdgeDetection(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()

	patterns := []struct {
		name  string
		flags uapi.LineFlag
	}{
		{"input", uapi.LineFlagInput},
		{"output", uapi.LineFlagOutput},
		{"rising", uapi.LineFlagInput | uapi.LineFlagEdgeRising},
		{"falling", uapi.LineFlagInput | uapi.LineFlagEdgeFalling},
		{"both", uapi.LineFlagInput | uapi.LineFlagEdgeBoth},
	}

	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := uint32(1)
	err = s.SetPull(int(offset), 0)
	require.Nil(t, err)
	for _, p1 := range patterns {
		for _, p2 := range patterns {
			tf := func(t *testing.T) {
				lr := uapi.LineRequest{
					Lines: 1,
					Config: uapi.LineConfig{
						Flags: p1.flags,
					},
				}
				lr.Offsets[0] = offset
				copy(lr.Consumer[:31], "test-set-config-edge-detection")
				err = uapi.GetLine(f.Fd(), &lr)
				require.Nil(t, err)
				defer unix.Close(int(lr.Fd))

				xevt := uapi.LineEvent{
					Offset:    offset,
					LineSeqno: 1,
					Seqno:     1,
				}
				testLineFlags(t, f.Fd(), offset, p1.flags)
				testEdgeDetectionEvents(t, s, lr.Fd, &xevt, p1.flags)

				config := uapi.LineConfig{
					Flags: p2.flags,
				}
				err = uapi.SetLineConfig(uintptr(lr.Fd), &config)
				require.Nil(t, err)
				testLineFlags(t, f.Fd(), offset, p2.flags)
				testEdgeDetectionEvents(t, s, lr.Fd, &xevt, p2.flags)

				config.Flags = p1.flags
				err = uapi.SetLineConfig(uintptr(lr.Fd), &config)
				require.Nil(t, err)
				testEdgeDetectionEvents(t, s, lr.Fd, &xevt, p1.flags)
			}
			t.Run(fmt.Sprintf("%s-to-%s", p1.name, p2.name), tf)
		}
	}
}

func testLineFlags(t *testing.T, fd uintptr, offset uint32, flags uapi.LineFlag) {
	li, err := uapi.GetLineInfo(fd, int(offset))
	assert.Nil(t, err)
	assert.Equal(t, flags|uapi.LineFlagUsed, li.Flags)
}

func testEdgeDetectionEvents(t *testing.T, s *gpiosim.Simpleton, fd int32, xevt *uapi.LineEvent, flags uapi.LineFlag) {
	offset := int(xevt.Offset)
	for i := 0; i < 2; i++ {
		s.SetPull(offset, 1)
		if flags&uapi.LineFlagEdgeRising == 0 {
			evt, err := readLineEventTimeout(fd, spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, evt, "spurious event")
		} else {
			xevt.ID = uapi.LineEventRisingEdge
			evt, err := readLineEventTimeout(fd, eventWaitTimeout)
			require.Nil(t, err)
			require.NotNil(t, evt, flags)
			evt.Timestamp = 0
			assert.Equal(t, *xevt, *evt)
			xevt.LineSeqno++
			xevt.Seqno++
		}

		s.SetPull(offset, 0)
		if flags&uapi.LineFlagEdgeFalling == 0 {
			evt, err := readLineEventTimeout(fd, spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, evt, "spurious event")
		} else {
			xevt.ID = uapi.LineEventFallingEdge
			evt, err := readLineEventTimeout(fd, eventWaitTimeout)
			require.Nil(t, err)
			require.NotNil(t, evt)
			evt.Timestamp = 0
			assert.Equal(t, *xevt, *evt)
			xevt.LineSeqno++
			xevt.Seqno++
		}
	}
}

func TestEventBufferOverflow(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	patterns := []struct {
		name string
		// The requested buffer size.
		size uint32
		// The actual buffer size allocated by the kernel.
		// This is not guaranteed stable.
		ksize int
	}{
		{"default",
			0,
			16,
		},
		{"smaller", // compared to default
			5,
			8,
		},
		{"larger",
			35,
			64,
		},
	}
	for _, p := range patterns {
		tf := func(t *testing.T) {
			err = s.SetPull(offset, 1)
			require.Nil(t, err)
			lr := uapi.LineRequest{
				Lines: 1,
				Config: uapi.LineConfig{
					Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
				},
				EventBufferSize: p.size,
			}
			lr.Offsets[0] = uint32(offset)
			copy(lr.Consumer[:31], "test-event-buffer-overflow-")
			err = uapi.GetLine(f.Fd(), &lr)
			require.Nil(t, err)
			defer unix.Close(int(lr.Fd))

			for i := 0; i < p.ksize+4; i++ {
				err = s.SetPull(1, i&1)
				require.Nil(t, err)
				time.Sleep(clkTick)
			}
			// first 4 events should be discarded by the kernel
			xevt := uapi.LineEvent{
				Offset:    uint32(offset),
				LineSeqno: 5,
				Seqno:     5,
			}
			for i := 0; i < p.ksize; i++ {
				evt, err := readLineEventTimeout(lr.Fd, eventWaitTimeout)
				require.Nil(t, err)
				require.NotNil(t, evt)
				evt.Timestamp = 0
				if i&1 == 0 {
					xevt.ID = uapi.LineEventFallingEdge
				} else {
					xevt.ID = uapi.LineEventRisingEdge
				}
				assert.Equal(t, xevt, *evt)
				xevt.LineSeqno++
				xevt.Seqno++
			}
			evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
			assert.Nil(t, err)
			assert.Nil(t, evt, "spurious event")
		}
		t.Run(p.name, tf)
	}
}

func TestSetConfigDebouncedEdges(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	err = s.SetPull(offset, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
		},
	}
	lr.Offsets[0] = uint32(offset)
	copy(lr.Consumer[:31], "test-set-config-debounced-edges")
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	periods := []int{-1, 1000, 0, 2000}
	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
		Offset:    uint32(offset),
	}

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	for _, period := range periods {
		if period >= 0 {
			config := uapi.LineConfig{
				Flags: lr.Config.Flags,
			}
			config.NumAttrs = 1
			config.Attrs[0].Mask = 1
			config.Attrs[0].Attr = uapi.DebouncePeriod(period).Encode()
			err = uapi.SetLineConfig(uintptr(lr.Fd), &config)
			require.Nil(t, err, period)
		}

		for i := 0; i < 2; i++ {
			xevt.ID = uapi.LineEventRisingEdge
			s.SetPull(offset, 1)
			evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
			require.Nil(t, err, i)
			require.NotNil(t, evt, i)
			evt.Timestamp = 0
			assert.Equal(t, xevt, *evt, i)

			xevt.LineSeqno++
			xevt.Seqno++
			xevt.ID = uapi.LineEventFallingEdge
			s.SetPull(offset, 0)
			evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
			require.Nil(t, err, i)
			require.NotNil(t, evt, i)
			evt.Timestamp = 0
			assert.Equal(t, xevt, *evt, i)

			xevt.LineSeqno++
			xevt.Seqno++
		}
	}
}

func TestGetLineDebouncedEdges(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	err = s.SetPull(offset, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags:    uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
			NumAttrs: 1,
		},
	}
	lr.Offsets[0] = uint32(offset)
	copy(lr.Consumer[:31], "test-get-line-debounced-edges")
	lr.Config.Attrs[0].Mask = 1
	lr.Config.Attrs[0].Attr = uapi.DebouncePeriod(20000).Encode()
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
		Offset:    uint32(offset),
	}

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	for i := 0; i < 2; i++ {
		xevt.ID = uapi.LineEventRisingEdge
		s.SetPull(offset, 1)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err, i)
		require.NotNil(t, evt, i)
		evt.Timestamp = 0
		assert.Equal(t, xevt, *evt, i)

		xevt.LineSeqno++
		xevt.Seqno++
		xevt.ID = uapi.LineEventFallingEdge
		s.SetPull(offset, 0)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err, i)
		require.NotNil(t, evt, i)
		evt.Timestamp = 0
		assert.Equal(t, xevt, *evt, i)

		xevt.LineSeqno++
		xevt.Seqno++
	}
}

func TestSetConfigEdgeDetectionPolarity(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	err = s.SetPull(offset, 0)
	require.Nil(t, err)
	lr := uapi.LineRequest{
		Lines: 1,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput | uapi.LineFlagEdgeRising,
		},
	}
	lr.Offsets[0] = uint32(offset)
	copy(lr.Consumer[:31], "test-set-config-edge-detection-polarity")
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	flags := []uapi.LineFlag{0, uapi.LineFlagActiveLow, 0, uapi.LineFlagActiveLow}
	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
		Offset:    uint32(offset),
		ID:        uapi.LineEventRisingEdge,
	}

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	for _, flag := range flags {
		config := uapi.LineConfig{
			Flags: lr.Config.Flags | flag,
		}
		err = uapi.SetLineConfig(uintptr(lr.Fd), &config)
		require.Nil(t, err, flag)

		if flag == 0 {
			for i := 0; i < 2; i++ {
				s.SetPull(offset, 1)
				evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
				require.Nil(t, err, i)
				require.NotNil(t, evt, i)
				evt.Timestamp = 0
				assert.Equal(t, xevt, *evt, i)

				xevt.LineSeqno++
				xevt.Seqno++
				s.SetPull(offset, 0)
				evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
				assert.Nil(t, err, i)
				assert.Nil(t, evt, "spurious event", i)
			}
		} else {
			for i := 0; i < 2; i++ {
				s.SetPull(offset, 1)
				evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
				assert.Nil(t, err, i)
				assert.Nil(t, evt, "spurious event", i)

				s.SetPull(offset, 0)
				evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
				require.Nil(t, err, i)
				require.NotNil(t, evt, i)
				evt.Timestamp = 0
				assert.Equal(t, xevt, *evt, i)
				xevt.LineSeqno++
				xevt.Seqno++
			}
		}
	}
}

func TestSetConfigDebouncedThenEdges(t *testing.T) {
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	offset := 1
	err = s.SetPull(offset, 0)
	require.Nil(t, err)
	config := uapi.LineConfig{
		Flags: uapi.LineFlagInput}

	lr := uapi.LineRequest{
		Lines:  1,
		Config: config,
	}
	lr.Offsets[0] = uint32(offset)
	copy(lr.Consumer[:31], "test-set-config-debounced-edges")
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	defer unix.Close(int(lr.Fd))

	config.NumAttrs = 1
	config.Attrs[0].Mask = 1
	config.Attrs[0].Attr = uapi.DebouncePeriod(1000).Encode()
	err = uapi.SetLineConfig(uintptr(lr.Fd), &config)
	require.Nil(t, err)

	config.Flags |= uapi.LineFlagEdgeBoth
	err = uapi.SetLineConfig(uintptr(lr.Fd), &config)
	require.Nil(t, err)

	xevt := uapi.LineEvent{
		Seqno:     1,
		LineSeqno: 1,
		Offset:    uint32(offset),
	}

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	for i := 0; i < 2; i++ {
		xevt.ID = uapi.LineEventRisingEdge
		s.SetPull(offset, 1)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err, i)
		require.NotNil(t, evt, i)
		evt.Timestamp = 0
		assert.Equal(t, xevt, *evt, i)

		xevt.LineSeqno++
		xevt.Seqno++
		xevt.ID = uapi.LineEventFallingEdge
		s.SetPull(offset, 0)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err, i)
		require.NotNil(t, evt, i)
		evt.Timestamp = 0
		assert.Equal(t, xevt, *evt, i)

		xevt.LineSeqno++
		xevt.Seqno++
	}
}

func TestOutputSetGets(t *testing.T) {
	t.Skip("contains known failures up to Linux 5.15")
	patterns := []struct {
		name string
		flag uapi.LineFlag
	}{
		{"o", uapi.LineFlagOutput},
		{"od", uapi.LineFlagOutput | uapi.LineFlagOpenDrain},
		{"os", uapi.LineFlagOutput | uapi.LineFlagOpenSource},
	}
	s, err := gpiosim.NewSimpleton(6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	offset := 0
	for _, p := range patterns {
		for initial := 0; initial <= 1; initial++ {
			for toggle := 0; toggle <= 1; toggle++ {
				for activeLow := 0; activeLow <= 1; activeLow++ {
					final := initial
					if toggle == 1 {
						final ^= 0x01
					}
					flags := p.flag
					name := p.name
					if activeLow == 1 {
						flags |= uapi.LineFlagActiveLow
						name += "al"
					}
					label := fmt.Sprintf("%s-%d-%d-%d", name, initial^1, initial, final)
					tf := func(t *testing.T) {
						testLine(t, s, offset, flags, initial, toggle)
					}
					t.Run(label, tf)
				}
			}
		}
	}
}

func TestEdgeDetectionLinesMax(t *testing.T) {
	s, err := gpiosim.NewSimpleton(uapi.LinesMax + 6)
	require.Nil(t, err)
	require.NotNil(t, s)
	defer s.Close()
	f, err := os.Open(s.DevPath())

	require.Nil(t, err)
	defer f.Close()
	offsets := [uapi.LinesMax]uint32{}
	for i := 0; i < uapi.LinesMax; i++ {
		offsets[i] = uint32(i)
		err = s.SetPull(i, 0)
		require.Nil(t, err)
	}
	lr := uapi.LineRequest{
		Lines:   uint32(uapi.LinesMax),
		Offsets: offsets,
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagInput | uapi.LineFlagEdgeBoth,
		},
	}
	copy(lr.Consumer[:31], "test-edge-detection-lines-max")
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)

	evt, err := readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	lv := uapi.LineValues{
		Mask: uapi.NewLineBitMask(uapi.LinesMax),
	}
	for i := 0; i < uapi.LinesMax; i++ {
		s.SetPull(i, 1)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		require.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uapi.LineEventRisingEdge, evt.ID)
		assert.Equal(t, uint32(i), evt.Offset)

		err = uapi.GetLineValues(uintptr(lr.Fd), &lv)
		assert.Nil(t, err)
		assert.Equal(t, 1, lv.Bits.Get(i))
	}

	for i := 0; i < uapi.LinesMax; i++ {
		s.SetPull(i, 0)
		evt, err = readLineEventTimeout(lr.Fd, eventWaitTimeout)
		assert.Nil(t, err)
		require.NotNil(t, evt)
		assert.Equal(t, uapi.LineEventFallingEdge, evt.ID)
		assert.Equal(t, uint32(i), evt.Offset)

		err = uapi.GetLineValues(uintptr(lr.Fd), &lv)
		assert.Nil(t, err)
		assert.Equal(t, 0, lv.Bits.Get(i))
	}

	evt, err = readLineEventTimeout(lr.Fd, spuriousEventWaitTimeout)
	assert.Nil(t, err)
	assert.Nil(t, evt, "spurious event")

	unix.Close(int(lr.Fd))
}

func testLine(t *testing.T, s *gpiosim.Simpleton, line int, flags uapi.LineFlag, initial, toggle int) {
	t.Helper()
	// set mock initial - opposing default
	s.SetPull(line, initial^0x01)
	f, err := os.Open(s.DevPath())
	require.Nil(t, err)
	defer f.Close()
	// request line output
	lr := uapi.LineRequest{
		Config: uapi.LineConfig{
			Flags: uapi.LineFlagOutput,
		},
		Lines: 1,
	}
	lr.Offsets[0] = uint32(line)
	copy(lr.Consumer[:31], "test-line")
	ov := uapi.OutputValues(initial)
	lca := uapi.LineConfigAttribute{Attr: ov.Encode(), Mask: 1}
	lr.Config.AddAttribute(lca)
	err = uapi.GetLine(f.Fd(), &lr)
	require.Nil(t, err)
	if toggle != 0 {
		var vv uapi.LineValues
		vv.Mask.Set(0, 1)
		vv.Bits.Set(0, initial^1)
		err = uapi.SetLineValues(uintptr(lr.Fd), vv)
		assert.Nil(t, err, "can't set value 1")
		err = uapi.GetLineValues(uintptr(lr.Fd), &vv)
		assert.Nil(t, err, "can't get value 1")
		assert.Equal(t, initial^1, vv.Get(0), "get value 1")
		vv.Bits.Set(0, initial)
		err = uapi.SetLineValues(uintptr(lr.Fd), vv)
		assert.Nil(t, err, "can't set value 2")
		err = uapi.GetLineValues(uintptr(lr.Fd), &vv)
		assert.Nil(t, err, "can't get value 2")
		assert.Equal(t, initial, vv.Get(0), "get value 2")
		vv.Bits.Set(0, initial^1)
		err = uapi.SetLineValues(uintptr(lr.Fd), vv)
		assert.Nil(t, err, "can't set value 3")
		err = uapi.GetLineValues(uintptr(lr.Fd), &vv)
		assert.Nil(t, err, "can't get value 3")
		assert.Equal(t, initial^1, vv.Get(0), "get value 3")
	}
	// release
	unix.Close(int(lr.Fd))
}
