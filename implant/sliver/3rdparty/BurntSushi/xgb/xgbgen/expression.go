package main

import (
	"fmt"
	"log"
)

// Expression represents all the different forms of expressions possible in
// side an XML protocol description file. It's also received a few custom
// addendums to make applying special functions (like padding) easier.
type Expression interface {
	// Concrete determines whether this particular expression can be computed
	// to some constant value inside xgbgen. (The alternative is that the
	// expression can only be computed with values at run time of the
	// generated code.)
	Concrete() bool

	// Eval evaluates a concrete expression. It is an error to call Eval
	// on any expression that is not concrete (or contains any sub-expression
	// that is not concrete).
	Eval() int

	// Reduce attempts to evaluate any concrete sub-expressions.
	// i.e., (1 + 2 * (5 + 1 + someSizeOfStruct) reduces to
	// (3 * (6 + someSizeOfStruct)).
	// 'prefix' is used preprended to any field reference name.
	Reduce(prefix string) string

	// String is an alias for Reduce("")
	String() string

	// Initialize makes sure all names in this expression and any subexpressions
	// have been translated to Go source names.
	Initialize(p *Protocol)
}

// Function is a custom expression not found in the XML. It's simply used
// to apply a function named in 'Name' to the Expr expression.
type Function struct {
	Name string
	Expr Expression
}

func (e *Function) Concrete() bool {
	return false
}

func (e *Function) Eval() int {
	log.Fatalf("Cannot evaluate a 'Function'. It is not concrete.")
	panic("unreachable")
}

func (e *Function) Reduce(prefix string) string {
	return fmt.Sprintf("%s(%s)", e.Name, e.Expr.Reduce(prefix))
}

func (e *Function) String() string {
	return e.Reduce("")
}

func (e *Function) Initialize(p *Protocol) {
	e.Expr.Initialize(p)
}

// BinaryOp is an expression that performs some operation (defined in the XML
// file) with Expr1 and Expr2 as operands.
type BinaryOp struct {
	Op    string
	Expr1 Expression
	Expr2 Expression
}

// newBinaryOp constructs a new binary expression when both expr1 and expr2
// are not nil. If one or both are nil, then the non-nil expression is
// returned unchanged or nil is returned.
func newBinaryOp(op string, expr1, expr2 Expression) Expression {
	switch {
	case expr1 != nil && expr2 != nil:
		return &BinaryOp{
			Op:    op,
			Expr1: expr1,
			Expr2: expr2,
		}
	case expr1 != nil && expr2 == nil:
		return expr1
	case expr1 == nil && expr2 != nil:
		return expr2
	case expr1 == nil && expr2 == nil:
		return nil
	}
	panic("unreachable")
}

func (e *BinaryOp) Concrete() bool {
	return e.Expr1.Concrete() && e.Expr2.Concrete()
}

func (e *BinaryOp) Eval() int {
	switch e.Op {
	case "+":
		return e.Expr1.Eval() + e.Expr2.Eval()
	case "-":
		return e.Expr1.Eval() - e.Expr2.Eval()
	case "*":
		return e.Expr1.Eval() * e.Expr2.Eval()
	case "/":
		return e.Expr1.Eval() / e.Expr2.Eval()
	case "&amp;":
		return e.Expr1.Eval() & e.Expr2.Eval()
	case "&lt;&lt;":
		return int(uint(e.Expr1.Eval()) << uint(e.Expr2.Eval()))
	}

	log.Fatalf("Invalid binary operator '%s' for expression.", e.Op)
	panic("unreachable")
}

func (e *BinaryOp) Reduce(prefix string) string {
	if e.Concrete() {
		return fmt.Sprintf("%d", e.Eval())
	}

	// An incredibly dirty hack to make sure any time we perform an operation
	// on a field, we're dealing with ints...
	expr1, expr2 := e.Expr1, e.Expr2
	switch expr1.(type) {
	case *FieldRef:
		expr1 = &Function{
			Name: "int",
			Expr: expr1,
		}
	}
	switch expr2.(type) {
	case *FieldRef:
		expr2 = &Function{
			Name: "int",
			Expr: expr2,
		}
	}
	return fmt.Sprintf("(%s %s %s)",
		expr1.Reduce(prefix), e.Op, expr2.Reduce(prefix))
}

func (e *BinaryOp) String() string {
	return e.Reduce("")
}

func (e *BinaryOp) Initialize(p *Protocol) {
	e.Expr1.Initialize(p)
	e.Expr2.Initialize(p)
}

// UnaryOp is the same as BinaryOp, except it's a unary operator with only
// one sub-expression.
type UnaryOp struct {
	Op   string
	Expr Expression
}

func (e *UnaryOp) Concrete() bool {
	return e.Expr.Concrete()
}

func (e *UnaryOp) Eval() int {
	switch e.Op {
	case "~":
		return ^e.Expr.Eval()
	}

	log.Fatalf("Invalid unary operator '%s' for expression.", e.Op)
	panic("unreachable")
}

