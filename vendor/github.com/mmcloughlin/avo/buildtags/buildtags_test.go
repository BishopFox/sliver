package buildtags

import "testing"

func TestGoString(t *testing.T) {
	cases := []struct {
		Constraint Interface
		Expect     string
	}{
		{Term("amd64"), "// +build amd64\n"},
		{Any(Opt(Term("linux"), Term("386")), Opt("darwin", Not("cgo"))), "// +build linux,386 darwin,!cgo\n"},
		{And(Any(Term("linux"), Term("darwin")), Term("386")), "// +build linux darwin\n// +build 386\n"},
	}
	for _, c := range cases {
		got := c.Constraint.ToConstraints().GoString()
		if got != c.Expect {
			t.Errorf("constraint %#v GoString() got %q; expected %q", c.Constraint, got, c.Expect)
		}
	}
}

func TestValidateOK(t *testing.T) {
	cases := []Interface{
		Term("name"),
		Term("!name"),
	}
	for _, c := range cases {
		if err := c.ToConstraints().Validate(); err != nil {
			t.Errorf("unexpected validation error for %#v: %q", c, err)
		}
	}
}

func TestValidateErrors(t *testing.T) {
	cases := []struct {
		Constraint    Interface
		ExpectMessage string
	}{
		{Term(""), "empty tag name"},
		{Term("!"), "empty tag name"},
		{Term("!!"), "at most one '!' allowed"},
		{Term("!abc!def"), "character '!' disallowed in tags"},
		{
			And(Any(Term("linux"), Term("my-os")), Term("386")).ToConstraints(),
			"invalid term \"my-os\": character '-' disallowed in tags",
		},
	}
	for _, c := range cases {
		err := c.Constraint.Validate()
		if err == nil {
			t.Fatalf("expect validation error for constraint:\n%s", c.Constraint.GoString())
		}
		if err.Error() != c.ExpectMessage {
			t.Fatalf("unexpected error message\n\tgot:\t%q\n\texpect:\t%q\n", err, c.ExpectMessage)
		}
	}
}

func TestParseConstraintRoundTrip(t *testing.T) {
	exprs := []string{
		"amd64",
		"amd64,linux",
		"!a",
		"!a,b c,!d,e",
		"linux,386 darwin,!cgo",
	}
	for _, expr := range exprs {
		c := AssertParseConstraint(t, expr)
		got := c.GoString()
		expect := "// +build " + expr + "\n"
		if got != expect {
			t.Fatalf("roundtrip error\n\tgot\t%q\n\texpect\t%q\n", got, expect)
		}
	}
}

func TestParseConstraintError(t *testing.T) {
	expr := "linux,386 my-os,!cgo"
	_, err := ParseConstraint(expr)
	if err == nil {
		t.Fatalf("expected error parsing %q", expr)
	}
}

func TestEvaluate(t *testing.T) {
	cases := []struct {
		Constraint Interface
		Values     map[string]bool
		Expect     bool
	}{
		{Term("a"), SetTags("a"), true},
		{Term("!a"), SetTags("a"), false},
		{Term("!a"), SetTags(), true},
		{Term("inval-id"), SetTags("inval-id"), false},

		{Opt(Term("a"), Term("b")), SetTags(), false},
		{Opt(Term("a"), Term("b")), SetTags("a"), false},
		{Opt(Term("a"), Term("b")), SetTags("b"), false},
		{Opt(Term("a"), Term("b")), SetTags("a", "b"), true},
		{Opt(Term("a"), Term("b-a-d")), SetTags("a", "b-a-d"), false},

		{
			Any(Opt(Term("linux"), Term("386")), Opt("darwin", Not("cgo"))),
			SetTags("linux", "386"),
			true,
		},
		{
			Any(Opt(Term("linux"), Term("386")), Opt("darwin", Not("cgo"))),
			SetTags("darwin"),
			true,
		},
		{
			Any(Opt(Term("linux"), Term("386")), Opt("darwin", Not("cgo"))),
			SetTags("linux", "darwin", "cgo"),
			false,
		},

		{
			And(Any(Term("linux"), Term("darwin")), Term("386")),
			SetTags("darwin", "386"),
			true,
		},
	}
	for _, c := range cases {
		got := c.Constraint.Evaluate(c.Values)
		if c.Constraint.Validate() != nil && got {
			t.Fatal("invalid expressions must evaluate false")
		}
		if got != c.Expect {
			t.Errorf("%#v evaluated with %#v got %v expect %v", c.Constraint, c.Values, got, c.Expect)
		}
	}
}

func AssertParseConstraint(t *testing.T, expr string) Constraint {
	t.Helper()
	c, err := ParseConstraint(expr)
	if err != nil {
		t.Fatalf("error parsing expression %q: %q", expr, err)
	}
	return c
}
