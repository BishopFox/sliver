package main

// Size corresponds to an expression that represents the number of bytes
// in some *thing*. Generally, sizes are used to allocate buffers and to
// inform X how big requests are.
// Size is basically a thin layer over an Expression that yields easy methods
// for adding and multiplying sizes.
type Size struct {
	Expression
	exact bool
}

// newFixedSize creates a new Size with some fixed and known value.
func newFixedSize(fixed uint, exact bool) Size {
	return Size{&Value{v: int(fixed)}, exact}
}

// newExpressionSize creates a new Size with some expression.
func newExpressionSize(variable Expression, exact bool) Size {
	return Size{variable, exact}
}

// Add adds s1 and s2 and returns a new Size.
func (s1 Size) Add(s2 Size) Size {
	return Size{newBinaryOp("+", s1, s2), s1.exact && s2.exact}
}

// Multiply mupltiplies s1 and s2 and returns a new Size.
func (s1 Size) Multiply(s2 Size) Size {
	return Size{newBinaryOp("*", s1, s2), s1.exact && s2.exact}
}
