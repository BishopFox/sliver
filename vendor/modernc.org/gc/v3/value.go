// Copyright 2022 The Gc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gc // modernc.org/gc/v3

import (
	"go/constant"
	"math"
)

var (
	_ Expression = (*BasicLitNode)(nil)
	_ Expression = (*BinaryExpressionNode)(nil)
	_ Expression = (*CompositeLitNode)(nil)
	_ Expression = (*ConversionNode)(nil)
	_ Expression = (*FunctionLitNode)(nil)
	_ Expression = (*KeyedElementNode)(nil)
	_ Expression = (*LiteralValueNode)(nil)
	_ Expression = (*MethodExprNode)(nil)
	_ Expression = (*OperandNameNode)(nil)
	_ Expression = (*OperandNode)(nil)
	_ Expression = (*OperandQualifiedNameNode)(nil)
	_ Expression = (*ParenthesizedExpressionNode)(nil)
	_ Expression = (*PrimaryExprNode)(nil)
	_ Expression = (*UnaryExprNode)(nil)
	_ Expression = (*ValueExpression)(nil)

	falseVal = constant.MakeBool(false)
	trueVal  = constant.MakeBool(true)
	unknown  = constant.MakeUnknown()
)

func known(v constant.Value) bool { return v != nil && v.Kind() != constant.Unknown }

type valueCache struct {
	v constant.Value
}

func (n *valueCache) Value() constant.Value {
	if n.v != nil {
		return n.v
	}

	return unknown
}

func (n *valueCache) setValue(v constant.Value) constant.Value {
	n.v = v
	return v
}

type valuer interface {
	Value() constant.Value
}

type Expression interface {
	Node
	checkExpr(c *ctx) Expression
	clone() Expression
	typer
	valuer
}

type ValueExpression struct {
	Node
	typeCache
	valueCache
}

func (n *ValueExpression) checkExpr(c *ctx) Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *ValueExpression) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *BasicLitNode) Type() Type {
	switch n.Ch() {
	case CHAR:
		return n.ctx.int32
	case INT:
		return n.ctx.untypedInt
	case FLOAT:
		return n.ctx.untypedFloat
	case STRING:
		return n.ctx.untypedString
	default:
		panic(todo("%v: %T %s %v", n.Position(), n, n.Source(false), n.Ch()))
	}
}

func (n *BasicLitNode) Value() constant.Value {
	return constant.MakeFromLiteral(n.Src(), n.Ch(), 0)
}

func (n *BasicLitNode) checkExpr(c *ctx) Expression {
	n.ctx = c
	if !known(n.Value()) {
		c.err(n, "invalid literal: %s", n.Source(false))
	}
	return n
}

func (n *BasicLitNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandNameNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandNameNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandNameNode) checkExpr(c *ctx) Expression {
	in, named := n.LexicalScope().lookup(n.Name)
	switch x := named.n.(type) {
	case *ConstSpecNode:
		switch in.kind {
		case UniverseScope:
			switch n.Name.Src() {
			case "iota":
				if c.iota < 0 {
					panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
				}

				r := &ValueExpression{Node: x}
				r.t = c.untypedInt
				r.v = constant.MakeInt64(c.iota)
				return r
			default:
				panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
			}
		default:
			return x.Expression
		}
	default:
		panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
	}
}

func (n *OperandNameNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *ParenthesizedExpressionNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *ParenthesizedExpressionNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *ParenthesizedExpressionNode) checkExpr(c *ctx) Expression {
	return n.Expression.checkExpr(c)
}

func (n *ParenthesizedExpressionNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *LiteralValueNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *LiteralValueNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *LiteralValueNode) checkExpr(c *ctx) Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *LiteralValueNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *KeyedElementNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *KeyedElementNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *KeyedElementNode) checkExpr(c *ctx) Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *KeyedElementNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *CompositeLitNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *CompositeLitNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *CompositeLitNode) checkExpr(c *ctx) Expression {
	if n == nil {
		return nil
	}

	if !n.enter(c, n) {
		return n
	}

	t := n.setType(c.checkType(n.LiteralType))
	n.LiteralValue.check(c, t)
	return n
}

func (n *CompositeLitNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *LiteralValueNode) check(c *ctx, t Type) {
	if n == nil {
		return
	}

	switch t.Kind() {
	case Array:
		n.checkArray(c, t.(*ArrayTypeNode))
	default:
		panic(todo("%v: %T %s %v", n.Position(), n, n.Source(false), t.Kind()))
	}
}

