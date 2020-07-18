package main

import (
	"fmt"
)

// BaseTypeMap is a map from X base types to Go types.
// X base types should correspond to the smallest set of X types
// that can be used to rewrite ALL X types in terms of Go types.
// That is, if you remove any of the following types, at least one
// XML protocol description will produce an invalid Go program.
// The types on the left *never* show themselves in the source.
var BaseTypeMap = map[string]string{
	"CARD8":  "byte",
	"CARD16": "uint16",
	"CARD32": "uint32",
	"INT8":   "int8",
	"INT16":  "int16",
	"INT32":  "int32",
	"BYTE":   "byte",
	"BOOL":   "bool",
	"float":  "float64",
	"double": "float64",
	"char":   "byte",
	"void":   "byte",
}

// BaseTypeSizes should have precisely the same keys as in BaseTypeMap,
// and the values should correspond to the size of the type in bytes.
var BaseTypeSizes = map[string]uint{
	"CARD8":  1,
	"CARD16": 2,
	"CARD32": 4,
	"INT8":   1,
	"INT16":  2,
	"INT32":  4,
	"BYTE":   1,
	"BOOL":   1,
	"float":  4,
	"double": 8,
	"char":   1,
	"void":   1,

	// Id is a special type used to determine the size of all Xid types.
	// "Id" is not actually written in the source.
	"Id": 4,
}

// TypeMap is a map from types in the XML to type names that is used
// in the functions that follow. Basically, every occurrence of the key
// type is replaced with the value type.
var TypeMap = map[string]string{
	"VISUALTYPE": "VisualInfo",
	"DEPTH":      "DepthInfo",
	"SCREEN":     "ScreenInfo",
	"Setup":      "SetupInfo",
}

// NameMap is the same as TypeMap, but for names.
var NameMap = map[string]string{}

// Reading, writing and defining...

// Base types
func (b *Base) Define(c *Context) {
	c.Putln("// Skipping definition for base type '%s'",
		SrcName(c.protocol, b.XmlName()))
	c.Putln("")
}

// Enum types
func (enum *Enum) Define(c *Context) {
	c.Putln("const (")
	for _, item := range enum.Items {
		c.Putln("%s%s = %d", enum.SrcName(), item.srcName, item.Expr.Eval())
	}
	c.Putln(")")
	c.Putln("")
}

// Resource types
func (res *Resource) Define(c *Context) {
	c.Putln("type %s uint32", res.SrcName())
	c.Putln("")
	c.Putln("func New%sId(c *xgb.Conn) (%s, error) {",
		res.SrcName(), res.SrcName())
	c.Putln("id, err := c.NewId()")
	c.Putln("if err != nil {")
	c.Putln("return 0, err")
	c.Putln("}")
	c.Putln("return %s(id), nil", res.SrcName())
	c.Putln("}")
	c.Putln("")
}

// TypeDef types
func (td *TypeDef) Define(c *Context) {
	c.Putln("type %s %s", td.srcName, td.Old.SrcName())
	c.Putln("")
}

// Field definitions, reads and writes.

// Pad fields
func (f *PadField) Define(c *Context) {
	if f.Align > 0 {
		c.Putln("// alignment gap to multiple of %d", f.Align)
	} else {
		c.Putln("// padding: %d bytes", f.Bytes)
	}
}

func (f *PadField) Read(c *Context, prefix string) {
	if f.Align > 0 {
		c.Putln("b = (b + %d) & ^%d // alignment gap", f.Align-1, f.Align-1)
	} else {
		c.Putln("b += %s // padding", f.Size())
	}
}

func (f *PadField) Write(c *Context, prefix string) {
	if f.Align > 0 {
		c.Putln("b = (b + %d) & ^%d // alignment gap", f.Align-1, f.Align-1)
	} else {
		c.Putln("b += %s // padding", f.Size())
	}
}

// Local fields
func (f *LocalField) Define(c *Context) {
	c.Putln("// local field: %s %s", f.SrcName(), f.Type.SrcName())
	panic("unreachable")
}

func (f *LocalField) Read(c *Context, prefix string) {
	c.Putln("// reading local field: %s (%s) :: %s",
		f.SrcName(), f.Size(), f.Type.SrcName())
	panic("unreachable")
}

func (f *LocalField) Write(c *Context, prefix string) {
	c.Putln("// skip writing local field: %s (%s) :: %s",
		f.SrcName(), f.Size(), f.Type.SrcName())
}

// Expr fields
func (f *ExprField) Define(c *Context) {
	c.Putln("// expression field: %s %s (%s)",
		f.SrcName(), f.Type.SrcName(), f.Expr)
	panic("unreachable")
}

func (f *ExprField) Read(c *Context, prefix string) {
	c.Putln("// reading expression field: %s (%s) (%s) :: %s",
		f.SrcName(), f.Size(), f.Expr, f.Type.SrcName())
	panic("unreachable")
}

func (f *ExprField) Write(c *Context, prefix string) {
	// Special case for bools, grrr.
	if f.Type.SrcName() == "bool" {
		c.Putln("buf[b] = byte(%s)", f.Expr.Reduce(prefix))
		c.Putln("b += 1")
	} else {
		WriteSimpleSingleField(c, f.Expr.Reduce(prefix), f.Type)
	}
}

// Value field
func (f *ValueField) Define(c *Context) {
	c.Putln("%s %s", f.MaskName, f.SrcType())
	c.Putln("%s []uint32", f.ListName)
}

func (f *ValueField) Read(c *Context, prefix string) {
	ReadSimpleSingleField(c,
		fmt.Sprintf("%s%s", prefix, f.MaskName), f.MaskType)
	c.Putln("")
	c.Putln("%s%s = make([]uint32, %s)",
		prefix, f.ListName, f.ListLength().Reduce(prefix))
	c.Putln("for i := 0; i < %s; i++ {", f.ListLength().Reduce(prefix))
	c.Putln("%s%s[i] = xgb.Get32(buf[b:])", prefix, f.ListName)
	c.Putln("b += 4")
	c.Putln("}")
	c.Putln("b = xgb.Pad(b)")
}

func (f *ValueField) Write(c *Context, prefix string) {
	// big time mofos
	if rq, ok := f.Parent.(*Request); !ok || rq.SrcName() != "ConfigureWindow" {
		WriteSimpleSingleField(c,
			fmt.Sprintf("%s%s", prefix, f.MaskName), f.MaskType)
	}
	c.Putln("for i := 0; i < %s; i++ {", f.ListLength().Reduce(prefix))
	c.Putln("xgb.Put32(buf[b:], %s%s[i])", prefix, f.ListName)
	c.Putln("b += 4")
	c.Putln("}")
	c.Putln("b = xgb.Pad(b)")
}

// Switch field
func (f *SwitchField) Define(c *Context) {
	c.Putln("// switch field: %s (%s)", f.Name, f.Expr)
	panic("todo")
}

func (f *SwitchField) Read(c *Context, prefix string) {
	c.Putln("// reading switch field: %s (%s)", f.Name, f.Expr)
	panic("todo")
}

func (f *SwitchField) Write(c *Context, prefix string) {
	c.Putln("// writing switch field: %s (%s)", f.Name, f.Expr)
	panic("todo")
}
