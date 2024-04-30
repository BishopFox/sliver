// Copyright 2022 The Gc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gc // modernc.org/gc/v3

import (
	"go/constant"
	"go/token"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"modernc.org/mathutil"
	"modernc.org/strutil"
)

// === RUN   TestTypeCheck/GOROOT
// === CONT  TestTypeCheck
//     all_test.go:1123: TOTAL packages 516, files 2,301, skip 0, ok 516, fail 0
//     all_test.go:1127: pkg count 516, heap 626,182,264
// --- PASS: TestTypeCheck (2.28s)
//     --- PASS: TestTypeCheck/GOROOT (2.12s)

//    all_test.go:1127: pkg count 516, heap 626,153,184
//    all_test.go:1128: pkg count 516, heap 590,057,064
//    all_test.go:1128: pkg count 516, heap 572,015,152
//    all_test.go:1129: pkg count 516, heap 567,709,952
//    all_test.go:1129: pkg count 516, heap 555,500,960
//    all_test.go:1129: pkg count 516, heap 551,777,488
//    all_test.go:1129: pkg count 516, heap 548,683,512
//    all_test.go:1129: pkg count 516, heap 548,447,936
//    all_test.go:1129: pkg count 516, heap 547,480,288
//    all_test.go:1129: pkg count 516, heap 546,915,592
//    all_test.go:1129: pkg count 516, heap 543,393,136
//    all_test.go:1129: pkg count 516, heap 544,638,544
//    all_test.go:1129: pkg count 516, heap 474,343,936
//    all_test.go:1129: pkg count 516, heap 459,353,840
//    all_test.go:1129: pkg count 516, heap 457,275,512
//    all_test.go:1129: pkg count 516, heap 455,355,680
//    all_test.go:1129: pkg count 516, heap 454,663,568
//    all_test.go:1129: pkg count 516, heap 454,581,072
//    all_test.go:1129: pkg count 516, heap 454,607,112
//    all_test.go:1129: pkg count 516, heap 454,709,968
//    all_test.go:1129: pkg count 516, heap 455,312,784
//    all_test.go:1129: pkg count 516, heap 456,016,824
//    all_test.go:1129: pkg count 516, heap 455,954,544
//    all_test.go:1129: pkg count 516, heap 456,016,592
//    all_test.go:1129: pkg count 516, heap 457,121,224

//    all_test.go:1129: pkg count 516, heap 427,262,960
//    all_test.go:1130: pkg count 516, heap 428,998,600
//    all_test.go:1130: pkg count 551, heap 448,395,152
//    all_test.go:1130: pkg count 551, heap 451,817,616
//    all_test.go:1131: pkg count 551, heap 452,091,200
//    all_test.go:1131: pkg count 551, heap 452,999,840

//                                         <total> x 16,603,469 =   892,265,816 á  54
//                                         <total> x 16,024,194 =   887,787,224 á  55
//                                         <total> x 16,025,144 =   888,006,760 á  55
//                                         <total> x 16,025,211 =   823,222,088 á  51
//                                         <total> x 16,025,281 =   822,404,264 á  51
//                                         <total> x 14,056,450 =   696,398,872 á  50
//                                         <total> x 14,056,581 =   696,851,856 á  50
//                                         <total> x 14,056,453 =   708,480,848 á  50
//                                         <total> x 14,422,414 =   719,035,680 á  50
//                                         <total> x 14,423,240 =   717,114,200 á  50
//                                         <total> x 14,425,901 =   711,567,152 á  49
//                                         <total> x 14,474,065 =   710,068,032 á  49
//                                         <total> x 14,481,041 =   710,373,680 á  49
//                                         <total> x 14,481,767 =   710,408,768 á  49
//                                         <total> x 14,484,493 =   710,543,264 á  49
//                                         <total> x 14,461,141 =   706,268,448 á  49
//                                         <total> x 14,461,182 =   707,678,232 á  49
//                                         <total> x 14,461,242 =   714,720,336 á  49
//                                         <total> x 14,461,219 =   797,198,184 á  55
//                                         <total> x 14,461,496 =   797,214,104 á  55
//                                         <total> x 14,461,329 =   716,132,376 á  50
//                                         <total> x 14,461,680 =   711,984,376 á  49
//                                         <total> x 14,160,586 =   702,969,536 á  50
//                                         <total> x 14,160,709 =   682,184,664 á  48
//                                         <total> x 14,160,848 =   673,044,152 á  48
//                                         <total> x 14,160,317 =   665,980,184 á  47
//                                         <total> x 14,005,861 =   661,267,672 á  47
//                                         <total> x 13,983,296 =   660,781,720 á  47
//                                         <total> x 13,943,950 =   660,175,016 á  47
//                                         <total> x 13,943,178 =   647,906,568 á  46
//                                         <total> x 13,924,463 =   648,999,976 á  47
//                                         <total> x 13,322,751 =   541,059,736 á  41
//                                         <total> x 12,815,541 =   510,052,400 á  40
//                                         <total> x 12,815,675 =   506,593,488 á  40
//                                         <total> x 12,639,779 =   500,965,136 á  40
//                                         <total> x 12,640,847 =   501,008,776 á  40
//                                         <total> x 12,603,003 =   499,658,832 á  40
//                                         <total> x 12,603,001 =   502,473,720 á  40
//                                         <total> x 12,602,667 =   505,274,416 á  40
//                                         <total> x 12,603,389 =   505,302,936 á  40
//                                         <total> x 12,604,481 =   507,314,552 á  40

//                                         <total> x 12,590,468 =   454,314,392 á  36
//                                         <total> x 12,591,896 =   456,980,832 á  36
//                                         <total> x 12,597,633 =   457,211,632 á  36
//                                         <total> x 12,597,637 =   458,714,592 á  36
//                                         <total> x 12,931,431 =   471,180,992 á  36
//                                         <total> x 12,931,309 =   481,877,912 á  37
//                                         <total> x 12,933,798 =   482,402,192 á  37
//                                         <total> x 12,934,587 =   483,606,808 á  37

const parserBudget = 1e7

var (
	noBack    bool
	panicBack bool
)

type visibiliter interface {
	Node
	Visible() int
	setVisible(int32)
}

type visible struct {
	visible int32 // first token index where n is visible
}

// Visible reports the first token index where n is visible (in scope). Applies
// to local scopes only.
func (n *visible) Visible() int { return int(n.visible) }

func (n *visible) setVisible(i int32) { n.visible = i }

type named struct {
	n       visibiliter
	declTok Token
}

type ScopeKind int

const (
	scZero ScopeKind = iota
	UniverseScope
	PackageScope
	FileScope
	OtherScope
)

type Scope struct {
	nodes  map[string]named
	parent *Scope

	kind ScopeKind
}

func newScope(parent *Scope, kind ScopeKind) *Scope { return &Scope{parent: parent, kind: kind} }

func (s *Scope) Iterate(f func(name string, n Node) (stop bool)) {
	for name, v := range s.nodes {
		if f(name, v.n) {
			return
		}
	}
}

func (s *Scope) Kind() ScopeKind { return s.kind }

func (s *Scope) Parent() *Scope { return s.parent }

func (s *Scope) declare(nm Token, n visibiliter, visible int32, p *parser, initOK bool) (r named) {
	snm := nm.Src()
	switch snm {
	case "_":
		return r
	case "init":
		if s.kind == PackageScope {
			if p != nil && !initOK && p.reportDeclarationErrors {
				p.err(nm.Position(), "in the package block, the identifier init may only be used for init function declarations")
			}
			return r
		}
	}

	if ex, ok := s.nodes[snm]; ok {
		return ex
	}

	if s.nodes == nil {
		s.nodes = map[string]named{}
	}
	// trc("%v: add %s %p", nm.Position(), snm, s)
	n.setVisible(visible)
	s.nodes[snm] = named{n, nm}
	return r
}

func (s *Scope) lookup(id Token) (in *Scope, r named) {
	nm := id.Src()
	ix := int(id.index)
	for s != nil {
		switch s.kind {
		case PackageScope, UniverseScope:
			ix = -1
		}

		sc, ok := s.nodes[nm]
		if ok && (ix < 0 || ix > sc.n.Visible()) {
			return s, sc
		}

		s = s.parent
	}
	return nil, r
}

type lexicalScoper struct{ s *Scope }

func newLexicalScoper(s *Scope) lexicalScoper { return lexicalScoper{s} }

func (n *lexicalScoper) LexicalScope() *Scope { return n.s }

// Node is an item of the CST tree.
type Node interface {
	Position() token.Position
	Source(full bool) string
}

var hooks = strutil.PrettyPrintHooks{
	reflect.TypeOf(Token{}): func(f strutil.Formatter, v interface{}, prefix, suffix string) {
		t := v.(Token)
		if !t.IsValid() {
			return
		}

		pos := t.Position()
		if pos.Filename != "" {
			pos.Filename = filepath.Base(pos.Filename)
		}
		f.Format(string(prefix)+"%10s %q %q\t(%v:)"+string(suffix), tokSource(t.Ch()), t.Sep(), t.Src(), pos)
	},
}

func dump(n Node) string { return strutil.PrettyString(n, "", "", hooks) }

// NodeSource returns the source text of 'n'. If 'full' is false, every non
// empty separator is replaced by a single space. Nodes found in 'kill' are
// skipped, transitively.
func NodeSource(n Node, full bool, kill map[Node]struct{}) string {
	return nodeSource2(n, full, kill)
}

func nodeSource(n interface{}, full bool) string {
	return nodeSource2(n, full, nil)
}

func nodeSource2(n interface{}, full bool, kill map[Node]struct{}) string {
	var a []int32
	var t Token
	nodeSource0(&t.source, &a, n, kill)
	if len(a) == 0 {
		return ""
	}

	var b strings.Builder
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
	for _, v := range a {
		t.index = v
		t.ch = t.source.toks[t.index].ch
		b.WriteString(t.Source(full))
	}
	return b.String()
}

func nodeSource0(ps **source, a *[]int32, n interface{}, kill map[Node]struct{}) {
	if x, ok := n.(Node); ok {
		if _, ok := kill[x]; ok {
			return
		}
	}

	switch x := n.(type) {
	case nil:
		// nop
	case Token:
		if x.IsValid() {
			*ps = x.source
			*a = append(*a, x.index)
		}
	case *BasicLitNode:
		if x.IsValid() {
			*ps = x.source
			*a = append(*a, x.index)
		}
	default:
		t := reflect.TypeOf(n)
		v := reflect.ValueOf(n)
		if v.IsZero() {
			break
		}

		switch t.Kind() {
		case reflect.Pointer:
			nodeSource0(ps, a, v.Elem().Interface(), kill)
		case reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				if token.IsExported(t.Field(i).Name) {
					nodeSource0(ps, a, v.Field(i).Interface(), kill)
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				nodeSource0(ps, a, v.Index(i).Interface(), kill)
			}
		default:
			panic(todo("", t.Name(), t.Kind()))
		}
	}
}

type AST struct {
	EOF          Token
	FileScope    *Scope
	SourceFile   *SourceFileNode
	packageScope *Scope // For the individual file, enables parallelism, consolidated by Package.check()
}

func (n *AST) Source(full bool) string { return nodeSource(n, full) }

func (n *AST) Position() (r token.Position) {
	if n == nil {
		return r
	}
	return n.SourceFile.Position()
}

type parser struct {
	a             *analyzer
	fileScope     *Scope
	maxBackOrigin string
	maxBackRange  [2]int
	packageScope  *Scope
	path          string
	s             *scanner
	sc            *Scope

	backs   int
	budget  int
	ix      int
	maxBack int
	maxIx   int

	isClosed                bool
	record                  bool
	reportDeclarationErrors bool
}

func newParser(pkgScope *Scope, path string, src []byte, record bool) *parser {
	return &parser{
		a:            newAnalyzer(),
		budget:       parserBudget,
		fileScope:    newScope(pkgScope, FileScope),
		packageScope: pkgScope,
		path:         path,
		record:       record,
		s:            newScanner(path, src),
		sc:           pkgScope,
	}
}

func (p *parser) c() token.Token                  { return p.peek(0) }
func (p *parser) closeScope()                     { p.sc = p.sc.parent }
func (p *parser) errPosition() (r token.Position) { return p.s.toks[p.maxIx].position(p.s.source) }
func (p *parser) openScope()                      { p.sc = newScope(p.sc, OtherScope) }

func (p *parser) pos() (r token.Position) {
	return p.s.toks[mathutil.MinInt32(int32(p.ix), int32(len(p.s.toks)-1))].position(p.s.source)
}

func (p *parser) err(pos token.Position, msg string, args ...interface{}) {
	p.s.errs.err(pos, msg, args...)
}

func (p *parser) declare(s *Scope, nm Token, n visibiliter, visible int32, initOK bool) {
	if ex := s.declare(nm, n, visible, p, initOK); ex.declTok.IsValid() && p.reportDeclarationErrors {
		p.err(nm.Position(), "%s redeclared, previous declaration at %v:", nm.Src(), ex.declTok.Position())
	}
}

func (p *parser) consume() (r Token) {
	r = Token{p.s.source, p.s.toks[p.ix].ch, int32(p.ix)}
	p.ix++
	p.budget--
	return r
}

func (p *parser) accept(t token.Token) (r Token, _ bool) {
	if p.c() == t {
		return p.consume(), true
	}
	return r, false
}

func (p *parser) expect(t token.Token) (r Token) {
	var ok bool
	if r, ok = p.accept(t); !ok {
		p.isClosed = true
	}
	return r
}

func (p *parser) peek(n int) token.Token {
	for p.ix+n >= len(p.s.toks) {
		if p.budget <= 0 || p.isClosed {
			return EOF
		}

		p.s.scan()
		if p.s.isClosed {
			p.isClosed = true
		}
	}
	p.maxIx = mathutil.Max(p.maxIx, p.ix)
	return token.Token(p.s.toks[p.ix+n].ch)
}

func (p *parser) recordBacktrack(ix int, record bool) {
	delta := p.ix - ix
	p.backs += delta
	if delta > p.maxBack {
		p.maxBack = delta
		p.maxBackRange = [2]int{ix, p.ix}
		p.maxBackOrigin = origin(3)
	}
	p.ix = ix
	if p.record && record {
		if _, _, line, ok := runtime.Caller(2); ok {
			p.a.record(line, delta)
		}
	}
}

func (p *parser) back(ix int) {
	p.recordBacktrack(ix, true)
	if p.isClosed {
		return
	}

	if noBack {
		p.isClosed = true
	}
	if panicBack {
		panic(todo("%v: (%v:)", p.errPosition(), origin(2)))
	}
}

func (p *parser) parse() (ast *AST, err error) {
	if p.c() != PACKAGE {
		p.s.errs.err(p.errPosition(), "syntax error")
		return nil, p.s.errs
	}

	sourceFile := p.sourceFile()
	if p.budget <= 0 {
		return nil, errorf("%s: resources exhausted", p.path)
	}

	if eof, ok := p.accept(EOF); ok && p.ix == len(p.s.toks) {
		return &AST{packageScope: p.packageScope, FileScope: p.fileScope, SourceFile: sourceFile, EOF: eof}, p.s.errs.Err()
	}

	p.s.errs.err(p.errPosition(), "syntax error")
	return nil, p.s.errs
}

type BinaryExpressionNode struct {
	LHS Expression
	Op  Token
	RHS Expression

	typeCache
	valueCache
}

// Position implements Node.
func (n *BinaryExpressionNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LHS.Position()
}

// Source implements Node.
func (n *BinaryExpressionNode) Source(full bool) string { return nodeSource(n, full) }

func (p *parser) additiveExpression(preBlock bool) (r Expression) {
	var multiplicativeExpression Expression
	// ebnf.Sequence MultiplicativeExpression { ( "+" | "-" | "|" | "^" ) MultiplicativeExpression } ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Name MultiplicativeExpression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if multiplicativeExpression = p.multiplicativeExpression(preBlock); multiplicativeExpression == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Repetition { ( "+" | "-" | "|" | "^" ) MultiplicativeExpression } ctx []
		r = multiplicativeExpression
	_0:
		{
			var op Token
			var multiplicativeExpression Expression
			switch p.c() {
			case ADD, OR, SUB, XOR:
				// ebnf.Sequence ( "+" | "-" | "|" | "^" ) MultiplicativeExpression ctx [ADD, OR, SUB, XOR]
				// *ebnf.Group ( "+" | "-" | "|" | "^" ) ctx [ADD, OR, SUB, XOR]
				// ebnf.Alternative "+" | "-" | "|" | "^" ctx [ADD, OR, SUB, XOR]
				op = p.consume()
				// *ebnf.Name MultiplicativeExpression ctx []
				switch p.c() {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
					if multiplicativeExpression = p.multiplicativeExpression(preBlock); multiplicativeExpression == nil {
						p.back(ix)
						goto _1
					}
				default:
					p.back(ix)
					goto _1
				}
				r = &BinaryExpressionNode{LHS: r, Op: op, RHS: multiplicativeExpression}
				goto _0
			}
		_1:
		}
	}
	return r
}

// AliasDeclNode represents the production
//
//	AliasDecl = identifier "=" Type .
type AliasDeclNode struct {
	IDENT    Token
	ASSIGN   Token
	TypeNode Type

	visible
}

// Source implements Node.
func (n *AliasDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *AliasDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IDENT.Position()
}

func (p *parser) aliasDecl() (r *AliasDeclNode) {
	var (
		identTok  Token
		assignTok Token
		typeNode  Type
	)
	// ebnf.Sequence identifier "=" Type ctx [IDENT]
	{
		if p.peek(1) != ASSIGN {
			return nil
		}
		ix := p.ix
		// *ebnf.Name identifier ctx [IDENT]
		identTok = p.expect(IDENT)
		// *ebnf.Token "=" ctx [ASSIGN]
		assignTok = p.expect(ASSIGN)
		// *ebnf.Name Type ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			if typeNode = p.type1(); typeNode == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	r = &AliasDeclNode{
		IDENT:    identTok,
		ASSIGN:   assignTok,
		TypeNode: typeNode,
	}
	p.declare(p.sc, identTok, r, int32(p.ix), false)
	return r
}

// ArgumentsNode represents the production
//
//	Arguments = "(" [ Expression ] ")" .
type ArgumentsNode struct {
	LPAREN     Token
	Expression Expression
	RPAREN     Token
}

// Source implements Node.
func (n *ArgumentsNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ArgumentsNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LPAREN.Position()
}

// Arguments1Node represents the production
//
//	Arguments = "(" [ ( Expression | Type [ "," Expression ] ) [ "..." ] [ "," ] ] ")" .
type Arguments1Node struct {
	LPAREN     Token
	Expression Expression
	TypeNode   Type
	COMMA      Token
	ELLIPSIS   Token
	COMMA2     Token
	RPAREN     Token
}

// Source implements Node.
func (n *Arguments1Node) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *Arguments1Node) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LPAREN.Position()
}

// Arguments2Node represents the production
//
//	Arguments = "(" ExpressionList ")" .
type Arguments2Node struct {
	LPAREN         Token
	ExpressionList *ExpressionListNode
	RPAREN         Token
}

// Source implements Node.
func (n *Arguments2Node) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *Arguments2Node) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LPAREN.Position()
}

// Arguments3Node represents the production
//
//	Arguments = "(" [ ( ExpressionList | Type [ "," ExpressionList ] ) [ "..." ] [ "," ] ] ")" .
type Arguments3Node struct {
	LPAREN         Token
	ExpressionList *ExpressionListNode
	TypeNode       Type
	COMMA          Token
	ELLIPSIS       Token
	COMMA2         Token
	RPAREN         Token
}

// Source implements Node.
func (n *Arguments3Node) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *Arguments3Node) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LPAREN.Position()
}

func (p *parser) arguments() Node {
	var (
		ok             bool
		lparenTok      Token
		expressionList *ExpressionListNode
		typeNode       Type
		commaTok       Token
		ellipsisTok    Token
		comma2Tok      Token
		rparenTok      Token
	)
	// ebnf.Sequence "(" [ ( ExpressionList | Type [ "," ExpressionList ] ) [ "..." ] [ "," ] ] ")" ctx [LPAREN]
	{
		ix := p.ix
		// *ebnf.Token "(" ctx [LPAREN]
		lparenTok = p.expect(LPAREN)
		// *ebnf.Option [ ( ExpressionList | Type [ "," ExpressionList ] ) [ "..." ] [ "," ] ] ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			// ebnf.Sequence ( ExpressionList | Type [ "," ExpressionList ] ) [ "..." ] [ "," ] ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			{
				ix := p.ix
				// *ebnf.Group ( ExpressionList | Type [ "," ExpressionList ] ) ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				// ebnf.Alternative ExpressionList | Type [ "," ExpressionList ] ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				switch p.c() {
				case ADD, AND, CHAR, FLOAT, IMAG, INT, NOT, STRING, SUB, XOR: // 0
					// *ebnf.Name ExpressionList ctx [ADD, AND, CHAR, FLOAT, IMAG, INT, NOT, STRING, SUB, XOR]
					if expressionList = p.expressionList(false); expressionList == nil {
						goto _2
					}
					break
				_2:
					expressionList = nil
					p.back(ix)
					goto _0
				case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT: // 0 1
					// *ebnf.Name ExpressionList ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
					if expressionList = p.expressionList(false); expressionList == nil {
						goto _4
					}
					break
				_4:
					expressionList = nil
					// ebnf.Sequence Type [ "," ExpressionList ] ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
					{
						ix := p.ix
						// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
						if typeNode = p.type1(); typeNode == nil {
							p.back(ix)
							goto _5
						}
						// *ebnf.Option [ "," ExpressionList ] ctx []
						switch p.c() {
						case COMMA:
							// ebnf.Sequence "," ExpressionList ctx [COMMA]
							{
								switch p.peek(1) {
								case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
								default:
									goto _6
								}
								ix := p.ix
								// *ebnf.Token "," ctx [COMMA]
								commaTok = p.expect(COMMA)
								// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
								if expressionList = p.expressionList(false); expressionList == nil {
									p.back(ix)
									goto _6
								}
							}
						}
						goto _7
					_6:
						commaTok = Token{}
						expressionList = nil
					_7:
					}
					break
				_5:
					commaTok = Token{}
					expressionList = nil
					typeNode = nil
					p.back(ix)
					goto _0
				default:
					p.back(ix)
					goto _0
				}
				// *ebnf.Option [ "..." ] ctx []
				switch p.c() {
				case ELLIPSIS:
					// *ebnf.Token "..." ctx [ELLIPSIS]
					ellipsisTok = p.expect(ELLIPSIS)
				}
				// *ebnf.Option [ "," ] ctx []
				switch p.c() {
				case COMMA:
					// *ebnf.Token "," ctx [COMMA]
					comma2Tok = p.expect(COMMA)
				}
			}
		}
		goto _1
	_0:
		comma2Tok = Token{}
		ellipsisTok = Token{}
		expressionList = nil
	_1:
		// *ebnf.Token ")" ctx []
		if rparenTok, ok = p.accept(RPAREN); !ok {
			p.back(ix)
			return nil
		}
	}
	switch expressionList.Len() {
	case 0, 1:
		if typeNode == nil && !commaTok.IsValid() && !ellipsisTok.IsValid() && !comma2Tok.IsValid() {
			return &ArgumentsNode{
				LPAREN:     lparenTok,
				Expression: expressionList.first(),
				RPAREN:     rparenTok,
			}
		}

		return &Arguments1Node{
			LPAREN:     lparenTok,
			Expression: expressionList.first(),
			TypeNode:   typeNode,
			COMMA:      commaTok,
			ELLIPSIS:   ellipsisTok,
			COMMA2:     comma2Tok,
			RPAREN:     rparenTok,
		}
	default:
		if typeNode == nil && !commaTok.IsValid() && !ellipsisTok.IsValid() && !comma2Tok.IsValid() {
			return &Arguments2Node{
				LPAREN:         lparenTok,
				ExpressionList: expressionList,
				RPAREN:         rparenTok,
			}
		}

		return &Arguments3Node{
			LPAREN:         lparenTok,
			ExpressionList: expressionList,
			TypeNode:       typeNode,
			COMMA:          commaTok,
			ELLIPSIS:       ellipsisTok,
			COMMA2:         comma2Tok,
			RPAREN:         rparenTok,
		}
	}
}

func (p *parser) arrayLength() Expression {
	// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	return p.expression(false)
}

// ArrayTypeNode represents the production
//
//	ArrayType = "[" ArrayLength "]" ElementType .
type ArrayTypeNode struct {
	LBRACK      Token
	ArrayLength Expression
	RBRACK      Token
	ElementType Type
}

// Source implements Node.
func (n *ArrayTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ArrayTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LBRACK.Position()
}

