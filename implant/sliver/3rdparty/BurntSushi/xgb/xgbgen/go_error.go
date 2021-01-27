package main

import (
	"fmt"
)

// Error types
func (e *Error) Define(c *Context) {
	c.Putln("// %s is the error number for a %s.", e.ErrConst(), e.ErrConst())
	c.Putln("const %s = %d", e.ErrConst(), e.Number)
	c.Putln("")
	c.Putln("type %s struct {", e.ErrType())
	c.Putln("Sequence uint16")
	c.Putln("NiceName string")
	for _, field := range e.Fields {
		field.Define(c)
	}
	c.Putln("}")
	c.Putln("")

	// Read defines a function that transforms a byte slice into this
	// error struct.
	e.Read(c)

	// Makes sure this error type implements the xgb.Error interface.
	e.ImplementsError(c)

	// Let's the XGB event loop read this error.
	c.Putln("func init() {")
	if c.protocol.isExt() {
		c.Putln("xgb.NewExtErrorFuncs[\"%s\"][%d] = %sNew",
			c.protocol.ExtXName, e.Number, e.ErrType())
	} else {
		c.Putln("xgb.NewErrorFuncs[%d] = %sNew", e.Number, e.ErrType())
	}
	c.Putln("}")
	c.Putln("")
}

func (e *Error) Read(c *Context) {
	c.Putln("// %sNew constructs a %s value that implements xgb.Error from "+
		"a byte slice.", e.ErrType(), e.ErrType())
	c.Putln("func %sNew(buf []byte) xgb.Error {", e.ErrType())
	c.Putln("v := %s{}", e.ErrType())
	c.Putln("v.NiceName = \"%s\"", e.SrcName())
	c.Putln("")
	c.Putln("b := 1 // skip error determinant")
	c.Putln("b += 1 // don't read error number")
	c.Putln("")
	c.Putln("v.Sequence = xgb.Get16(buf[b:])")
	c.Putln("b += 2")
	c.Putln("")
	for _, field := range e.Fields {
		field.Read(c, "v.")
		c.Putln("")
	}
	c.Putln("return v")
	c.Putln("}")
	c.Putln("")
}

// ImplementsError writes functions to implement the XGB Error interface.
func (e *Error) ImplementsError(c *Context) {
	c.Putln("// SequenceId returns the sequence id attached to the %s error.",
		e.ErrConst())
	c.Putln("// This is mostly used internally.")
	c.Putln("func (err %s) SequenceId() uint16 {", e.ErrType())
	c.Putln("return err.Sequence")
	c.Putln("}")
	c.Putln("")
	c.Putln("// BadId returns the 'BadValue' number if one exists for the "+
		"%s error. If no bad value exists, 0 is returned.", e.ErrConst())
	c.Putln("func (err %s) BadId() uint32 {", e.ErrType())
	if !c.protocol.isExt() {
		c.Putln("return err.BadValue")
	} else {
		c.Putln("return 0")
	}
	c.Putln("}")
	c.Putln("// Error returns a rudimentary string representation of the %s "+
		"error.", e.ErrConst())
	c.Putln("")
	c.Putln("func (err %s) Error() string {", e.ErrType())
	ErrorFieldString(c, e.Fields, e.ErrConst())
	c.Putln("}")
	c.Putln("")
}

// ErrorCopy types
func (e *ErrorCopy) Define(c *Context) {
	c.Putln("// %s is the error number for a %s.", e.ErrConst(), e.ErrConst())
	c.Putln("const %s = %d", e.ErrConst(), e.Number)
	c.Putln("")
	c.Putln("type %s %s", e.ErrType(), e.Old.(*Error).ErrType())
	c.Putln("")

	// Read defines a function that transforms a byte slice into this
	// error struct.
	e.Read(c)

	// Makes sure this error type implements the xgb.Error interface.
	e.ImplementsError(c)

	// Let's the XGB know how to read this error.
	c.Putln("func init() {")
	if c.protocol.isExt() {
		c.Putln("xgb.NewExtErrorFuncs[\"%s\"][%d] = %sNew",
			c.protocol.ExtXName, e.Number, e.ErrType())
	} else {
		c.Putln("xgb.NewErrorFuncs[%d] = %sNew", e.Number, e.ErrType())
	}
	c.Putln("}")
	c.Putln("")
}

func (e *ErrorCopy) Read(c *Context) {
	c.Putln("// %sNew constructs a %s value that implements xgb.Error from "+
		"a byte slice.", e.ErrType(), e.ErrType())
	c.Putln("func %sNew(buf []byte) xgb.Error {", e.ErrType())
	c.Putln("v := %s(%sNew(buf).(%s))",
		e.ErrType(), e.Old.(*Error).ErrType(), e.Old.(*Error).ErrType())
	c.Putln("v.NiceName = \"%s\"", e.SrcName())
	c.Putln("return v")
	c.Putln("}")
	c.Putln("")
}

// ImplementsError writes functions to implement the XGB Error interface.
func (e *ErrorCopy) ImplementsError(c *Context) {
	c.Putln("// SequenceId returns the sequence id attached to the %s error.",
		e.ErrConst())
	c.Putln("// This is mostly used internally.")
	c.Putln("func (err %s) SequenceId() uint16 {", e.ErrType())
	c.Putln("return err.Sequence")
	c.Putln("}")
	c.Putln("")
	c.Putln("// BadId returns the 'BadValue' number if one exists for the "+
		"%s error. If no bad value exists, 0 is returned.", e.ErrConst())
	c.Putln("func (err %s) BadId() uint32 {", e.ErrType())
	if !c.protocol.isExt() {
		c.Putln("return err.BadValue")
	} else {
		c.Putln("return 0")
	}
	c.Putln("}")
	c.Putln("")
	c.Putln("// Error returns a rudimentary string representation of the %s "+
		"error.", e.ErrConst())
	c.Putln("func (err %s) Error() string {", e.ErrType())
	ErrorFieldString(c, e.Old.(*Error).Fields, e.ErrConst())
	c.Putln("}")
	c.Putln("")
}

// ErrorFieldString works for both Error and ErrorCopy. It assembles all of the
// fields in an error and formats them into a single string.
func ErrorFieldString(c *Context, fields []Field, errName string) {
	c.Putln("fieldVals := make([]string, 0, %d)", len(fields))
	c.Putln("fieldVals = append(fieldVals, \"NiceName: \" + err.NiceName)")
	c.Putln("fieldVals = append(fieldVals, "+
		"xgb.Sprintf(\"Sequence: %s\", err.Sequence))", "%d")
	for _, field := range fields {
		switch field.(type) {
		case *PadField:
			continue
		default:
			if field.SrcType() == "string" {
				c.Putln("fieldVals = append(fieldVals, \"%s: \" + err.%s)",
					field.SrcName(), field.SrcName())
			} else {
				format := fmt.Sprintf("xgb.Sprintf(\"%s: %s\", err.%s)",
					field.SrcName(), "%d", field.SrcName())
				c.Putln("fieldVals = append(fieldVals, %s)", format)
			}
		}
	}
	c.Putln("return \"%s {\" + xgb.StringsJoin(fieldVals, \", \") + \"}\"",
		errName)
}
