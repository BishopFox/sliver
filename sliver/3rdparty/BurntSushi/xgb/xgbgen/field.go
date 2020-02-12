package main

import (
	"fmt"
	"log"
	"strings"
)

// Field corresponds to any field described in an XML protocol description
// file. This includes struct fields, union fields, request fields,
// reply fields and so on.
// To make code generation easier, fields that have types are also stored.
// Note that not all fields support all methods defined in this interface.
// For instance, a padding field does not have a source name.
type Field interface {
	// Initialize sets up the source name of this field.
	Initialize(p *Protocol)

	// SrcName is the Go source name of this field.
	SrcName() string

	// XmlName is the name of this field from the XML file.
	XmlName() string

	// SrcType is the Go source type name of this field.
	SrcType() string

	// Size returns an expression that computes the size (in bytes)
	// of this field.
	Size() Size

	// Define writes the Go code to declare this field (in a struct definition).
	Define(c *Context)

	// Read writes the Go code to convert a byte slice to a Go value
	// of this field.
	// 'prefix' is the prefix of the name of the Go value.
	Read(c *Context, prefix string)

	// Write writes the Go code to convert a Go value to a byte slice of
	// this field.
	// 'prefix' is the prefix of the name of the Go value.
	Write(c *Context, prefix string)
}

func (pad *PadField) Initialize(p *Protocol) {}

// PadField represents any type of padding. It is omitted from
// definitions, but is used in Read/Write to increment the buffer index.
// It is also used in size calculation.
type PadField struct {
	Bytes uint
	Align uint16
}

func (p *PadField) SrcName() string {
	panic("illegal to take source name of a pad field")
}

func (p *PadField) XmlName() string {
	panic("illegal to take XML name of a pad field")
}

func (f *PadField) SrcType() string {
	panic("it is illegal to call SrcType on a PadField field")
}

func (p *PadField) Size() Size {
	if p.Align > 0 {
		return newFixedSize(uint(p.Align), false)
	} else {
		return newFixedSize(p.Bytes, true)
	}
}

// SingleField represents most of the fields in an XML protocol description.
// It corresponds to any single value.
type SingleField struct {
	srcName string
	xmlName string
	Type    Type
}

func (f *SingleField) Initialize(p *Protocol) {
	f.srcName = SrcName(p, f.XmlName())
	f.Type = f.Type.(*Translation).RealType(p)
}

func (f *SingleField) SrcName() string {
	if f.srcName == "Bytes" {
		return "Bytes_"
	}
	return f.srcName
}

func (f *SingleField) XmlName() string {
	return f.xmlName
}

func (f *SingleField) SrcType() string {
	return f.Type.SrcName()
}

func (f *SingleField) Size() Size {
	return f.Type.Size()
}

// ListField represents a list of values.
type ListField struct {
	srcName    string
	xmlName    string
	Type       Type
	LengthExpr Expression
}

func (f *ListField) SrcName() string {
	return f.srcName
}

func (f *ListField) XmlName() string {
	return f.xmlName
}

func (f *ListField) SrcType() string {
	if strings.ToLower(f.Type.XmlName()) == "char" {
		return fmt.Sprintf("string")
	}
	return fmt.Sprintf("[]%s", f.Type.SrcName())
}

// Length computes the *number* of values in a list.
// If this ListField does not have any length expression, we throw our hands
// up and simply compute the 'len' of the field name of this list.
func (f *ListField) Length() Size {
	if f.LengthExpr == nil {
		return newExpressionSize(&Function{
			Name: "len",
			Expr: &FieldRef{
				Name: f.SrcName(),
			},
		}, true)
	}
	return newExpressionSize(f.LengthExpr, true)
}

// Size computes the *size* of a list (in bytes).
// It it typically a simple matter of multiplying the length of the list by
// the size of the type of the list.
// But if it's a list of struct where the struct has a list field, we use a
// special function written in go_struct.go to compute the size (since the
// size in this case can only be computed recursively).
func (f *ListField) Size() Size {
	elsz := f.Type.Size()
	simpleLen := &Padding{
		Expr: newBinaryOp("*", f.Length().Expression, elsz.Expression),
	}

	switch field := f.Type.(type) {
	case *Struct:
		if field.HasList() {
			sizeFun := &Function{
				Name: fmt.Sprintf("%sListSize", f.Type.SrcName()),
				Expr: &FieldRef{Name: f.SrcName()},
			}
			return newExpressionSize(sizeFun, elsz.exact)
		} else {
			return newExpressionSize(simpleLen, elsz.exact)
		}
	case *Union:
		return newExpressionSize(simpleLen, elsz.exact)
	case *Base:
		return newExpressionSize(simpleLen, elsz.exact)
	case *Resource:
		return newExpressionSize(simpleLen, elsz.exact)
	case *TypeDef:
		return newExpressionSize(simpleLen, elsz.exact)
	default:
		log.Panicf("Cannot compute list size with type '%T'.", f.Type)
	}
	panic("unreachable")
}