func (p *parser) arrayType() *ArrayTypeNode {
	var (
		ok          bool
		lbrackTok   Token
		arrayLength Expression
		rbrackTok   Token
		elementType Type
	)
	// ebnf.Sequence "[" ArrayLength "]" ElementType ctx [LBRACK]
	{
		switch p.peek(1) {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "[" ctx [LBRACK]
		lbrackTok = p.expect(LBRACK)
		// *ebnf.Name ArrayLength ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if arrayLength = p.arrayLength(); arrayLength == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token "]" ctx []
		if rbrackTok, ok = p.accept(RBRACK); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Name ElementType ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			if elementType = p.type1(); elementType == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &ArrayTypeNode{
		LBRACK:      lbrackTok,
		ArrayLength: arrayLength,
		RBRACK:      rbrackTok,
		ElementType: elementType,
	}
}

// AssignmentNode represents the production
//
//	Assignment = ExpressionList ( "=" | "+=" | "-=" | "|=" | "^=" | "*=" | "/=" | "%=" | "<<=" | ">>=" | "&=" | "&^=" ) ExpressionList .
type AssignmentNode struct {
	ExpressionList  *ExpressionListNode
	Op              Token
	ExpressionList2 *ExpressionListNode
}

// Source implements Node.
func (n *AssignmentNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *AssignmentNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) assignment(expressionList *ExpressionListNode, preBlock bool) *AssignmentNode {
	var (
		op              Token
		expressionList2 *ExpressionListNode
	)
	// ebnf.Sequence ( "=" | "+=" | "-=" | "|=" | "^=" | "*=" | "/=" | "%=" | "<<=" | ">>=" | "&=" | "&^=" ) ExpressionList ctx [ADD_ASSIGN, AND_ASSIGN, AND_NOT_ASSIGN, ASSIGN, MUL_ASSIGN, OR_ASSIGN, QUO_ASSIGN, REM_ASSIGN, SHL_ASSIGN, SHR_ASSIGN, SUB_ASSIGN, XOR_ASSIGN]
	{
		ix := p.ix
		// *ebnf.Group ( "=" | "+=" | "-=" | "|=" | "^=" | "*=" | "/=" | "%=" | "<<=" | ">>=" | "&=" | "&^=" ) ctx [ADD_ASSIGN, AND_ASSIGN, AND_NOT_ASSIGN, ASSIGN, MUL_ASSIGN, OR_ASSIGN, QUO_ASSIGN, REM_ASSIGN, SHL_ASSIGN, SHR_ASSIGN, SUB_ASSIGN, XOR_ASSIGN]
		// ebnf.Alternative "=" | "+=" | "-=" | "|=" | "^=" | "*=" | "/=" | "%=" | "<<=" | ">>=" | "&=" | "&^=" ctx [ADD_ASSIGN, AND_ASSIGN, AND_NOT_ASSIGN, ASSIGN, MUL_ASSIGN, OR_ASSIGN, QUO_ASSIGN, REM_ASSIGN, SHL_ASSIGN, SHR_ASSIGN, SUB_ASSIGN, XOR_ASSIGN]
		op = p.consume()
		// *ebnf.Name ExpressionList ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			if expressionList2 = p.expressionList(preBlock); expressionList2 == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &AssignmentNode{
		ExpressionList:  expressionList,
		Op:              op,
		ExpressionList2: expressionList2,
	}
}

// BasicLitNode represents the production
//
//	BasicLit = int_lit | float_lit | imaginary_lit | rune_lit | string_lit .
type BasicLitNode struct {
	Token
	ctx *ctx
}

//TODO- // Source implements Node.
//TODO- func (n *BasicLitNode) Source(full bool) string { return nodeSource(n, full) }
//TODO-
//TODO- // Position implements Node.
//TODO- func (n *BasicLitNode) Position() (r token.Position) {
//TODO- 	if !n.IsValid() {
//TODO- 		return r
//TODO- 	}
//TODO-
//TODO- 	return Token(*n).Position()
//TODO- }
//TODO-
//TODO- func (n *BasicLitNode) Ch() token.Token { return Token(*n).Ch() }
//TODO- func (n *BasicLitNode) IsValid() bool   { return Token(*n).IsValid() }

func (p *parser) basicLit() Expression {
	// ebnf.Alternative int_lit | float_lit | imaginary_lit | rune_lit | string_lit ctx [CHAR, FLOAT, IMAG, INT, STRING]
	t := p.consume()
	v := constant.MakeFromLiteral(t.Src(), token.Token(t.ch), 0)
	if v.Kind() == constant.Unknown {
		p.err(t.Position(), "invalid literal: %s", t.Src())
	}
	return &BasicLitNode{Token: t}
}

// BlockNode represents the production
//
//	Block = "{" StatementList "}" .
type BlockNode struct {
	LBRACE        Token
	StatementList *StatementListNode
	RBRACE        Token
}

// Source implements Node.
func (n *BlockNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *BlockNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LBRACE.Position()
}

func (p *parser) block(rx *ParametersNode, s *SignatureNode) *BlockNode {
	var (
		ok            bool
		lbraceTok     Token
		statementList *StatementListNode
		rbraceTok     Token
	)
	// ebnf.Sequence "{" StatementList "}" ctx [LBRACE]
	{
		p.openScope()

		defer p.closeScope()

		ix := p.ix
		// *ebnf.Token "{" ctx [LBRACE]
		lbraceTok = p.expect(LBRACE)
		if rx != nil {
			rx.declare(p, p.sc)
		}
		if s != nil {
			s.Parameters.declare(p, p.sc)
			if s.Result != nil {
				s.Result.Parameters.declare(p, p.sc)
			}
		}
		// *ebnf.Name StatementList ctx []
		switch p.c() {
		case ADD, AND, ARROW, BREAK, CHAN, CHAR, CONST, CONTINUE, DEFER, FALLTHROUGH, FLOAT, FOR, FUNC, GO, GOTO, IDENT, IF, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, RETURN, SELECT, SEMICOLON, STRING, STRUCT, SUB, SWITCH, TYPE, VAR, XOR /* ε */ :
			if statementList = p.statementList(); statementList == nil {
				p.back(ix)
				return nil
			}
		}
		// *ebnf.Token "}" ctx []
		if rbraceTok, ok = p.accept(RBRACE); !ok {
			p.back(ix)
			return nil
		}
	}
	return &BlockNode{
		LBRACE:        lbraceTok,
		StatementList: statementList,
		RBRACE:        rbraceTok,
	}
}

// BreakStmtNode represents the production
//
//	BreakStmt = "break" [ Label ] .
type BreakStmtNode struct {
	BREAK Token
	Label *LabelNode
}

// Source implements Node.
func (n *BreakStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *BreakStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.BREAK.Position()
}

func (p *parser) breakStmt() *BreakStmtNode {
	var (
		breakTok Token
		label    *LabelNode
	)
	// ebnf.Sequence "break" [ Label ] ctx [BREAK]
	{
		// *ebnf.Token "break" ctx [BREAK]
		breakTok = p.expect(BREAK)
		// *ebnf.Option [ Label ] ctx []
		switch p.c() {
		case IDENT:
			// *ebnf.Name Label ctx [IDENT]
			if label = p.label(); label == nil {
				goto _0
			}
		}
		goto _1
	_0:
		label = nil
	_1:
	}
	return &BreakStmtNode{
		BREAK: breakTok,
		Label: label,
	}
}

func (p *parser) channel() Expression {
	// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	return p.expression(false)
}

// ChannelTypeNode represents the production
//
//	ChannelType = ( "chan" "<-" | "chan" | "<-" "chan" ) ElementType .
type ChannelTypeNode struct {
	CHAN        Token
	ARROW       Token
	ElementType Type
}

// Source implements Node.
func (n *ChannelTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ChannelTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.ARROW.IsValid() && n.ARROW.index < n.CHAN.index {
		return n.ARROW.Position()
	}

	return n.CHAN.Position()
}

func (p *parser) channelType() *ChannelTypeNode {
	var (
		chanTok     Token
		arrowTok    Token
		elementType Type
	)
	// ebnf.Sequence ( "chan" "<-" | "chan" | "<-" "chan" ) ElementType ctx [ARROW, CHAN]
	{
		ix := p.ix
		// *ebnf.Group ( "chan" "<-" | "chan" | "<-" "chan" ) ctx [ARROW, CHAN]
		// ebnf.Alternative "chan" "<-" | "chan" | "<-" "chan" ctx [ARROW, CHAN]
		switch p.c() {
		case CHAN: // 0 1
			// ebnf.Sequence "chan" "<-" ctx [CHAN]
			{
				if p.peek(1) != ARROW {
					goto _0
				}
				// *ebnf.Token "chan" ctx [CHAN]
				chanTok = p.expect(CHAN)
				// *ebnf.Token "<-" ctx [ARROW]
				arrowTok = p.expect(ARROW)
			}
			break
		_0:
			arrowTok = Token{}
			chanTok = Token{}
			// *ebnf.Token "chan" ctx [CHAN]
			chanTok = p.expect(CHAN)
			break
			p.back(ix)
			return nil
		case ARROW: // 2
			// ebnf.Sequence "<-" "chan" ctx [ARROW]
			{
				if p.peek(1) != CHAN {
					goto _2
				}
				// *ebnf.Token "<-" ctx [ARROW]
				arrowTok = p.expect(ARROW)
				// *ebnf.Token "chan" ctx [CHAN]
				chanTok = p.expect(CHAN)
			}
			break
		_2:
			p.back(ix)
			return nil
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Name ElementType ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			if elementType = p.type1(); elementType == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &ChannelTypeNode{
		CHAN:        chanTok,
		ARROW:       arrowTok,
		ElementType: elementType,
	}
}

// CommCaseNode represents the production
//
//	CommCase = "case" ( SendStmt | RecvStmt ) | "default" .
type CommCaseNode struct {
	CASE     Token
	SendStmt *SendStmtNode
	RecvStmt *RecvStmtNode
	DEFAULT  Token
}

// Source implements Node.
func (n *CommCaseNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *CommCaseNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) commCase() *CommCaseNode {
	var (
		caseTok    Token
		sendStmt   *SendStmtNode
		recvStmt   *RecvStmtNode
		defaultTok Token
	)
	// ebnf.Alternative "case" ( SendStmt | RecvStmt ) | "default" ctx [CASE, DEFAULT]
	switch p.c() {
	case CASE: // 0
		// ebnf.Sequence "case" ( SendStmt | RecvStmt ) ctx [CASE]
		{
			switch p.peek(1) {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			default:
				goto _0
			}
			ix := p.ix
			// *ebnf.Token "case" ctx [CASE]
			caseTok = p.expect(CASE)
			// *ebnf.Group ( SendStmt | RecvStmt ) ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			// ebnf.Alternative SendStmt | RecvStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR: // 0 1
				// *ebnf.Name SendStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if sendStmt = p.sendStmt(); sendStmt == nil {
					goto _2
				}
				break
			_2:
				sendStmt = nil
				// *ebnf.Name RecvStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if recvStmt = p.recvStmt(); recvStmt == nil {
					goto _3
				}
				break
			_3:
				recvStmt = nil
				p.back(ix)
				goto _0
			default:
				p.back(ix)
				goto _0
			}
		}
		break
	_0:
		caseTok = Token{}
		return nil
	case DEFAULT: // 1
		// *ebnf.Token "default" ctx [DEFAULT]
		defaultTok = p.expect(DEFAULT)
	default:
		return nil
	}
	return &CommCaseNode{
		CASE:     caseTok,
		SendStmt: sendStmt,
		RecvStmt: recvStmt,
		DEFAULT:  defaultTok,
	}
}

// CommClauseNode represents the production
//
//	CommClause = CommCase ":" StatementList .
type CommClauseNode struct {
	CommCase      *CommCaseNode
	COLON         Token
	StatementList *StatementListNode
}

// Source implements Node.
func (n *CommClauseNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *CommClauseNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.CommCase.Position()
}

func (p *parser) commClause() *CommClauseNode {
	var (
		ok            bool
		commCase      *CommCaseNode
		colonTok      Token
		statementList *StatementListNode
	)
	// ebnf.Sequence CommCase ":" StatementList ctx [CASE, DEFAULT]
	{
		p.openScope()

		defer p.closeScope()

		ix := p.ix
		// *ebnf.Name CommCase ctx [CASE, DEFAULT]
		if commCase = p.commCase(); commCase == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token ":" ctx []
		if colonTok, ok = p.accept(COLON); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Name StatementList ctx []
		switch p.c() {
		case ADD, AND, ARROW, BREAK, CHAN, CHAR, CONST, CONTINUE, DEFER, FALLTHROUGH, FLOAT, FOR, FUNC, GO, GOTO, IDENT, IF, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, RETURN, SELECT, SEMICOLON, STRING, STRUCT, SUB, SWITCH, TYPE, VAR, XOR /* ε */ :
			if statementList = p.statementList(); statementList == nil {
				p.back(ix)
				return nil
			}
		}
	}
	return &CommClauseNode{
		CommCase:      commCase,
		COLON:         colonTok,
		StatementList: statementList,
	}
}

// CompositeLitNode represents the production
//
//	CompositeLit = LiteralType LiteralValue .
type CompositeLitNode struct {
	LiteralType  Node
	LiteralValue *LiteralValueNode

	typeCache
}

// Source implements Node.
func (n *CompositeLitNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *CompositeLitNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LiteralType.Position()
}

func (p *parser) compositeLit() *CompositeLitNode {
	var (
		literalType  Node
		literalValue *LiteralValueNode
	)
	// ebnf.Sequence LiteralType LiteralValue ctx [LBRACK, MAP, STRUCT]
	{
		ix := p.ix
		// *ebnf.Name LiteralType ctx [LBRACK, MAP, STRUCT]
		if literalType = p.literalType(); literalType == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Name LiteralValue ctx []
		switch p.c() {
		case LBRACE:
			if literalValue = p.literalValue(); literalValue == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &CompositeLitNode{
		LiteralType:  literalType,
		LiteralValue: literalValue,
	}
}

func (p *parser) condition(preBlock bool) Expression {
	// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	return p.expression(preBlock)
}

// ConstSpecListNode represents the production
//
//	ConstSpecListNode = { ConstSpec ";" } .
type ConstSpecListNode struct {
	ConstSpec Node
	SEMICOLON Token
	List      *ConstSpecListNode
}

// Source implements Node.
func (n *ConstSpecListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ConstSpecListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.ConstSpec.Position()
}

// ConstDeclNode represents the production
//
//	ConstDecl = "const" ( ConstSpec | "(" { ConstSpec ";" } ")" ) .
type ConstDeclNode struct {
	CONST     Token
	LPAREN    Token
	ConstSpec Node
	RPAREN    Token
}

// Source implements Node.
func (n *ConstDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ConstDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.CONST.Position()
}

func (p *parser) constDecl() *ConstDeclNode {
	var (
		ok        bool
		constTok  Token
		constSpec Node
		lparenTok Token
		list      *ConstSpecListNode
		rparenTok Token
		iota      int64
	)
	// ebnf.Sequence "const" ( ConstSpec | "(" { ConstSpec ";" } [ ConstSpec ] ")" ) ctx [CONST]
	{
		switch p.peek(1) {
		case IDENT, LPAREN:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "const" ctx [CONST]
		constTok = p.expect(CONST)
		// *ebnf.Group ( ConstSpec | "(" { ConstSpec ";" } [ ConstSpec ] ")" ) ctx [IDENT, LPAREN]
		// ebnf.Alternative ConstSpec | "(" { ConstSpec ";" } [ ConstSpec ] ")" ctx [IDENT, LPAREN]
		switch p.c() {
		case IDENT: // 0
			// *ebnf.Name ConstSpec ctx [IDENT]
			if constSpec = p.constSpec(iota); constSpec == nil {
				goto _0
			}
			list = &ConstSpecListNode{
				ConstSpec: constSpec,
			}
			break
		_0:
			constSpec = nil
			p.back(ix)
			return nil
		case LPAREN: // 1
			// ebnf.Sequence "(" { ConstSpec ";" } [ ConstSpec ] ")" ctx [LPAREN]
			{
				ix := p.ix
				// *ebnf.Token "(" ctx [LPAREN]
				lparenTok = p.expect(LPAREN)
				// *ebnf.Repetition { ConstSpec ";" } ctx []
				var item *ConstSpecListNode
			_4:
				{
					var constSpec Node
					var semicolonTok Token
					switch p.c() {
					case IDENT:
						// ebnf.Sequence ConstSpec ";" ctx [IDENT]
						ix := p.ix
						// *ebnf.Name ConstSpec ctx [IDENT]
						if constSpec = p.constSpec(iota); constSpec == nil {
							p.back(ix)
							goto _5
						}
						if p.c() == RPAREN {
							next := &ConstSpecListNode{
								ConstSpec: constSpec,
							}
							if item != nil {
								item.List = next
							}
							item = next
							if list == nil {
								list = item
							}
							break
						}

						// *ebnf.Token ";" ctx []
						if semicolonTok, ok = p.accept(SEMICOLON); !ok {
							p.back(ix)
							goto _5
						}
						next := &ConstSpecListNode{
							ConstSpec: constSpec,
							SEMICOLON: semicolonTok,
						}
						iota++
						if item != nil {
							item.List = next
						}
						item = next
						if list == nil {
							list = item
						}
						goto _4
					}
				_5:
				}
				if rparenTok, ok = p.accept(RPAREN); !ok {
					p.back(ix)
					goto _2
				}
			}
			break
		_2:
			lparenTok = Token{}
			rparenTok = Token{}
			p.back(ix)
			return nil
		default:
			p.back(ix)
			return nil
		}
	}
	if list != nil && list.List == nil && !list.SEMICOLON.IsValid() {
		return &ConstDeclNode{
			CONST:     constTok,
			LPAREN:    lparenTok,
			ConstSpec: list.ConstSpec,
			RPAREN:    rparenTok,
		}
	}

	return &ConstDeclNode{
		CONST:     constTok,
		LPAREN:    lparenTok,
		ConstSpec: list,
		RPAREN:    rparenTok,
	}
}

// ConstSpecNode represents the production
//
//	ConstSpec = Identifier [ [ Type ] "=" Expression ] .
type ConstSpecNode struct {
	IDENT      Token
	TypeNode   Type
	ASSIGN     Token
	Expression Expression

	iota int64
	visible

	guard
}

// Source implements Node.
func (n *ConstSpecNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ConstSpecNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IDENT.Position()
}

// ConstSpec2Node represents the production
//
//	ConstSpec = IdentifierList [ [ Type ] "=" ExpressionList ] .
type ConstSpec2Node struct {
	IdentifierList *IdentifierListNode
	TypeNode       Type
	ASSIGN         Token
	ExpressionList *ExpressionListNode

	iota int64

	visible
}

// Source implements Node.
func (n *ConstSpec2Node) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ConstSpec2Node) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IdentifierList.Position()
}

func (p *parser) constSpec(iota int64) Node {
	var (
		ok             bool
		identifierList *IdentifierListNode
		typeNode       Type
		assignTok      Token
		expressionList *ExpressionListNode
	)
	// ebnf.Sequence IdentifierList [ [ Type ] "=" ExpressionList ] ctx [IDENT]
	{
		ix := p.ix
		// *ebnf.Name IdentifierList ctx [IDENT]
		if identifierList = p.identifierList(); identifierList == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ [ Type ] "=" ExpressionList ] ctx []
		switch p.c() {
		case ARROW, ASSIGN, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			// ebnf.Sequence [ Type ] "=" ExpressionList ctx [ARROW, ASSIGN, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
			{
				ix := p.ix
				// *ebnf.Option [ Type ] ctx [ARROW, ASSIGN, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
				switch p.c() {
				case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
					// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
					if typeNode = p.type1(); typeNode == nil {
						goto _2
					}
				}
				goto _3
			_2:
				typeNode = nil
			_3:
				// *ebnf.Token "=" ctx []
				if assignTok, ok = p.accept(ASSIGN); !ok {
					p.back(ix)
					goto _0
				}
				// *ebnf.Name ExpressionList ctx []
				switch p.c() {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
					if expressionList = p.expressionList(false); expressionList == nil {
						p.back(ix)
						goto _0
					}
				default:
					p.back(ix)
					goto _0
				}
			}
		}
		goto _1
	_0:
		assignTok = Token{}
		expressionList = nil
		typeNode = nil
	_1:
	}
	sc := p.sc
	visible := int32(p.ix)
	if expressionList.Len() < 2 && identifierList.Len() < 2 {
		r := &ConstSpecNode{
			IDENT:      identifierList.first(),
			TypeNode:   typeNode,
			ASSIGN:     assignTok,
			Expression: expressionList.first(),
			iota:       iota,
		}
		ids := identifierList.Len()
		exprs := expressionList.Len()
		if exprs != 0 && ids != exprs {
			p.err(r.ASSIGN.Position(), "different number of identifiers and expressions: %v %v", ids, exprs)
		}
		for l := identifierList; l != nil; l = l.List {
			p.declare(sc, l.IDENT, r, visible, false)
		}
		return r
	}

	r := &ConstSpec2Node{
		IdentifierList: identifierList,
		TypeNode:       typeNode,
		ASSIGN:         assignTok,
		ExpressionList: expressionList,
		iota:           iota,
	}
	ids := r.IdentifierList.Len()
	exprs := r.ExpressionList.Len()
	if exprs != 0 && ids != exprs {
		p.err(r.ASSIGN.Position(), "different number of identifiers and expressions: %v %v", ids, exprs)
	}
	for l := r.IdentifierList; l != nil; l = l.List {
		p.declare(sc, l.IDENT, r, visible, false)
	}
	return r
}

// ContinueStmtNode represents the production
//
//	ContinueStmt = "continue" [ Label ] .
type ContinueStmtNode struct {
	CONTINUE Token
	Label    *LabelNode
}

// Source implements Node.
func (n *ContinueStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ContinueStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.CONTINUE.Position()
}

func (p *parser) continueStmt() *ContinueStmtNode {
	var (
		continueTok Token
		label       *LabelNode
	)
	// ebnf.Sequence "continue" [ Label ] ctx [CONTINUE]
	{
		// *ebnf.Token "continue" ctx [CONTINUE]
		continueTok = p.expect(CONTINUE)
		// *ebnf.Option [ Label ] ctx []
		switch p.c() {
		case IDENT:
			// *ebnf.Name Label ctx [IDENT]
			if label = p.label(); label == nil {
				goto _0
			}
		}
		goto _1
	_0:
		label = nil
	_1:
	}
	return &ContinueStmtNode{
		CONTINUE: continueTok,
		Label:    label,
	}
}

// ConversionNode represents the production
//
//	Conversion = Type "(" Expression [ "," ] ")" .
type ConversionNode struct {
	TypeNode   Type
	LPAREN     Token
	Expression Expression
	COMMA      Token
	RPAREN     Token

	valueCache
}

// Source implements Node.
func (n *ConversionNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ConversionNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TypeNode.Position()
}

func (p *parser) conversion() *ConversionNode {
	var (
		ok         bool
		typeNode   Type
		lparenTok  Token
		expression Expression
		commaTok   Token
		rparenTok  Token
	)
	// ebnf.Sequence Type "(" Expression [ "," ] ")" ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
	{
		ix := p.ix
		// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		if typeNode = p.type1(); typeNode == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token "(" ctx []
		if lparenTok, ok = p.accept(LPAREN); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Name Expression ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			if expression = p.expression(false); expression == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ "," ] ctx []
		switch p.c() {
		case COMMA:
			// *ebnf.Token "," ctx [COMMA]
			commaTok = p.expect(COMMA)
		}
		// *ebnf.Token ")" ctx []
		if rparenTok, ok = p.accept(RPAREN); !ok {
			p.back(ix)
			return nil
		}
	}
	return &ConversionNode{
		TypeNode:   typeNode,
		LPAREN:     lparenTok,
		Expression: expression,
		COMMA:      commaTok,
		RPAREN:     rparenTok,
	}
}

func (p *parser) declaration() Node {
	// ebnf.Alternative ConstDecl | TypeDecl | VarDecl ctx [CONST, TYPE, VAR]
	switch p.c() {
	case CONST: // 0
		// *ebnf.Name ConstDecl ctx [CONST]
		return p.constDecl()
	case TYPE: // 1
		// *ebnf.Name TypeDecl ctx [TYPE]
		return p.typeDecl()
	case VAR: // 2
		// *ebnf.Name VarDecl ctx [VAR]
		return p.varDecl()
	default:
		return nil
	}
}

// DeferStmtNode represents the production
//
//	DeferStmt = "defer" Expression .
type DeferStmtNode struct {
	DEFER      Token
	Expression Expression
}

// Source implements Node.
func (n *DeferStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *DeferStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.DEFER.Position()
}

func (p *parser) deferStmt() *DeferStmtNode {
	var (
		deferTok   Token
		expression Expression
	)
	// ebnf.Sequence "defer" Expression ctx [DEFER]
	{
		switch p.peek(1) {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "defer" ctx [DEFER]
		deferTok = p.expect(DEFER)
		// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if expression = p.expression(false); expression == nil {
			p.back(ix)
			return nil
		}
	}
	return &DeferStmtNode{
		DEFER:      deferTok,
		Expression: expression,
	}
}

func (p *parser) element() Expression {
	var (
		expression   Expression
		literalValue *LiteralValueNode
	)
	// ebnf.Alternative Expression | LiteralValue ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	switch p.c() {
	case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR: // 0
		// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if expression = p.expression(false); expression != nil {
			return expression
		}
	case LBRACE: // 1
		// *ebnf.Name LiteralValue ctx [LBRACE]
		if literalValue = p.literalValue(); literalValue != nil {
			return literalValue
		}
	}
	return nil
}

// EmbeddedFieldNode represents the production
//
//	EmbeddedField = [ "*" ] TypeName [ TypeArgs ] .
type EmbeddedFieldNode struct {
	MUL      Token
	TypeName *TypeNameNode
	TypeArgs *TypeArgsNode
}

// Source implements Node.
func (n *EmbeddedFieldNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *EmbeddedFieldNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) embeddedField() *EmbeddedFieldNode {
	var (
		mulTok   Token
		typeName *TypeNameNode
		typeArgs *TypeArgsNode
	)
	// ebnf.Sequence [ "*" ] TypeName [ TypeArgs ] ctx [IDENT, MUL]
	{
		ix := p.ix
		// *ebnf.Option [ "*" ] ctx [IDENT, MUL]
		switch p.c() {
		case MUL:
			// *ebnf.Token "*" ctx [MUL]
			mulTok = p.expect(MUL)
		}
		// *ebnf.Name TypeName ctx []
		switch p.c() {
		case IDENT:
			if typeName = p.typeName(); typeName == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ TypeArgs ] ctx []
		switch p.c() {
		case LBRACK:
			// *ebnf.Name TypeArgs ctx [LBRACK]
			if typeArgs = p.typeArgs(); typeArgs == nil {
				goto _2
			}
		}
		goto _3
	_2:
		typeArgs = nil
	_3:
	}
	return &EmbeddedFieldNode{
		MUL:      mulTok,
		TypeName: typeName,
		TypeArgs: typeArgs,
	}
}

// EmptyStmtNode represents the production
//
//	EmptyStmt =  .
type EmptyStmtNode struct {
}

// Source implements Node.
func (n *EmptyStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *EmptyStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) emptyStmt() *EmptyStmtNode {
	return &EmptyStmtNode{}
}

// ExprCaseClauseListNode represents the production
//
//	ExprCaseClause = ExprSwitchCase ":" StatementList .
type ExprCaseClauseListNode struct {
	ExprSwitchCase Node
	COLON          Token
	StatementList  *StatementListNode
	List           *ExprCaseClauseListNode
}

// Source implements Node.
func (n *ExprCaseClauseListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ExprCaseClauseListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.ExprSwitchCase.Position()
}

func (p *parser) exprCaseClause() *ExprCaseClauseListNode {
	var (
		ok             bool
		exprSwitchCase Node
		colonTok       Token
		statementList  *StatementListNode
	)
	// ebnf.Sequence ExprSwitchCase ":" StatementList ctx [CASE, DEFAULT]
	{
		p.openScope()

		defer p.closeScope()

		ix := p.ix
		// *ebnf.Name ExprSwitchCase ctx [CASE, DEFAULT]
		if exprSwitchCase = p.exprSwitchCase(); exprSwitchCase == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token ":" ctx []
		if colonTok, ok = p.accept(COLON); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Name StatementList ctx []
		switch p.c() {
		case ADD, AND, ARROW, BREAK, CHAN, CHAR, CONST, CONTINUE, DEFER, FALLTHROUGH, FLOAT, FOR, FUNC, GO, GOTO, IDENT, IF, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, RETURN, SELECT, SEMICOLON, STRING, STRUCT, SUB, SWITCH, TYPE, VAR, XOR /* ε */ :
			if statementList = p.statementList(); statementList == nil {
				p.back(ix)
				return nil
			}
		}
	}
	return &ExprCaseClauseListNode{
		ExprSwitchCase: exprSwitchCase,
		COLON:          colonTok,
		StatementList:  statementList,
	}
}

// ExprSwitchCaseNode represents the production
//
//	ExprSwitchCase = "case" ExpressionList | "default" .
type ExprSwitchCaseNode struct {
	CASE       Token
	Expression Expression
	DEFAULT    Token
}

// Source implements Node.
func (n *ExprSwitchCaseNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ExprSwitchCaseNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.CASE.Position()
}

// ExprSwitchCase2Node represents the production
//
//	ExprSwitchCase = "case" ExpressionList | "default" .
type ExprSwitchCase2Node struct {
	CASE           Token
	ExpressionList *ExpressionListNode
	DEFAULT        Token
}

// Source implements Node.
func (n *ExprSwitchCase2Node) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ExprSwitchCase2Node) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.CASE.Position()
}

func (p *parser) exprSwitchCase() Node {
	var (
		caseTok        Token
		expressionList *ExpressionListNode
		defaultTok     Token
	)
	// ebnf.Alternative "case" ExpressionList | "default" ctx [CASE, DEFAULT]
	switch p.c() {
	case CASE: // 0
		// ebnf.Sequence "case" ExpressionList ctx [CASE]
		{
			switch p.peek(1) {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			default:
				goto _0
			}
			ix := p.ix
			// *ebnf.Token "case" ctx [CASE]
			caseTok = p.expect(CASE)
			// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if expressionList = p.expressionList(false); expressionList == nil {
				p.back(ix)
				goto _0
			}
		}
		break
	_0:
		caseTok = Token{}
		expressionList = nil
		return nil
	case DEFAULT: // 1
		// *ebnf.Token "default" ctx [DEFAULT]
		defaultTok = p.expect(DEFAULT)
	default:
		return nil
	}
	if expressionList.Len() == 1 {
		return &ExprSwitchCaseNode{
			CASE:       caseTok,
			Expression: expressionList.first(),
			DEFAULT:    defaultTok,
		}
	}

	return &ExprSwitchCase2Node{
		CASE:           caseTok,
		ExpressionList: expressionList,
		DEFAULT:        defaultTok,
	}
}

// ExprSwitchStmtNode represents the production
//
//	ExprSwitchStmt = "switch" [ Expression ] "{" { ExprCaseClause } "}" | "switch" SimpleStmt ";" [ Expression ] "{" { ExprCaseClause } "}" .
type ExprSwitchStmtNode struct {
	SWITCH             Token
	Expression         Expression
	LBRACE             Token
	ExprCaseClauseList *ExprCaseClauseListNode
	RBRACE             Token
	SimpleStmt         Node
	SEMICOLON          Token
}

