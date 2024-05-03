// Copyright 2022 The Gc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gc // modernc.org/gc/v3

import (
	"fmt"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
)

// The list of tokens.
const (
	// Special tokens
	ILLEGAL = token.ILLEGAL
	EOF     = token.EOF
	COMMENT = token.COMMENT

	// Identifiers and basic type literals
	// (these tokens stand for classes of literals)
	IDENT  = token.IDENT  // main
	INT    = token.INT    // 12345
	FLOAT  = token.FLOAT  // 123.45
	IMAG   = token.IMAG   // 123.45i
	CHAR   = token.CHAR   // 'a'
	STRING = token.STRING // "abc"

	// Operators and delimiters
	ADD = token.ADD // +
	SUB = token.SUB // -
	MUL = token.MUL // *
	QUO = token.QUO // /
	REM = token.REM // %

	AND     = token.AND     // &
	OR      = token.OR      // |
	XOR     = token.XOR     // ^
	SHL     = token.SHL     // <<
	SHR     = token.SHR     // >>
	AND_NOT = token.AND_NOT // &^

	ADD_ASSIGN = token.ADD_ASSIGN // +=
	SUB_ASSIGN = token.SUB_ASSIGN // -=
	MUL_ASSIGN = token.MUL_ASSIGN // *=
	QUO_ASSIGN = token.QUO_ASSIGN // /=
	REM_ASSIGN = token.REM_ASSIGN // %=

	AND_ASSIGN     = token.AND_ASSIGN     // &=
	OR_ASSIGN      = token.OR_ASSIGN      // |=
	XOR_ASSIGN     = token.XOR_ASSIGN     // ^=
	SHL_ASSIGN     = token.SHL_ASSIGN     // <<=
	SHR_ASSIGN     = token.SHR_ASSIGN     // >>=
	AND_NOT_ASSIGN = token.AND_NOT_ASSIGN // &^=

	LAND  = token.LAND  // &&
	LOR   = token.LOR   // ||
	ARROW = token.ARROW // <-
	INC   = token.INC   // ++
	DEC   = token.DEC   // --

	EQL    = token.EQL    // ==
	LSS    = token.LSS    // <
	GTR    = token.GTR    // >
	ASSIGN = token.ASSIGN // =
	NOT    = token.NOT    // !

	NEQ      = token.NEQ      // !=
	LEQ      = token.LEQ      // <=
	GEQ      = token.GEQ      // >=
	DEFINE   = token.DEFINE   // :=
	ELLIPSIS = token.ELLIPSIS // ...

	LPAREN = token.LPAREN // (
	LBRACK = token.LBRACK // [
	LBRACE = token.LBRACE // {
	COMMA  = token.COMMA  // ,
	PERIOD = token.PERIOD // .

	RPAREN    = token.RPAREN    // )
	RBRACK    = token.RBRACK    // ]
	RBRACE    = token.RBRACE    // }
	SEMICOLON = token.SEMICOLON // ;
	COLON     = token.COLON     // :

	// Keywords
	BREAK    = token.BREAK
	CASE     = token.CASE
	CHAN     = token.CHAN
	CONST    = token.CONST
	CONTINUE = token.CONTINUE

	DEFAULT     = token.DEFAULT
	DEFER       = token.DEFER
	ELSE        = token.ELSE
	FALLTHROUGH = token.FALLTHROUGH
	FOR         = token.FOR

	FUNC   = token.FUNC
	GO     = token.GO
	GOTO   = token.GOTO
	IF     = token.IF
	IMPORT = token.IMPORT

	INTERFACE = token.INTERFACE
	MAP       = token.MAP
	PACKAGE   = token.PACKAGE
	RANGE     = token.RANGE
	RETURN    = token.RETURN

	SELECT = token.SELECT
	STRUCT = token.STRUCT
	SWITCH = token.SWITCH
	TYPE   = token.TYPE
	VAR    = token.VAR

	// additional tokens, handled in an ad-hoc manner
	TILDE = token.TILDE
)

var (
	trcTODOs       bool
	extendedErrors bool
)

// origin returns caller's short position, skipping skip frames.
func origin(skip int) string {
	pc, fn, fl, _ := runtime.Caller(skip)
	f := runtime.FuncForPC(pc)
	var fns string
	if f != nil {
		fns = f.Name()
		if x := strings.LastIndex(fns, "."); x > 0 {
			fns = fns[x+1:]
		}
		if strings.HasPrefix(fns, "func") {
			num := true
			for _, c := range fns[len("func"):] {
				if c < '0' || c > '9' {
					num = false
					break
				}
			}
			if num {
				return origin(skip + 2)
			}
		}
	}
	return fmt.Sprintf("%s:%d:%s", filepath.Base(fn), fl, fns)
}