func (n *LiteralValueNode) checkArray(c *ctx, t *ArrayTypeNode) {
	panic(todo("%v: %T %s %s", n.Position(), n, t, n.Source(false)))
}

func (n *FunctionLitNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *FunctionLitNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *FunctionLitNode) checkExpr(c *ctx) Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *FunctionLitNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandNode) checkExpr(c *ctx) Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandQualifiedNameNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandQualifiedNameNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandQualifiedNameNode) checkExpr(c *ctx) Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *OperandQualifiedNameNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *ConversionNode) Type() Type {
	return n.TypeNode
}

func (n *ConversionNode) checkExpr(c *ctx) Expression {
	t := n.TypeNode.check(c)
	n.Expression = n.Expression.checkExpr(c)
	v := n.Expression.Value()
	n.v = c.convertValue(n.Expression, v, t)
	return n
}

func (n *ConversionNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *MethodExprNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *MethodExprNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *MethodExprNode) checkExpr(c *ctx) Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *MethodExprNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *PrimaryExprNode) Type() Type {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *PrimaryExprNode) Value() constant.Value {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *PrimaryExprNode) checkExpr(c *ctx) Expression {
	switch x := n.PrimaryExpr.(type) {
	case *OperandNameNode:
		_, named := x.LexicalScope().lookup(x.Name)
		switch y := named.n.(type) {
		case *TypeDefNode:
			switch z := n.Postfix.(type) {
			case *ArgumentsNode:
				cnv := &ConversionNode{
					TypeNode: &TypeNameNode{
						Name:          x.Name,
						lexicalScoper: x.lexicalScoper,
					},
					LPAREN:     z.LPAREN,
					Expression: z.Expression,
					RPAREN:     z.RPAREN,
				}
				return cnv.checkExpr(c)
			default:
				panic(todo("%v: %T %s", n.Position(), z, n.Source(false)))
			}
		default:
			panic(todo("%v: %T %s", n.Position(), y, n.Source(false)))
		}
	default:
		panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
	}

	n.PrimaryExpr = n.PrimaryExpr.checkExpr(c)
	switch x := n.Postfix.(type) {
	default:
		panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
	}
}

func (n *PrimaryExprNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *BinaryExpressionNode) checkExpr(c *ctx) (r Expression) {
	if n == nil {
		return nil
	}

	if n.typeCache.Type() != Invalid {
		return n
	}

	n.LHS = n.LHS.checkExpr(c)
	n.RHS = n.RHS.checkExpr(c)
	lv := n.LHS.Value()
	lt := n.LHS.Type()
	rv := n.RHS.Value()
	rt := n.RHS.Type()

	defer func() {
		if known(lv) && known(rv) && r != nil && !known(r.Value()) {
			c.err(n.Op, "operation value not determined: %v %s %v", lv, n.Op.Src(), rv)
		}
	}()

	switch n.Op.Ch() {
	case SHL, SHR:
		var u uint64
		var uOk bool
		n.t = lt
		// The right operand in a shift expression must have integer type or be an
		// untyped constant representable by a value of type uint.
		switch {
		case isIntegerType(rt):
			// ok
		case known(rv):
			if isAnyArithmeticType(rt) {
				rv = c.convertValue(n.RHS, rv, c.cfg.uint)
				if known(rv) {
					u, uOk = constant.Uint64Val(rv)
				}
				break
			}

			c.err(n.Op, "TODO %v", n.Op.Src())
			return n
		default:
			panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
			return n
		}

		// If the left operand of a non-constant shift expression is an untyped
		// constant, it is first implicitly converted to the type it would assume if
		// the shift expression were replaced by its left operand alone.
		switch {
		case known(lv) && !known(rv):
			panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
			// c.err(n.Op, "TODO %v", n.Op.Ch.str())
			// return n
		case known(lv) && known(rv):
			if !uOk {
				panic(todo(""))
			}

			n.t = lt
			n.v = constant.Shift(lv, n.Op.Ch(), uint(u))
		default:
			trc("", known(lv), known(rv), u, uOk)
			panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
			// n.t = lt
			// n.v = constant.BinaryOp(lv, n.Op.Ch(), rv)
		}
	case ADD, SUB, MUL, QUO, REM:
		if !isAnyArithmeticType(lt) || !isAnyArithmeticType(rt) {
			c.err(n.Op, "TODO %v %v", lt, rt)
			break
		}

		// For other binary operators, the operand types must be identical unless the
		// operation involves shifts or untyped constants.
		//
		// Except for shift operations, if one operand is an untyped constant and the
		// other operand is not, the constant is implicitly converted to the type of
		// the other operand.
		switch {
		case isAnyUntypedType(lt) && isAnyUntypedType(rt):
			n.v = constant.BinaryOp(lv, n.Op.Ch(), rv)
			switch n.v.Kind() {
			case constant.Int:
				n.t = c.untypedInt
			case constant.Float:
				n.t = c.untypedFloat
			default:
				c.err(n.Op, "TODO %v %v %q %v %v -> %v %v", lv, lt, n.Op.Src(), rv, rt, n.v, n.v.Kind())
			}
		case isAnyUntypedType(lt) && !isAnyUntypedType(rt):
			c.err(n.Op, "TODO %v %v %q %v %v", lv, lt, n.Op.Src(), rv, rt)
		case !isAnyUntypedType(lt) && isAnyUntypedType(rt):
			c.err(n.Op, "TODO %v %v %q %v %v", lv, lt, n.Op.Src(), rv, rt)
		default: // case !isAnyUntypedType(lt) && !isAnyUntypedType(rt):
			c.err(n.Op, "TODO %v %v %q %v %v", lv, lt, n.Op.Src(), rv, rt)
		}
	default:
		c.err(n.Op, "TODO %v %v %q %v %v", lv, lt, n.Op.Src(), rv, rt)
	}
	return n
}