// Source implements Node.
func (n *ExprSwitchStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ExprSwitchStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) exprSwitchStmt() *ExprSwitchStmtNode {
	var (
		ok           bool
		switchTok    Token
		expression   Expression
		lbraceTok    Token
		list         *ExprCaseClauseListNode
		rbraceTok    Token
		simpleStmt   Node
		semicolonTok Token
	)
	// ebnf.Alternative "switch" [ Expression ] "{" { ExprCaseClause } "}" | "switch" SimpleStmt ";" [ Expression ] "{" { ExprCaseClause } "}" ctx [SWITCH]
	switch p.c() {
	case SWITCH: // 0 1
		// ebnf.Sequence "switch" [ Expression ] "{" { ExprCaseClause } "}" ctx [SWITCH]
		{
			ix := p.ix
			// *ebnf.Token "switch" ctx [SWITCH]
			switchTok = p.expect(SWITCH)
			// *ebnf.Option [ Expression ] ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if expression = p.expression(true); expression == nil {
					goto _1
				}
			}
			goto _2
		_1:
			expression = nil
		_2:
			// *ebnf.Token "{" ctx []
			if lbraceTok, ok = p.accept(LBRACE); !ok {
				p.back(ix)
				goto _0
			}
			// *ebnf.Repetition { ExprCaseClause } ctx []
			var item *ExprCaseClauseListNode
		_3:
			{
				var exprCaseClause *ExprCaseClauseListNode
				switch p.c() {
				case CASE, DEFAULT:
					// *ebnf.Name ExprCaseClause ctx [CASE, DEFAULT]
					if exprCaseClause = p.exprCaseClause(); exprCaseClause == nil {
						goto _4
					}
					if item != nil {
						item.List = exprCaseClause
					}
					item = exprCaseClause
					if list == nil {
						list = item
					}
					goto _3
				}
			_4:
			}
			// *ebnf.Token "}" ctx []
			if rbraceTok, ok = p.accept(RBRACE); !ok {
				p.back(ix)
				goto _0
			}
		}
		break
	_0:
		expression = nil
		lbraceTok = Token{}
		rbraceTok = Token{}
		switchTok = Token{}
		// ebnf.Sequence "switch" SimpleStmt ";" [ Expression ] "{" { ExprCaseClause } "}" ctx [SWITCH]
		{
			ix := p.ix
			// *ebnf.Token "switch" ctx [SWITCH]
			switchTok = p.expect(SWITCH)
			// *ebnf.Name SimpleStmt ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */ :
				if simpleStmt = p.simpleStmt(false); simpleStmt == nil {
					p.back(ix)
					goto _5
				}
			}
			// *ebnf.Token ";" ctx []
			if semicolonTok, ok = p.accept(SEMICOLON); !ok {
				p.back(ix)
				goto _5
			}
			// *ebnf.Option [ Expression ] ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if expression = p.expression(true); expression == nil {
					goto _6
				}
			}
			goto _7
		_6:
			expression = nil
		_7:
			// *ebnf.Token "{" ctx []
			if lbraceTok, ok = p.accept(LBRACE); !ok {
				p.back(ix)
				goto _5
			}
			// *ebnf.Repetition { ExprCaseClause } ctx []
			var item *ExprCaseClauseListNode
		_8:
			{
				var exprCaseClause *ExprCaseClauseListNode
				switch p.c() {
				case CASE, DEFAULT:
					// *ebnf.Name ExprCaseClause ctx [CASE, DEFAULT]
					if exprCaseClause = p.exprCaseClause(); exprCaseClause == nil {
						goto _9
					}
					if item != nil {
						item.List = exprCaseClause
					}
					item = exprCaseClause
					if list == nil {
						list = item
					}
					goto _8
				}
			_9:
			}
			// *ebnf.Token "}" ctx []
			if rbraceTok, ok = p.accept(RBRACE); !ok {
				p.back(ix)
				goto _5
			}
		}
		break
	_5:
		expression = nil
		lbraceTok = Token{}
		rbraceTok = Token{}
		semicolonTok = Token{}
		simpleStmt = nil
		switchTok = Token{}
		return nil
	default:
		return nil
	}
	return &ExprSwitchStmtNode{
		SWITCH:             switchTok,
		Expression:         expression,
		LBRACE:             lbraceTok,
		ExprCaseClauseList: list,
		RBRACE:             rbraceTok,
		SimpleStmt:         simpleStmt,
		SEMICOLON:          semicolonTok,
	}
}

func (p *parser) expression(preBlock bool) (r Expression) {
	var logicalAndExpression Expression
	// ebnf.Sequence LogicalAndExpression { "||" LogicalAndExpression } ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Name LogicalAndExpression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if logicalAndExpression = p.logicalAndExpression(preBlock); logicalAndExpression == nil {
			p.back(ix)
			return nil
		}
		r = logicalAndExpression
		// *ebnf.Repetition { "||" LogicalAndExpression } ctx []
	_0:
		{
			var lorTok Token
			var logicalAndExpression Expression
			switch p.c() {
			case LOR:
				// ebnf.Sequence "||" LogicalAndExpression ctx [LOR]
				switch p.peek(1) {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				default:
					goto _1
				}
				ix := p.ix
				// *ebnf.Token "||" ctx [LOR]
				lorTok = p.expect(LOR)
				// *ebnf.Name LogicalAndExpression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if logicalAndExpression = p.logicalAndExpression(preBlock); logicalAndExpression == nil {
					p.back(ix)
					goto _1
				}
				r = &BinaryExpressionNode{LHS: r, Op: lorTok, RHS: logicalAndExpression}
				goto _0
			}
		_1:
		}
	}
	return r
}

// ExpressionListNode represents the production
//
//	ExpressionList = Expression { "," Expression } .
type ExpressionListNode struct {
	COMMA      Token
	Expression Expression
	List       *ExpressionListNode
}

func (n *ExpressionListNode) first() Expression {
	if n == nil {
		return nil
	}

	return n.Expression
}

// Len reports the number of items in n.
func (n *ExpressionListNode) Len() (r int) {
	for ; n != nil; n = n.List {
		r++
	}
	return r
}

// Source implements Node.
func (n *ExpressionListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ExpressionListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.COMMA.IsValid() {
		return n.COMMA.Position()
	}

	return n.Expression.Position()
}

func (p *parser) expressionList(preBlock bool) *ExpressionListNode {
	var (
		expression           Expression
		expressionList, last *ExpressionListNode
	)
	// ebnf.Sequence Expression { "," Expression } ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if expression = p.expression(preBlock); expression == nil {
			p.back(ix)
			return nil
		}
		expressionList = &ExpressionListNode{
			Expression: expression,
		}
		last = expressionList
		// *ebnf.Repetition { "," Expression } ctx []
	_0:
		{
			var commaTok Token
			switch p.c() {
			case COMMA:
				// ebnf.Sequence "," Expression ctx [COMMA]
				switch p.peek(1) {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				default:
					goto _1
				}
				ix := p.ix
				// *ebnf.Token "," ctx [COMMA]
				commaTok = p.expect(COMMA)
				// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if expression = p.expression(preBlock); expression == nil {
					p.back(ix)
					goto _1
				}
				next := &ExpressionListNode{
					COMMA:      commaTok,
					Expression: expression,
				}
				last.List = next
				last = next
				goto _0
			}
		_1:
		}
	}
	return expressionList
}

// FallthroughStmtNode represents the production
//
//	FallthroughStmt = "fallthrough" .
type FallthroughStmtNode struct {
	FALLTHROUGH Token
}

// Source implements Node.
func (n *FallthroughStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *FallthroughStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) fallthroughStmt() *FallthroughStmtNode {
	var (
		fallthroughTok Token
	)
	// *ebnf.Token "fallthrough" ctx [FALLTHROUGH]
	fallthroughTok = p.expect(FALLTHROUGH)
	return &FallthroughStmtNode{
		FALLTHROUGH: fallthroughTok,
	}
}

// FieldDeclNode represents the production
//
//	FieldDecl = ( IdentifierList Type | EmbeddedField ) [ Tag ] .
type FieldDeclNode struct {
	IdentifierList *IdentifierListNode
	TypeNode       Type
	EmbeddedField  *EmbeddedFieldNode
	Tag            *TagNode
}

// Source implements Node.
func (n *FieldDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *FieldDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.IdentifierList != nil {
		return n.IdentifierList.Position()
	}

	return n.EmbeddedField.Position()
}

func (p *parser) fieldDecl() *FieldDeclNode {
	var (
		identifierList *IdentifierListNode
		typeNode       Type
		embeddedField  *EmbeddedFieldNode
		tag            *TagNode
	)
	// ebnf.Sequence ( IdentifierList Type | EmbeddedField ) [ Tag ] ctx [IDENT, MUL]
	{
		ix := p.ix
		// *ebnf.Group ( IdentifierList Type | EmbeddedField ) ctx [IDENT, MUL]
		// ebnf.Alternative IdentifierList Type | EmbeddedField ctx [IDENT, MUL]
		switch p.c() {
		case IDENT: // 0 1
			// ebnf.Sequence IdentifierList Type ctx [IDENT]
			{
				ix := p.ix
				// *ebnf.Name IdentifierList ctx [IDENT]
				if identifierList = p.identifierList(); identifierList == nil {
					p.back(ix)
					goto _0
				}
				// *ebnf.Name Type ctx []
				switch p.c() {
				case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
					if typeNode = p.type1(); typeNode == nil {
						p.back(ix)
						goto _0
					}
				default:
					p.back(ix)
					goto _0
				}
			}
			break
		_0:
			identifierList = nil
			typeNode = nil
			// *ebnf.Name EmbeddedField ctx [IDENT]
			if embeddedField = p.embeddedField(); embeddedField == nil {
				goto _1
			}
			break
		_1:
			embeddedField = nil
			p.back(ix)
			return nil
		case MUL: // 1
			// *ebnf.Name EmbeddedField ctx [MUL]
			if embeddedField = p.embeddedField(); embeddedField == nil {
				goto _2
			}
			break
		_2:
			embeddedField = nil
			p.back(ix)
			return nil
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ Tag ] ctx []
		switch p.c() {
		case STRING:
			// *ebnf.Name Tag ctx [STRING]
			if tag = p.tag(); tag == nil {
				goto _4
			}
		}
		goto _5
	_4:
		tag = nil
	_5:
	}
	return &FieldDeclNode{
		IdentifierList: identifierList,
		TypeNode:       typeNode,
		EmbeddedField:  embeddedField,
		Tag:            tag,
	}
}

// ForClauseNode represents the production
//
//	ForClause = [ InitStmt ] ";" [ Condition ] ";" [ PostStmt ] .
type ForClauseNode struct {
	InitStmt   Node
	SEMICOLON  Token
	Condition  Expression
	SEMICOLON2 Token
	PostStmt   Node
}

// Source implements Node.
func (n *ForClauseNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ForClauseNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) forClause() *ForClauseNode {
	var (
		ok            bool
		initStmt      Node
		semicolonTok  Token
		condition     Expression
		semicolon2Tok Token
		postStmt      Node
	)
	// ebnf.Sequence [ InitStmt ] ";" [ Condition ] ";" [ PostStmt ] ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, SEMICOLON, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Option [ InitStmt ] ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, SEMICOLON, STRING, STRUCT, SUB, XOR]
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			// *ebnf.Name InitStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if initStmt = p.initStmt(); initStmt == nil {
				goto _0
			}
		}
		goto _1
	_0:
		initStmt = nil
	_1:
		// *ebnf.Token ";" ctx []
		if semicolonTok, ok = p.accept(SEMICOLON); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ Condition ] ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			// *ebnf.Name Condition ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if condition = p.condition(false); condition == nil {
				goto _2
			}
		}
		goto _3
	_2:
		condition = nil
	_3:
		// *ebnf.Token ";" ctx []
		if semicolon2Tok, ok = p.accept(SEMICOLON); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ PostStmt ] ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */ :
			// *ebnf.Name PostStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */]
			if postStmt = p.postStmt(); postStmt == nil {
				goto _4
			}
		}
		goto _5
	_4:
		postStmt = nil
	_5:
	}
	return &ForClauseNode{
		InitStmt:   initStmt,
		SEMICOLON:  semicolonTok,
		Condition:  condition,
		SEMICOLON2: semicolon2Tok,
		PostStmt:   postStmt,
	}
}

// ForStmtNode represents the production
//
//	ForStmt = "for" [ ForClause | RangeClause | Condition ] Block .
type ForStmtNode struct {
	FOR         Token
	ForClause   *ForClauseNode
	RangeClause *RangeClauseNode
	Condition   Expression
	Block       *BlockNode
}

// Source implements Node.
func (n *ForStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ForStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.FOR.Position()
}

func (p *parser) forStmt() *ForStmtNode {
	var (
		forTok      Token
		forClause   *ForClauseNode
		rangeClause *RangeClauseNode
		condition   Expression
		block       *BlockNode
	)
	// ebnf.Sequence "for" [ ForClause | RangeClause | Condition ] Block ctx [FOR]
	{
		p.openScope()

		defer p.closeScope()

		ix := p.ix
		// *ebnf.Token "for" ctx [FOR]
		forTok = p.expect(FOR)
		// *ebnf.Option [ ForClause | RangeClause | Condition ] ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, RANGE, SEMICOLON, STRING, STRUCT, SUB, XOR:
			// ebnf.Alternative ForClause | RangeClause | Condition ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, RANGE, SEMICOLON, STRING, STRUCT, SUB, XOR]
			switch p.c() {
			case SEMICOLON: // 0
				// *ebnf.Name ForClause ctx [SEMICOLON]
				if forClause = p.forClause(); forClause == nil {
					goto _2
				}
				break
			_2:
				forClause = nil
				goto _0
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR: // 0 1 2
				// *ebnf.Name ForClause ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if forClause = p.forClause(); forClause == nil {
					goto _4
				}
				break
			_4:
				forClause = nil
				// *ebnf.Name RangeClause ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if rangeClause = p.rangeClause(); rangeClause == nil {
					goto _5
				}
				break
			_5:
				rangeClause = nil
				// *ebnf.Name Condition ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if condition = p.condition(true); condition == nil {
					goto _6
				}
				break
			_6:
				condition = nil
				goto _0
			case RANGE: // 1
				// *ebnf.Name RangeClause ctx [RANGE]
				if rangeClause = p.rangeClause(); rangeClause == nil {
					goto _7
				}
				break
			_7:
				rangeClause = nil
				goto _0
			default:
				goto _0
			}
		}
		goto _1
	_0:
		forClause = nil
		rangeClause = nil
	_1:
		// *ebnf.Name Block ctx []
		switch p.c() {
		case LBRACE:
			if block = p.block(nil, nil); block == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &ForStmtNode{
		FOR:         forTok,
		ForClause:   forClause,
		RangeClause: rangeClause,
		Condition:   condition,
		Block:       block,
	}
}

// FunctionBodyNode represents the production
//
//	FunctionBody = Block .
type FunctionBodyNode struct {
	Block *BlockNode
}

// Source implements Node.
func (n *FunctionBodyNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *FunctionBodyNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.Block.Position()
}

func (p *parser) functionBody(rx *ParametersNode, s *SignatureNode) *FunctionBodyNode {
	var (
		block *BlockNode
	)
	// *ebnf.Name Block ctx [LBRACE]
	if block = p.block(rx, s); block == nil {
		return nil
	}
	return &FunctionBodyNode{
		Block: block,
	}
}

// FunctionDeclNode represents the production
//
//	FunctionDecl = "func" FunctionName [ TypeParameters ] Signature [ FunctionBody ] .
type FunctionDeclNode struct {
	FUNC           Token
	FunctionName   *FunctionNameNode
	TypeParameters *TypeParametersNode
	Signature      *SignatureNode
	FunctionBody   *FunctionBodyNode

	visible
}

// Source implements Node.
func (n *FunctionDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *FunctionDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.FUNC.Position()
}

func (p *parser) functionDecl() (r *FunctionDeclNode) {
	var (
		funcTok        Token
		functionName   *FunctionNameNode
		typeParameters *TypeParametersNode
		signature      *SignatureNode
		functionBody   *FunctionBodyNode
	)
	// ebnf.Sequence "func" FunctionName [ TypeParameters ] Signature [ FunctionBody ] ctx [FUNC]
	{
		if p.peek(1) != IDENT {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "func" ctx [FUNC]
		funcTok = p.expect(FUNC)
		// *ebnf.Name FunctionName ctx [IDENT]
		if functionName = p.functionName(); functionName == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ TypeParameters ] ctx []
		switch p.c() {
		case LBRACK:
			// *ebnf.Name TypeParameters ctx [LBRACK]
			if typeParameters = p.typeParameters(); typeParameters == nil {
				goto _0
			}
		}
		goto _1
	_0:
		typeParameters = nil
	_1:
		// *ebnf.Name Signature ctx []
		switch p.c() {
		case LPAREN:
			if signature = p.signature(); signature == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ FunctionBody ] ctx []
		switch p.c() {
		case LBRACE:
			// *ebnf.Name FunctionBody ctx [LBRACE]
			if functionBody = p.functionBody(nil, signature); functionBody == nil {
				goto _2
			}
		}
		goto _3
	_2:
		functionBody = nil
	_3:
	}
	sc := p.sc
	r = &FunctionDeclNode{
		FUNC:           funcTok,
		FunctionName:   functionName,
		TypeParameters: typeParameters,
		Signature:      signature,
		FunctionBody:   functionBody,
	}
	p.declare(sc, functionName.IDENT, r, 0, true)
	return r
}

// FunctionLitNode represents the production
//
//	FunctionLit = "func" Signature FunctionBody .
type FunctionLitNode struct {
	FUNC         Token
	Signature    *SignatureNode
	FunctionBody *FunctionBodyNode
}

// Source implements Node.
func (n *FunctionLitNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *FunctionLitNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.FUNC.Position()
}

