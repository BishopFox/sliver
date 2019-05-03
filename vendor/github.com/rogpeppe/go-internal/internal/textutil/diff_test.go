// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textutil_test

import (
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/internal/textutil"
)

var diffTests = []struct {
	text1 string
	text2 string
	diff  string
}{
	{"a b c", "a b d e f", "a b -c +d +e +f"},
	{"", "a b c", "+a +b +c"},
	{"a b c", "", "-a -b -c"},
	{"a b c", "d e f", "-a -b -c +d +e +f"},
	{"a b c d e f", "a b d e f", "a b -c d e f"},
	{"a b c e f", "a b c d e f", "a b c +d e f"},
}

func TestDiff(t *testing.T) {
	for _, tt := range diffTests {
		// Turn spaces into \n.
		text1 := strings.Replace(tt.text1, " ", "\n", -1)
		if text1 != "" {
			text1 += "\n"
		}
		text2 := strings.Replace(tt.text2, " ", "\n", -1)
		if text2 != "" {
			text2 += "\n"
		}
		out := textutil.Diff(text1, text2)
		// Cut final \n, cut spaces, turn remaining \n into spaces.
		out = strings.Replace(strings.Replace(strings.TrimSuffix(out, "\n"), " ", "", -1), "\n", " ", -1)
		if out != tt.diff {
			t.Errorf("diff(%q, %q) = %q, want %q", text1, text2, out, tt.diff)
		}
	}
}
