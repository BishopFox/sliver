// Package ast defines the syntax tree shared by the parser and evaluator.
//
// The tree intentionally distinguishes Sleep syntax from the meaning supplied
// by an embedding host.  In particular, environment declarations and object
// expressions remain generic: Aggressor Script assigns meaning to names such
// as on, alias, popup, and set after parsing.
package ast

import "github.com/sliverarmory/opfor/internal/lexer"

// Span is the end-exclusive source range attached to every syntax node.
type Span = lexer.Span

// Node is implemented by every syntax tree node.
type Node interface {
	Span() Span
}

// Expr is implemented by expression nodes.
type Expr interface {
	Node
	exprNode()
}

// Stmt is implemented by statement nodes.
type Stmt interface {
	Node
	stmtNode()
}

// Base supplies the source range implementation embedded by concrete nodes.
type Base struct {
	Range Span
}

func (n Base) Span() Span { return n.Range }

// ExprBase marks a node as an expression.
type ExprBase struct{ Base }

func (ExprBase) exprNode() {}

// StmtBase marks a node as a statement.
type StmtBase struct{ Base }

func (StmtBase) stmtNode() {}

// Script is a parsed source unit.
type Script struct {
	Base
	Statements []Stmt
}

// BlockStmt is a braced statement sequence. Closures contain a BlockStmt, but
// a block is not itself an expression; this keeps statement blocks and closure
// literals unambiguous to consumers.
type BlockStmt struct {
	StmtBase
	Statements []Stmt
}

// ExprStmt evaluates an expression for its side effects.
type ExprStmt struct {
	StmtBase
	Expr Expr
}

// IdentifierExpr is a bare identifier. Sleep permits bare identifiers in
// named arguments, environment selectors, Java messages, and host extensions.
type IdentifierExpr struct {
	ExprBase
	Name string
}

// VariableKind identifies the sigil used by a Sleep variable.
type VariableKind uint8

const (
	ScalarVariable VariableKind = iota
	ArrayVariable
	HashVariable
)

// VariableExpr is a scalar, array, or hash variable reference.
type VariableExpr struct {
	ExprBase
	Kind VariableKind
	Name string // excludes the sigil
	Raw  string // includes the sigil
}

// NumberKind identifies Sleep's numeric literal representation.
type NumberKind uint8

const (
	IntegerNumber NumberKind = iota
	LongNumber
	DoubleNumber
)

// NumberExpr retains both the source spelling and lexer-normalized text.
type NumberExpr struct {
	ExprBase
	Kind NumberKind
	Raw  string
	Text string
}

// BoolExpr represents Sleep's bare true and false literal spellings.
type BoolExpr struct {
	ExprBase
	Value bool
	Raw   string
}

// NullExpr represents an inert null operand synthesized for parser recovery
// and serialized bytecode. Sleep source spells the null scalar $null, which
// remains a VariableExpr.
type NullExpr struct {
	ExprBase
	Raw string
}

// StringKind identifies Sleep's three quoted forms.
type StringKind uint8

const (
	SingleQuotedString StringKind = iota
	DoubleQuotedString
	BacktickString
)

// StringExpr retains both the quoted source spelling and its lexer-normalized
// contents. Interpolation remains a runtime operation because it depends on
// the active variable environment.
type StringExpr struct {
	ExprBase
	Kind      StringKind
	Raw       string
	Text      string
	TextRange Span // exact source range of Text, excluding quote delimiters
}

// FunctionRefExpr is an &name function reference.
type FunctionRefExpr struct {
	ExprBase
	Name string
	Raw  string
}

// ReferenceExpr is Sleep's pass-by-name form (for example, \$value).
type ReferenceExpr struct {
	ExprBase
	Target Expr
}

// ClassExpr is a ^qualified.Java.Class literal.
type ClassExpr struct {
	ExprBase
	Name string
	Raw  string
}

// ClosureExpr is an anonymous braced closure.
type ClosureExpr struct {
	ExprBase
	Body *BlockStmt
}