func (e *UnaryOp) Reduce(prefix string) string {
	if e.Concrete() {
		return fmt.Sprintf("%d", e.Eval())
	}
	return fmt.Sprintf("(%s (%s))", e.Op, e.Expr.Reduce(prefix))
}

func (e *UnaryOp) String() string {
	return e.Reduce("")
}

func (e *UnaryOp) Initialize(p *Protocol) {
	e.Expr.Initialize(p)
}

// Padding represents the application of the 'pad' function to some
// sub-expression.
type Padding struct {
	Expr Expression
}

func (e *Padding) Concrete() bool {
	return e.Expr.Concrete()
}

func (e *Padding) Eval() int {
	return pad(e.Expr.Eval())
}

func (e *Padding) Reduce(prefix string) string {
	if e.Concrete() {
		return fmt.Sprintf("%d", e.Eval())
	}
	return fmt.Sprintf("xgb.Pad(%s)", e.Expr.Reduce(prefix))
}

func (e *Padding) String() string {
	return e.Reduce("")
}

func (e *Padding) Initialize(p *Protocol) {
	e.Expr.Initialize(p)
}

// PopCount represents the application of the 'PopCount' function to
// some sub-expression.
type PopCount struct {
	Expr Expression
}

func (e *PopCount) Concrete() bool {
	return e.Expr.Concrete()
}

func (e *PopCount) Eval() int {
	return int(popCount(uint(e.Expr.Eval())))
}

func (e *PopCount) Reduce(prefix string) string {
	if e.Concrete() {
		return fmt.Sprintf("%d", e.Eval())
	}
	return fmt.Sprintf("xgb.PopCount(%s)", e.Expr.Reduce(prefix))
}

func (e *PopCount) String() string {
	return e.Reduce("")
}

func (e *PopCount) Initialize(p *Protocol) {
	e.Expr.Initialize(p)
}

// Value represents some constant integer.
type Value struct {
	v int
}

func (e *Value) Concrete() bool {
	return true
}

func (e *Value) Eval() int {
	return e.v
}

func (e *Value) Reduce(prefix string) string {
	return fmt.Sprintf("%d", e.v)
}

func (e *Value) String() string {
	return e.Reduce("")
}

func (e *Value) Initialize(p *Protocol) {}

// Bit represents some bit whose value is computed by '1 << bit'.
type Bit struct {
	b int
}

func (e *Bit) Concrete() bool {
	return true
}

func (e *Bit) Eval() int {
	return int(1 << uint(e.b))
}

func (e *Bit) Reduce(prefix string) string {
	return fmt.Sprintf("%d", e.Eval())
}

func (e *Bit) String() string {
	return e.Reduce("")
}

func (e *Bit) Initialize(p *Protocol) {}

// FieldRef represents a reference to some variable in the generated code
// with name Name.
type FieldRef struct {
	Name string
}

func (e *FieldRef) Concrete() bool {
	return false
}

func (e *FieldRef) Eval() int {
	log.Fatalf("Cannot evaluate a 'FieldRef'. It is not concrete.")
	panic("unreachable")
}

func (e *FieldRef) Reduce(prefix string) string {
	val := e.Name
	if len(prefix) > 0 {
		val = fmt.Sprintf("%s%s", prefix, val)
	}
	return val
}

func (e *FieldRef) String() string {
	return e.Reduce("")
}

func (e *FieldRef) Initialize(p *Protocol) {
	e.Name = SrcName(p, e.Name)
}

// EnumRef represents a reference to some enumeration field.
// EnumKind is the "group" an EnumItem is the name of the specific enumeration
// value inside that group.
type EnumRef struct {
	EnumKind Type
	EnumItem string
}

func (e *EnumRef) Concrete() bool {
	return false
}

func (e *EnumRef) Eval() int {
	log.Fatalf("Cannot evaluate an 'EnumRef'. It is not concrete.")
	panic("unreachable")
}

func (e *EnumRef) Reduce(prefix string) string {
	return fmt.Sprintf("%s%s", e.EnumKind, e.EnumItem)
}

func (e *EnumRef) String() string {
	return e.Reduce("")
}

func (e *EnumRef) Initialize(p *Protocol) {
	e.EnumKind = e.EnumKind.(*Translation).RealType(p)
	e.EnumItem = SrcName(p, e.EnumItem)
}

// SumOf represents a summation of the variable in the generated code named by
// Name. It is not currently used. (It's XKB voodoo.)
type SumOf struct {
	Name string
}

func (e *SumOf) Concrete() bool {
	return false
}

func (e *SumOf) Eval() int {
	log.Fatalf("Cannot evaluate a 'SumOf'. It is not concrete.")
	panic("unreachable")
}

func (e *SumOf) Reduce(prefix string) string {
	if len(prefix) > 0 {
		return fmt.Sprintf("sum(%s%s)", prefix, e.Name)
	}
	return fmt.Sprintf("sum(%s)", e.Name)
}

func (e *SumOf) String() string {
	return e.Reduce("")
}

func (e *SumOf) Initialize(p *Protocol) {
	e.Name = SrcName(p, e.Name)
}