func (p *parser) functionLit() *FunctionLitNode {
	var (
		funcTok      Token
		signature    *SignatureNode
		functionBody *FunctionBodyNode
	)
	// ebnf.Sequence "func" Signature FunctionBody ctx [FUNC]
	{
		switch p.peek(1) {
		case LPAREN:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "func" ctx [FUNC]
		funcTok = p.expect(FUNC)
		// *ebnf.Name Signature ctx [LPAREN]
		if signature = p.signature(); signature == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Name FunctionBody ctx []
		switch p.c() {
		case LBRACE:
			if functionBody = p.functionBody(nil, signature); functionBody == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &FunctionLitNode{
		FUNC:         funcTok,
		Signature:    signature,
		FunctionBody: functionBody,
	}
}

// FunctionNameNode represents the production
//
//	FunctionName = identifier .
type FunctionNameNode struct {
	IDENT Token
}

// Source implements Node.
func (n *FunctionNameNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *FunctionNameNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IDENT.Position()
}

func (p *parser) functionName() *FunctionNameNode {
	var (
		identTok Token
	)
	// *ebnf.Name identifier ctx [IDENT]
	identTok = p.expect(IDENT)
	return &FunctionNameNode{
		IDENT: identTok,
	}
}

// FunctionTypeNode represents the production
//
//	FunctionType = "func" Signature .
type FunctionTypeNode struct {
	FUNC      Token
	Signature *SignatureNode

	guard
}

// Source implements Node.
func (n *FunctionTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *FunctionTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.FUNC.Position()
}

func (p *parser) functionType() *FunctionTypeNode {
	var (
		funcTok   Token
		signature *SignatureNode
	)
	// ebnf.Sequence "func" Signature ctx [FUNC]
	{
		switch p.peek(1) {
		case LPAREN:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "func" ctx [FUNC]
		funcTok = p.expect(FUNC)
		// *ebnf.Name Signature ctx [LPAREN]
		if signature = p.signature(); signature == nil {
			p.back(ix)
			return nil
		}
	}
	return &FunctionTypeNode{
		FUNC:      funcTok,
		Signature: signature,
	}
}

// GoStmtNode represents the production
//
//	GoStmt = "go" Expression .
type GoStmtNode struct {
	GO         Token
	Expression Expression
}

// Source implements Node.
func (n *GoStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *GoStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.GO.Position()
}

func (p *parser) goStmt() *GoStmtNode {
	var (
		goTok      Token
		expression Expression
	)
	// ebnf.Sequence "go" Expression ctx [GO]
	{
		switch p.peek(1) {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "go" ctx [GO]
		goTok = p.expect(GO)
		// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if expression = p.expression(false); expression == nil {
			p.back(ix)
			return nil
		}
	}
	return &GoStmtNode{
		GO:         goTok,
		Expression: expression,
	}
}

// GotoStmtNode represents the production
//
//	GotoStmt = "goto" Label .
type GotoStmtNode struct {
	GOTO  Token
	Label *LabelNode
}

// Source implements Node.
func (n *GotoStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *GotoStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.GOTO.Position()
}

func (p *parser) gotoStmt() *GotoStmtNode {
	var (
		gotoTok Token
		label   *LabelNode
	)
	// ebnf.Sequence "goto" Label ctx [GOTO]
	{
		if p.peek(1) != IDENT {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "goto" ctx [GOTO]
		gotoTok = p.expect(GOTO)
		// *ebnf.Name Label ctx [IDENT]
		if label = p.label(); label == nil {
			p.back(ix)
			return nil
		}
	}
	return &GotoStmtNode{
		GOTO:  gotoTok,
		Label: label,
	}
}

// IdentifierListNode represents the production
//
//	IdentifierList = identifier { "," identifier } .
type IdentifierListNode struct {
	COMMA Token
	IDENT Token
	List  *IdentifierListNode
}

// Len reports the number of items in n.
func (n *IdentifierListNode) Len() (r int) {
	for ; n != nil; n = n.List {
		r++
	}
	return r
}

func (n *IdentifierListNode) first() (r Token) {
	if n != nil {
		r = n.IDENT
	}
	return r
}

// Source implements Node.
func (n *IdentifierListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *IdentifierListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.COMMA.IsValid() {
		return n.COMMA.Position()
	}

	return n.IDENT.Position()
}

func (p *parser) identifierList() *IdentifierListNode {
	var (
		identTok   Token
		list, last *IdentifierListNode
	)
	// ebnf.Sequence identifier { "," identifier } ctx [IDENT]
	{
		// *ebnf.Name identifier ctx [IDENT]
		identTok = p.expect(IDENT)
		// *ebnf.Repetition { "," identifier } ctx []
		list = &IdentifierListNode{
			IDENT: identTok,
		}
		last = list
	_0:
		{
			var commaTok Token
			var identTok Token
			switch p.c() {
			case COMMA:
				// ebnf.Sequence "," identifier ctx [COMMA]
				if p.peek(1) != IDENT {
					goto _1
				}
				// *ebnf.Token "," ctx [COMMA]
				commaTok = p.expect(COMMA)
				// *ebnf.Name identifier ctx [IDENT]
				identTok = p.expect(IDENT)
				next := &IdentifierListNode{
					COMMA: commaTok,
					IDENT: identTok,
				}
				last.List = next
				last = next
				goto _0
			}
		_1:
		}
	}
	return list
}

// IfElseStmtNode represents the production
//
//	IfStmt = "if" Expression Block [ "else" ( IfStmt | Block ) ] | "if" SimpleStmt ";" Expression Block [ "else" ( IfStmt | Block ) ] .
type IfElseStmtNode struct {
	IF         Token
	Expression Expression
	Block      *BlockNode
	ELSE       Token
	ElseClause Node
	SimpleStmt Node
	SEMICOLON  Token
}

// Source implements Node.
func (n *IfElseStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *IfElseStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

// IfStmtNode represents the production
//
//	IfStmt = "if" Expression Block [ "else" ( IfStmt | Block ) ] | "if" SimpleStmt ";" Expression Block [ "else" ( IfStmt | Block ) ] .
type IfStmtNode struct {
	IF         Token
	Expression Expression
	Block      *BlockNode
	SimpleStmt Node
	SEMICOLON  Token
}

// Source implements Node.
func (n *IfStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *IfStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) ifStmt() Node {
	var (
		ok           bool
		ifTok        Token
		expression   Expression
		block        *BlockNode
		elseTok      Token
		ifStmt       Node
		block2       *BlockNode
		simpleStmt   Node
		semicolonTok Token
	)
	// ebnf.Alternative "if" Expression Block [ "else" ( IfStmt | Block ) ] | "if" SimpleStmt ";" Expression Block [ "else" ( IfStmt | Block ) ] ctx [IF]
	switch p.c() {
	case IF: // 0 1
		p.openScope()

		defer p.closeScope()

		// ebnf.Sequence "if" Expression Block [ "else" ( IfStmt | Block ) ] ctx [IF]
		{
			switch p.peek(1) {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			default:
				goto _0
			}
			ix := p.ix
			// *ebnf.Token "if" ctx [IF]
			ifTok = p.expect(IF)
			// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if expression = p.expression(true); expression == nil {
				p.back(ix)
				goto _0
			}
			// *ebnf.Name Block ctx []
			switch p.c() {
			case LBRACE:
				if block = p.block(nil, nil); block == nil {
					p.back(ix)
					goto _0
				}
			default:
				p.back(ix)
				goto _0
			}
			// *ebnf.Option [ "else" ( IfStmt | Block ) ] ctx []
			switch p.c() {
			case ELSE:
				// ebnf.Sequence "else" ( IfStmt | Block ) ctx [ELSE]
				{
					switch p.peek(1) {
					case IF, LBRACE:
					default:
						goto _1
					}
					ix := p.ix
					// *ebnf.Token "else" ctx [ELSE]
					elseTok = p.expect(ELSE)
					// *ebnf.Group ( IfStmt | Block ) ctx [IF, LBRACE]
					// ebnf.Alternative IfStmt | Block ctx [IF, LBRACE]
					switch p.c() {
					case IF: // 0
						// *ebnf.Name IfStmt ctx [IF]
						if ifStmt = p.ifStmt(); ifStmt == nil {
							goto _3
						}
						break
					_3:
						ifStmt = nil
						p.back(ix)
						goto _1
					case LBRACE: // 1
						// *ebnf.Name Block ctx [LBRACE]
						if block2 = p.block(nil, nil); block2 == nil {
							goto _5
						}
						break
					_5:
						block2 = nil
						p.back(ix)
						goto _1
					default:
						p.back(ix)
						goto _1
					}
				}
			default:
				return &IfStmtNode{
					IF:         ifTok,
					Expression: expression,
					Block:      block,
					SimpleStmt: simpleStmt,
					SEMICOLON:  semicolonTok,
				}
			}
			goto _2
		_1:
			block2 = nil
			elseTok = Token{}
			ifStmt = nil
		_2:
		}
		break
	_0:
		block = nil
		block2 = nil
		elseTok = Token{}
		expression = nil
		ifStmt = nil
		ifTok = Token{}
		// ebnf.Sequence "if" SimpleStmt ";" Expression Block [ "else" ( IfStmt | Block ) ] ctx [IF]
		{
			ix := p.ix
			// *ebnf.Token "if" ctx [IF]
			ifTok = p.expect(IF)
			// *ebnf.Name SimpleStmt ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */ :
				if simpleStmt = p.simpleStmt(false); simpleStmt == nil {
					p.back(ix)
					goto _7
				}
			}
			// *ebnf.Token ";" ctx []
			if semicolonTok, ok = p.accept(SEMICOLON); !ok {
				p.back(ix)
				goto _7
			}
			// *ebnf.Name Expression ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				if expression = p.expression(true); expression == nil {
					p.back(ix)
					goto _7
				}
			default:
				p.back(ix)
				goto _7
			}
			// *ebnf.Name Block ctx []
			switch p.c() {
			case LBRACE:
				if block = p.block(nil, nil); block == nil {
					p.back(ix)
					goto _7
				}
			default:
				p.back(ix)
				goto _7
			}
			// *ebnf.Option [ "else" ( IfStmt | Block ) ] ctx []
			switch p.c() {
			case ELSE:
				// ebnf.Sequence "else" ( IfStmt | Block ) ctx [ELSE]
				{
					switch p.peek(1) {
					case IF, LBRACE:
					default:
						goto _8
					}
					ix := p.ix
					// *ebnf.Token "else" ctx [ELSE]
					elseTok = p.expect(ELSE)
					// *ebnf.Group ( IfStmt | Block ) ctx [IF, LBRACE]
					// ebnf.Alternative IfStmt | Block ctx [IF, LBRACE]
					switch p.c() {
					case IF: // 0
						// *ebnf.Name IfStmt ctx [IF]
						if ifStmt = p.ifStmt(); ifStmt == nil {
							goto _10
						}
						break
					_10:
						ifStmt = nil
						p.back(ix)
						goto _8
					case LBRACE: // 1
						// *ebnf.Name Block ctx [LBRACE]
						if block2 = p.block(nil, nil); block2 == nil {
							goto _12
						}
						break
					_12:
						block2 = nil
						p.back(ix)
						goto _8
					default:
						p.back(ix)
						goto _8
					}
				}
			}
			goto _9
		_8:
			block2 = nil
			elseTok = Token{}
			ifStmt = nil
		_9:
		}
		break
	_7:
		block = nil
		block2 = nil
		elseTok = Token{}
		expression = nil
		ifStmt = nil
		ifTok = Token{}
		semicolonTok = Token{}
		simpleStmt = nil
		return nil
	default:
		return nil
	}
	var elseClause Node
	switch {
	case ifStmt != nil:
		elseClause = ifStmt
	case block2 != nil:
		elseClause = block2
	}
	return &IfElseStmtNode{
		IF:         ifTok,
		Expression: expression,
		Block:      block,
		ELSE:       elseTok,
		ElseClause: elseClause,
		SimpleStmt: simpleStmt,
		SEMICOLON:  semicolonTok,
	}
}

// ImportSpecListNode represents the production
//
//	ImportSpecListNode = { ImportSpec ";" } .
type ImportSpecListNode struct {
	ImportSpec *ImportSpecNode
	SEMICOLON  Token
	List       *ImportSpecListNode
}

// Source implements Node.
func (n *ImportSpecListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ImportSpecListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.ImportSpec.Position()
}

// ImportDeclNode represents the production
//
//	ImportDecl = "import" ( ImportSpec | "(" { ImportSpec ";" } ")" ) .
type ImportDeclNode struct {
	IMPORT         Token
	LPAREN         Token
	ImportSpecList *ImportSpecListNode
	RPAREN         Token
}

// Source implements Node.
func (n *ImportDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ImportDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IMPORT.Position()
}

func (p *parser) importDecl() *ImportDeclNode {
	var (
		ok         bool
		importTok  Token
		importSpec *ImportSpecNode
		lparenTok  Token
		list, last *ImportSpecListNode
		rparenTok  Token
	)
	// ebnf.Sequence "import" ( ImportSpec | "(" { ImportSpec ";" } [ ImportSpec ] ")" ) ctx [IMPORT]
	{
		switch p.peek(1) {
		case IDENT, LPAREN, PERIOD, STRING:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "import" ctx [IMPORT]
		importTok = p.expect(IMPORT)
		// *ebnf.Group ( ImportSpec | "(" { ImportSpec ";" } [ ImportSpec ] ")" ) ctx [IDENT, LPAREN, PERIOD, STRING]
		// ebnf.Alternative ImportSpec | "(" { ImportSpec ";" } [ ImportSpec ] ")" ctx [IDENT, LPAREN, PERIOD, STRING]
		switch p.c() {
		case IDENT, PERIOD, STRING: // 0
			// *ebnf.Name ImportSpec ctx [IDENT, PERIOD, STRING]
			if importSpec = p.importSpec(); importSpec == nil {
				goto _0
			}
			list = &ImportSpecListNode{
				ImportSpec: importSpec,
			}
			break
		_0:
			importSpec = nil
			p.back(ix)
			return nil
		case LPAREN: // 1
			// ebnf.Sequence "(" { ImportSpec ";" } [ ImportSpec ] ")" ctx [LPAREN]
			{
				ix := p.ix
				// *ebnf.Token "(" ctx [LPAREN]
				lparenTok = p.expect(LPAREN)
				// *ebnf.Repetition { ImportSpec ";" } ctx []
			_4:
				{
					var importSpec *ImportSpecNode
					var semicolonTok Token
					switch p.c() {
					case IDENT, PERIOD, STRING:
						// ebnf.Sequence ImportSpec ";" ctx [IDENT, PERIOD, STRING]
						ix := p.ix
						// *ebnf.Name ImportSpec ctx [IDENT, PERIOD, STRING]
						if importSpec = p.importSpec(); importSpec == nil {
							p.back(ix)
							goto _5
						}
						// *ebnf.Token ";" ctx []
						if semicolonTok, ok = p.accept(SEMICOLON); !ok {
							next := &ImportSpecListNode{
								ImportSpec: importSpec,
							}
							if last != nil {
								last.List = next
							}
							if list == nil {
								list = next
							}
							last = next
							goto _5
						}
						next := &ImportSpecListNode{
							ImportSpec: importSpec,
							SEMICOLON:  semicolonTok,
						}
						if last != nil {
							last.List = next
						}
						if list == nil {
							list = next
						}
						last = next
						goto _4
					}
				_5:
				}
				// *ebnf.Token ")" ctx []
				if rparenTok, ok = p.accept(RPAREN); !ok {
					p.back(ix)
					goto _2
				}
			}
			break
		_2:
			lparenTok = Token{}
			rparenTok = Token{}
			p.back(ix)
			return nil
		default:
			p.back(ix)
			return nil
		}
	}
	return &ImportDeclNode{
		IMPORT:         importTok,
		LPAREN:         lparenTok,
		ImportSpecList: list,
		RPAREN:         rparenTok,
	}
}

// ImportSpecNode represents the production
//
//	ImportSpec = [ "." | PackageName ] ImportPath .
type ImportSpecNode struct {
	PERIOD      Token
	PackageName Token
	ImportPath  *BasicLitNode

	pkg *Package
	visible
}

// Source implements Node.
func (n *ImportSpecNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ImportSpecNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.PERIOD.IsValid() {
		return n.PERIOD.Position()
	}

	if n.PackageName.IsValid() {
		return n.PackageName.Position()
	}

	return n.ImportPath.Position()
}

func (p *parser) importSpec() *ImportSpecNode {
	var (
		periodTok   Token
		packageName Token
		importPath  *BasicLitNode
	)
	// ebnf.Sequence [ "." | PackageName ] ImportPath ctx [IDENT, PERIOD, STRING]
	{
		ix := p.ix
		// *ebnf.Option [ "." | PackageName ] ctx [IDENT, PERIOD, STRING]
		switch p.c() {
		case IDENT, PERIOD:
			// ebnf.Alternative "." | PackageName ctx [IDENT, PERIOD]
			switch p.c() {
			case PERIOD: // 0
				// *ebnf.Token "." ctx [PERIOD]
				periodTok = p.expect(PERIOD)
			case IDENT: // 1
				// *ebnf.Name PackageName ctx [IDENT]
				if packageName = p.packageName(); !packageName.IsValid() {
					goto _4
				}
				break
			_4:
				packageName = Token{}
				goto _0
			default:
				goto _0
			}
		}
		goto _1
	_0:
		packageName = Token{}
		periodTok = Token{}
	_1:
		// *ebnf.Name ImportPath ctx []
		switch p.c() {
		case STRING:
			if importPath = p.basicLit().(*BasicLitNode); importPath == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &ImportSpecNode{
		PERIOD:      periodTok,
		PackageName: packageName,
		ImportPath:  importPath,
	}
}

// IndexNode represents the production
//
//	Index = "[" Expression "]" .
type IndexNode struct {
	LBRACK     Token
	Expression Expression
	RBRACK     Token
}

// Source implements Node.
func (n *IndexNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *IndexNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LBRACK.Position()
}

func (p *parser) index() *IndexNode {
	var (
		ok         bool
		lbrackTok  Token
		expression Expression
		rbrackTok  Token
	)
	// ebnf.Sequence "[" Expression "]" ctx [LBRACK]
	{
		switch p.peek(1) {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "[" ctx [LBRACK]
		lbrackTok = p.expect(LBRACK)
		// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if expression = p.expression(false); expression == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token "]" ctx []
		if rbrackTok, ok = p.accept(RBRACK); !ok {
			p.back(ix)
			return nil
		}
	}
	return &IndexNode{
		LBRACK:     lbrackTok,
		Expression: expression,
		RBRACK:     rbrackTok,
	}
}
func (p *parser) initStmt() Node {
	// *ebnf.Name SimpleStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */]
	return p.simpleStmt(true)
}

// InterfaceElemNode represents the production
//
//	InterfaceElem = MethodElem | TypeElem .
type InterfaceElemNode struct {
	MethodElem *MethodElemNode
	TypeElem   *TypeElemListNode
}

// Source implements Node.
func (n *InterfaceElemNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *InterfaceElemNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.MethodElem != nil {
		return n.MethodElem.Position()
	}

	return n.TypeElem.Position()
}

func (p *parser) interfaceElem() *InterfaceElemNode {
	var (
		methodElem *MethodElemNode
		typeElem   *TypeElemListNode
	)
	// ebnf.Alternative MethodElem | TypeElem ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
	switch p.c() {
	case IDENT: // 0 1
		// *ebnf.Name MethodElem ctx [IDENT]
		if methodElem = p.methodElem(); methodElem == nil {
			goto _0
		}
		break
	_0:
		methodElem = nil
		// *ebnf.Name TypeElem ctx [IDENT]
		if typeElem = p.typeElem(); typeElem == nil {
			goto _1
		}
		break
	_1:
		typeElem = nil
		return nil
	case ARROW, CHAN, FUNC, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE: // 1
		// *ebnf.Name TypeElem ctx [ARROW, CHAN, FUNC, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
		if typeElem = p.typeElem(); typeElem == nil {
			goto _2
		}
		break
	_2:
		typeElem = nil
		return nil
	default:
		return nil
	}
	return &InterfaceElemNode{
		MethodElem: methodElem,
		TypeElem:   typeElem,
	}
}

// InterfaceElemListNode represents the production
//
//	InterfaceElemListNode = { InterfaceElem ";" } .
type InterfaceElemListNode struct {
	InterfaceElem *InterfaceElemNode
	SEMICOLON     Token
	List          *InterfaceElemListNode
}

// Source implements Node.
func (n *InterfaceElemListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *InterfaceElemListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.InterfaceElem.Position()
}

// InterfaceTypeNode represents the production
//
//	InterfaceType = "interface" "{" { InterfaceElem ";" } "}" .
type InterfaceTypeNode struct {
	INTERFACE         Token
	LBRACE            Token
	InterfaceElemList *InterfaceElemListNode
	RBRACE            Token

	guard

	methods map[string]*MethodElemNode
}

// Source implements Node.
func (n *InterfaceTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *InterfaceTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.INTERFACE.Position()
}

func (p *parser) interfaceType() *InterfaceTypeNode {
	var (
		ok           bool
		interfaceTok Token
		lbraceTok    Token
		list, last   *InterfaceElemListNode
		rbraceTok    Token
	)
	// ebnf.Sequence "interface" "{" { InterfaceElem ";" } [ InterfaceElem ] "}" ctx [INTERFACE]
	{
		if p.peek(1) != LBRACE {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "interface" ctx [INTERFACE]
		interfaceTok = p.expect(INTERFACE)
		// *ebnf.Token "{" ctx [LBRACE]
		lbraceTok = p.expect(LBRACE)
		// *ebnf.Repetition { InterfaceElem ";" } ctx []
	_0:
		{
			var interfaceElem *InterfaceElemNode
			var semicolonTok Token
			switch p.c() {
			case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE:
				// ebnf.Sequence InterfaceElem ";" ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
				ix := p.ix
				// *ebnf.Name InterfaceElem ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
				if interfaceElem = p.interfaceElem(); interfaceElem == nil {
					p.back(ix)
					goto _1
				}
				// *ebnf.Token ";" ctx []
				if semicolonTok, ok = p.accept(SEMICOLON); !ok {
					next := &InterfaceElemListNode{
						InterfaceElem: interfaceElem,
					}
					if last != nil {
						last.List = next
					}
					if list == nil {
						list = next
					}
					last = next
					goto _1
				}
				next := &InterfaceElemListNode{
					InterfaceElem: interfaceElem,
					SEMICOLON:     semicolonTok,
				}
				if last != nil {
					last.List = next
				}
				if list == nil {
					list = next
				}
				last = next
				goto _0
			}
		_1:
		}
		// *ebnf.Token "}" ctx []
		if rbraceTok, ok = p.accept(RBRACE); !ok {
			p.back(ix)
			return nil
		}
	}
	return &InterfaceTypeNode{
		INTERFACE:         interfaceTok,
		LBRACE:            lbraceTok,
		InterfaceElemList: list,
		RBRACE:            rbraceTok,
	}
}

// KeyedElementNode represents the production
//
//	KeyedElement = Element [ ":" Element ] .
type KeyedElementNode struct {
	Element  Expression
	COLON    Token
	Element2 Expression
}

// Source implements Node.
func (n *KeyedElementNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *KeyedElementNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.Element.Position()
}

func (p *parser) keyedElement() Expression {
	var (
		element  Expression
		colonTok Token
		element2 Expression
	)
	// ebnf.Sequence Element [ ":" Element ] ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Name Element ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if element = p.element(); element == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ ":" Element ] ctx []
		switch p.c() {
		case COLON:
			// ebnf.Sequence ":" Element ctx [COLON]
			{
				switch p.peek(1) {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				default:
					goto _0
				}
				ix := p.ix
				// *ebnf.Token ":" ctx [COLON]
				colonTok = p.expect(COLON)
				// *ebnf.Name Element ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if element2 = p.element(); element2 == nil {
					p.back(ix)
					goto _0
				}
			}
		}
		goto _1
	_0:
		colonTok = Token{}
		element2 = nil
	_1:
	}
	if !colonTok.IsValid() {
		return element
	}

	return &KeyedElementNode{
		Element:  element,
		COLON:    colonTok,
		Element2: element2,
	}
}

// LabelNode represents the production
//
//	Label = identifier .
type LabelNode struct {
	IDENT Token
}

// Source implements Node.
func (n *LabelNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *LabelNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IDENT.Position()
}

func (p *parser) label() *LabelNode {
	var (
		identTok Token
	)
	// *ebnf.Name identifier ctx [IDENT]
	identTok = p.expect(IDENT)
	return &LabelNode{
		IDENT: identTok,
	}
}

// LabeledStmtNode represents the production
//
//	LabeledStmt = Label ":" Statement .
type LabeledStmtNode struct {
	Label     *LabelNode
	COLON     Token
	Statement Node
}

// Source implements Node.
func (n *LabeledStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *LabeledStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.Label.Position()
}

func (p *parser) labeledStmt() *LabeledStmtNode {
	var (
		label     *LabelNode
		colonTok  Token
		statement Node
	)
	// ebnf.Sequence Label ":" Statement ctx [IDENT]
	{
		if p.peek(1) != COLON {
			return nil
		}
		ix := p.ix
		// *ebnf.Name Label ctx [IDENT]
		if label = p.label(); label == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token ":" ctx [COLON]
		colonTok = p.expect(COLON)
		// *ebnf.Name Statement ctx []
		switch p.c() {
		case ADD, AND, ARROW, BREAK, CHAN, CHAR, CONST, CONTINUE, DEFER, FALLTHROUGH, FLOAT, FOR, FUNC, GO, GOTO, IDENT, IF, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, RETURN, SELECT, STRING, STRUCT, SUB, SWITCH, TYPE, VAR, XOR /* ε */ :
			if statement = p.statement(); statement == nil {
				p.back(ix)
				return nil
			}
		}
	}
	return &LabeledStmtNode{
		Label:     label,
		COLON:     colonTok,
		Statement: statement,
	}
}

func (p *parser) literal() Expression {
	var (
		basicLit     Expression
		compositeLit *CompositeLitNode
		functionLit  *FunctionLitNode
	)
	// ebnf.Alternative BasicLit | CompositeLit | FunctionLit ctx [CHAR, FLOAT, FUNC, IMAG, INT, LBRACK, MAP, STRING, STRUCT]
	switch p.c() {
	case CHAR, FLOAT, IMAG, INT, STRING: // 0
		// *ebnf.Name BasicLit ctx [CHAR, FLOAT, IMAG, INT, STRING]
		if basicLit = p.basicLit(); basicLit == nil {
			return nil
		}
		return basicLit
	case LBRACK, MAP, STRUCT: // 1
		// *ebnf.Name CompositeLit ctx [LBRACK, MAP, STRUCT]
		if compositeLit = p.compositeLit(); compositeLit == nil {
			return nil
		}
		return compositeLit
	case FUNC: // 2
		// *ebnf.Name FunctionLit ctx [FUNC]
		if functionLit = p.functionLit(); functionLit == nil {
			return nil
		}
		return functionLit
	default:
		return nil
	}
}

// ArrayLiteralTypeNode represents the production
//
//	ArrayLiteralType = StructType | ArrayType | "[" "..." "]" ElementType | SliceType | MapType .
type ArrayLiteralTypeNode struct {
	LBRACK      Token
	ELLIPSIS    Token
	RBRACK      Token
	ElementType Node
}

// Source implements Node.
func (n *ArrayLiteralTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ArrayLiteralTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LBRACK.Position()
}

func (p *parser) literalType() Node {
	var (
		ok          bool
		structType  *StructTypeNode
		arrayType   *ArrayTypeNode
		lbrackTok   Token
		ellipsisTok Token
		rbrackTok   Token
		elementType Node
		sliceType   *SliceTypeNode
		mapType     *MapTypeNode
	)
	// ebnf.Alternative StructType | ArrayType | "[" "..." "]" ElementType | SliceType | MapType ctx [LBRACK, MAP, STRUCT]
	switch p.c() {
	case STRUCT: // 0
		// *ebnf.Name StructType ctx [STRUCT]
		if structType = p.structType(); structType != nil {
			return structType
		}
	case LBRACK: // 1 2 3
		// *ebnf.Name ArrayType ctx [LBRACK]
		if arrayType = p.arrayType(); arrayType != nil {
			return arrayType
		}

		// ebnf.Sequence "[" "..." "]" ElementType ctx [LBRACK]
		{
			if p.peek(1) != ELLIPSIS {
				goto _3
			}
			ix := p.ix
			// *ebnf.Token "[" ctx [LBRACK]
			lbrackTok = p.expect(LBRACK)
			// *ebnf.Token "..." ctx [ELLIPSIS]
			ellipsisTok = p.expect(ELLIPSIS)
			// *ebnf.Token "]" ctx []
			if rbrackTok, ok = p.accept(RBRACK); !ok {
				p.back(ix)
				goto _3
			}
			// *ebnf.Name ElementType ctx []
			switch p.c() {
			case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
				if elementType = p.type1(); elementType == nil {
					p.back(ix)
					goto _3
				}
			default:
				p.back(ix)
				goto _3
			}
		}
		return &ArrayLiteralTypeNode{
			LBRACK:      lbrackTok,
			ELLIPSIS:    ellipsisTok,
			RBRACK:      rbrackTok,
			ElementType: elementType,
		}
	_3:
		elementType = nil
		ellipsisTok = Token{}
		lbrackTok = Token{}
		rbrackTok = Token{}
		// *ebnf.Name SliceType ctx [LBRACK]
		if sliceType = p.sliceType(); sliceType != nil {
			return sliceType
		}

		return nil
	case MAP: // 4
		// *ebnf.Name MapType ctx [MAP]
		if mapType = p.mapType(); mapType != nil {
			return mapType
		}
	}
	return nil
}

// KeyedElementListNode represents the production
//
//	KeyedElementListNode = { KeyedElement "," } .
type KeyedElementListNode struct {
	KeyedElement Expression
	COMMA        Token
	List         *KeyedElementListNode
}

// Source implements Node.
func (n *KeyedElementListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *KeyedElementListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.KeyedElement.Position()
}

// LiteralValueNode represents the production
//
//	LiteralValue = "{" { KeyedElement "," } "}" .
type LiteralValueNode struct {
	LBRACE           Token
	KeyedElementList *KeyedElementListNode
	RBRACE           Token
}

// Source implements Node.
func (n *LiteralValueNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *LiteralValueNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LBRACE.Position()
}

func (p *parser) literalValue() *LiteralValueNode {
	var (
		ok           bool
		lbraceTok    Token
		list, last   *KeyedElementListNode
		keyedElement Expression
		rbraceTok    Token
	)
	// ebnf.Sequence "{" [ ElementList [ "," ] ] "}" ctx [LBRACE]
	ix := p.ix
	// *ebnf.Token "{" ctx [LBRACE]
	lbraceTok = p.expect(LBRACE)
	for {
		// *ebnf.Option [ ElementList [ "," ] ] ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			{
				ix := p.ix
				if keyedElement = p.keyedElement(); keyedElement == nil {
					p.back(ix)
					goto _1
				}
				switch p.c() {
				case COMMA:
					// *ebnf.Token "," ctx [COMMA]
					next := &KeyedElementListNode{
						KeyedElement: keyedElement,
						COMMA:        p.consume(),
					}
					if last != nil {
						last.List = next
					}
					if list == nil {
						list = next
					}
					last = next
					continue
				case RBRACE:
					next := &KeyedElementListNode{
						KeyedElement: keyedElement,
					}
					if last != nil {
						last.List = next
					}
					if list == nil {
						list = next
					}
					last = next
				}
			}
		}
		goto _1
	}
_1:
	// *ebnf.Token "}" ctx []
	if rbraceTok, ok = p.accept(RBRACE); !ok {
		p.back(ix)
		return nil
	}
	return &LiteralValueNode{
		LBRACE:           lbraceTok,
		KeyedElementList: list,
		RBRACE:           rbraceTok,
	}
}

func (p *parser) logicalAndExpression(preBlock bool) (r Expression) {
	var relationalExpression Expression
	// ebnf.Sequence RelationalExpression { "&&" RelationalExpression } ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Name RelationalExpression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if relationalExpression = p.relationalExpression(preBlock); relationalExpression == nil {
			p.back(ix)
			return nil
		}
		r = relationalExpression
		// *ebnf.Repetition { "&&" RelationalExpression } ctx []
	_0:
		{
			var landTok Token
			var relationalExpression Expression
			switch p.c() {
			case LAND:
				// ebnf.Sequence "&&" RelationalExpression ctx [LAND]
				switch p.peek(1) {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				default:
					goto _1
				}
				ix := p.ix
				// *ebnf.Token "&&" ctx [LAND]
				landTok = p.expect(LAND)
				// *ebnf.Name RelationalExpression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if relationalExpression = p.relationalExpression(preBlock); relationalExpression == nil {
					p.back(ix)
					goto _1
				}
				r = &BinaryExpressionNode{LHS: r, Op: landTok, RHS: relationalExpression}
				goto _0
			}
		_1:
		}
	}
	return r
}

// MapTypeNode represents the production
//
//	MapType = "map" "[" KeyType "]" ElementType .
type MapTypeNode struct {
	MAP         Token
	LBRACK      Token
	KeyType     Node
	RBRACK      Token
	ElementType Node
}

// Source implements Node.
func (n *MapTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *MapTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.MAP.Position()
}