// GroupExpr records explicit parentheses even when they contain one value.
type GroupExpr struct {
	ExprBase
	Expr Expr
}

// TupleExpr is a parenthesized, comma-separated expression sequence. It is
// most commonly used as an assignment target.
type TupleExpr struct {
	ExprBase
	Elements []Expr
}

// ArrayLiteralExpr represents @(...).
type ArrayLiteralExpr struct {
	ExprBase
	Elements []Expr
}

// HashLiteralExpr represents %(...). Entries are normally PairExpr values,
// though preserving Expr allows the parser to diagnose malformed input while
// retaining a useful tree.
type HashLiteralExpr struct {
	ExprBase
	Entries []Expr
}

// PairExpr is Sleep's key/value and named-argument operator (=>).
type PairExpr struct {
	ExprBase
	Key    Expr
	Value  Expr
	RawKey string // predicate-shaped keys are parser data, not evaluated expressions
}

// UnaryExpr represents prefix predicates/operators and Sleep's adjacent bare
// scalar ++/-- rewrite. The parser rejects general postfix operands.
type UnaryExpr struct {
	ExprBase
	Op      string
	Operand Expr
	Postfix bool
}

// BinaryExpr represents ordinary operators and predicates. Negated predicates
// retain their full spelling (for example !iswm) in Op.
type BinaryExpr struct {
	ExprBase
	Left  Expr
	Op    string
	Right Expr
}

// AssignExpr represents simple, compound, and tuple assignment. Target is
// either a normal assignable expression or a TupleExpr.
type AssignExpr struct {
	ExprBase
	Target Expr
	Op     string
	Value  Expr
}

// SequenceExpr preserves the reference parser's shared evaluation frame for
// multiple ideas on the right-hand side of an assignment. This malformed form
// is observable when a closure literal is missing its terminating semicolon:
// every following idea is evaluated before Assign rejects the extra values.
type SequenceExpr struct {
	ExprBase
	Elements []Expr
}

// ParameterTermExpr preserves the values produced by a nested Sleep parameter
// term. The expressions execute left-to-right, but only the last value remains
// visible to the enclosing operator frame.
type ParameterTermExpr struct {
	ExprBase
	Ideas []Expr
}

// ParameterOperatorExpr is Sleep's legacy three-or-more-idea parameter form.
// The second idea is an operator spelling rather than an evaluated value; the
// remaining ideas form its right-hand parameter term.
type ParameterOperatorExpr struct {
	ExprBase
	Left  Expr
	Op    string
	Right []Expr
}

// AdjacentEmptyGroupExpr records a non-function value followed by (). Sleep
// treats the empty parenthesized term as a second idea which pushes no scalar,
// so the wrapped value survives in value contexts without becoming a call.
type AdjacentEmptyGroupExpr struct {
	ExprBase
	Value Expr
}

// IndexExpr indexes an array, hash, closure, function result, or host object.
type IndexExpr struct {
	ExprBase
	Target Expr
	Index  Expr
}

// CallExpr is a normal parenthesized call. Array and hash literal calls are
// represented by ArrayLiteralExpr and HashLiteralExpr instead.
type CallExpr struct {
	ExprBase
	Callee    Expr
	Args      []Expr
	ArgGroups []int // source-order lengths of comma-delimited parameter terms
}

// ObjectMessage is the optional Java/Sleep message in a bracket expression.
// Qualified names are retained as one string.
type ObjectMessage struct {
	Range Span
	Name  string
}

func (m ObjectMessage) Span() Span { return m.Range }

// ObjectExpr represents [target], [target: args], [target message], and
// [target message: args]. A nil Message denotes direct closure/function
// invocation or the one-term bracket form.
type ObjectExpr struct {
	ExprBase
	Target  Expr
	Message *ObjectMessage
	Args    []Expr
}

// ImportPathExpr preserves Sleep's unquoted import-from path form, such as
// data/extensions.jar. Quoted paths remain StringExpr values.
type ImportPathExpr struct {
	ExprBase
	Raw string
}

