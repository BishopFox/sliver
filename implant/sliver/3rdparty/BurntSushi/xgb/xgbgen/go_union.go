package main

// Union types
func (u *Union) Define(c *Context) {
	c.Putln("// %s is a represention of the %s union type.",
		u.SrcName(), u.SrcName())
	c.Putln("// Note that to *create* a Union, you should *never* create")
	c.Putln("// this struct directly (unless you know what you're doing).")
	c.Putln("// Instead use one of the following constructors for '%s':",
		u.SrcName())
	for _, field := range u.Fields {
		c.Putln("//     %s%sNew(%s %s) %s", u.SrcName(), field.SrcName(),
			field.SrcName(), field.SrcType(), u.SrcName())
	}

	c.Putln("type %s struct {", u.SrcName())
	for _, field := range u.Fields {
		field.Define(c)
	}
	c.Putln("}")
	c.Putln("")

	// Write functions for each field that create instances of this
	// union using the corresponding field.
	u.New(c)

	// Write function that reads bytes and produces this union.
	u.Read(c)

	// Write function that reads bytes and produces a list of this union.
	u.ReadList(c)

	// Write function that writes bytes given this union.
	u.Write(c)

	// Write function that writes a list of this union.
	u.WriteList(c)
}

func (u *Union) New(c *Context) {
	for _, field := range u.Fields {
		c.Putln("// %s%sNew constructs a new %s union type with the %s field.",
			u.SrcName(), field.SrcName(), u.SrcName(), field.SrcName())
		c.Putln("func %s%sNew(%s %s) %s {",
			u.SrcName(), field.SrcName(), field.SrcName(),
			field.SrcType(), u.SrcName())
		c.Putln("var b int")
		c.Putln("buf := make([]byte, %s)", u.Size())
		c.Putln("")
		field.Write(c, "")
		c.Putln("")
		c.Putln("// Create the Union type")
		c.Putln("v := %s{}", u.SrcName())
		c.Putln("")
		c.Putln("// Now copy buf into all fields")
		c.Putln("")
		for _, field2 := range u.Fields {
			c.Putln("b = 0 // always read the same bytes")
			field2.Read(c, "v.")
			c.Putln("")
		}
		c.Putln("return v")
		c.Putln("}")
		c.Putln("")
	}
}

func (u *Union) Read(c *Context) {
	c.Putln("// %sRead reads a byte slice into a %s value.",
		u.SrcName(), u.SrcName())
	c.Putln("func %sRead(buf []byte, v *%s) int {", u.SrcName(), u.SrcName())
	c.Putln("var b int")
	c.Putln("")
	for _, field := range u.Fields {
		c.Putln("b = 0 // re-read the same bytes")
		field.Read(c, "v.")
		c.Putln("")
	}
	c.Putln("return %s", u.Size())
	c.Putln("}")
	c.Putln("")
}

func (u *Union) ReadList(c *Context) {
	c.Putln("// %sReadList reads a byte slice into a list of %s values.",
		u.SrcName(), u.SrcName())
	c.Putln("func %sReadList(buf []byte, dest []%s) int {",
		u.SrcName(), u.SrcName())
	c.Putln("b := 0")
	c.Putln("for i := 0; i < len(dest); i++ {")
	c.Putln("dest[i] = %s{}", u.SrcName())
	c.Putln("b += %sRead(buf[b:], &dest[i])", u.SrcName())
	c.Putln("}")
	c.Putln("return xgb.Pad(b)")
	c.Putln("}")
	c.Putln("")
}

// This is a bit tricky since writing from a Union implies that only
// the data inside ONE of the elements is actually written.
// However, we only currently support unions where every field has the
// *same* *fixed* size. Thus, we make sure to always read bytes into
// every field which allows us to simply pick the first field and write it.
func (u *Union) Write(c *Context) {
	c.Putln("// Bytes writes a %s value to a byte slice.", u.SrcName())
	c.Putln("// Each field in a union must contain the same data.")
	c.Putln("// So simply pick the first field and write that to the wire.")
	c.Putln("func (v %s) Bytes() []byte {", u.SrcName())
	c.Putln("buf := make([]byte, %s)", u.Size().Reduce("v."))
	c.Putln("b := 0")
	c.Putln("")
	u.Fields[0].Write(c, "v.")
	c.Putln("return buf")
	c.Putln("}")
	c.Putln("")
}

func (u *Union) WriteList(c *Context) {
	c.Putln("// %sListBytes writes a list of %s values to a byte slice.",
		u.SrcName(), u.SrcName())
	c.Putln("func %sListBytes(buf []byte, list []%s) int {",
		u.SrcName(), u.SrcName())
	c.Putln("b := 0")
	c.Putln("var unionBytes []byte")
	c.Putln("for _, item := range list {")
	c.Putln("unionBytes = item.Bytes()")
	c.Putln("copy(buf[b:], unionBytes)")
	c.Putln("b += xgb.Pad(len(unionBytes))")
	c.Putln("}")
	c.Putln("return b")
	c.Putln("}")
	c.Putln("")
}

func (u *Union) WriteListSize(c *Context) {
	c.Putln("// Union list size %s", u.SrcName())
	c.Putln("// %sListSize computes the size (bytes) of a list of %s values.",
		u.SrcName())
	c.Putln("func %sListSize(list []%s) int {", u.SrcName(), u.SrcName())
	c.Putln("size := 0")
	c.Putln("for _, item := range list {")
	c.Putln("size += %s", u.Size().Reduce("item."))
	c.Putln("}")
	c.Putln("return size")
	c.Putln("}")
	c.Putln("")
}