func (p *parser) mapType() *MapTypeNode {
	var (
		ok          bool
		mapTok      Token
		lbrackTok   Token
		keyType     Node
		rbrackTok   Token
		elementType Node
	)
	// ebnf.Sequence "map" "[" KeyType "]" ElementType ctx [MAP]
	{
		if p.peek(1) != LBRACK {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "map" ctx [MAP]
		mapTok = p.expect(MAP)
		// *ebnf.Token "[" ctx [LBRACK]
		lbrackTok = p.expect(LBRACK)
		// *ebnf.Name KeyType ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			if keyType = p.type1(); keyType == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Token "]" ctx []
		if rbrackTok, ok = p.accept(RBRACK); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Name ElementType ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			if elementType = p.type1(); elementType == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &MapTypeNode{
		MAP:         mapTok,
		LBRACK:      lbrackTok,
		KeyType:     keyType,
		RBRACK:      rbrackTok,
		ElementType: elementType,
	}
}

// MethodDeclNode represents the production
//
//	MethodDecl = "func" Receiver MethodName Signature [ FunctionBody ] .
type MethodDeclNode struct {
	FUNC         Token
	Receiver     *ParametersNode
	MethodName   Token
	Signature    *SignatureNode
	FunctionBody *FunctionBodyNode
}

// Source implements Node.
func (n *MethodDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *MethodDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.FUNC.Position()
}

func (p *parser) methodDecl() *MethodDeclNode {
	var (
		funcTok      Token
		receiver     *ParametersNode
		methodName   Token
		signature    *SignatureNode
		functionBody *FunctionBodyNode
	)
	// ebnf.Sequence "func" Receiver MethodName Signature [ FunctionBody ] ctx [FUNC]
	{
		switch p.peek(1) {
		case LPAREN:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "func" ctx [FUNC]
		funcTok = p.expect(FUNC)
		// *ebnf.Name Receiver ctx [LPAREN]
		if receiver = p.receiver(); receiver == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Name MethodName ctx []
		switch p.c() {
		case IDENT:
			if methodName = p.methodName(); !methodName.IsValid() {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Name Signature ctx []
		switch p.c() {
		case LPAREN:
			if signature = p.signature(); signature == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ FunctionBody ] ctx []
		switch p.c() {
		case LBRACE:
			// *ebnf.Name FunctionBody ctx [LBRACE]
			if functionBody = p.functionBody(receiver, signature); functionBody == nil {
				goto _0
			}
		}
		goto _1
	_0:
		functionBody = nil
	_1:
	}
	return &MethodDeclNode{
		FUNC:         funcTok,
		Receiver:     receiver,
		MethodName:   methodName,
		Signature:    signature,
		FunctionBody: functionBody,
	}
}

// MethodElemNode represents the production
//
//	MethodElem = MethodName Signature .
type MethodElemNode struct {
	MethodName Token
	Signature  *SignatureNode

	typ Type
}

// Source implements Node.
func (n *MethodElemNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *MethodElemNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.MethodName.Position()
}

func (p *parser) methodElem() *MethodElemNode {
	var (
		methodName Token
		signature  *SignatureNode
	)
	// ebnf.Sequence MethodName Signature ctx [IDENT]
	{
		switch p.peek(1) {
		case LPAREN:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Name MethodName ctx [IDENT]
		if methodName = p.methodName(); !methodName.IsValid() {
			p.back(ix)
			return nil
		}
		// *ebnf.Name Signature ctx [LPAREN]
		if signature = p.signature(); signature == nil {
			p.back(ix)
			return nil
		}
	}
	return &MethodElemNode{
		MethodName: methodName,
		Signature:  signature,
	}
}

// MethodExprNode represents the production
//
//	MethodExpr = ReceiverType "." MethodName .
type MethodExprNode struct {
	ReceiverType Node
	PERIOD       Token
	MethodName   Token
}

// Source implements Node.
func (n *MethodExprNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *MethodExprNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.ReceiverType.Position()
}

func (p *parser) methodExpr() *MethodExprNode {
	var (
		ok           bool
		receiverType Node
		periodTok    Token
		methodName   Token
	)
	// ebnf.Sequence ReceiverType "." MethodName ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
	{
		ix := p.ix
		// *ebnf.Name ReceiverType ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		if receiverType = p.type1(); receiverType == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token "." ctx []
		if periodTok, ok = p.accept(PERIOD); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Name MethodName ctx []
		switch p.c() {
		case IDENT:
			if methodName = p.methodName(); !methodName.IsValid() {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &MethodExprNode{
		ReceiverType: receiverType,
		PERIOD:       periodTok,
		MethodName:   methodName,
	}
}

func (p *parser) methodName() Token {
	// *ebnf.Name identifier ctx [IDENT]
	return p.expect(IDENT)
}

func (p *parser) multiplicativeExpression(preBlock bool) (r Expression) {
	var unaryExpr Expression
	// ebnf.Sequence UnaryExpr { ( "*" | "/" | "%" | "<<" | ">>" | "&" | "&^" ) UnaryExpr } ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Name UnaryExpr ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if unaryExpr = p.unaryExpr(preBlock); unaryExpr == nil {
			p.back(ix)
			return nil
		}
		r = unaryExpr
		// *ebnf.Repetition { ( "*" | "/" | "%" | "<<" | ">>" | "&" | "&^" ) UnaryExpr } ctx []
	_0:
		{
			var op Token
			var unaryExpr Expression
			switch p.c() {
			case AND, AND_NOT, MUL, QUO, REM, SHL, SHR:
				// ebnf.Sequence ( "*" | "/" | "%" | "<<" | ">>" | "&" | "&^" ) UnaryExpr ctx [AND, AND_NOT, MUL, QUO, REM, SHL, SHR]
				// *ebnf.Group ( "*" | "/" | "%" | "<<" | ">>" | "&" | "&^" ) ctx [AND, AND_NOT, MUL, QUO, REM, SHL, SHR]
				// ebnf.Alternative "*" | "/" | "%" | "<<" | ">>" | "&" | "&^" ctx [AND, AND_NOT, MUL, QUO, REM, SHL, SHR]
				op = p.consume()
				// *ebnf.Name UnaryExpr ctx []
				switch p.c() {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
					if unaryExpr = p.unaryExpr(preBlock); unaryExpr == nil {
						p.back(ix)
						goto _1
					}
				default:
					p.back(ix)
					goto _1
				}
				r = &BinaryExpressionNode{LHS: r, Op: op, RHS: unaryExpr}
				goto _0
			}
		_1:
		}
	}
	return r
}

// OperandNode represents the production
//
//	Operand = Literal | OperandName [ TypeArgs ] [ LiteralValue ] .
type OperandNode struct {
	OperandName  Expression
	TypeArgs     *TypeArgsNode
	LiteralValue *LiteralValueNode
}

// Source implements Node.
func (n *OperandNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *OperandNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.OperandName.Position()
}

type ParenthesizedExpressionNode struct {
	LPAREN     Token
	Expression Expression
	RPAREN     Token
}

// Source implements Node.
func (n *ParenthesizedExpressionNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ParenthesizedExpressionNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LPAREN.Position()
}

func (p *parser) operand(preBlock bool) Expression {
	var (
		ok           bool
		literal      Expression
		operandName  Expression
		typeArgs     *TypeArgsNode
		literalValue *LiteralValueNode
		lparenTok    Token
		expression   Expression
		rparenTok    Token
	)
	// ebnf.Alternative Literal | OperandName [ TypeArgs ] [ LiteralValue ] | "(" Expression ")" ctx [CHAR, FLOAT, FUNC, IDENT, IMAG, INT, LBRACK, LPAREN, MAP, STRING, STRUCT]
	switch p.c() {
	case CHAR, FLOAT, FUNC, IMAG, INT, LBRACK, MAP, STRING, STRUCT: // 0
		// *ebnf.Name Literal ctx [CHAR, FLOAT, FUNC, IMAG, INT, LBRACK, MAP, STRING, STRUCT]
		if literal = p.literal(); literal == nil {
			return nil
		}
		return literal
	case IDENT: // 1
		// ebnf.Sequence OperandName [ TypeArgs ] [ LiteralValue ] ctx [IDENT]
		{
			ix := p.ix
			// *ebnf.Name OperandName ctx [IDENT]
			if operandName = p.operandName(); operandName == nil {
				p.back(ix)
				goto _2
			}
			// *ebnf.Option [ TypeArgs ] ctx []
			switch p.c() {
			case LBRACK:
				// *ebnf.Name TypeArgs ctx [LBRACK]
				if typeArgs = p.typeArgs(); typeArgs == nil {
					goto _4
				}
			}
			goto _5
		_4:
			typeArgs = nil
		_5:
			if !preBlock {
				// *ebnf.Option [ LiteralValue ] ctx []
				switch p.c() {
				case LBRACE:
					// *ebnf.Name LiteralValue ctx [LBRACE]
					if literalValue = p.literalValue(); literalValue == nil {
						goto _6
					}
				}
				goto _7
			_6:
				literalValue = nil
			_7:
			}
		}
		break
	_2:
		literalValue = nil
		operandName = nil
		typeArgs = nil
		return nil
	case LPAREN: // 2
		// ebnf.Sequence "(" Expression ")" ctx [LPAREN]
		{
			switch p.peek(1) {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			default:
				goto _8
			}
			ix := p.ix
			// *ebnf.Token "(" ctx [LPAREN]
			lparenTok = p.expect(LPAREN)
			// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if expression = p.expression(false); expression == nil {
				p.back(ix)
				goto _8
			}
			// *ebnf.Token ")" ctx []
			if rparenTok, ok = p.accept(RPAREN); !ok {
				p.back(ix)
				goto _8
			}
		}
		return &ParenthesizedExpressionNode{LPAREN: lparenTok, Expression: expression, RPAREN: rparenTok}
	_8:
		expression = nil
		lparenTok = Token{}
		rparenTok = Token{}
		return nil
	default:
		return nil
	}
	if operandName != nil && typeArgs == nil && literalValue == nil {
		return operandName
	}

	return &OperandNode{
		OperandName:  operandName,
		TypeArgs:     typeArgs,
		LiteralValue: literalValue,
	}
}

// IotaNode represents the production
//
//	IotaNode = identifier .
type IotaNode struct {
	Iota Token
	lexicalScoper
}

// Source implements Node.
func (n *IotaNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *IotaNode) Position() (r token.Position) {
	if n == nil || !n.Iota.IsValid() {
		return r
	}

	return n.Iota.Position()
}

// OperandNameNode represents the production
//
//	OperandName = identifier .
type OperandNameNode struct {
	Name Token
	lexicalScoper
}

// Source implements Node.
func (n *OperandNameNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *OperandNameNode) Position() (r token.Position) {
	if n == nil || !n.Name.IsValid() {
		return r
	}

	return n.Name.Position()
}

// OperandQualifiedNameNode represents the production
//
//	OperandQualifiedName = QualifiedIdent .
type OperandQualifiedNameNode struct {
	Name *QualifiedIdentNode
	lexicalScoper
}

// Source implements Node.
func (n *OperandQualifiedNameNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *OperandQualifiedNameNode) Position() (r token.Position) {
	if n == nil || n.Name == nil {
		return r
	}

	return n.Name.Position()
}

func (p *parser) operandName() Expression {
	var (
		qualifiedIdent *QualifiedIdentNode
	)
	// ebnf.Alternative QualifiedIdent | identifier ctx [IDENT]
	switch p.c() {
	case IDENT: // 0 1
		// *ebnf.Name QualifiedIdent ctx [IDENT]
		if qualifiedIdent = p.qualifiedIdent(); qualifiedIdent != nil {
			return &OperandQualifiedNameNode{
				Name:          qualifiedIdent,
				lexicalScoper: newLexicalScoper(p.sc),
			}
		}

		// *ebnf.Name identifier ctx [IDENT]
		return &OperandNameNode{
			Name:          p.expect(IDENT),
			lexicalScoper: newLexicalScoper(p.sc),
		}
	default:
		return nil
	}
}

// PackageClauseNode represents the production
//
//	PackageClause = "package" PackageName .
type PackageClauseNode struct {
	PACKAGE     Token
	PackageName Token
}

// Source implements Node.
func (n *PackageClauseNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *PackageClauseNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.PACKAGE.Position()
}

func (p *parser) packageClause() *PackageClauseNode {
	var (
		packageTok  Token
		packageName Token
	)
	// ebnf.Sequence "package" PackageName ctx [PACKAGE]
	{
		if p.peek(1) != IDENT {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "package" ctx [PACKAGE]
		packageTok = p.expect(PACKAGE)
		// *ebnf.Name PackageName ctx [IDENT]
		if packageName = p.packageName(); !packageName.IsValid() {
			p.back(ix)
			return nil
		}
	}
	return &PackageClauseNode{
		PACKAGE:     packageTok,
		PackageName: packageName,
	}
}

func (p *parser) packageName() Token {
	// *ebnf.Name identifier ctx [IDENT]
	return p.expect(IDENT)
}

// ParameterDeclNode represents the production
//
//	ParameterDecl = [ IdentifierList ] [ "..." ] Type .
type ParameterDeclNode struct {
	IdentifierList *IdentifierListNode
	ELLIPSIS       Token
	TypeNode       Type

	visible
}

// Source implements Node.
func (n *ParameterDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ParameterDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.IdentifierList != nil {
		return n.IdentifierList.Position()
	}

	if n.ELLIPSIS.IsValid() {
		return n.ELLIPSIS.Position()
	}

	return n.TypeNode.Position()
}

func (p *parser) parameterDecl() *ParameterDeclNode {
	var (
		identTok    Token
		ellipsisTok Token
		typeNode    Type
	)
	// ebnf.Alternative identifier "..." Type | identifier Type | "..." Type | Type ctx [ARROW, CHAN, ELLIPSIS, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
	switch p.c() {
	case IDENT: // 0 1 3
		// ebnf.Sequence identifier "..." Type ctx [IDENT]
		{
			if p.peek(1) != ELLIPSIS {
				goto _0
			}
			ix := p.ix
			// *ebnf.Name identifier ctx [IDENT]
			identTok = p.expect(IDENT)
			// *ebnf.Token "..." ctx [ELLIPSIS]
			ellipsisTok = p.expect(ELLIPSIS)
			// *ebnf.Name Type ctx []
			switch p.c() {
			case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
				if typeNode = p.type1(); typeNode == nil {
					p.back(ix)
					goto _0
				}
			default:
				p.back(ix)
				goto _0
			}
		}
		break
	_0:
		ellipsisTok = Token{}
		identTok = Token{}
		typeNode = nil
		// ebnf.Sequence identifier Type ctx [IDENT]
		{
			switch p.peek(1) {
			case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			default:
				goto _1
			}
			ix := p.ix
			// *ebnf.Name identifier ctx [IDENT]
			identTok = p.expect(IDENT)
			// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
			if typeNode = p.type1(); typeNode == nil {
				p.back(ix)
				goto _1
			}
		}
		break
	_1:
		identTok = Token{}
		typeNode = nil
		// *ebnf.Name Type ctx [IDENT]
		if typeNode = p.type1(); typeNode == nil {
			goto _2
		}
		break
	_2:
		typeNode = nil
		return nil
	case ELLIPSIS: // 2
		// ebnf.Sequence "..." Type ctx [ELLIPSIS]
		{
			switch p.peek(1) {
			case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			default:
				goto _3
			}
			ix := p.ix
			// *ebnf.Token "..." ctx [ELLIPSIS]
			ellipsisTok = p.expect(ELLIPSIS)
			// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
			if typeNode = p.type1(); typeNode == nil {
				p.back(ix)
				goto _3
			}
		}
		break
	_3:
		ellipsisTok = Token{}
		typeNode = nil
		return nil
	case ARROW, CHAN, FUNC, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT: // 3
		// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		if typeNode = p.type1(); typeNode == nil {
			goto _5
		}
		break
	_5:
		typeNode = nil
		return nil
	default:
		return nil
	}
	var idl *IdentifierListNode
	if identTok.IsValid() {
		idl = &IdentifierListNode{IDENT: identTok}
	}
	return &ParameterDeclNode{
		IdentifierList: idl,
		ELLIPSIS:       ellipsisTok,
		TypeNode:       typeNode,
	}
}

// ParameterDeclListNode represents the production
//
//	ParameterDeclListNode = { ParameterDecl  "," } .
type ParameterDeclListNode struct {
	ParameterDecl *ParameterDeclNode
	COMMA         Token
	List          *ParameterDeclListNode
}

// Source implements Node.
func (n *ParameterDeclListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ParameterDeclListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.ParameterDecl.Position()
}

// ParametersNode represents the production
//
//	Parameters = "(" { ParameterDecl  "," } ")" .
type ParametersNode struct {
	LPAREN            Token
	ParameterDeclList *ParameterDeclListNode
	RPAREN            Token
}

// Source implements Node.
func (n *ParametersNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ParametersNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LPAREN.Position()
}

func (p *parser) parameters() (r *ParametersNode) {
	var (
		ok            bool
		lparenTok     Token
		parameterDecl *ParameterDeclNode
		list0         []*ParameterDeclListNode
		list, last    *ParameterDeclListNode
		rparenTok     Token
	)
	ix := p.ix
	// *ebnf.Token "(" ctx [LPAREN]
	lparenTok = p.expect(LPAREN)
	for {
		// *ebnf.Option [ ParameterList [ "," ] ] ctx []
		switch p.c() {
		case ARROW, CHAN, ELLIPSIS, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			// ebnf.Sequence ParameterList [ "," ] ctx [ARROW, CHAN, ELLIPSIS, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
			{
				ix := p.ix
				if parameterDecl = p.parameterDecl(); parameterDecl == nil {
					p.back(ix)
					goto _1
				}
				// *ebnf.Option [ "," ] ctx []
				switch p.c() {
				case COMMA:
					// *ebnf.Token "," ctx [COMMA]
					next := &ParameterDeclListNode{
						ParameterDecl: parameterDecl,
						COMMA:         p.consume(),
					}
					list0 = append(list0, next)
					if last != nil {
						last.List = next
					}
					if list == nil {
						list = next
					}
					last = next
					continue
				case RPAREN:
					next := &ParameterDeclListNode{
						ParameterDecl: parameterDecl,
					}
					list0 = append(list0, next)
					if last != nil {
						last.List = next
					}
					if list == nil {
						list = next
					}
					last = next
				}
			}
		}
		goto _1
	}
_1:
	// *ebnf.Token ")" ctx []
	if rparenTok, ok = p.accept(RPAREN); !ok {
		p.back(ix)
		return nil
	}
	var ids []int
	for i, v := range list0 {
		if v.ParameterDecl.IdentifierList != nil {
			ids = append(ids, i)
		}
	}
	r = &ParametersNode{
		LPAREN:            lparenTok,
		ParameterDeclList: list,
		RPAREN:            rparenTok,
	}
	if len(ids) != 0 && len(ids) != len(list0) {
		//                                                    len(ids)
		//                                                    | ids
		//                                                    | |   len(list0)
		// TODO gc.go:74:20 (rel, importPath, version string) 1 [2] 3
		last = nil
		for _, v := range list0 {
			x := *v
			x.List = nil
		}
		for firstX := 0; len(ids) != 0; {
			lastX := ids[0]
			ids = ids[1:]
			grp := list0[lastX]
			grp.List = nil
			idl := grp.ParameterDecl.IdentifierList
			for i := lastX - 1; i >= firstX; i-- {
				item := list0[i]
				decl := item.ParameterDecl
				typ := decl.TypeNode
				switch x := typ.(type) {
				case *TypeNode:
					if x.TypeArgs != nil {
						panic(todo("%v: %s", decl.Position(), decl.Source(false)))
					}

					if x.TypeName == nil {
						p.err(x.Position(), "syntax error: mixed named and unnamed parameters")
						continue
					}

					switch y := x.TypeName.Name.(type) {
					case Token:
						// { param , } vs { , ident }
						idl.COMMA = item.COMMA
						li := &IdentifierListNode{IDENT: y}
						li.List = idl
						idl = li
					default:
						p.err(y.Position(), "syntax error: mixed named and unnamed parameters")
					}
				case *TypeNameNode:
					switch y := x.Name.(type) {
					case Token:
						// { param , } vs { , ident }
						idl.COMMA = item.COMMA
						li := &IdentifierListNode{IDENT: y}
						li.List = idl
						idl = li
					default:
						p.err(y.Position(), "syntax error: mixed named and unnamed parameters")
					}
				default:
					p.err(x.Position(), "syntax error: mixed named and unnamed parameters")
				}
			}
			grp.ParameterDecl.IdentifierList = idl
			firstX = lastX + 1
			if last == nil {
				r.ParameterDeclList = grp
				last = grp
				continue
			}

			last.List = grp
			last = grp
		}
	}
	return r
}

func (n *ParametersNode) declare(p *parser, s *Scope) {
	if n == nil {
		return
	}

	for l := n.ParameterDeclList; l != nil; l = l.List {
		pd := l.ParameterDecl
		for l := pd.IdentifierList; l != nil; l = l.List {
			p.declare(s, l.IDENT, pd, 0, true)
		}
	}
}

// PointerTypeNode represents the production
//
//	PointerType = "*" BaseType .
type PointerTypeNode struct {
	MUL      Token
	BaseType Type

	guard
}

// Source implements Node.
func (n *PointerTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *PointerTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.MUL.Position()
}

func (p *parser) pointerType() *PointerTypeNode {
	var (
		mulTok   Token
		baseType Type
	)
	// ebnf.Sequence "*" BaseType ctx [MUL]
	{
		switch p.peek(1) {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "*" ctx [MUL]
		mulTok = p.expect(MUL)
		// *ebnf.Name BaseType ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		if baseType = p.type1(); baseType == nil {
			p.back(ix)
			return nil
		}
	}
	return &PointerTypeNode{
		MUL:      mulTok,
		BaseType: baseType,
	}
}

func (p *parser) postStmt() Node {
	// *ebnf.Name SimpleStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */]
	return p.simpleStmt(true)
}

// PrimaryExprNode represents the production
//
//	PrimaryExpr = Operand | Conversion | MethodExpr { Selector | Index | Slice | TypeAssertion | Arguments } .
type PrimaryExprNode struct {
	PrimaryExpr Expression
	Postfix     Node
}

// Source implements Node.
func (n *PrimaryExprNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *PrimaryExprNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.PrimaryExpr.Position()
}

func (p *parser) primaryExpr(preBlock bool) (r Expression) {
	var (
		item0      Expression
		operand    Expression
		conversion *ConversionNode
		methodExpr *MethodExprNode
		list       []Node
	)
	// ebnf.Sequence ( Operand | Conversion | MethodExpr ) { Selector | Index | Slice | TypeAssertion | Arguments } ctx [ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRING, STRUCT]
	{
		ix := p.ix
		// *ebnf.Group ( Operand | Conversion | MethodExpr ) ctx [ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRING, STRUCT]
		// ebnf.Alternative Operand | Conversion | MethodExpr ctx [ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRING, STRUCT]
		switch p.c() {
		case CHAR, FLOAT, IMAG, INT, STRING: // 0
			// *ebnf.Name Operand ctx [CHAR, FLOAT, IMAG, INT, STRING]
			if operand = p.operand(preBlock); operand == nil {
				goto _0
			}
			item0 = operand
			break
		_0:
			operand = nil
			p.back(ix)
			return nil
		case FUNC, IDENT, LBRACK, LPAREN, MAP, STRUCT: // 0 1 2
			// *ebnf.Name Operand ctx [FUNC, IDENT, LBRACK, LPAREN, MAP, STRUCT]
			if operand = p.operand(preBlock); operand == nil {
				goto _2
			}
			item0 = operand
			break
		_2:
			operand = nil
			// *ebnf.Name Conversion ctx [FUNC, IDENT, LBRACK, LPAREN, MAP, STRUCT]
			if conversion = p.conversion(); conversion == nil {
				goto _3
			}
			item0 = conversion
			break
		_3:
			conversion = nil
			// *ebnf.Name MethodExpr ctx [FUNC, IDENT, LBRACK, LPAREN, MAP, STRUCT]
			if methodExpr = p.methodExpr(); methodExpr == nil {
				goto _4
			}
			item0 = methodExpr
			break
		_4:
			methodExpr = nil
			p.back(ix)
			return nil
		case ARROW, CHAN, INTERFACE, MUL: // 1 2
			// *ebnf.Name Conversion ctx [ARROW, CHAN, INTERFACE, MUL]
			if conversion = p.conversion(); conversion == nil {
				goto _5
			}
			item0 = conversion
			break
		_5:
			conversion = nil
			// *ebnf.Name MethodExpr ctx [ARROW, CHAN, INTERFACE, MUL]
			if methodExpr = p.methodExpr(); methodExpr == nil {
				goto _6
			}
			item0 = methodExpr
			break
		_6:
			methodExpr = nil
			p.back(ix)
			return nil
		default:
			p.back(ix)
			return nil
		}

		r = item0

		// *ebnf.Repetition { Selector | Index | Slice | TypeAssertion | Arguments } ctx []
	_7:
		{
			var item Node
			var selector *SelectorNode
			var index *IndexNode
			var slice *SliceNode
			var typeAssertion *TypeAssertionNode
			var arguments Node
			switch p.c() {
			case LBRACK, LPAREN, PERIOD:
				// ebnf.Alternative Selector | Index | Slice | TypeAssertion | Arguments ctx [LBRACK, LPAREN, PERIOD]
				switch p.c() {
				case PERIOD: // 0 3
					// *ebnf.Name Selector ctx [PERIOD]
					if selector = p.selector(); selector == nil {
						goto _9
					}
					item = selector
					break
				_9:
					// *ebnf.Name TypeAssertion ctx [PERIOD]
					if typeAssertion = p.typeAssertion(); typeAssertion == nil {
						goto _10
					}
					item = typeAssertion
					break
				_10:
					goto _8
				case LBRACK: // 1 2
					// *ebnf.Name Index ctx [LBRACK]
					if index = p.index(); index == nil {
						goto _11
					}
					item = index
					break
				_11:
					// *ebnf.Name Slice ctx [LBRACK]
					if slice = p.slice(); slice == nil {
						goto _12
					}
					item = slice
					break
				_12:
					goto _8
				case LPAREN: // 4
					// *ebnf.Name Arguments ctx [LPAREN]
					if arguments = p.arguments(); arguments == nil {
						goto _13
					}
					item = arguments
					break
				_13:
					goto _8
				default:
					goto _8
				}
				list = append(list, item)
				r = &PrimaryExprNode{PrimaryExpr: r, Postfix: item}
				goto _7
			}
		_8:
		}
	}
	return r
}

// QualifiedIdentNode represents the production
//
//	QualifiedIdent = PackageName "." identifier .
type QualifiedIdentNode struct {
	PackageName Token
	PERIOD      Token
	IDENT       Token
}

// Source implements Node.
func (n *QualifiedIdentNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *QualifiedIdentNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.PackageName.Position()
}

func (p *parser) qualifiedIdent() *QualifiedIdentNode {
	var (
		ok          bool
		packageName Token
		periodTok   Token
		identTok    Token
	)
	// ebnf.Sequence PackageName "." identifier ctx [IDENT]
	{
		if p.peek(1) != PERIOD {
			return nil
		}
		ix := p.ix
		// *ebnf.Name PackageName ctx [IDENT]
		if packageName = p.packageName(); !packageName.IsValid() {
			p.back(ix)
			return nil
		}
		// *ebnf.Token "." ctx [PERIOD]
		periodTok = p.expect(PERIOD)
		// *ebnf.Name identifier ctx []
		if identTok, ok = p.accept(IDENT); !ok {
			p.back(ix)
			return nil
		}
	}
	return &QualifiedIdentNode{
		PackageName: packageName,
		PERIOD:      periodTok,
		IDENT:       identTok,
	}
}

// RangeClauseNode represents the production
//
//	RangeClause = "range" Expression | ExpressionList "=" "range" Expression | IdentifierList ":=" "range" Expression .
type RangeClauseNode struct {
	RANGE          Token
	Expression     Expression
	ExpressionList *ExpressionListNode
	ASSIGN         Token
	IdentifierList *IdentifierListNode
	DEFINE         Token
}

// Source implements Node.
func (n *RangeClauseNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *RangeClauseNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) rangeClause() *RangeClauseNode {
	var (
		ok             bool
		rangeTok       Token
		expression     Expression
		expressionList *ExpressionListNode
		assignTok      Token
		identifierList *IdentifierListNode
		defineTok      Token
	)
	// ebnf.Alternative "range" Expression | ExpressionList "=" "range" Expression | IdentifierList ":=" "range" Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, RANGE, STRING, STRUCT, SUB, XOR]
	switch p.c() {
	case RANGE: // 0
		// ebnf.Sequence "range" Expression ctx [RANGE]
		{
			switch p.peek(1) {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			default:
				goto _0
			}
			ix := p.ix
			// *ebnf.Token "range" ctx [RANGE]
			rangeTok = p.expect(RANGE)
			// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if expression = p.expression(true); expression == nil {
				p.back(ix)
				goto _0
			}
		}
		break
	_0:
		expression = nil
		rangeTok = Token{}
		return nil
	case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR: // 1
		// ebnf.Sequence ExpressionList "=" "range" Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		{
			ix := p.ix
			// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if expressionList = p.expressionList(false); expressionList == nil {
				p.back(ix)
				goto _2
			}
			// *ebnf.Token "=" ctx []
			if assignTok, ok = p.accept(ASSIGN); !ok {
				p.back(ix)
				goto _2
			}
			// *ebnf.Token "range" ctx []
			if rangeTok, ok = p.accept(RANGE); !ok {
				p.back(ix)
				goto _2
			}
			// *ebnf.Name Expression ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				if expression = p.expression(true); expression == nil {
					p.back(ix)
					goto _2
				}
			default:
				p.back(ix)
				goto _2
			}
		}
		break
	_2:
		assignTok = Token{}
		expressionList = nil
		expression = nil
		rangeTok = Token{}
		return nil
	case IDENT: // 1 2
		// ebnf.Sequence ExpressionList "=" "range" Expression ctx [IDENT]
		{
			ix := p.ix
			// *ebnf.Name ExpressionList ctx [IDENT]
			if expressionList = p.expressionList(false); expressionList == nil {
				p.back(ix)
				goto _4
			}
			// *ebnf.Token "=" ctx []
			if assignTok, ok = p.accept(ASSIGN); !ok {
				p.back(ix)
				goto _4
			}
			// *ebnf.Token "range" ctx []
			if rangeTok, ok = p.accept(RANGE); !ok {
				p.back(ix)
				goto _4
			}
			// *ebnf.Name Expression ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				if expression = p.expression(true); expression == nil {
					p.back(ix)
					goto _4
				}
			default:
				p.back(ix)
				goto _4
			}
		}
		break
	_4:
		assignTok = Token{}
		expressionList = nil
		expression = nil
		rangeTok = Token{}
		// ebnf.Sequence IdentifierList ":=" "range" Expression ctx [IDENT]
		{
			ix := p.ix
			// *ebnf.Name IdentifierList ctx [IDENT]
			if identifierList = p.identifierList(); identifierList == nil {
				p.back(ix)
				goto _5
			}
			// *ebnf.Token ":=" ctx []
			if defineTok, ok = p.accept(DEFINE); !ok {
				p.back(ix)
				goto _5
			}
			// *ebnf.Token "range" ctx []
			if rangeTok, ok = p.accept(RANGE); !ok {
				p.back(ix)
				goto _5
			}
			// *ebnf.Name Expression ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				if expression = p.expression(true); expression == nil {
					p.back(ix)
					goto _5
				}
			default:
				p.back(ix)
				goto _5
			}
		}
		break
	_5:
		defineTok = Token{}
		expression = nil
		identifierList = nil
		rangeTok = Token{}
		return nil
	default:
		return nil
	}
	return &RangeClauseNode{
		RANGE:          rangeTok,
		Expression:     expression,
		ExpressionList: expressionList,
		ASSIGN:         assignTok,
		IdentifierList: identifierList,
		DEFINE:         defineTok,
	}
}

func (p *parser) receiver() *ParametersNode {
	// *ebnf.Name Parameters ctx [LPAREN]
	return p.parameters()
}

func (p *parser) recvExpr() Expression {
	// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	return p.expression(false)
}

// RecvStmtNode represents the production
//
//	RecvStmt = [ ExpressionList "=" | IdentifierList ":=" ] RecvExpr .
type RecvStmtNode struct {
	ExpressionList *ExpressionListNode
	Token          Token
	IdentifierList *IdentifierListNode
	RecvExpr       Expression
}