func (n *BinaryExpressionNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (n *UnaryExprNode) checkExpr(c *ctx) Expression {
	if n == nil {
		return nil
	}

	if n.typeCache.Type() != Invalid {
		return n
	}

	n.UnaryExpr = n.UnaryExpr.checkExpr(c)
	v := n.UnaryExpr.Value()
	t := n.UnaryExpr.Type()
	switch n.Op.Ch() {
	default:
		trc("", v, t)
		panic(todo("%v: %T %s", n.Op.Position(), n, n.Source(false)))
	}
}

func (n *UnaryExprNode) clone() Expression {
	panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
}

func (c *ctx) convertValue(n Node, v constant.Value, to Type) (r constant.Value) {
	if !known(v) {
		return unknown
	}

	switch to.Kind() {
	case
		Complex128,
		Complex64,
		Function,
		Interface,
		Map,
		Pointer,
		Slice,
		String,
		Struct,
		Tuple,
		UnsafePointer,
		UntypedBool,
		UntypedComplex,
		UntypedFloat,
		UntypedInt,
		UntypedNil,
		UntypedRune,
		UntypedString:

		c.err(n, "TODO %v -> %v", v, to)
	case Int:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		i64, ok := constant.Int64Val(w)
		if !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		switch c.cfg.goarch {
		case "386", "arm":
			if i64 < math.MinInt32 || i64 > math.MaxInt32 {
				c.err(n, "value %s overflows %s", v, to)
				return unknown
			}
		}
		return w
	case Int8:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		i64, ok := constant.Int64Val(w)
		if !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		if i64 < math.MinInt8 || i64 > math.MaxInt8 {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		return w
	case Int16:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		i64, ok := constant.Int64Val(w)
		if !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		if i64 < math.MinInt16 || i64 > math.MaxInt16 {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		return w
	case Int32:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		i64, ok := constant.Int64Val(w)
		if !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		if i64 < math.MinInt32 || i64 > math.MaxInt32 {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		return w
	case Int64:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		if _, ok := constant.Int64Val(w); !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		return w
	case Uint, Uintptr:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		u64, ok := constant.Uint64Val(w)
		if !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		switch c.cfg.goarch {
		case "386", "arm":
			if u64 > math.MaxUint32 {
				c.err(n, "value %s overflows %s", v, to)
				return unknown
			}
		}
		return w
	case Uint8:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		u64, ok := constant.Uint64Val(w)
		if !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		if u64 > math.MaxUint8 {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		return w
	case Uint16:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		u64, ok := constant.Uint64Val(w)
		if !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		if u64 > math.MaxUint16 {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		return w
	case Uint32:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		u64, ok := constant.Uint64Val(w)
		if !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		if u64 > math.MaxUint32 {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		return w
	case Uint64:
		w := constant.ToInt(v)
		if !known(w) {
			c.err(n, "cannot convert %s to %s", v, to)
			return unknown
		}

		if _, ok := constant.Uint64Val(w); !ok {
			c.err(n, "value %s overflows %s", v, to)
			return unknown
		}

		return w
	case Float32, Float64:
		return constant.ToFloat(v)
	case Bool:
		if v.Kind() == constant.Bool {
			return v
		}
	}
	return unknown
}