// todo prints and returns caller's position and an optional message tagged with TODO. Output goes to stderr.
//
//lint:ignore U1000 whatever
func todo(s string, args ...interface{}) string {
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	r := fmt.Sprintf("%s\n\tTODO (%s)", origin(2), s)
	// fmt.Fprintf(os.Stderr, "%s\n", r)
	// os.Stdout.Sync()
	return r
}

// trc prints and returns caller's position and an optional message tagged with TRC. Output goes to stderr.
//
//lint:ignore U1000 whatever
func trc(s string, args ...interface{}) string {
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	r := fmt.Sprintf("%s: TRC (%s)", origin(2), s)
	fmt.Fprintf(os.Stderr, "%s\n", r)
	os.Stderr.Sync()
	return r
}

func extractPos(s string) (p token.Position, ok bool) {
	var prefix string
	if len(s) > 1 && s[1] == ':' { // c:\foo
		prefix = s[:2]
		s = s[2:]
	}
	// "testdata/parser/bug/001.c:1193: ..."
	a := strings.Split(s, ":")
	// ["testdata/parser/bug/001.c" "1193" "..."]
	if len(a) < 2 {
		return p, false
	}

	line, err := strconv.Atoi(a[1])
	if err != nil {
		return p, false
	}

	col, err := strconv.Atoi(a[2])
	if err != nil {
		col = 1
	}

	return token.Position{Filename: prefix + a[0], Line: line, Column: col}, true
}

// errorf constructs an error value. If extendedErrors is true, the error will
// contain its origin.
func errorf(s string, args ...interface{}) error {
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	if trcTODOs && strings.HasPrefix(s, "TODO") {
		fmt.Fprintf(os.Stderr, "%s (%v)\n", s, origin(2))
		os.Stderr.Sync()
	}
	switch {
	case extendedErrors:
		return fmt.Errorf("%s (%v: %v: %v)", s, origin(4), origin(3), origin(2))
	default:
		return fmt.Errorf("%s", s)
	}
}

func tokSource(t token.Token) string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case COMMENT:
		return "COMMENT"
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case IMAG:
		return "IMAG"
	case CHAR:
		return "CHAR"
	case STRING:
		return "STRING"
	case ADD:
		return "ADD"
	case SUB:
		return "SUB"
	case MUL:
		return "MUL"
	case QUO:
		return "QUO"
	case REM:
		return "REM"
	case AND:
		return "AND"
	case OR:
		return "OR"
	case XOR:
		return "XOR"
	case SHL:
		return "SHL"
	case SHR:
		return "SHR"
	case AND_NOT:
		return "AND_NOT"
	case ADD_ASSIGN:
		return "ADD_ASSIGN"
	case SUB_ASSIGN:
		return "SUB_ASSIGN"
	case MUL_ASSIGN:
		return "MUL_ASSIGN"
	case QUO_ASSIGN:
		return "QUO_ASSIGN"
	case REM_ASSIGN:
		return "REM_ASSIGN"
	case AND_ASSIGN:
		return "AND_ASSIGN"
	case OR_ASSIGN:
		return "OR_ASSIGN"
	case XOR_ASSIGN:
		return "XOR_ASSIGN"
	case SHL_ASSIGN:
		return "SHL_ASSIGN"
	case SHR_ASSIGN:
		return "SHR_ASSIGN"
	case AND_NOT_ASSIGN:
		return "AND_NOT_ASSIGN"
	case LAND:
		return "LAND"
	case LOR:
		return "LOR"
	case ARROW:
		return "ARROW"
	case INC:
		return "INC"
	case DEC:
		return "DEC"
	case EQL:
		return "EQL"
	case LSS:
		return "LSS"
	case GTR:
		return "GTR"
	case ASSIGN:
		return "ASSIGN"
	case NOT:
		return "NOT"
	case NEQ:
		return "NEQ"
	case LEQ:
		return "LEQ"
	case GEQ:
		return "GEQ"
	case DEFINE:
		return "DEFINE"
	case ELLIPSIS:
		return "ELLIPSIS"
	case LPAREN:
		return "LPAREN"
	case LBRACK:
		return "LBRACK"
	case LBRACE:
		return "LBRACE"
	case COMMA:
		return "COMMA"
	case PERIOD:
		return "PERIOD"
	case RPAREN:
		return "RPAREN"
	case RBRACK:
		return "RBRACK"
	case RBRACE:
		return "RBRACE"
	case SEMICOLON:
		return "SEMICOLON"
	case COLON:
		return "COLON"
	case BREAK:
		return "BREAK"
	case CASE:
		return "CASE"
	case CHAN:
		return "CHAN"
	case CONST:
		return "CONST"
	case CONTINUE:
		return "CONTINUE"
	case DEFAULT:
		return "DEFAULT"
	case DEFER:
		return "DEFER"
	case ELSE:
		return "ELSE"
	case FALLTHROUGH:
		return "FALLTHROUGH"
	case FOR:
		return "FOR"
	case FUNC:
		return "FUNC"
	case GO:
		return "GO"
	case GOTO:
		return "GOTO"
	case IF:
		return "IF"
	case IMPORT:
		return "IMPORT"
	case INTERFACE:
		return "INTERFACE"
	case MAP:
		return "MAP"
	case PACKAGE:
		return "PACKAGE"
	case RANGE:
		return "RANGE"
	case RETURN:
		return "RETURN"
	case SELECT:
		return "SELECT"
	case STRUCT:
		return "STRUCT"
	case SWITCH:
		return "SWITCH"
	case TYPE:
		return "TYPE"
	case VAR:
		return "VAR"
	case TILDE:
		return "TILDE"
	default:
		panic(todo("", int(t), t))
	}
}