// Source implements Node.
func (n *RecvStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *RecvStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) recvStmt() *RecvStmtNode {
	var (
		ok             bool
		expressionList *ExpressionListNode
		tok            Token
		identifierList *IdentifierListNode
		recvExpr       Expression
	)
	// ebnf.Sequence [ ExpressionList "=" | IdentifierList ":=" ] RecvExpr ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Option [ ExpressionList "=" | IdentifierList ":=" ] ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		// ebnf.Alternative ExpressionList "=" | IdentifierList ":=" ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR: // 0
			// ebnf.Sequence ExpressionList "=" ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			{
				ix := p.ix
				// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if expressionList = p.expressionList(false); expressionList == nil {
					p.back(ix)
					goto _2
				}
				// *ebnf.Token "=" ctx []
				if tok, ok = p.accept(ASSIGN); !ok {
					p.back(ix)
					goto _2
				}
			}
			break
		_2:
			tok = Token{}
			expressionList = nil
			goto _0
		case IDENT: // 0 1
			// ebnf.Sequence ExpressionList "=" ctx [IDENT]
			{
				ix := p.ix
				// *ebnf.Name ExpressionList ctx [IDENT]
				if expressionList = p.expressionList(false); expressionList == nil {
					p.back(ix)
					goto _4
				}
				// *ebnf.Token "=" ctx []
				if tok, ok = p.accept(ASSIGN); !ok {
					p.back(ix)
					goto _4
				}
			}
			break
		_4:
			tok = Token{}
			expressionList = nil
			// ebnf.Sequence IdentifierList ":=" ctx [IDENT]
			{
				ix := p.ix
				// *ebnf.Name IdentifierList ctx [IDENT]
				if identifierList = p.identifierList(); identifierList == nil {
					p.back(ix)
					goto _5
				}
				// *ebnf.Token ":=" ctx []
				if tok, ok = p.accept(DEFINE); !ok {
					p.back(ix)
					goto _5
				}
			}
			break
		_5:
			tok = Token{}
			identifierList = nil
			goto _0
		default:
			goto _0
		}
		goto _1
	_0:
		tok = Token{}
		expressionList = nil
	_1:
		// *ebnf.Name RecvExpr ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			if recvExpr = p.recvExpr(); recvExpr == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &RecvStmtNode{
		ExpressionList: expressionList,
		Token:          tok,
		IdentifierList: identifierList,
		RecvExpr:       recvExpr,
	}
}

func (p *parser) relationalExpression(preBlock bool) (r Expression) {
	var additiveExpression Expression
	// ebnf.Sequence AdditiveExpression { ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) AdditiveExpression } ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Name AdditiveExpression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if additiveExpression = p.additiveExpression(preBlock); additiveExpression == nil {
			p.back(ix)
			return nil
		}
		r = additiveExpression
		// *ebnf.Repetition { ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) AdditiveExpression } ctx []
	_0:
		{
			var op Token
			var additiveExpression Expression
			switch p.c() {
			case EQL, GEQ, GTR, LEQ, LSS, NEQ:
				// ebnf.Sequence ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) AdditiveExpression ctx [EQL, GEQ, GTR, LEQ, LSS, NEQ]
				// *ebnf.Group ( "==" | "!=" | "<" | "<=" | ">" | ">=" ) ctx [EQL, GEQ, GTR, LEQ, LSS, NEQ]
				// ebnf.Alternative "==" | "!=" | "<" | "<=" | ">" | ">=" ctx [EQL, GEQ, GTR, LEQ, LSS, NEQ]
				op = p.consume()
				// *ebnf.Name AdditiveExpression ctx []
				switch p.c() {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
					if additiveExpression = p.additiveExpression(preBlock); additiveExpression == nil {
						p.back(ix)
						goto _1
					}
				default:
					p.back(ix)
					goto _1
				}
				r = &BinaryExpressionNode{LHS: r, Op: op, RHS: additiveExpression}
				goto _0
			}
		_1:
		}
	}
	return r
}

// ResultNode represents the production
//
//	Result = Parameters | Type .
type ResultNode struct {
	Parameters *ParametersNode
	TypeNode   Type
}

// Source implements Node.
func (n *ResultNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ResultNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.Parameters != nil {
		return n.Parameters.Position()
	}

	return n.TypeNode.Position()
}

func (p *parser) result() *ResultNode {
	var (
		parameters *ParametersNode
		typeNode   Type
	)
	// ebnf.Alternative Parameters | Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
	switch p.c() {
	case LPAREN: // 0 1
		// *ebnf.Name Parameters ctx [LPAREN]
		if parameters = p.parameters(); parameters == nil {
			goto _0
		}
		break
	_0:
		parameters = nil
		// *ebnf.Name Type ctx [LPAREN]
		if typeNode = p.type1(); typeNode == nil {
			goto _1
		}
		break
	_1:
		typeNode = nil
		return nil
	case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, MAP, MUL, STRUCT: // 1
		// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, MAP, MUL, STRUCT]
		if typeNode = p.type1(); typeNode == nil {
			goto _2
		}
		break
	_2:
		typeNode = nil
		return nil
	default:
		return nil
	}
	return &ResultNode{
		Parameters: parameters,
		TypeNode:   typeNode,
	}
}

// ReturnStmtNode represents the production
//
//	ReturnStmt = "return" [ ExpressionList ] .
type ReturnStmtNode struct {
	RETURN         Token
	ExpressionList *ExpressionListNode
}

// Source implements Node.
func (n *ReturnStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ReturnStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.RETURN.Position()
}

func (p *parser) returnStmt() *ReturnStmtNode {
	var (
		returnTok      Token
		expressionList *ExpressionListNode
	)
	// ebnf.Sequence "return" [ ExpressionList ] ctx [RETURN]
	{
		// *ebnf.Token "return" ctx [RETURN]
		returnTok = p.expect(RETURN)
		// *ebnf.Option [ ExpressionList ] ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if expressionList = p.expressionList(false); expressionList == nil {
				goto _0
			}
		}
		goto _1
	_0:
		expressionList = nil
	_1:
	}
	return &ReturnStmtNode{
		RETURN:         returnTok,
		ExpressionList: expressionList,
	}
}

// CommClauseListNode represents the production
//
//	CommClauseListNode = { CommClause } .
type CommClauseListNode struct {
	CommClause *CommClauseNode
	List       *CommClauseListNode
}

// Source implements Node.
func (n *CommClauseListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *CommClauseListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.CommClause.Position()
}

// SelectStmtNode represents the production
//
//	SelectStmt = "select" "{" { CommClause } "}" .
type SelectStmtNode struct {
	SELECT         Token
	LBRACE         Token
	CommClauseList *CommClauseListNode
	RBRACE         Token
}

// Source implements Node.
func (n *SelectStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *SelectStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.SELECT.Position()
}

func (p *parser) selectStmt() *SelectStmtNode {
	var (
		ok         bool
		selectTok  Token
		lbraceTok  Token
		list, last *CommClauseListNode
		rbraceTok  Token
	)
	// ebnf.Sequence "select" "{" { CommClause } "}" ctx [SELECT]
	{
		if p.peek(1) != LBRACE {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "select" ctx [SELECT]
		selectTok = p.expect(SELECT)
		// *ebnf.Token "{" ctx [LBRACE]
		lbraceTok = p.expect(LBRACE)
		// *ebnf.Repetition { CommClause } ctx []
	_0:
		{
			var commClause *CommClauseNode
			switch p.c() {
			case CASE, DEFAULT:
				// *ebnf.Name CommClause ctx [CASE, DEFAULT]
				if commClause = p.commClause(); commClause == nil {
					goto _1
				}
				next := &CommClauseListNode{
					CommClause: commClause,
				}
				if last != nil {
					last.List = next
				}
				if list == nil {
					list = next
				}
				last = next
				goto _0
			}
		_1:
		}
		// *ebnf.Token "}" ctx []
		if rbraceTok, ok = p.accept(RBRACE); !ok {
			p.back(ix)
			return nil
		}
	}
	return &SelectStmtNode{
		SELECT:         selectTok,
		LBRACE:         lbraceTok,
		CommClauseList: list,
		RBRACE:         rbraceTok,
	}
}

// SelectorNode represents the production
//
//	Selector = "." identifier .
type SelectorNode struct {
	PERIOD Token
	IDENT  Token
}

// Source implements Node.
func (n *SelectorNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *SelectorNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.PERIOD.Position()
}

func (p *parser) selector() *SelectorNode {
	var (
		periodTok Token
		identTok  Token
	)
	// ebnf.Sequence "." identifier ctx [PERIOD]
	{
		if p.peek(1) != IDENT {
			return nil
		}
		// *ebnf.Token "." ctx [PERIOD]
		periodTok = p.expect(PERIOD)
		// *ebnf.Name identifier ctx [IDENT]
		identTok = p.expect(IDENT)
	}
	return &SelectorNode{
		PERIOD: periodTok,
		IDENT:  identTok,
	}
}

// SendStmtNode represents the production
//
//	SendStmt = Channel "<-" Expression .
type SendStmtNode struct {
	Channel    Expression
	ARROW      Token
	Expression Expression
}

// Source implements Node.
func (n *SendStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *SendStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.Channel.Position()
}

func (p *parser) sendStmt() *SendStmtNode {
	var (
		ok         bool
		channel    Expression
		arrowTok   Token
		expression Expression
	)
	// ebnf.Sequence Channel "<-" Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	{
		ix := p.ix
		// *ebnf.Name Channel ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if channel = p.channel(); channel == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token "<-" ctx []
		if arrowTok, ok = p.accept(ARROW); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Name Expression ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
			if expression = p.expression(false); expression == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &SendStmtNode{
		Channel:    channel,
		ARROW:      arrowTok,
		Expression: expression,
	}
}

// ShortVarDeclNode represents the production
//
//	ShortVarDecl = IdentifierList ":=" ExpressionList .
type ShortVarDeclNode struct {
	IdentifierList *IdentifierListNode
	DEFINE         Token
	ExpressionList *ExpressionListNode

	visible
}

// Source implements Node.
func (n *ShortVarDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ShortVarDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IdentifierList.Position()
}

func (p *parser) shortVarDecl(lhs *ExpressionListNode, preBlock bool) (r *ShortVarDeclNode) {
	var (
		defineTok      Token
		expressionList *ExpressionListNode
	)
	// ebnf.Sequence ":=" ExpressionList ctx [DEFINE]
	{
		switch p.peek(1) {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token ":=" ctx [DEFINE]
		defineTok = p.expect(DEFINE)
		// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		if expressionList = p.expressionList(preBlock); expressionList == nil {
			p.back(ix)
			return nil
		}
	}
	list := p.exprList2identList(lhs)
	sc := p.sc
	r = &ShortVarDeclNode{
		IdentifierList: list,
		DEFINE:         defineTok,
		ExpressionList: expressionList,
	}
	visible := int32(p.ix)
	hasNew := false
	for n := r.IdentifierList; n != nil; n = n.List {
		id := n.IDENT
		ex := sc.declare(id, r, visible, p, false)
		if !ex.declTok.IsValid() {
			hasNew = true
		}
	}
	if !hasNew {
		for n := r.IdentifierList; n != nil; n = n.List {
			id := n.IDENT
			nm := id.Src()
			ex := sc.nodes[nm]
			if ex.declTok.IsValid() {
				p.err(id.Position(), "%s redeclared, previous declaration at %v: (%p)", nm, ex.declTok.Position(), sc)
			}
		}
	}
	return r
}

func (p *parser) exprList2identList(list *ExpressionListNode) (r *IdentifierListNode) {
	var last *IdentifierListNode
	for n := list; n != nil; n = n.List {
		next := &IdentifierListNode{
			COMMA: n.COMMA,
			IDENT: p.expr2ident(n.Expression),
		}
		if !next.IDENT.IsValid() {
			continue
		}

		if r == nil {
			r = next
		}
		if last != nil {
			last.List = next
		}
		last = next
	}
	return r
}

func (p *parser) expr2ident(e Expression) (r Token) {
	switch x := e.(type) {
	case *OperandNode:
		if (x.TypeArgs != nil || x.LiteralValue != nil) && p.reportDeclarationErrors {
			p.err(x.Position(), "expected identifier")
			break
		}

		return p.expr2ident(x.OperandName)
	case *OperandNameNode:
		return x.Name

		p.err(x.Position(), "expected identifier")
	default:
		if p.reportDeclarationErrors {
			p.err(x.Position(), "expected identifier")
		}
	}
	return r
}

// SignatureNode represents the production
//
//	Signature = Parameters [ Result ] .
type SignatureNode struct {
	Parameters *ParametersNode
	Result     *ResultNode

	typeCache
}

// Source implements Node.
func (n *SignatureNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *SignatureNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.Parameters.Position()
}

func (p *parser) signature() *SignatureNode {
	var (
		parameters *ParametersNode
		result     *ResultNode
	)
	// ebnf.Sequence Parameters [ Result ] ctx [LPAREN]
	{
		ix := p.ix
		// *ebnf.Name Parameters ctx [LPAREN]
		if parameters = p.parameters(); parameters == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ Result ] ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			// *ebnf.Name Result ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
			if result = p.result(); result == nil {
				goto _0
			}
		}
		goto _1
	_0:
		result = nil
	_1:
	}
	return &SignatureNode{
		Parameters: parameters,
		Result:     result,
	}
}

// IncDecStmtNode represents the production
//
//	IncDecStmt = Expression ( "++" | "--" ) .
type IncDecStmtNode struct {
	Expression Expression
	Token      Token
}

// Source implements Node.
func (n *IncDecStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *IncDecStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.Expression.Position()
}

func (p *parser) simpleStmt(preBlock bool) Node {
	var (
		expressionList *ExpressionListNode
		assignment     *AssignmentNode
		shortVarDecl   *ShortVarDeclNode
		arrowTok       Token
		expression     Expression
		emptyStmt      *EmptyStmtNode
	)
	// ebnf.Alternative ExpressionList [ Assignment | ShortVarDecl | "<-" Expression | "++" | "--" ] | EmptyStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */]
	switch p.c() {
	case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR: // 0
		// ebnf.Sequence ExpressionList [ Assignment | ShortVarDecl | "<-" Expression | "++" | "--" ] ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
		{
			ix := p.ix
			// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
			if expressionList = p.expressionList(preBlock); expressionList == nil {
				p.back(ix)
				goto _0
			}
			// *ebnf.Option [ Assignment | ShortVarDecl | "<-" Expression | "++" | "--" ] ctx []
			switch p.c() {
			case ADD_ASSIGN, AND_ASSIGN, AND_NOT_ASSIGN, ARROW, ASSIGN, DEC, DEFINE, INC, MUL_ASSIGN, OR_ASSIGN, QUO_ASSIGN, REM_ASSIGN, SHL_ASSIGN, SHR_ASSIGN, SUB_ASSIGN, XOR_ASSIGN:
				// ebnf.Alternative Assignment | ShortVarDecl | "<-" Expression | "++" | "--" ctx [ADD_ASSIGN, AND_ASSIGN, AND_NOT_ASSIGN, ARROW, ASSIGN, DEC, DEFINE, INC, MUL_ASSIGN, OR_ASSIGN, QUO_ASSIGN, REM_ASSIGN, SHL_ASSIGN, SHR_ASSIGN, SUB_ASSIGN, XOR_ASSIGN]
				switch p.c() {
				case ADD_ASSIGN, AND_ASSIGN, AND_NOT_ASSIGN, ASSIGN, MUL_ASSIGN, OR_ASSIGN, QUO_ASSIGN, REM_ASSIGN, SHL_ASSIGN, SHR_ASSIGN, SUB_ASSIGN, XOR_ASSIGN: // 0
					// *ebnf.Name Assignment ctx [ADD_ASSIGN, AND_ASSIGN, AND_NOT_ASSIGN, ASSIGN, MUL_ASSIGN, OR_ASSIGN, QUO_ASSIGN, REM_ASSIGN, SHL_ASSIGN, SHR_ASSIGN, SUB_ASSIGN, XOR_ASSIGN]
					if assignment = p.assignment(expressionList, preBlock); assignment == nil {
						goto _4
					}
					return assignment
				_4:
					assignment = nil
					goto _2
				case DEFINE: // 1
					// *ebnf.Name ShortVarDecl ctx [DEFINE]
					if shortVarDecl = p.shortVarDecl(expressionList, preBlock); shortVarDecl == nil {
						goto _6
					}
					return shortVarDecl
				_6:
					shortVarDecl = nil
					goto _2
				case ARROW: // 2
					// ebnf.Sequence "<-" Expression ctx [ARROW]
					{
						switch p.peek(1) {
						case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
						default:
							goto _8
						}
						ix := p.ix
						// *ebnf.Token "<-" ctx [ARROW]
						arrowTok = p.expect(ARROW)
						// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
						if expression = p.expression(preBlock); expression == nil {
							p.back(ix)
							goto _8
						}
					}
					if expressionList.Len() > 1 {
						p.err(expressionList.Position(), "expected one expression: %s", expressionList.Source(false))
					}
					return &SendStmtNode{
						Channel:    expressionList.first(),
						ARROW:      arrowTok,
						Expression: expression,
					}

				_8:
					arrowTok = Token{}
					expression = nil
					goto _2
				case INC: // 3
					// *ebnf.Token "++" ctx [INC]
					if expressionList.Len() > 1 {
						p.err(expressionList.Position(), "expected one expression: %s", expressionList.Source(false))
					}
					return &IncDecStmtNode{
						Expression: expressionList.first(),
						Token:      p.expect(INC),
					}
				case DEC: // 4
					// *ebnf.Token "--" ctx [DEC]
					if expressionList.Len() > 1 {
						p.err(expressionList.Position(), "expected one expression: %s", expressionList.Source(false))
					}
					return &IncDecStmtNode{
						Expression: expressionList.first(),
						Token:      p.expect(DEC),
					}
				default:
					goto _2
				}
			}
			goto _3
		_2:
			arrowTok = Token{}
			assignment = nil
			expression = nil
			shortVarDecl = nil
		_3:
		}
		break
	_0:
		arrowTok = Token{}
		assignment = nil
		expression = nil
		expressionList = nil
		shortVarDecl = nil
		return nil
	default: //  /* ε */ 1
		// *ebnf.Name EmptyStmt ctx [ /* ε */]
		if emptyStmt = p.emptyStmt(); emptyStmt == nil {
			goto _14
		}
		return emptyStmt
	_14:
		emptyStmt = nil
		return nil
	}
	if expressionList == nil || expressionList.Len() > 1 {
		return nil
	}

	return expressionList.first()
}

// SliceNode represents the production
//
//	Slice = "[" [ Expression ] ":" [ Expression ] "]" | "[" [ Expression ] ":" Expression ":" Expression "]" .
type SliceNode struct {
	LBRACK      Token
	Expression  Expression
	COLON       Token
	Expression2 Expression
	RBRACK      Token
	COLON2      Token
	Expression3 Expression
}

// Source implements Node.
func (n *SliceNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *SliceNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) slice() *SliceNode {
	var (
		ok          bool
		lbrackTok   Token
		expression  Expression
		colonTok    Token
		expression2 Expression
		rbrackTok   Token
		colon2Tok   Token
		expression3 Expression
	)
	// ebnf.Alternative "[" [ Expression ] ":" [ Expression ] "]" | "[" [ Expression ] ":" Expression ":" Expression "]" ctx [LBRACK]
	switch p.c() {
	case LBRACK: // 0 1
		// ebnf.Sequence "[" [ Expression ] ":" [ Expression ] "]" ctx [LBRACK]
		{
			ix := p.ix
			// *ebnf.Token "[" ctx [LBRACK]
			lbrackTok = p.expect(LBRACK)
			// *ebnf.Option [ Expression ] ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if expression = p.expression(false); expression == nil {
					goto _1
				}
			}
			goto _2
		_1:
			expression = nil
		_2:
			// *ebnf.Token ":" ctx []
			if colonTok, ok = p.accept(COLON); !ok {
				p.back(ix)
				goto _0
			}
			// *ebnf.Option [ Expression ] ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if expression2 = p.expression(false); expression2 == nil {
					goto _3
				}
			}
			goto _4
		_3:
			expression2 = nil
		_4:
			// *ebnf.Token "]" ctx []
			if rbrackTok, ok = p.accept(RBRACK); !ok {
				p.back(ix)
				goto _0
			}
		}
		break
	_0:
		colonTok = Token{}
		expression = nil
		expression2 = nil
		lbrackTok = Token{}
		rbrackTok = Token{}
		// ebnf.Sequence "[" [ Expression ] ":" Expression ":" Expression "]" ctx [LBRACK]
		{
			ix := p.ix
			// *ebnf.Token "[" ctx [LBRACK]
			lbrackTok = p.expect(LBRACK)
			// *ebnf.Option [ Expression ] ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				// *ebnf.Name Expression ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if expression = p.expression(false); expression == nil {
					goto _6
				}
			}
			goto _7
		_6:
			expression = nil
		_7:
			// *ebnf.Token ":" ctx []
			if colonTok, ok = p.accept(COLON); !ok {
				p.back(ix)
				goto _5
			}
			// *ebnf.Name Expression ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				if expression2 = p.expression(false); expression2 == nil {
					p.back(ix)
					goto _5
				}
			default:
				p.back(ix)
				goto _5
			}
			// *ebnf.Token ":" ctx []
			if colon2Tok, ok = p.accept(COLON); !ok {
				p.back(ix)
				goto _5
			}
			// *ebnf.Name Expression ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				if expression3 = p.expression(false); expression3 == nil {
					p.back(ix)
					goto _5
				}
			default:
				p.back(ix)
				goto _5
			}
			// *ebnf.Token "]" ctx []
			if rbrackTok, ok = p.accept(RBRACK); !ok {
				p.back(ix)
				goto _5
			}
		}
		break
	_5:
		colon2Tok = Token{}
		colonTok = Token{}
		expression = nil
		expression2 = nil
		expression3 = nil
		lbrackTok = Token{}
		rbrackTok = Token{}
		return nil
	default:
		return nil
	}
	return &SliceNode{
		LBRACK:      lbrackTok,
		Expression:  expression,
		COLON:       colonTok,
		Expression2: expression2,
		RBRACK:      rbrackTok,
		COLON2:      colon2Tok,
		Expression3: expression3,
	}
}

// SliceTypeNode represents the production
//
//	SliceType = "[" "]" ElementType .
type SliceTypeNode struct {
	LBRACK      Token
	RBRACK      Token
	ElementType Node

	guard
}

// Source implements Node.
func (n *SliceTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *SliceTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LBRACK.Position()
}

