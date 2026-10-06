// SPDX-License-Identifier: MIT
//
// Copyright © 2020 Kent Gibson <warthog618@gmail.com>.

//go:build linux

package uapi_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/warthog618/go-gpiosim"
)

func BenchmarkChipOpenClose(b *testing.B) {
	s, err := gpiosim.NewSimpleton(4)
	require.Nil(b, err)
	defer s.Close()
	for i := 0; i < b.N; i++ {
		f, _ := os.Open(s.DevPath())
		f.Close()
	}
}