func (f *ListField) Initialize(p *Protocol) {
	f.srcName = SrcName(p, f.XmlName())
	f.Type = f.Type.(*Translation).RealType(p)
	if f.LengthExpr != nil {
		f.LengthExpr.Initialize(p)
	}
}

// LocalField is exactly the same as a regular SingleField, except it isn't
// sent over the wire. (i.e., it's probably used to compute an ExprField).
type LocalField struct {
	*SingleField
}

// ExprField is a field that is not parameterized, but is computed from values
// of other fields.
type ExprField struct {
	srcName string
	xmlName string
	Type    Type
	Expr    Expression
}

func (f *ExprField) SrcName() string {
	return f.srcName
}

func (f *ExprField) XmlName() string {
	return f.xmlName
}

func (f *ExprField) SrcType() string {
	return f.Type.SrcName()
}

func (f *ExprField) Size() Size {
	return f.Type.Size()
}

func (f *ExprField) Initialize(p *Protocol) {
	f.srcName = SrcName(p, f.XmlName())
	f.Type = f.Type.(*Translation).RealType(p)
	f.Expr.Initialize(p)
}

// ValueField represents two fields in one: a mask and a list of 4-byte
// integers. The mask specifies which kinds of values are in the list.
// (i.e., See ConfigureWindow, CreateWindow, ChangeWindowAttributes, etc.)
type ValueField struct {
	Parent   interface{}
	MaskType Type
	MaskName string
	ListName string
}

func (f *ValueField) SrcName() string {
	panic("it is illegal to call SrcName on a ValueField field")
}

func (f *ValueField) XmlName() string {
	panic("it is illegal to call XmlName on a ValueField field")
}

func (f *ValueField) SrcType() string {
	return f.MaskType.SrcName()
}

// Size computes the size in bytes of the combination of the mask and list
// in this value field.
// The expression to compute this looks complicated, but it's really just
// the number of bits set in the mask multiplied 4 (and padded of course).
func (f *ValueField) Size() Size {
	maskSize := f.MaskType.Size()
	listSize := newExpressionSize(&Function{
		Name: "xgb.Pad",
		Expr: &BinaryOp{
			Op:    "*",
			Expr1: &Value{v: 4},
			Expr2: &PopCount{
				Expr: &Function{
					Name: "int",
					Expr: &FieldRef{
						Name: f.MaskName,
					},
				},
			},
		},
	}, true)
	return maskSize.Add(listSize)
}

func (f *ValueField) ListLength() Size {
	return newExpressionSize(&PopCount{
		Expr: &Function{
			Name: "int",
			Expr: &FieldRef{
				Name: f.MaskName,
			},
		},
	}, true)
}

func (f *ValueField) Initialize(p *Protocol) {
	f.MaskType = f.MaskType.(*Translation).RealType(p)
	f.MaskName = SrcName(p, f.MaskName)
	f.ListName = SrcName(p, f.ListName)
}

// SwitchField represents a 'switch' element in the XML protocol description
// file. It is not currently used. (i.e., it is XKB voodoo.)
type SwitchField struct {
	Name     string
	Expr     Expression
	Bitcases []*Bitcase
}

func (f *SwitchField) SrcName() string {
	panic("it is illegal to call SrcName on a SwitchField field")
}

func (f *SwitchField) XmlName() string {
	panic("it is illegal to call XmlName on a SwitchField field")
}

func (f *SwitchField) SrcType() string {
	panic("it is illegal to call SrcType on a SwitchField field")
}

// XXX: This is a bit tricky. The size has to be represented as a non-concrete
// expression that finds *which* bitcase fields are included, and sums the
// sizes of those fields.
func (f *SwitchField) Size() Size {
	return newFixedSize(0, true)
}

func (f *SwitchField) Initialize(p *Protocol) {
	f.Name = SrcName(p, f.Name)
	f.Expr.Initialize(p)
	for _, bitcase := range f.Bitcases {
		bitcase.Expr.Initialize(p)
		for _, field := range bitcase.Fields {
			field.Initialize(p)
		}
	}
}

// Bitcase represents a single bitcase inside a switch expression.
// It is not currently used. (i.e., it's XKB voodoo.)
type Bitcase struct {
	Fields []Field
	Expr   Expression
}