func (p *parser) sliceType() *SliceTypeNode {
	var (
		lbrackTok   Token
		rbrackTok   Token
		elementType Node
	)
	// ebnf.Sequence "[" "]" ElementType ctx [LBRACK]
	{
		if p.peek(1) != RBRACK {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "[" ctx [LBRACK]
		lbrackTok = p.expect(LBRACK)
		// *ebnf.Token "]" ctx [RBRACK]
		rbrackTok = p.expect(RBRACK)
		// *ebnf.Name ElementType ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			if elementType = p.type1(); elementType == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &SliceTypeNode{
		LBRACK:      lbrackTok,
		RBRACK:      rbrackTok,
		ElementType: elementType,
	}
}

// ImportDeclListNode represents the production
//
//	ImportDeclListNode = { ImportDecl ";" } .
type ImportDeclListNode struct {
	ImportDecl *ImportDeclNode
	SEMICOLON  Token
	List       *ImportDeclListNode
}

// Source implements Node.
func (n *ImportDeclListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ImportDeclListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.ImportDecl.Position()
}

// TopLevelDeclListNode represents the production
//
//	TopLevelDeclListNode = { TopLevelDecl ";" .
type TopLevelDeclListNode struct {
	TopLevelDecl Node
	SEMICOLON    Token
	List         *TopLevelDeclListNode
}

// Source implements Node.
func (n *TopLevelDeclListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TopLevelDeclListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TopLevelDecl.Position()
}

// SourceFileNode represents the production
//
//	SourceFile = PackageClause ";" { ImportDecl ";" } { TopLevelDecl ";" } .
type SourceFileNode struct {
	PackageClause    *PackageClauseNode
	SEMICOLON        Token
	ImportDeclList   *ImportDeclListNode
	TopLevelDeclList *TopLevelDeclListNode
}

// Source implements Node.
func (n *SourceFileNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *SourceFileNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.PackageClause.Position()
}

func (p *parser) sourceFile() *SourceFileNode {
	var (
		ok            bool
		packageClause *PackageClauseNode
		semicolonTok  Token
		list, last    *ImportDeclListNode
		list2, last2  *TopLevelDeclListNode
	)
	// ebnf.Sequence PackageClause ";" { ImportDecl ";" } { TopLevelDecl ";" } ctx [PACKAGE]
	{
		ix := p.ix
		// *ebnf.Name PackageClause ctx [PACKAGE]
		if packageClause = p.packageClause(); packageClause == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token ";" ctx []
		if semicolonTok, ok = p.accept(SEMICOLON); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Repetition { ImportDecl ";" } ctx []
	_0:
		{
			var importDecl *ImportDeclNode
			var semicolonTok Token
			switch p.c() {
			case IMPORT:
				// ebnf.Sequence ImportDecl ";" ctx [IMPORT]
				ix := p.ix
				// *ebnf.Name ImportDecl ctx [IMPORT]
				if importDecl = p.importDecl(); importDecl == nil {
					p.back(ix)
					goto _1
				}
				// *ebnf.Token ";" ctx []
				if semicolonTok, ok = p.accept(SEMICOLON); !ok {
					p.back(ix)
					goto _1
				}
				next := &ImportDeclListNode{
					ImportDecl: importDecl,
					SEMICOLON:  semicolonTok,
				}
				if last != nil {
					last.List = next
				}
				if list == nil {
					list = next
				}
				last = next
				goto _0
			}
		_1:
		}
		// *ebnf.Repetition { TopLevelDecl ";" } ctx []
	_2:
		{
			var topLevelDecl Node
			var semicolonTok Token
			switch p.c() {
			case CONST, FUNC, TYPE, VAR:
				// ebnf.Sequence TopLevelDecl ";" ctx [CONST, FUNC, TYPE, VAR]
				ix := p.ix
				// *ebnf.Name TopLevelDecl ctx [CONST, FUNC, TYPE, VAR]
				if topLevelDecl = p.topLevelDecl(); topLevelDecl == nil {
					p.back(ix)
					goto _3
				}
				// *ebnf.Token ";" ctx []
				if semicolonTok, ok = p.accept(SEMICOLON); !ok {
					p.back(ix)
					goto _3
				}
				next := &TopLevelDeclListNode{
					TopLevelDecl: topLevelDecl,
					SEMICOLON:    semicolonTok,
				}
				if last2 != nil {
					last2.List = next
				}
				if list2 == nil {
					list2 = next
				}
				last2 = next
				goto _2
			}
		_3:
		}
	}
	return &SourceFileNode{
		PackageClause:    packageClause,
		SEMICOLON:        semicolonTok,
		ImportDeclList:   list,
		TopLevelDeclList: list2,
	}
}

func (p *parser) statement() Node {
	var (
		declaration     Node
		labeledStmt     *LabeledStmtNode
		goStmt          *GoStmtNode
		returnStmt      *ReturnStmtNode
		breakStmt       *BreakStmtNode
		continueStmt    *ContinueStmtNode
		gotoStmt        *GotoStmtNode
		fallthroughStmt *FallthroughStmtNode
		block           *BlockNode
		ifStmt          Node
		switchStmt      *SwitchStmtNode
		selectStmt      *SelectStmtNode
		forStmt         *ForStmtNode
		deferStmt       *DeferStmtNode
		simpleStmt      Node
	)
	// ebnf.Alternative Declaration | LabeledStmt | GoStmt | ReturnStmt | BreakStmt | ContinueStmt | GotoStmt | FallthroughStmt | Block | IfStmt | SwitchStmt | SelectStmt | ForStmt | DeferStmt | SimpleStmt ctx [ADD, AND, ARROW, BREAK, CHAN, CHAR, CONST, CONTINUE, DEFER, FALLTHROUGH, FLOAT, FOR, FUNC, GO, GOTO, IDENT, IF, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, RETURN, SELECT, STRING, STRUCT, SUB, SWITCH, TYPE, VAR, XOR /* ε */]
	switch p.c() {
	case CONST, TYPE, VAR: // 0
		// *ebnf.Name Declaration ctx [CONST, TYPE, VAR]
		if declaration = p.declaration(); declaration == nil {
			return nil
		}
		return declaration
	case IDENT: // 1 14
		// *ebnf.Name LabeledStmt ctx [IDENT]
		if labeledStmt = p.labeledStmt(); labeledStmt == nil {
			goto _2
		}
		return labeledStmt
	_2:
		labeledStmt = nil
		// *ebnf.Name SimpleStmt ctx [IDENT]
		if simpleStmt = p.simpleStmt(false); simpleStmt == nil {
			return nil
		}
		return simpleStmt
	case GO: // 2
		// *ebnf.Name GoStmt ctx [GO]
		if goStmt = p.goStmt(); goStmt == nil {
			return nil
		}
		return goStmt
	case RETURN: // 3
		// *ebnf.Name ReturnStmt ctx [RETURN]
		if returnStmt = p.returnStmt(); returnStmt == nil {
			return nil
		}
		return returnStmt
	case BREAK: // 4
		// *ebnf.Name BreakStmt ctx [BREAK]
		if breakStmt = p.breakStmt(); breakStmt == nil {
			return nil
		}
		return breakStmt
	case CONTINUE: // 5
		// *ebnf.Name ContinueStmt ctx [CONTINUE]
		if continueStmt = p.continueStmt(); continueStmt == nil {
			return nil
		}
		return continueStmt
	case GOTO: // 6
		// *ebnf.Name GotoStmt ctx [GOTO]
		if gotoStmt = p.gotoStmt(); gotoStmt == nil {
			return nil
		}
		return gotoStmt
	case FALLTHROUGH: // 7
		// *ebnf.Name FallthroughStmt ctx [FALLTHROUGH]
		if fallthroughStmt = p.fallthroughStmt(); fallthroughStmt == nil {
			return nil
		}
		return fallthroughStmt
	case LBRACE: // 8
		// *ebnf.Name Block ctx [LBRACE]
		if block = p.block(nil, nil); block == nil {
			return nil
		}
		return block
		return nil
	case IF: // 9
		// *ebnf.Name IfStmt ctx [IF]
		if ifStmt = p.ifStmt(); ifStmt == nil {
			return nil
		}
		return ifStmt
	case SWITCH: // 10
		// *ebnf.Name SwitchStmt ctx [SWITCH]
		if switchStmt = p.switchStmt(); switchStmt == nil {
			return nil
		}
		return switchStmt
	case SELECT: // 11
		// *ebnf.Name SelectStmt ctx [SELECT]
		if selectStmt = p.selectStmt(); selectStmt == nil {
			return nil
		}
		return selectStmt
	case FOR: // 12
		// *ebnf.Name ForStmt ctx [FOR]
		if forStmt = p.forStmt(); forStmt == nil {
			return nil
		}
		return forStmt
	case DEFER: // 13
		// *ebnf.Name DeferStmt ctx [DEFER]
		if deferStmt = p.deferStmt(); deferStmt == nil {
			return nil
		}
		return deferStmt
	case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */ : // 14
		// *ebnf.Name SimpleStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR /* ε */]
		if simpleStmt = p.simpleStmt(false); simpleStmt == nil {
			return nil
		}
		return simpleStmt
	}
	return nil
}

// StatementListNode represents the production
//
//	StatementList = { Statement ";" } .
type StatementListNode struct {
	Statement Node
	SEMICOLON Token
	List      *StatementListNode
}

// Source implements Node.
func (n *StatementListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *StatementListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) statementList() *StatementListNode {
	var (
		statement  Node
		list, last *StatementListNode
	)
	for {
		ix := p.ix
		switch p.c() {
		case ADD, AND, ARROW, BREAK, CHAN, CHAR, CONST, CONTINUE, DEFER, FALLTHROUGH, FLOAT, FOR, FUNC, GO, GOTO, IDENT, IF, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, RETURN, SELECT, STRING, STRUCT, SUB, SWITCH, TYPE, VAR, XOR /* ε */ :
			if statement = p.statement(); statement == nil {
				p.back(ix)
				goto _1
			}
		case SEMICOLON:
			next := &StatementListNode{
				SEMICOLON: p.consume(),
			}
			if last != nil {
				last.List = next
			}
			if list == nil {
				list = next
			}
			last = next
			continue
		default:
			goto _1
		}

		if p.c() != SEMICOLON {
			next := &StatementListNode{
				Statement: statement,
			}
			if last != nil {
				last.List = next
			}
			if list == nil {
				list = next
			}
			last = next
			goto _1
		}
		next := &StatementListNode{
			Statement: statement,
			SEMICOLON: p.consume(),
		}
		if last != nil {
			last.List = next
		}
		if list == nil {
			list = next
		}
		last = next
	}
_1:
	return list
}

// FieldDeclListNode represents the production
//
//	FieldDeclListNode = { FieldDecl ";" } .
type FieldDeclListNode struct {
	FieldDecl *FieldDeclNode
	SEMICOLON Token
	List      *FieldDeclListNode
}

// Source implements Node.
func (n *FieldDeclListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *FieldDeclListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.FieldDecl.Position()
}

// StructTypeNode represents the production
//
//	StructType = "struct" "{" { FieldDecl ";" } "}" .
type StructTypeNode struct {
	STRUCT        Token
	LBRACE        Token
	FieldDeclList *FieldDeclListNode
	RBRACE        Token
	fields        []Field

	guard
}

// Source implements Node.
func (n *StructTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *StructTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.STRUCT.Position()
}

func (p *parser) structType() *StructTypeNode {
	var (
		ok         bool
		structTok  Token
		lbraceTok  Token
		list, last *FieldDeclListNode
		rbraceTok  Token
	)
	// ebnf.Sequence "struct" "{" { FieldDecl ";" } [ FieldDecl ] "}" ctx [STRUCT]
	{
		if p.peek(1) != LBRACE {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "struct" ctx [STRUCT]
		structTok = p.expect(STRUCT)
		// *ebnf.Token "{" ctx [LBRACE]
		lbraceTok = p.expect(LBRACE)
		// *ebnf.Repetition { FieldDecl ";" } ctx []
	_0:
		{
			var fieldDecl *FieldDeclNode
			var semicolonTok Token
			switch p.c() {
			case IDENT, MUL:
				// ebnf.Sequence FieldDecl ";" ctx [IDENT, MUL]
				ix := p.ix
				// *ebnf.Name FieldDecl ctx [IDENT, MUL]
				if fieldDecl = p.fieldDecl(); fieldDecl == nil {
					p.back(ix)
					goto _1
				}
				if p.c() == RBRACE {
					next := &FieldDeclListNode{
						FieldDecl: fieldDecl,
					}
					if last != nil {
						last.List = next
					}
					if list == nil {
						list = next
					}
					last = next
					goto _1
				}
				// *ebnf.Token ";" ctx []
				if semicolonTok, ok = p.accept(SEMICOLON); !ok {
					p.back(ix)
					goto _1
				}
				next := &FieldDeclListNode{
					FieldDecl: fieldDecl,
					SEMICOLON: semicolonTok,
				}
				if last != nil {
					last.List = next
				}
				if list == nil {
					list = next
				}
				last = next
				goto _0
			}
		_1:
		}
		// *ebnf.Token "}" ctx []
		if rbraceTok, ok = p.accept(RBRACE); !ok {
			p.back(ix)
			return nil
		}
	}
	return &StructTypeNode{
		STRUCT:        structTok,
		LBRACE:        lbraceTok,
		FieldDeclList: list,
		RBRACE:        rbraceTok,
	}
}

// SwitchStmtNode represents the production
//
//	SwitchStmt = ExprSwitchStmt | TypeSwitchStmt .
type SwitchStmtNode struct {
	ExprSwitchStmt *ExprSwitchStmtNode
	TypeSwitchStmt *TypeSwitchStmtNode
}

// Source implements Node.
func (n *SwitchStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *SwitchStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) switchStmt() *SwitchStmtNode {
	var (
		exprSwitchStmt *ExprSwitchStmtNode
		typeSwitchStmt *TypeSwitchStmtNode
	)
	p.openScope()

	defer p.closeScope()

	// ebnf.Alternative ExprSwitchStmt | TypeSwitchStmt ctx [SWITCH]
	switch p.c() {
	case SWITCH: // 0 1
		// *ebnf.Name ExprSwitchStmt ctx [SWITCH]
		if exprSwitchStmt = p.exprSwitchStmt(); exprSwitchStmt == nil {
			goto _0
		}
		break
	_0:
		exprSwitchStmt = nil
		p.closeScope()
		p.openScope()
		// *ebnf.Name TypeSwitchStmt ctx [SWITCH]
		if typeSwitchStmt = p.typeSwitchStmt(); typeSwitchStmt == nil {
			goto _1
		}
		break
	_1:
		typeSwitchStmt = nil
		return nil
	default:
		return nil
	}
	return &SwitchStmtNode{
		ExprSwitchStmt: exprSwitchStmt,
		TypeSwitchStmt: typeSwitchStmt,
	}
}

// TagNode represents the production
//
//	Tag = string_lit .
type TagNode struct {
	STRING Token
}

// Source implements Node.
func (n *TagNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TagNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.STRING.Position()
}

func (p *parser) tag() *TagNode {
	var (
		stringTok Token
	)
	// *ebnf.Name string_lit ctx [STRING]
	stringTok = p.expect(STRING)
	return &TagNode{
		STRING: stringTok,
	}
}

func (p *parser) topLevelDecl() (r Node) {
	// ebnf.Alternative Declaration | FunctionDecl | MethodDecl ctx [CONST, FUNC, TYPE, VAR]
	switch p.c() {
	case CONST, TYPE, VAR: // 0
		// *ebnf.Name Declaration ctx [CONST, TYPE, VAR]
		return p.declaration()
	case FUNC: // 1 2
		// *ebnf.Name FunctionDecl ctx [FUNC]
		if functionDecl := p.functionDecl(); functionDecl != nil {
			return functionDecl
		}
		// *ebnf.Name MethodDecl ctx [FUNC]
		return p.methodDecl()
	}
	return nil
}

// TypeNode represents the production
//
//	Type = TypeName TypeArgs .
type TypeNode struct {
	TypeName *TypeNameNode
	TypeArgs *TypeArgsNode
}

// Source implements Node.
func (n *TypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.TypeName != nil {
		return n.TypeName.Position()
	}

	return r
}

// ParenthesizedTypeNode represents the production
//
//	ParenthesizedType = "(" Type ")" .
type ParenthesizedTypeNode struct {
	LPAREN   Token
	TypeNode Type
	RPAREN   Token
}

// Source implements Node.
func (n *ParenthesizedTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *ParenthesizedTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.LPAREN.IsValid() {
		return n.LPAREN.Position()
	}

	return r
}

func (p *parser) type1() Type {
	var (
		ok        bool
		typeName  *TypeNameNode
		typeArgs  *TypeArgsNode
		typeLit   Type
		lparenTok Token
		typeNode  Type
		rparenTok Token
	)
	// ebnf.Alternative TypeName [ TypeArgs ] | TypeLit | "(" Type ")" ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
	switch p.c() {
	case IDENT: // 0
		// ebnf.Sequence TypeName [ TypeArgs ] ctx [IDENT]
		{
			ix := p.ix
			// *ebnf.Name TypeName ctx [IDENT]
			if typeName = p.typeName(); typeName == nil {
				p.back(ix)
				goto _0
			}
			// *ebnf.Option [ TypeArgs ] ctx []
			switch p.c() {
			case LBRACK:
				// *ebnf.Name TypeArgs ctx [LBRACK]
				if typeArgs = p.typeArgs(); typeArgs == nil {
					goto _2
				}
			}
			goto _3
		_2:
			typeArgs = nil
		_3:
		}
		if typeArgs == nil {
			return typeName
		}

		break
	_0:
		typeArgs = nil
		typeName = nil
		return nil
	case ARROW, CHAN, FUNC, INTERFACE, LBRACK, MAP, MUL, STRUCT: // 1
		// *ebnf.Name TypeLit ctx [ARROW, CHAN, FUNC, INTERFACE, LBRACK, MAP, MUL, STRUCT]
		if typeLit = p.typeLit(); typeLit == nil {
			goto _4
		}
		return typeLit
	_4:
		typeLit = nil
		return nil
	case LPAREN: // 2
		// ebnf.Sequence "(" Type ")" ctx [LPAREN]
		{
			switch p.peek(1) {
			case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			default:
				goto _6
			}
			ix := p.ix
			// *ebnf.Token "(" ctx [LPAREN]
			lparenTok = p.expect(LPAREN)
			// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
			if typeNode = p.type1(); typeNode == nil {
				p.back(ix)
				goto _6
			}
			// *ebnf.Token ")" ctx []
			if rparenTok, ok = p.accept(RPAREN); !ok {
				p.back(ix)
				goto _6
			}
		}
		return &ParenthesizedTypeNode{LPAREN: lparenTok, TypeNode: typeNode, RPAREN: rparenTok}
	_6:
		lparenTok = Token{}
		rparenTok = Token{}
		typeNode = nil
		return nil
	default:
		return nil
	}
	return &TypeNode{
		TypeName: typeName,
		TypeArgs: typeArgs,
	}
}

// TypeArgsNode represents the production
//
//	TypeArgs = "[" TypeList [ "," ] "]" .
type TypeArgsNode struct {
	LBRACK   Token
	TypeList *TypeListNode
	COMMA    Token
	RBRACK   Token
}

// Source implements Node.
func (n *TypeArgsNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeArgsNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LBRACK.Position()
}

func (p *parser) typeArgs() *TypeArgsNode {
	var (
		ok        bool
		lbrackTok Token
		typeList  *TypeListNode
		commaTok  Token
		rbrackTok Token
	)
	// ebnf.Sequence "[" TypeList [ "," ] "]" ctx [LBRACK]
	{
		switch p.peek(1) {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "[" ctx [LBRACK]
		lbrackTok = p.expect(LBRACK)
		// *ebnf.Name TypeList ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		if typeList = p.typeList(); typeList == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ "," ] ctx []
		switch p.c() {
		case COMMA:
			// *ebnf.Token "," ctx [COMMA]
			commaTok = p.expect(COMMA)
		}
		// *ebnf.Token "]" ctx []
		if rbrackTok, ok = p.accept(RBRACK); !ok {
			p.back(ix)
			return nil
		}
	}
	return &TypeArgsNode{
		LBRACK:   lbrackTok,
		TypeList: typeList,
		COMMA:    commaTok,
		RBRACK:   rbrackTok,
	}
}

// TypeAssertionNode represents the production
//
//	TypeAssertion = "." "(" Type ")" .
type TypeAssertionNode struct {
	PERIOD   Token
	LPAREN   Token
	TypeNode Type
	RPAREN   Token
}

// Source implements Node.
func (n *TypeAssertionNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeAssertionNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.PERIOD.Position()
}

func (p *parser) typeAssertion() *TypeAssertionNode {
	var (
		ok        bool
		periodTok Token
		lparenTok Token
		typeNode  Type
		rparenTok Token
	)
	// ebnf.Sequence "." "(" Type ")" ctx [PERIOD]
	{
		if p.peek(1) != LPAREN {
			return nil
		}
		ix := p.ix
		// *ebnf.Token "." ctx [PERIOD]
		periodTok = p.expect(PERIOD)
		// *ebnf.Token "(" ctx [LPAREN]
		lparenTok = p.expect(LPAREN)
		// *ebnf.Name Type ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			if typeNode = p.type1(); typeNode == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Token ")" ctx []
		if rparenTok, ok = p.accept(RPAREN); !ok {
			p.back(ix)
			return nil
		}
	}
	return &TypeAssertionNode{
		PERIOD:   periodTok,
		LPAREN:   lparenTok,
		TypeNode: typeNode,
		RPAREN:   rparenTok,
	}
}

// TypeCaseClauseNode represents the production
//
//	TypeCaseClause = TypeSwitchCase ":" StatementList .
type TypeCaseClauseNode struct {
	TypeSwitchCase *TypeSwitchCaseNode
	COLON          Token
	StatementList  *StatementListNode
}

// Source implements Node.
func (n *TypeCaseClauseNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeCaseClauseNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TypeSwitchCase.Position()
}

func (p *parser) typeCaseClause() *TypeCaseClauseNode {
	var (
		ok             bool
		typeSwitchCase *TypeSwitchCaseNode
		colonTok       Token
		statementList  *StatementListNode
	)
	// ebnf.Sequence TypeSwitchCase ":" StatementList ctx [CASE, DEFAULT]
	{
		p.openScope()

		defer p.closeScope()

		ix := p.ix
		// *ebnf.Name TypeSwitchCase ctx [CASE, DEFAULT]
		if typeSwitchCase = p.typeSwitchCase(); typeSwitchCase == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Token ":" ctx []
		if colonTok, ok = p.accept(COLON); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Name StatementList ctx []
		switch p.c() {
		case ADD, AND, ARROW, BREAK, CHAN, CHAR, CONST, CONTINUE, DEFER, FALLTHROUGH, FLOAT, FOR, FUNC, GO, GOTO, IDENT, IF, IMAG, INT, INTERFACE, LBRACE, LBRACK, LPAREN, MAP, MUL, NOT, RETURN, SELECT, SEMICOLON, STRING, STRUCT, SUB, SWITCH, TYPE, VAR, XOR /* ε */ :
			if statementList = p.statementList(); statementList == nil {
				p.back(ix)
				return nil
			}
		}
	}
	return &TypeCaseClauseNode{
		TypeSwitchCase: typeSwitchCase,
		COLON:          colonTok,
		StatementList:  statementList,
	}
}

// TypeConstraintNode represents the production
//
//	TypeConstraint = TypeElem .
type TypeConstraintNode struct {
	TypeElem *TypeElemListNode
}

// Source implements Node.
func (n *TypeConstraintNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeConstraintNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TypeElem.Position()
}

func (p *parser) typeConstraint() *TypeConstraintNode {
	var (
		typeElem *TypeElemListNode
	)
	// *ebnf.Name TypeElem ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
	if typeElem = p.typeElem(); typeElem == nil {
		return nil
	}
	return &TypeConstraintNode{
		TypeElem: typeElem,
	}
}

// TypeSpecListNode represents the production
//
//	TypeSpecListNode = "type" ( TypeSpec | "(" { TypeSpec ";" } ")" ) .
type TypeSpecListNode struct {
	TypeSpec  Node
	SEMICOLON Token
	List      *TypeSpecListNode
}

// Source implements Node.
func (n *TypeSpecListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeSpecListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TypeSpec.Position()
}

// TypeDeclNode represents the production
//
//	TypeDecl = "type" ( TypeSpec | "(" { TypeSpec ";" } ")" ) .
type TypeDeclNode struct {
	TYPE         Token
	LPAREN       Token
	TypeSpecList *TypeSpecListNode
	RPAREN       Token
}

// Source implements Node.
func (n *TypeDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TYPE.Position()
}

func (p *parser) typeDecl() *TypeDeclNode {
	var (
		ok         bool
		typeTok    Token
		typeSpec   Node
		lparenTok  Token
		list, last *TypeSpecListNode
		rparenTok  Token
	)
	// ebnf.Sequence "type" ( TypeSpec | "(" { TypeSpec ";" } ")" ) ctx [TYPE]
	{
		switch p.peek(1) {
		case IDENT, LPAREN:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "type" ctx [TYPE]
		typeTok = p.expect(TYPE)
		// *ebnf.Group ( TypeSpec | "(" { TypeSpec ";" } ")" ) ctx [IDENT, LPAREN]
		// ebnf.Alternative TypeSpec | "(" { TypeSpec ";" } ")" ctx [IDENT, LPAREN]
		switch p.c() {
		case IDENT: // 0
			// *ebnf.Name TypeSpec ctx [IDENT]
			if typeSpec = p.typeSpec(); typeSpec == nil {
				goto _0
			}
			list = &TypeSpecListNode{
				TypeSpec: typeSpec,
			}
			break
		_0:
			typeSpec = nil
			p.back(ix)
			return nil
		case LPAREN: // 1
			// ebnf.Sequence "(" { TypeSpec ";" } ")" ctx [LPAREN]
			{
				ix := p.ix
				// *ebnf.Token "(" ctx [LPAREN]
				lparenTok = p.expect(LPAREN)
				// *ebnf.Repetition { TypeSpec ";" } ctx []
			_4:
				{
					var typeSpec Node
					var semicolonTok Token
					switch p.c() {
					case IDENT:
						// ebnf.Sequence TypeSpec ";" ctx [IDENT]
						ix := p.ix
						// *ebnf.Name TypeSpec ctx [IDENT]
						if typeSpec = p.typeSpec(); typeSpec == nil {
							p.back(ix)
							goto _5
						}
						// *ebnf.Token ";" ctx []
						if semicolonTok, ok = p.accept(SEMICOLON); !ok {
							p.back(ix)
							goto _5
						}
						next := &TypeSpecListNode{
							TypeSpec:  typeSpec,
							SEMICOLON: semicolonTok,
						}
						if last != nil {
							last.List = next
						}
						if list == nil {
							list = next
						}
						last = next
						goto _4
					}
				_5:
				}
				// *ebnf.Token ")" ctx []
				if rparenTok, ok = p.accept(RPAREN); !ok {
					p.back(ix)
					goto _2
				}
			}
			break
		_2:
			lparenTok = Token{}
			rparenTok = Token{}
			p.back(ix)
			return nil
		default:
			p.back(ix)
			return nil
		}
	}
	return &TypeDeclNode{
		TYPE:         typeTok,
		LPAREN:       lparenTok,
		TypeSpecList: list,
		RPAREN:       rparenTok,
	}
}

// TypeDefNode represents the production
//
//	TypeDef = identifier [ TypeParameters ] Type .
type TypeDefNode struct {
	IDENT          Token
	TypeParameters *TypeParametersNode
	TypeNode       Type

	pkg *Package
	visible
}

// Source implements Node.
func (n *TypeDefNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeDefNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IDENT.Position()
}

func (p *parser) typeDef() (r *TypeDefNode) {
	var (
		identTok       Token
		typeParameters *TypeParametersNode
		typeNode       Type
	)
	// ebnf.Sequence identifier [ TypeParameters ] Type ctx [IDENT]
	{
		ix := p.ix
		// *ebnf.Name identifier ctx [IDENT]
		identTok = p.expect(IDENT)
		// *ebnf.Option [ TypeParameters ] ctx []
		switch p.c() {
		case LBRACK:
			// *ebnf.Name TypeParameters ctx [LBRACK]
			if typeParameters = p.typeParameters(); typeParameters == nil {
				goto _0
			}
		}
		goto _1
	_0:
		typeParameters = nil
	_1:
		// *ebnf.Name Type ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			if typeNode = p.type1(); typeNode == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	r = &TypeDefNode{
		IDENT:          identTok,
		TypeParameters: typeParameters,
		TypeNode:       typeNode,
	}
	p.declare(p.sc, identTok, r, int32(p.ix), false)
	return r
}

// TypeElemListNode represents the production
//
//	TypeElem = TypeTerm { "|" TypeTerm } .
type TypeElemListNode struct {
	OR       Token
	TypeTerm *TypeTermNode
	List     *TypeElemListNode
}

// Source implements Node.
func (n *TypeElemListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeElemListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.OR.IsValid() {
		return n.OR.Position()
	}

	return n.TypeTerm.Position()
}

func (p *parser) typeElem() *TypeElemListNode {
	var (
		typeTerm   *TypeTermNode
		list, last *TypeElemListNode
	)
	// ebnf.Sequence TypeTerm { "|" TypeTerm } ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
	{
		ix := p.ix
		// *ebnf.Name TypeTerm ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
		if typeTerm = p.typeTerm(); typeTerm == nil {
			p.back(ix)
			return nil
		}
		list = &TypeElemListNode{
			TypeTerm: typeTerm,
		}
		last = list
		// *ebnf.Repetition { "|" TypeTerm } ctx []
	_0:
		{
			var orTok Token
			var typeTerm *TypeTermNode
			switch p.c() {
			case OR:
				// ebnf.Sequence "|" TypeTerm ctx [OR]
				switch p.peek(1) {
				case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE:
				default:
					goto _1
				}
				ix := p.ix
				// *ebnf.Token "|" ctx [OR]
				orTok = p.expect(OR)
				// *ebnf.Name TypeTerm ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
				if typeTerm = p.typeTerm(); typeTerm == nil {
					p.back(ix)
					goto _1
				}
				next := &TypeElemListNode{
					OR:       orTok,
					TypeTerm: typeTerm,
				}
				last.List = next
				last = next
				goto _0
			}
		_1:
		}
	}
	return list
}

// TypeListNode represents the production
//
//	TypeList = Type { "," Type } .
type TypeListNode struct {
	COMMA    Token
	TypeNode Type
	List     *TypeListNode
}

// Source implements Node.
func (n *TypeListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.COMMA.IsValid() {
		return n.COMMA.Position()
	}

	return n.TypeNode.Position()
}

func (p *parser) typeList() *TypeListNode {
	var (
		typeNode   Type
		list, last *TypeListNode
	)
	// ebnf.Sequence Type { "," Type } ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
	{
		ix := p.ix
		// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		if typeNode = p.type1(); typeNode == nil {
			p.back(ix)
			return nil
		}
		list = &TypeListNode{
			TypeNode: typeNode,
		}
		last = list
		// *ebnf.Repetition { "," Type } ctx []
	_0:
		{
			var commaTok Token
			var typeNode Type
			switch p.c() {
			case COMMA:
				// ebnf.Sequence "," Type ctx [COMMA]
				switch p.peek(1) {
				case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
				default:
					goto _1
				}
				ix := p.ix
				// *ebnf.Token "," ctx [COMMA]
				commaTok = p.expect(COMMA)
				// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
				if typeNode = p.type1(); typeNode == nil {
					p.back(ix)
					goto _1
				}
				next := &TypeListNode{
					COMMA:    commaTok,
					TypeNode: typeNode,
				}
				last.List = next
				last = next
				goto _0
			}
		_1:
		}
	}
	return list
}

func (p *parser) typeLit() Type {
	var (
		arrayType     *ArrayTypeNode
		structType    *StructTypeNode
		pointerType   *PointerTypeNode
		functionType  *FunctionTypeNode
		interfaceType *InterfaceTypeNode
		mapType       *MapTypeNode
		channelType   *ChannelTypeNode
	)
	// ebnf.Alternative ArrayType | StructType | PointerType | FunctionType | InterfaceType | SliceType | MapType | ChannelType ctx [ARROW, CHAN, FUNC, INTERFACE, LBRACK, MAP, MUL, STRUCT]
	switch p.c() {
	case LBRACK: // 0 5
		if p.peek(1) == RBRACK {
			return p.sliceType()
		}

		// *ebnf.Name ArrayType ctx [LBRACK]
		if arrayType = p.arrayType(); arrayType != nil {
			return arrayType
		}
	case STRUCT: // 1
		// *ebnf.Name StructType ctx [STRUCT]
		if structType = p.structType(); structType != nil {
			return structType
		}
	case MUL: // 2
		// *ebnf.Name PointerType ctx [MUL]
		if pointerType = p.pointerType(); pointerType != nil {
			return pointerType
		}
	case FUNC: // 3
		// *ebnf.Name FunctionType ctx [FUNC]
		if functionType = p.functionType(); functionType != nil {
			return functionType
		}
	case INTERFACE: // 4
		// *ebnf.Name InterfaceType ctx [INTERFACE]
		if interfaceType = p.interfaceType(); interfaceType != nil {
			return interfaceType
		}
	case MAP: // 6
		// *ebnf.Name MapType ctx [MAP]
		if mapType = p.mapType(); mapType != nil {
			return mapType
		}
	case ARROW, CHAN: // 7
		// *ebnf.Name ChannelType ctx [ARROW, CHAN]
		if channelType = p.channelType(); channelType != nil {
			return channelType
		}
	}
	return nil
}

// TypeNameNode represents the production
//
//	TypeName = QualifiedIdent | identifier .
type TypeNameNode struct {
	Name Node
	lexicalScoper
}

// Source implements Node.
func (n *TypeNameNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeNameNode) Position() (r token.Position) {
	if n == nil || n.Name == nil {
		return r
	}

	return n.Name.Position()
}

func (p *parser) typeName() *TypeNameNode {
	var (
		qualifiedIdent *QualifiedIdentNode
	)
	// ebnf.Alternative QualifiedIdent | identifier ctx [IDENT]
	switch p.c() {
	case IDENT: // 0 1
		// *ebnf.Name QualifiedIdent ctx [IDENT]
		if qualifiedIdent = p.qualifiedIdent(); qualifiedIdent != nil {
			return &TypeNameNode{
				Name:          qualifiedIdent,
				lexicalScoper: newLexicalScoper(p.sc),
			}
		}

		// *ebnf.Name identifier ctx [IDENT]
		return &TypeNameNode{
			Name:          p.expect(IDENT),
			lexicalScoper: newLexicalScoper(p.sc),
		}
	default:
		return nil
	}
}

// TypeParamDeclNode represents the production
//
//	TypeParamDecl = IdentifierList TypeConstraint .
type TypeParamDeclNode struct {
	IdentifierList *IdentifierListNode
	TypeConstraint *TypeConstraintNode
}

// Source implements Node.
func (n *TypeParamDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeParamDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IdentifierList.Position()
}

func (p *parser) typeParamDecl() *TypeParamDeclNode {
	var (
		identifierList *IdentifierListNode
		typeConstraint *TypeConstraintNode
	)
	// ebnf.Sequence IdentifierList TypeConstraint ctx [IDENT]
	{
		ix := p.ix
		// *ebnf.Name IdentifierList ctx [IDENT]
		if identifierList = p.identifierList(); identifierList == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Name TypeConstraint ctx []
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE:
			if typeConstraint = p.typeConstraint(); typeConstraint == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
	}
	return &TypeParamDeclNode{
		IdentifierList: identifierList,
		TypeConstraint: typeConstraint,
	}
}

// TypeParamListNode represents the production
//
//	TypeParamList = TypeParamDecl { "," TypeParamDecl } .
type TypeParamListNode struct {
	COMMA         Token
	TypeParamDecl *TypeParamDeclNode
	List          *TypeParamListNode
}

// Source implements Node.
func (n *TypeParamListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeParamListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TypeParamDecl.Position()
}

func (p *parser) typeParamList() *TypeParamListNode {
	var (
		typeParamDecl *TypeParamDeclNode
		list, last    *TypeParamListNode
	)
	// ebnf.Sequence TypeParamDecl { "," TypeParamDecl } ctx [IDENT]
	{
		ix := p.ix
		// *ebnf.Name TypeParamDecl ctx [IDENT]
		if typeParamDecl = p.typeParamDecl(); typeParamDecl == nil {
			p.back(ix)
			return nil
		}
		list = &TypeParamListNode{
			TypeParamDecl: typeParamDecl,
		}
		last = list
		// *ebnf.Repetition { "," TypeParamDecl } ctx []
	_0:
		{
			var commaTok Token
			var typeParamDecl *TypeParamDeclNode
			switch p.c() {
			case COMMA:
				// ebnf.Sequence "," TypeParamDecl ctx [COMMA]
				switch p.peek(1) {
				case IDENT:
				default:
					goto _1
				}
				ix := p.ix
				// *ebnf.Token "," ctx [COMMA]
				commaTok = p.expect(COMMA)
				// *ebnf.Name TypeParamDecl ctx [IDENT]
				if typeParamDecl = p.typeParamDecl(); typeParamDecl == nil {
					p.back(ix)
					goto _1
				}
				next := &TypeParamListNode{
					COMMA:         commaTok,
					TypeParamDecl: typeParamDecl,
				}
				last.List = next
				last = next
				goto _0
			}
		_1:
		}
	}
	return list
}

// TypeParametersNode represents the production
//
//	TypeParameters = "[" TypeParamList [ "," ] "]" .
type TypeParametersNode struct {
	LBRACK        Token
	TypeParamList *TypeParamListNode
	COMMA         Token
	RBRACK        Token
}

// Source implements Node.
func (n *TypeParametersNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeParametersNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.LBRACK.Position()
}

func (p *parser) typeParameters() *TypeParametersNode {
	var (
		ok            bool
		lbrackTok     Token
		typeParamList *TypeParamListNode
		commaTok      Token
		rbrackTok     Token
	)
	// ebnf.Sequence "[" TypeParamList [ "," ] "]" ctx [LBRACK]
	{
		switch p.peek(1) {
		case IDENT:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "[" ctx [LBRACK]
		lbrackTok = p.expect(LBRACK)
		// *ebnf.Name TypeParamList ctx [IDENT]
		if typeParamList = p.typeParamList(); typeParamList == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Option [ "," ] ctx []
		switch p.c() {
		case COMMA:
			// *ebnf.Token "," ctx [COMMA]
			commaTok = p.expect(COMMA)
		}
		// *ebnf.Token "]" ctx []
		if rbrackTok, ok = p.accept(RBRACK); !ok {
			p.back(ix)
			return nil
		}
	}
	return &TypeParametersNode{
		LBRACK:        lbrackTok,
		TypeParamList: typeParamList,
		COMMA:         commaTok,
		RBRACK:        rbrackTok,
	}
}

func (p *parser) typeSpec() Node {
	var (
		aliasDecl *AliasDeclNode
		typeDef   *TypeDefNode
	)
	// ebnf.Alternative AliasDecl | TypeDef ctx [IDENT]
	switch p.c() {
	case IDENT: // 0 1
		// *ebnf.Name AliasDecl ctx [IDENT]
		if aliasDecl = p.aliasDecl(); aliasDecl == nil {
			goto _0
		}
		return aliasDecl
	_0:
		aliasDecl = nil
		// *ebnf.Name TypeDef ctx [IDENT]
		if typeDef = p.typeDef(); typeDef == nil {
			return nil
		}
		return typeDef
	default:
		return nil
	}
}

// TypeSwitchCaseNode represents the production
//
//	TypeSwitchCase = "case" TypeList | "default" .
type TypeSwitchCaseNode struct {
	CASE     Token
	TypeList *TypeListNode
	DEFAULT  Token
}

// Source implements Node.
func (n *TypeSwitchCaseNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeSwitchCaseNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) typeSwitchCase() *TypeSwitchCaseNode {
	var (
		caseTok    Token
		typeList   *TypeListNode
		defaultTok Token
	)
	// ebnf.Alternative "case" TypeList | "default" ctx [CASE, DEFAULT]
	switch p.c() {
	case CASE: // 0
		// ebnf.Sequence "case" TypeList ctx [CASE]
		{
			switch p.peek(1) {
			case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
			default:
				goto _0
			}
			ix := p.ix
			// *ebnf.Token "case" ctx [CASE]
			caseTok = p.expect(CASE)
			// *ebnf.Name TypeList ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
			if typeList = p.typeList(); typeList == nil {
				p.back(ix)
				goto _0
			}
		}
		break
	_0:
		caseTok = Token{}
		typeList = nil
		return nil
	case DEFAULT: // 1
		// *ebnf.Token "default" ctx [DEFAULT]
		defaultTok = p.expect(DEFAULT)
	default:
		return nil
	}
	return &TypeSwitchCaseNode{
		CASE:     caseTok,
		TypeList: typeList,
		DEFAULT:  defaultTok,
	}
}

// TypeSwitchGuardNode represents the production
//
//	TypeSwitchGuard = [ identifier ":=" ] PrimaryExpr "." "(" "type" ")" .
type TypeSwitchGuardNode struct {
	IDENT       Token
	DEFINE      Token
	PrimaryExpr Expression
	PERIOD      Token
	LPAREN      Token
	TYPE        Token
	RPAREN      Token
}

// Source implements Node.
func (n *TypeSwitchGuardNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeSwitchGuardNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	panic("TODO")
}

func (p *parser) typeSwitchGuard() *TypeSwitchGuardNode {
	var (
		ok          bool
		identTok    Token
		defineTok   Token
		primaryExpr Expression
		periodTok   Token
		lparenTok   Token
		typeTok     Token
		rparenTok   Token
	)
	// ebnf.Sequence [ identifier ":=" ] PrimaryExpr "." "(" "type" ")" ctx [ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRING, STRUCT]
	{
		ix := p.ix
		// *ebnf.Option [ identifier ":=" ] ctx [ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRING, STRUCT]
		switch p.c() {
		case IDENT:
			// ebnf.Sequence identifier ":=" ctx [IDENT]
			{
				if p.peek(1) != DEFINE {
					goto _0
				}
				// *ebnf.Name identifier ctx [IDENT]
				identTok = p.expect(IDENT)
				// *ebnf.Token ":=" ctx [DEFINE]
				defineTok = p.expect(DEFINE)
			}
		}
		goto _1
	_0:
		defineTok = Token{}
		identTok = Token{}
	_1:
		// *ebnf.Name PrimaryExpr ctx []
		switch p.c() {
		case ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRING, STRUCT:
			if primaryExpr = p.primaryExpr(false); primaryExpr == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Token "." ctx []
		if periodTok, ok = p.accept(PERIOD); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Token "(" ctx []
		if lparenTok, ok = p.accept(LPAREN); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Token "type" ctx []
		if typeTok, ok = p.accept(TYPE); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Token ")" ctx []
		if rparenTok, ok = p.accept(RPAREN); !ok {
			p.back(ix)
			return nil
		}
	}
	return &TypeSwitchGuardNode{
		IDENT:       identTok,
		DEFINE:      defineTok,
		PrimaryExpr: primaryExpr,
		PERIOD:      periodTok,
		LPAREN:      lparenTok,
		TYPE:        typeTok,
		RPAREN:      rparenTok,
	}
}

// TypeCaseClauseListNode represents the production
//
//	TypeCaseClauseListNode = { TypeCaseClause } .
type TypeCaseClauseListNode struct {
	TypeCaseClause *TypeCaseClauseNode
	List           *TypeCaseClauseListNode
}

// Source implements Node.
func (n *TypeCaseClauseListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeCaseClauseListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TypeCaseClause.Position()
}

// TypeSwitchStmtNode represents the production
//
//	TypeSwitchStmt = "switch" [ SimpleStmt ";" ] TypeSwitchGuard "{" { TypeCaseClause } "}" .
type TypeSwitchStmtNode struct {
	SWITCH             Token
	SimpleStmt         Node
	SEMICOLON          Token
	TypeSwitchGuard    *TypeSwitchGuardNode
	LBRACE             Token
	TypeCaseClauseList *TypeCaseClauseListNode
	RBRACE             Token
}

// Source implements Node.
func (n *TypeSwitchStmtNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeSwitchStmtNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.SWITCH.Position()
}

func (p *parser) typeSwitchStmt() *TypeSwitchStmtNode {
	var (
		ok              bool
		switchTok       Token
		simpleStmt      Node
		semicolonTok    Token
		typeSwitchGuard *TypeSwitchGuardNode
		lbraceTok       Token
		list, last      *TypeCaseClauseListNode
		rbraceTok       Token
	)
	// ebnf.Sequence "switch" [ SimpleStmt ";" ] TypeSwitchGuard "{" { TypeCaseClause } "}" ctx [SWITCH]
	{
		ix := p.ix
		// *ebnf.Token "switch" ctx [SWITCH]
		switchTok = p.expect(SWITCH)
		// *ebnf.Option [ SimpleStmt ";" ] ctx []
		switch p.c() {
		case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, SEMICOLON, STRING, STRUCT, SUB, XOR:
			// ebnf.Sequence SimpleStmt ";" ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, SEMICOLON, STRING, STRUCT, SUB, XOR]
			{
				ix := p.ix
				// *ebnf.Name SimpleStmt ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, SEMICOLON, STRING, STRUCT, SUB, XOR]
				switch p.c() {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
					if simpleStmt = p.simpleStmt(false); simpleStmt == nil {
						p.back(ix)
						goto _0
					}
				default:
					p.back(ix)
					goto _0
				}
				// *ebnf.Token ";" ctx []
				if semicolonTok, ok = p.accept(SEMICOLON); !ok {
					p.back(ix)
					goto _0
				}
			}
		}
		goto _1
	_0:
		semicolonTok = Token{}
		simpleStmt = nil
	_1:
		// *ebnf.Name TypeSwitchGuard ctx []
		switch p.c() {
		case ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRING, STRUCT:
			if typeSwitchGuard = p.typeSwitchGuard(); typeSwitchGuard == nil {
				p.back(ix)
				return nil
			}
		default:
			p.back(ix)
			return nil
		}
		// *ebnf.Token "{" ctx []
		if lbraceTok, ok = p.accept(LBRACE); !ok {
			p.back(ix)
			return nil
		}
		// *ebnf.Repetition { TypeCaseClause } ctx []
	_2:
		{
			var typeCaseClause *TypeCaseClauseNode
			switch p.c() {
			case CASE, DEFAULT:
				// *ebnf.Name TypeCaseClause ctx [CASE, DEFAULT]
				if typeCaseClause = p.typeCaseClause(); typeCaseClause == nil {
					goto _3
				}
				next := &TypeCaseClauseListNode{
					TypeCaseClause: typeCaseClause,
				}
				if last != nil {
					last.List = next
				}
				if list == nil {
					list = next
				}
				last = next
				goto _2
			}
		_3:
		}
		// *ebnf.Token "}" ctx []
		if rbraceTok, ok = p.accept(RBRACE); !ok {
			p.back(ix)
			return nil
		}
	}
	return &TypeSwitchStmtNode{
		SWITCH:             switchTok,
		SimpleStmt:         simpleStmt,
		SEMICOLON:          semicolonTok,
		TypeSwitchGuard:    typeSwitchGuard,
		LBRACE:             lbraceTok,
		TypeCaseClauseList: list,
		RBRACE:             rbraceTok,
	}
}

// TypeTermNode represents the production
//
//	TypeTerm = Type | UnderlyingType .
type TypeTermNode struct {
	TypeNode       Type
	UnderlyingType *UnderlyingTypeNode
}

// Source implements Node.
func (n *TypeTermNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *TypeTermNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	if n.TypeNode != nil {
		return n.TypeNode.Position()
	}

	return n.UnderlyingType.Position()
}

func (p *parser) typeTerm() *TypeTermNode {
	var (
		typeNode       Type
		underlyingType *UnderlyingTypeNode
	)
	// ebnf.Alternative Type | UnderlyingType ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT, TILDE]
	switch p.c() {
	case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT: // 0
		// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		if typeNode = p.type1(); typeNode == nil {
			goto _0
		}
		break
	_0:
		typeNode = nil
		return nil
	case TILDE: // 1
		// *ebnf.Name UnderlyingType ctx [TILDE]
		if underlyingType = p.underlyingType(); underlyingType == nil {
			goto _2
		}
		break
	_2:
		underlyingType = nil
		return nil
	default:
		return nil
	}
	return &TypeTermNode{
		TypeNode:       typeNode,
		UnderlyingType: underlyingType,
	}
}

// UnaryExprNode represents the production
//
//	UnaryExpr = PrimaryExpr | ( "+" | "-" | "!" | "^" | "*" | "&" | "<-" ) UnaryExpr .
type UnaryExprNode struct {
	Op        Token
	UnaryExpr Expression

	typeCache
	valueCache
}

// Source implements Node.
func (n *UnaryExprNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *UnaryExprNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.Op.Position()
}

func (p *parser) unaryExpr(preBlock bool) Expression {
	var (
		primaryExpr Expression
		op          Token
		unaryExpr   Expression
	)
	// ebnf.Alternative PrimaryExpr | ( "+" | "-" | "!" | "^" | "*" | "&" | "<-" ) UnaryExpr ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
	switch p.c() {
	case CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, STRING, STRUCT: // 0
		// *ebnf.Name PrimaryExpr ctx [CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, STRING, STRUCT]
		if primaryExpr = p.primaryExpr(preBlock); primaryExpr == nil {
			return nil
		}
		return primaryExpr
	case ARROW, MUL: // 0 1
		// *ebnf.Name PrimaryExpr ctx [ARROW, MUL]
		if primaryExpr = p.primaryExpr(preBlock); primaryExpr == nil {
			goto _2
		}
		return primaryExpr
	_2:
		primaryExpr = nil
		// ebnf.Sequence ( "+" | "-" | "!" | "^" | "*" | "&" | "<-" ) UnaryExpr ctx [ARROW, MUL]
		{
			ix := p.ix
			// *ebnf.Group ( "+" | "-" | "!" | "^" | "*" | "&" | "<-" ) ctx [ARROW, MUL]
			// ebnf.Alternative "+" | "-" | "!" | "^" | "*" | "&" | "<-" ctx [ARROW, MUL]
			op = p.consume()
			// *ebnf.Name UnaryExpr ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				if unaryExpr = p.unaryExpr(preBlock); unaryExpr == nil {
					p.back(ix)
					goto _3
				}
			default:
				p.back(ix)
				goto _3
			}
		}
		break
	_3:
		unaryExpr = nil
		return nil
	case ADD, AND, NOT, SUB, XOR: // 1
		// ebnf.Sequence ( "+" | "-" | "!" | "^" | "*" | "&" | "<-" ) UnaryExpr ctx [ADD, AND, NOT, SUB, XOR]
		{
			ix := p.ix
			// *ebnf.Group ( "+" | "-" | "!" | "^" | "*" | "&" | "<-" ) ctx [ADD, AND, NOT, SUB, XOR]
			// ebnf.Alternative "+" | "-" | "!" | "^" | "*" | "&" | "<-" ctx [ADD, AND, NOT, SUB, XOR]
			op = p.consume()
			// *ebnf.Name UnaryExpr ctx []
			switch p.c() {
			case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				if unaryExpr = p.unaryExpr(preBlock); unaryExpr == nil {
					p.back(ix)
					goto _8
				}
			default:
				p.back(ix)
				goto _8
			}
		}
		break
	_8:
		op = Token{}
		unaryExpr = nil
		return nil
	default:
		return nil
	}
	return &UnaryExprNode{
		Op:        op,
		UnaryExpr: unaryExpr,
	}
}

// UnderlyingTypeNode represents the production
//
//	UnderlyingType = "~" Type .
type UnderlyingTypeNode struct {
	TILDE    Token
	TypeNode Type
}

// Source implements Node.
func (n *UnderlyingTypeNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *UnderlyingTypeNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.TILDE.Position()
}

func (p *parser) underlyingType() *UnderlyingTypeNode {
	var (
		tildeTok Token
		typeNode Type
	)
	// ebnf.Sequence "~" Type ctx [TILDE]
	{
		switch p.peek(1) {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "~" ctx [TILDE]
		tildeTok = p.expect(TILDE)
		// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		if typeNode = p.type1(); typeNode == nil {
			p.back(ix)
			return nil
		}
	}
	return &UnderlyingTypeNode{
		TILDE:    tildeTok,
		TypeNode: typeNode,
	}
}

// VarSpecListNode represents the production
//
//	VarSpecListNode = { VarSpec ";" } .
type VarSpecListNode struct {
	VarSpec   Node
	SEMICOLON Token
	List      *VarSpecListNode
}

// Source implements Node.
func (n *VarSpecListNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *VarSpecListNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.VarSpec.Position()
}

// VarDeclNode represents the production
//
//	VarDecl = "var" ( VarSpec | "(" { VarSpec ";" } ")" ) .
type VarDeclNode struct {
	VAR     Token
	LPAREN  Token
	VarSpec Node
	RPAREN  Token
}

// Source implements Node.
func (n *VarDeclNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *VarDeclNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.VAR.Position()
}

func (p *parser) varDecl() *VarDeclNode {
	var (
		ok         bool
		varTok     Token
		varSpec    Node
		lparenTok  Token
		list, last *VarSpecListNode
		rparenTok  Token
	)
	// ebnf.Sequence "var" ( VarSpec | "(" { VarSpec ";" } ")" ) ctx [VAR]
	{
		switch p.peek(1) {
		case IDENT, LPAREN:
		default:
			return nil
		}
		ix := p.ix
		// *ebnf.Token "var" ctx [VAR]
		varTok = p.expect(VAR)
		// *ebnf.Group ( VarSpec | "(" { VarSpec ";" } ")" ) ctx [IDENT, LPAREN]
		// ebnf.Alternative VarSpec | "(" { VarSpec ";" } ")" ctx [IDENT, LPAREN]
		switch p.c() {
		case IDENT: // 0
			// *ebnf.Name VarSpec ctx [IDENT]
			if varSpec = p.varSpec(); varSpec == nil {
				goto _0
			}
			list = &VarSpecListNode{
				VarSpec: varSpec,
			}
			break
		_0:
			varSpec = nil
			p.back(ix)
			return nil
		case LPAREN: // 1
			// ebnf.Sequence "(" { VarSpec ";" } ")" ctx [LPAREN]
			{
				ix := p.ix
				// *ebnf.Token "(" ctx [LPAREN]
				lparenTok = p.expect(LPAREN)
				// *ebnf.Repetition { VarSpec ";" } ctx []
			_4:
				{
					var varSpec Node
					var semicolonTok Token
					switch p.c() {
					case IDENT:
						// ebnf.Sequence VarSpec ";" ctx [IDENT]
						ix := p.ix
						// *ebnf.Name VarSpec ctx [IDENT]
						if varSpec = p.varSpec(); varSpec == nil {
							p.back(ix)
							goto _5
						}
						// *ebnf.Token ";" ctx []
						if semicolonTok, ok = p.accept(SEMICOLON); !ok {
							p.back(ix)
							goto _5
						}
						next := &VarSpecListNode{
							VarSpec:   varSpec,
							SEMICOLON: semicolonTok,
						}
						if last != nil {
							last.List = next
						}
						if list == nil {
							list = next
						}
						last = next
						goto _4
					}
				_5:
				}
				// *ebnf.Token ")" ctx []
				if rparenTok, ok = p.accept(RPAREN); !ok {
					p.back(ix)
					goto _2
				}
			}
			break
		_2:
			lparenTok = Token{}
			rparenTok = Token{}
			p.back(ix)
			return nil
		default:
			p.back(ix)
			return nil
		}
	}
	if list != nil && list.List == nil && !list.SEMICOLON.IsValid() {
		return &VarDeclNode{
			VAR:     varTok,
			LPAREN:  lparenTok,
			VarSpec: list.VarSpec,
			RPAREN:  rparenTok,
		}
	}

	return &VarDeclNode{
		VAR:     varTok,
		LPAREN:  lparenTok,
		VarSpec: list,
		RPAREN:  rparenTok,
	}
}

// VarSpecNode represents the production
//
//	VarSpec = identifier ( Type [ "=" ExpressionList ] | "=" ExpressionList ) .
type VarSpecNode struct {
	IDENT          Token
	TypeNode       Type
	ASSIGN         Token
	ExpressionList *ExpressionListNode
	lexicalScoper

	visible
}

// Source implements Node.
func (n *VarSpecNode) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *VarSpecNode) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IDENT.Position()
}