type data struct {
	line  int
	cases int
	cnt   int
}

type analyzer struct {
	sync.Mutex
	m map[int]*data // line: data
}

func newAnalyzer() *analyzer {
	return &analyzer{m: map[int]*data{}}
}

func (a *analyzer) record(line, cnt int) {
	d := a.m[line]
	if d == nil {
		d = &data{line: line}
		a.m[line] = d
	}
	d.cases++
	d.cnt += cnt
}

func (a *analyzer) merge(b *analyzer) {
	a.Lock()
	defer a.Unlock()

	for k, v := range b.m {
		d := a.m[k]
		if d == nil {
			d = &data{line: k}
			a.m[k] = d
		}
		d.cases += v.cases
		d.cnt += v.cnt
	}
}

func (a *analyzer) report() string {
	var rows []*data
	for _, v := range a.m {
		rows = append(rows, v)
	}
	sort.Slice(rows, func(i, j int) bool {
		a := rows[i]
		b := rows[j]
		if a.cases < b.cases {
			return true
		}

		if a.cases > b.cases {
			return false
		}

		// a.cases == b.cases
		if a.cnt < b.cnt {
			return true
		}

		if a.cnt > b.cnt {
			return false
		}

		// a.cnt == b.cnt
		return a.line < b.line
	})
	var b strings.Builder
	var cases, cnt int
	for _, row := range rows {
		cases += row.cases
		cnt += row.cnt
		avg := float64(row.cnt) / float64(row.cases)
		fmt.Fprintf(&b, "parser.go:%d:\t%16s %16s %8.1f\n", row.line, h(row.cases), h(row.cnt), avg)
	}
	avg := float64(cnt) / float64(cases)
	fmt.Fprintf(&b, "<total>\t\t%16s %16s %8.1f\n", h(cases), h(cnt), avg)
	return b.String()
}

func h(v interface{}) string {
	switch x := v.(type) {
	case int:
		return humanize.Comma(int64(x))
	case int32:
		return humanize.Comma(int64(x))
	case int64:
		return humanize.Comma(x)
	case uint32:
		return humanize.Comma(int64(x))
	case uint64:
		if x <= math.MaxInt64 {
			return humanize.Comma(int64(x))
		}

		return "-" + humanize.Comma(-int64(x))
	}
	return fmt.Sprint(v)
}

type parallel struct {
	limiter chan struct{}
}

func newParallel() *parallel {
	return &parallel{
		limiter: make(chan struct{}, runtime.GOMAXPROCS(0)),
	}
}

func (p *parallel) throttle(f func()) {
	p.limiter <- struct{}{}

	defer func() {
		<-p.limiter
	}()

	f()
}

func extraTags(verMajor, verMinor int, goos, goarch string) (r []string) {
	// https://github.com/golang/go/commit/eeb7899137cda1c2cd60dab65ff41f627436db5b
	//
	// In Go 1.17 we added register ABI on AMD64 on Linux/macOS/Windows
	// as a GOEXPERIMENT, on by default. In Go 1.18, we commit to always
	// enabling register ABI on AMD64.
	//
	// Now "go build" for AMD64 always have goexperiment.regabi* tags
	// set. However, at bootstrapping cmd/dist does not set the tags
	// when building go_bootstrap. For this to work, unfortunately, we
	// need to hard-code AMD64 to use register ABI in runtime code.
	if verMajor == 1 {
		switch {
		case verMinor == 17:
			switch goos {
			case "linux", "darwin", "windows":
				if goarch == "amd64" {
					r = append(r, "goexperiment.regabiargs", "goexperiment.regabiwrappers")
				}
			}
		case verMinor >= 18:
			if goarch == "amd64" {
				r = append(r, "goexperiment.regabiargs", "goexperiment.regabiwrappers")
			}
		}
	}
	return r
}