// ImportStmt imports a Java class/package, optionally from a JAR path.
type ImportStmt struct {
	StmtBase
	Target string
	From   Expr
}

// IfStmt represents an if/else-if/else chain. Else is either another IfStmt,
// a BlockStmt, or nil.
type IfStmt struct {
	StmtBase
	Condition Expr
	Then      *BlockStmt
	Else      Stmt
}

// WhileStmt represents both while (condition) and Sleep's assignment loop
// while $value (generator), whose binding may use any variable sigil. Binding
// is nil for the ordinary form.
type WhileStmt struct {
	StmtBase
	Binding   *VariableExpr
	Condition Expr
	Body      *BlockStmt
}

// ForStmt represents a C-style for loop. Init and Post permit comma-separated
// expression sequences, and Condition may be nil.
type ForStmt struct {
	StmtBase
	Init      []Expr
	Condition Expr
	Post      []Expr
	Body      *BlockStmt
}

// ForeachStmt represents both foreach $value (...) and
// foreach $key => $value (...). Key is nil in the one-variable form.
type ForeachStmt struct {
	StmtBase
	Key      Expr
	Value    Expr
	Iterable Expr
	Body     *BlockStmt
}

// TryStmt represents try { ... } catch $error { ... }.
type TryStmt struct {
	StmtBase
	Body     *BlockStmt
	CatchVar *VariableExpr
	Catch    *BlockStmt
}

// ReturnStmt returns an optional value from a function.
type ReturnStmt struct {
	StmtBase
	Value Expr
}

// BreakStmt and ContinueStmt carry Sleep's optional operand. Return evaluates
// that operand before requesting flow control, but the loop operation itself
// always carries the empty scalar.
type BreakStmt struct {
	StmtBase
	Value Expr
}

type ContinueStmt struct {
	StmtBase
	Value Expr
}

type HaltStmt struct{ StmtBase }
type DoneStmt struct{ StmtBase }

// YieldStmt yields an optional value from a coroutine.
type YieldStmt struct {
	StmtBase
	Value Expr
}

// ThrowStmt raises a scalar value.
type ThrowStmt struct {
	StmtBase
	Value Expr
}

// CallCCStmt invokes a closure with the current continuation.
type CallCCStmt struct {
	StmtBase
	Closure Expr
}

// AssertStmt optionally carries a colon-separated failure message.
type AssertStmt struct {
	StmtBase
	Condition Expr
	Message   Expr
}

// SelectorKind classifies the parameter bound to an environment declaration.
type SelectorKind uint8

const (
	IdentifierSelector SelectorKind = iota
	StringSelector
	WildcardSelector
	RawSelector
)

// Selector preserves an environment declaration parameter without assigning
// host-specific meaning to it.
type Selector struct {
	Range Span
	Kind  SelectorKind
	Value string
	Raw   string
	// Expr retains an evaluable quoted selector. Sleep evaluates quoted names
	// when the declaration executes, so interpolation can select the function
	// or host-binding name dynamically. Unquoted selectors remain literal and
	// leave Expr nil.
	Expr Expr
}

func (s Selector) Span() Span { return s.Range }

// EnvironmentForm identifies which Sleep environment bridge ABI a
// declaration uses. The three forms are syntactically and semantically
// distinct in the reference runtime: ordinary environments evaluate their
// name, filter environments retain an unevaluated parameter, and predicate
// environments retain a callable condition.
type EnvironmentForm uint8

const (
	OrdinaryEnvironment EnvironmentForm = iota
	FilterEnvironment
	PredicateEnvironment
)

// EnvironmentStmt is Sleep's generic "keyword selector { block }" extension
// form. Aggressor uses it for sub, on, alias, command, set, popup, item, bind,
// ssh_alias, and other host-defined declarations.
type EnvironmentStmt struct {
	StmtBase
	Keyword   string
	Form      EnvironmentForm
	Selectors []Selector
	Predicate Expr
	Body      *BlockStmt
}