// VarSpec2Node represents the production
//
//	VarSpec = IdentifierList ( Type [ "=" ExpressionList ] | "=" ExpressionList ) .
type VarSpec2Node struct {
	IdentifierList *IdentifierListNode
	TypeNode       Type
	ASSIGN         Token
	ExpressionList *ExpressionListNode
	lexicalScoper

	visible
}

// Source implements Node.
func (n *VarSpec2Node) Source(full bool) string { return nodeSource(n, full) }

// Position implements Node.
func (n *VarSpec2Node) Position() (r token.Position) {
	if n == nil {
		return r
	}

	return n.IdentifierList.Position()
}

func (p *parser) varSpec() Node {
	var (
		identifierList *IdentifierListNode
		typeNode       Type
		assignTok      Token
		expressionList *ExpressionListNode
	)
	// ebnf.Sequence IdentifierList ( Type [ "=" ExpressionList ] | "=" ExpressionList ) ctx [IDENT]
	{
		ix := p.ix
		// *ebnf.Name IdentifierList ctx [IDENT]
		if identifierList = p.identifierList(); identifierList == nil {
			p.back(ix)
			return nil
		}
		// *ebnf.Group ( Type [ "=" ExpressionList ] | "=" ExpressionList ) ctx []
		// ebnf.Alternative Type [ "=" ExpressionList ] | "=" ExpressionList ctx [ARROW, ASSIGN, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
		switch p.c() {
		case ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT: // 0
			// ebnf.Sequence Type [ "=" ExpressionList ] ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
			{
				ix := p.ix
				// *ebnf.Name Type ctx [ARROW, CHAN, FUNC, IDENT, INTERFACE, LBRACK, LPAREN, MAP, MUL, STRUCT]
				if typeNode = p.type1(); typeNode == nil {
					p.back(ix)
					goto _0
				}
				// *ebnf.Option [ "=" ExpressionList ] ctx []
				switch p.c() {
				case ASSIGN:
					// ebnf.Sequence "=" ExpressionList ctx [ASSIGN]
					{
						switch p.peek(1) {
						case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
						default:
							goto _2
						}
						ix := p.ix
						// *ebnf.Token "=" ctx [ASSIGN]
						assignTok = p.expect(ASSIGN)
						// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
						if expressionList = p.expressionList(false); expressionList == nil {
							p.back(ix)
							goto _2
						}
					}
				}
				goto _3
			_2:
				assignTok = Token{}
				expressionList = nil
			_3:
			}
			break
		_0:
			assignTok = Token{}
			expressionList = nil
			typeNode = nil
			p.back(ix)
			return nil
		case ASSIGN: // 1
			// ebnf.Sequence "=" ExpressionList ctx [ASSIGN]
			{
				switch p.peek(1) {
				case ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR:
				default:
					goto _4
				}
				ix := p.ix
				// *ebnf.Token "=" ctx [ASSIGN]
				assignTok = p.expect(ASSIGN)
				// *ebnf.Name ExpressionList ctx [ADD, AND, ARROW, CHAN, CHAR, FLOAT, FUNC, IDENT, IMAG, INT, INTERFACE, LBRACK, LPAREN, MAP, MUL, NOT, STRING, STRUCT, SUB, XOR]
				if expressionList = p.expressionList(false); expressionList == nil {
					p.back(ix)
					goto _4
				}
			}
			break
		_4:
			assignTok = Token{}
			expressionList = nil
			p.back(ix)
			return nil
		default:
			p.back(ix)
			return nil
		}
	}
	sc := p.sc
	if identifierList.Len() == 1 {
		r := &VarSpecNode{
			lexicalScoper:  newLexicalScoper(sc),
			IDENT:          identifierList.IDENT,
			TypeNode:       typeNode,
			ASSIGN:         assignTok,
			ExpressionList: expressionList,
		}
		visible := int32(p.ix)
		p.declare(sc, r.IDENT, r, visible, false)
		return r
	}

	r := &VarSpec2Node{
		lexicalScoper:  newLexicalScoper(sc),
		IdentifierList: identifierList,
		TypeNode:       typeNode,
		ASSIGN:         assignTok,
		ExpressionList: expressionList,
	}
	visible := int32(p.ix)
	for l := r.IdentifierList; l != nil; l = l.List {
		p.declare(sc, l.IDENT, r, visible, false)
	}
	return r
}

const (
	balanceZero = iota
	balanceTuple
	balanceEqual
	balanceExtraRhs
	balanceExtraLhs
)

func checkBalance(lhs, rhs int) int {
	switch {
	case lhs == rhs:
		return balanceEqual
	case lhs > 1 && rhs == 1:
		return balanceTuple
	case lhs > rhs:
		return balanceExtraLhs
	case lhs < rhs:
		return balanceExtraRhs
	default:
		panic(todo("", lhs, rhs))
	}
}
