package build

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/mmcloughlin/avo/src"
)

func TestLogErrorNil(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, "prefix: ", 0)
	LogError(l, nil, 0)
	if buf.String() != "" {
		t.Fatalf("should print nothing for nil error")
	}
}

func TestLogErrorNonErrorList(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, "prefix: ", 0)
	err := errors.New("not an ErrorList")
	LogError(l, err, 0)
	got := buf.String()
	expect := "prefix: " + err.Error() + "\n"
	if got != expect {
		t.Fatalf("got\t%q\nexpect\t%q\n", got, expect)
	}
}

func TestLogErrorList(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf, "prefix: ", 0)

	e := ErrorList{}
	filename := "asm.go"
	err := errors.New("some kind of error")
	n := 7
	for i := 1; i <= n; i++ {
		p := src.Position{Filename: filename, Line: i}
		e.AddAt(p, err)
	}

	expect := ""
	// Unlimited print.
	LogError(l, e, 0)
	for i := 1; i <= n; i++ {
		expect += fmt.Sprintf("prefix: %s:%d: some kind of error\n", filename, i)
	}

	// Max equal to number of errors.
	LogError(l, e, n)
	for i := 1; i <= n; i++ {
		expect += fmt.Sprintf("prefix: %s:%d: some kind of error\n", filename, i)
	}

	// Max less than number of errors.
	m := n / 2
	LogError(l, e, m)
	for i := 1; i <= m; i++ {
		expect += fmt.Sprintf("prefix: %s:%d: some kind of error\n", filename, i)
	}
	expect += fmt.Sprintf("prefix: too many errors\n")

	got := buf.String()
	if got != expect {
		t.Fatalf("got\n%s\nexpect\n%s\n", got, expect)
	}
}
