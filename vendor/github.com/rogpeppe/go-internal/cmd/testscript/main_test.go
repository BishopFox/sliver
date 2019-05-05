// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"testscript": main1,
	}))
}

func TestScripts(t *testing.T) {
	var stderr bytes.Buffer
	cmd := exec.Command("go", "env", "GOMOD")
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, stderr.String())
	}
	gomod := string(out)

	if gomod == "" {
		t.Fatalf("apparently we are not running in module mode?")
	}

	p := testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			env.Vars = append(env.Vars,
				"GOINTERNALMODPATH="+filepath.Dir(gomod),
			)
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"unquote": unquote,
		},
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}

func unquote(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! unquote")
	}
	for _, arg := range args {
		file := ts.MkAbs(arg)
		data, err := ioutil.ReadFile(file)
		ts.Check(err)
		data = bytes.Replace(data, []byte("\n>"), []byte("\n"), -1)
		data = bytes.TrimPrefix(data, []byte(">"))
		err = ioutil.WriteFile(file, data, 0666)
		ts.Check(err)
	}
}
