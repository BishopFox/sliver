// Copyright 2022 The Gc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gc // modernc.org/gc/v3

import (
	"fmt"
	"go/constant"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type ctx struct {
	ast  *AST
	cfg  *Config
	errs errList
	iota int64
	pkg  *Package

	int32         Type // Set by newCtx
	untypedFloat  Type // Set by newCtx
	untypedInt    Type // Set by newCtx
	untypedString Type // Set by newCtx
}

func newCtx(cfg *Config) (r *ctx) {
	r = &ctx{
		cfg:  cfg,
		iota: -1, // -> Invalid
	}
	r.int32 = r.newPredeclaredType(znode, Int32)
	r.untypedFloat = r.newPredeclaredType(znode, UntypedFloat)
	r.untypedInt = r.newPredeclaredType(znode, UntypedInt)
	r.untypedString = r.newPredeclaredType(znode, UntypedString)
	return r
}

func (c *ctx) err(n Node, msg string, args ...interface{}) {
	var pos token.Position
	if n != nil {
		pos = n.Position()
	}
	s := fmt.Sprintf(msg, args...)
	if trcTODOs && strings.HasPrefix(s, "TODO") {
		fmt.Fprintf(os.Stderr, "%v: %s (%v)\n", pos, s, origin(2))
		os.Stderr.Sync()
	}
	switch {
	case extendedErrors:
		c.errs.err(pos, "%s (%v: %v: %v)", s, origin(4), origin(3), origin(2))
	default:
		c.errs.err(pos, s)
	}
}

func (c *ctx) isBuiltin() bool { return c.pkg.Scope.kind == UniverseScope }
func (c *ctx) isUnsafe() bool  { return c.pkg.isUnsafe }

func (c *ctx) lookup(sc *Scope, id Token) (pkg *Package, in *Scope, r named) {
	sc0 := sc
	pkg = c.pkg
	for {
		switch in, nm := sc.lookup(id); x := nm.n.(type) {
		case *TypeDefNode:
			if sc.kind == UniverseScope {
				if sc0.kind != UniverseScope && token.IsExported(id.Src()) {
					// trc("%v: %q %v %v", id.Position(), id.Src(), sc0.kind, sc.kind)
					return nil, nil, r
				}
			}

			return x.pkg, in, nm
		default:
			panic(todo("%v: %q %T", id.Position(), id.Src(), x))
		}
	}
}

func (n *Package) check(c *ctx) (err error) {
	if n == nil {
		return nil
	}

	c.pkg = n
	// trc("PKG %q", n.ImportPath)
	// defer func() { trc("PKG %q -> err: %v", n.ImportPath, err) }()
	for _, v := range n.GoFiles {
		path := filepath.Join(n.FSPath, v.Name())
		n.AST[path].check(c)
	}
	return c.errs.Err()
}

func (n *AST) check(c *ctx) {
	if n == nil {
		return
	}

	c.ast = n
	n.SourceFile.check(c)
}

func (n *SourceFileNode) check(c *ctx) {
	if n == nil {
		return
	}

	n.PackageClause.check(c)
	for l := n.ImportDeclList; l != nil; l = l.List {
		l.ImportDecl.check(c)
	}
	for l := n.TopLevelDeclList; l != nil; l = l.List {
		switch x := l.TopLevelDecl.(type) {
		case *TypeDeclNode:
			x.check(c)
		case *ConstDeclNode:
			x.check(c)
		case *VarDeclNode:
			x.check(c)
		case *FunctionDeclNode:
			x.check(c)
		case *MethodDeclNode:
			x.check(c)
		default:
			panic(todo("%v: %T %s", x.Position(), x, x.Source(false)))
		}
	}
}

func (n *MethodDeclNode) check(c *ctx) {
	if n == nil {
		return
	}

	n.Receiver.check(c)
	n.Signature.check(c)
}

func (n *FunctionDeclNode) check(c *ctx) {
	if n == nil {
		return
	}

	if c.isBuiltin() {
		switch nm := n.FunctionName.IDENT.Src(); nm {
		case
			"append",
			"cap",
			"close",
			"complex",
			"copy",
			"delete",
			"imag",
			"len",
			"make",
			"new",
			"panic",
			"print",
			"println",
			"real",
			"recover",

			// Go 1.21
			"max",
			"min",
			"clear":

			n.Signature.t = c.newPredeclaredType(n, Function)
		default:
			panic(todo("%v: %q %s", n.Position(), nm, n.Source(false)))
		}
		return
	}

	n.Signature.check(c)
	if n.TypeParameters != nil {
		panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
	}
}

func (n *SignatureNode) check(c *ctx) Type {
	if n == nil {
		return Invalid
	}

	if !n.enter(c, n) {
		return n.Type()
	}

	in := n.Parameters.check(c)
	out := n.Result.check(c)
	return n.setType(newTupleType(n.Parameters, []Type{in, out}))
}

func (n *ResultNode) check(c *ctx) Type {
	if n == nil {
		return Invalid
	}

	switch {
	case n.Parameters != nil:
		return n.Parameters.check(c)
	case n.TypeNode != nil:
		return n.TypeNode.check(c)
	default:
		panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
	}
}

func (n *ParametersNode) check(c *ctx) Type {
	if n == nil {
		return Invalid
	}

	r := newTupleType(n, nil)
	for l := n.ParameterDeclList; l != nil; l = l.List {
		r.Types = append(r.Types, l.ParameterDecl.check(c)...)
	}
	return r
}

func (n *ParameterDeclNode) check(c *ctx) (r []Type) {
	if n == nil {
		return nil
	}

	t := n.TypeNode.check(c)
	for l := n.IdentifierList; l != nil; l = l.List {
		r = append(r, t)
	}
	return r
}

func (n *VarDeclNode) check(c *ctx) {
	if n == nil {
		return
	}

	switch x := n.VarSpec.(type) {
	case *VarSpecNode:
		x.check(c)
	default:
		panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
	}
}

func (n *VarSpecNode) check(c *ctx) {
	if n == nil {
		return
	}

	if c.isBuiltin() {
		switch nm := n.IDENT.Src(); nm {
		case "nil":
			n.TypeNode = c.newPredeclaredType(n, UntypedNil)
		default:
			panic(todo("%v: %q", n.IDENT.Position(), nm))
		}
		return
	}

	if n.TypeNode != nil {
		c.err(n, "TODO %v", n.TypeNode.Source(false))
	}
	var e []Expression
	for l := n.ExpressionList; l != nil; l = l.List {
		e = append(e, l.Expression.checkExpr(c))
	}
	switch len(e) {
	default:
		panic(todo("", len(e)))
		c.err(n, "TODO %v", len(e))
	}
}

func (n *ConstDeclNode) check(c *ctx) {
	if n == nil {
		return
	}

	switch x := n.ConstSpec.(type) {
	case *ConstSpecListNode:
		var prev Node
		for l := x; l != nil; l = l.List {
			switch y := l.ConstSpec.(type) {
			case *ConstSpecNode:
				y.check(c, prev)
				if y.Expression != nil || y.TypeNode != nil {
					prev = y
				}
			default:
				panic(todo("%v: %T %s", n.Position(), y, n.Source(false)))
			}
		}
	case *ConstSpecNode:
		x.check(c, nil)
	default:
		panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
	}

}

func (n *ConstSpecNode) check(c *ctx, prev Node) {
	if n == nil {
		return
	}

	if !n.enter(c, n) {
		if n.guard == guardChecking {
			panic(todo("")) // report recursive
		}
		return
	}

	defer func() { n.guard = guardChecked }()

	if c.isBuiltin() {
		switch n.IDENT.Src() {
		case "true":
			switch x := n.Expression.(type) {
			case *BinaryExpressionNode:
				x.setValue(trueVal)
				x.setType(c.newPredeclaredType(x, UntypedBool))
			default:
				panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
			}
		case "false":
			switch x := n.Expression.(type) {
			case *BinaryExpressionNode:
				x.setValue(falseVal)
				x.setType(c.newPredeclaredType(x, UntypedBool))
			default:
				panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
			}
		case "iota":
			switch x := n.Expression.(type) {
			case *BasicLitNode:
				// ok
			default:
				panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
			}
		default:
			panic(todo("", n.Position(), n.Source(false)))
		}
		return
	}

	save := c.iota
	c.iota = n.iota

	defer func() { c.iota = save }()

	switch {
	case n.Expression != nil:
		n.Expression = n.Expression.checkExpr(c)
		if n.TypeNode == nil {
			n.TypeNode = n.Expression.Type()
			return
		}

		t := n.TypeNode.check(c)
		trc("", t)
		panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
	default:
		// var e Expression
		// var pe *Expression
		// switch {
		// case n.Expression != nil:
		// 	e = n.Expression
		// 	pe = &n.Expression
		// default:
		// 	switch x := prev.(type) {
		// 	case *ConstSpecNode:
		// 		e = x.Expression.clone()
		// 		pe = &e
		// 	default:
		// 		panic(todo("%v: %T %s", n.Position(), x, n.Source(false)))
		// 	}
		// }
		// ev, et := e.checkExpr(c, pe)
		// e = *pe
		// if ev.Kind() == constant.Unknown {
		// 	c.err(e, "%s is not a constant", e.Source(false))
		// 	n.t = Invalid
		// 	n.setValue(unknown)
		// 	return Invalid
		// }
		// switch {
		// case n.t == nil:
		// 	n.t = et
		// default:

		// 		c.err(n.Expression, "cannot assign %v (type %v) to type %v", ev, et, n.Type())
		// 		return Invalid
		// 	} else {
		// 		n.setValue(convertValue(c, e, ev, n.Type()))
		// 	}
		// }
		// return n.Type()
		panic(todo("%v: %T %s", n.Position(), n, n.Source(false)))
	}

}

func (n *TypeDeclNode) check(c *ctx) {
	if n == nil {
		return
	}

	for l := n.TypeSpecList; l != nil; l = l.List {
		switch x := l.TypeSpec.(type) {
		case *TypeDefNode:
			switch {
			case c.isBuiltin():
				x.pkg = c.pkg
				switch nm := x.IDENT.Src(); nm {
				case "bool":
					x.TypeNode = c.newPredeclaredType(x, Bool)
				case "int":
					x.TypeNode = c.newPredeclaredType(x, Int)
					c.cfg.int = x.TypeNode
				case "int8":
					x.TypeNode = c.newPredeclaredType(x, Int8)
				case "int16":
					x.TypeNode = c.newPredeclaredType(x, Int16)
				case "int32":
					x.TypeNode = c.newPredeclaredType(x, Int32)
				case "int64":
					x.TypeNode = c.newPredeclaredType(x, Int64)
				case "uint":
					x.TypeNode = c.newPredeclaredType(x, Uint)
					c.cfg.uint = x.TypeNode
				case "uint8":
					x.TypeNode = c.newPredeclaredType(x, Uint8)
				case "uint16":
					x.TypeNode = c.newPredeclaredType(x, Uint16)
				case "uint32":
					x.TypeNode = c.newPredeclaredType(x, Uint32)
				case "uint64":
					x.TypeNode = c.newPredeclaredType(x, Uint64)
				case "uintptr":
					x.TypeNode = c.newPredeclaredType(x, Uintptr)
				case "string":
					x.TypeNode = c.newPredeclaredType(x, String)
				case "float32":
					x.TypeNode = c.newPredeclaredType(x, Float32)
				case "float64":
					x.TypeNode = c.newPredeclaredType(x, Float64)
				case "complex64":
					x.TypeNode = c.newPredeclaredType(x, Complex64)
				case "complex128":
					x.TypeNode = c.newPredeclaredType(x, Complex128)
				case "comparable":
					x.TypeNode = c.newPredeclaredType(x, Interface)
				case "error":
					x.check(c)
				default:
					if token.IsExported(nm) {
						delete(c.pkg.Scope.nodes, nm)
						return
					}

					panic(todo("%v: %T %s", x.Position(), x, x.Source(false)))
				}
			case c.isUnsafe():
				switch nm := x.IDENT.Src(); nm {
				case "ArbitraryType", "IntegerType", "Pointer":
					x.TypeNode.check(c)
				default:
					panic(todo("%v: %T %s", x.Position(), x, x.Source(false)))
				}
			default:
				switch {
				case x.TypeParameters != nil:
					panic(todo("%v: %T %s", x.Position(), x, x.Source(false)))
				default:
					x.check(c)
				}
			}
		case *AliasDeclNode:
			x.check(c)
		default:
			panic(todo("%v: %T %s", x.Position(), x, x.Source(false)))
		}
	}
}

func (n *AliasDeclNode) check(c *ctx) {
	if n == nil {
		return
	}

	n.TypeNode.check(c)
}

func (n *ImportDeclNode) check(c *ctx) {
	if n == nil {
		return
	}

	type result struct {
		spec *ImportSpecNode
		pkg  *Package
		err  error
	}
	var a []*result
	var wg sync.WaitGroup
	for l := n.ImportSpecList; l != nil; l = l.List {
		r := &result{}
		a = append(a, r)
		wg.Add(1)
		go func(isln *ImportSpecListNode, r *result) {

			defer wg.Done()

			r.spec = isln.ImportSpec
			r.pkg, r.err = r.spec.check(c)
			r.spec.pkg = r.pkg
		}(l, r)
	}
	wg.Wait()
	fileScope := c.ast.FileScope
	pkgScope := c.pkg.Scope
	for _, v := range a {
		switch x := v.err.(type) {
		case nil:
			// ok
		default:
			panic(todo("%v: %T: %s", v.spec.Position(), x, x))
		}
		if c.pkg.ImportPath == "builtin" && v.spec.ImportPath.Src() == `"cmp"` {
			continue
		}

		switch ex := fileScope.declare(v.pkg.Name, v.spec, 0, nil, true); {
		case ex.declTok.IsValid():
			c.err(n, "%s redeclared, previous declaration at %v:", v.pkg.Name.Src(), ex.declTok.Position())
			continue
		}

		switch ex := pkgScope.declare(v.pkg.Name, v.spec, 0, nil, true); {
		case ex.declTok.IsValid():
			c.err(n, "%s redeclared, previous declaration at %v:", v.pkg.Name.Src(), ex.declTok.Position())
			continue
		}
	}
}

func (n *ImportSpecNode) check(c *ctx) (*Package, error) {
	if n == nil {
		return nil, nil
	}

	switch {
	case n.PERIOD.IsValid():
		panic(todo("", n.Position(), n.Source(false)))
	case n.PackageName.IsValid():
		//TODO version
		check := c.pkg.typeCheck
		switch check {
		case TypeCheckAll:
			// nop
		default:
			panic(todo("", check))
		}
		return c.cfg.newPackage(c.pkg.FSPath, constant.StringVal(n.ImportPath.Value()), "", nil, false, check, c.pkg.guard)
	default:
		//TODO version
		check := c.pkg.typeCheck
		switch check {
		case TypeCheckAll:
			// nop
		default:
			if c.pkg.ImportPath == "builtin" && n.ImportPath.Src() == `"cmp"` {
				return nil, nil
			}
		}
		return c.cfg.newPackage(c.pkg.FSPath, constant.StringVal(n.ImportPath.Value()), "", nil, false, check, c.pkg.guard)
	}
}

func (n *PackageClauseNode) check(c *ctx) {
	if n == nil {
		return
	}

	nm := n.PackageName.Src()
	if ex := c.pkg.Name; ex.IsValid() && ex.Src() != nm {
		c.err(n.PackageName, "found different packages %q and %q", ex.Src(), nm)
		return
	}

	c.pkg.Name = n.PackageName
}
