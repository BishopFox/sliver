package main

func (s *Struct) Define(c *Context) {
	c.Putln("type %s struct {", s.SrcName())
	for _, field := range s.Fields {
		field.Define(c)
	}
	c.Putln("}")
	c.Putln("")

	// Write function that reads bytes and produces this struct.
	s.Read(c)

	// Write function that reads bytes and produces a list of this struct.
	s.ReadList(c)

	// Write function that writes bytes given this struct.
	s.Write(c)

	// Write function that writes a list of this struct.
	s.WriteList(c)

	// Write function that computes the size of a list of these structs,
	// IF there is a list field in this struct.
	if s.HasList() {
		s.WriteListSize(c)
	}
}

// Read for a struct creates a function 'ReadStructName' that takes a source
// byte slice (i.e., the buffer) and a destination struct, and returns
// the number of bytes read off the buffer.
// 'ReadStructName' should only be used to read raw reply data from the wire.
func (s *Struct) Read(c *Context) {
	c.Putln("// %sRead reads a byte slice into a %s value.",
		s.SrcName(), s.SrcName())
	c.Putln("func %sRead(buf []byte, v *%s) int {", s.SrcName(), s.SrcName())

	c.Putln("b := 0")
	c.Putln("")
	for _, field := range s.Fields {
		field.Read(c, "v.")
		c.Putln("")
	}
	c.Putln("return b")

	c.Putln("}")
	c.Putln("")
}

// ReadList for a struct creates a function 'ReadStructNameList' that takes
// a source (i.e., the buffer) byte slice, and a destination slice and returns
// the number of bytes read from the byte slice.
func (s *Struct) ReadList(c *Context) {
	c.Putln("// %sReadList reads a byte slice into a list of %s values.",
		s.SrcName(), s.SrcName())
	c.Putln("func %sReadList(buf []byte, dest []%s) int {",
		s.SrcName(), s.SrcName())
	c.Putln("b := 0")
	c.Putln("for i := 0; i < len(dest); i++ {")
	c.Putln("dest[i] = %s{}", s.SrcName())
	c.Putln("b += %sRead(buf[b:], &dest[i])", s.SrcName())
	c.Putln("}")

	c.Putln("return xgb.Pad(b)")

	c.Putln("}")
	c.Putln("")
}

func (s *Struct) Write(c *Context) {
	c.Putln("// Bytes writes a %s value to a byte slice.", s.SrcName())
	c.Putln("func (v %s) Bytes() []byte {", s.SrcName())
	c.Putln("buf := make([]byte, %s)", s.Size().Reduce("v."))
	c.Putln("b := 0")
	c.Putln("")
	for _, field := range s.Fields {
		field.Write(c, "v.")
		c.Putln("")
	}
	c.Putln("return buf[:b]")
	c.Putln("}")
	c.Putln("")
}

func (s *Struct) WriteList(c *Context) {
	c.Putln("// %sListBytes writes a list of %s values to a byte slice.",
		s.SrcName(), s.SrcName())
	c.Putln("func %sListBytes(buf []byte, list []%s) int {",
		s.SrcName(), s.SrcName())
	c.Putln("b := 0")
	c.Putln("var structBytes []byte")
	c.Putln("for _, item := range list {")
	c.Putln("structBytes = item.Bytes()")
	c.Putln("copy(buf[b:], structBytes)")
	c.Putln("b += len(structBytes)")
	c.Putln("}")
	c.Putln("return xgb.Pad(b)")
	c.Putln("}")
	c.Putln("")
}

func (s *Struct) WriteListSize(c *Context) {
	c.Putln("// %sListSize computes the size (bytes) of a list of %s values.",
		s.SrcName(), s.SrcName())
	c.Putln("func %sListSize(list []%s) int {", s.SrcName(), s.SrcName())
	c.Putln("size := 0")
	if s.Size().Expression.Concrete() {
		c.Putln("for _ = range list {")
	} else {
		c.Putln("for _, item := range list {")
	}
	c.Putln("size += %s", s.Size().Reduce("item."))
	c.Putln("}")
	c.Putln("return size")
	c.Putln("}")
	c.Putln("")
}
