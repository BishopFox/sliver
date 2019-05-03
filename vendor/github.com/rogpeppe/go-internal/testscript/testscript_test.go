// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testscript

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func printArgs() int {
	fmt.Printf("%q\n", os.Args)
	return 0
}

func exitWithStatus() int {
	n, _ := strconv.Atoi(os.Args[1])
	return n
}

func TestMain(m *testing.M) {
	os.Exit(RunMain(m, map[string]func() int{
		"printargs": printArgs,
		"status":    exitWithStatus,
	}))
}

func TestCRLFInput(t *testing.T) {
	td, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create TempDir: %v", err)
	}
	defer func() {
		os.RemoveAll(td)
	}()
	tf := filepath.Join(td, "script.txt")
	contents := []byte("exists output.txt\r\n-- output.txt --\r\noutput contents")
	if err := ioutil.WriteFile(tf, contents, 0644); err != nil {
		t.Fatalf("failed to write to %v: %v", tf, err)
	}
	t.Run("_", func(t *testing.T) {
		Run(t, Params{Dir: td})
	})
}

func TestSimple(t *testing.T) {
	// TODO set temp directory.
	Run(t, Params{
		Dir: "scripts",
		Cmds: map[string]func(ts *TestScript, neg bool, args []string){
			"setSpecialVal":    setSpecialVal,
			"ensureSpecialVal": ensureSpecialVal,
		},
	})
	// TODO check that the temp directory has been removed.
}

func setSpecialVal(ts *TestScript, neg bool, args []string) {
	ts.Setenv("SPECIALVAL", "42")
}

func ensureSpecialVal(ts *TestScript, neg bool, args []string) {
	want := "42"
	if got := ts.Getenv("SPECIALVAL"); got != want {
		ts.Fatalf("expected SPECIALVAL to be %q; got %q", want, got)
	}
}
