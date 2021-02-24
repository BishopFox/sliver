package main

import (
	"fmt"
	"strings"
)

type Type interface {
	Initialize(p *Protocol)
	SrcName() string
	XmlName() string
	Size() Size

	Define(c *Context)
}

type Types []Type

func (ts Types) Len() int      { return len(ts) }
func (ts Types) Swap(i, j int) { ts[i], ts[j] = ts[j], ts[i] }
func (ts Types) Less(i, j int) bool {
	x1, x2 := ts[i].XmlName(), ts[j].XmlName()
	s1, s2 := ts[i].SrcName(), ts[j].SrcName()
	return (s1 == s2 && x1 < x2) || s1 < s2
}

// Translation is used *only* when transitioning from XML types to
// our better representation. They are placeholders for the real types (below)
// that will replace them.
type Translation struct {
	xmlName string
}

func newTranslation(name string) *Translation {
	return &Translation{xmlName: name}
}

// RealType takes 'XmlName' and finds its real concrete type in our Protocol.
// It is an error if we can't find such a type.
func (t *Translation) RealType(p *Protocol) Type {
	// Check to see if there is a namespace. If so, strip it and use it to
	// make sure we only look for a type in that protocol.
	namespace, typeName := "", t.XmlName()
	if ni := strings.Index(t.XmlName(), ":"); ni > -1 {
		namespace, typeName = strings.ToLower(typeName[:ni]), typeName[ni+1:]
	}

	if len(namespace) == 0 || namespace == strings.ToLower(p.Name) {
		for _, typ := range p.Types {
			if typeName == typ.XmlName() {
				return typ
			}
		}
	}
	for _, imp := range p.Imports {
		if len(namespace) == 0 || namespace == strings.ToLower(imp.Name) {
			for _, typ := range imp.Types {
				if typeName == typ.XmlName() {
					return typ
				}
			}
		}
	}
	panic("Could not find real type for translation type: " + t.XmlName())
}

func (t *Translation) SrcName() string {
	panic("it is illegal to call SrcName on a translation type")
}

func (t *Translation) XmlName() string {
	return t.xmlName
}

func (t *Translation) Size() Size {
	panic("it is illegal to call Size on a translation type")
}

func (t *Translation) Define(c *Context) {
	panic("it is illegal to call Define on a translation type")
}

func (t *Translation) Initialize(p *Protocol) {
	panic("it is illegal to call Initialize on a translation type")
}

type Base struct {
	srcName string
	xmlName string
	size    Size
}

func (b *Base) SrcName() string {
	return b.srcName
}

func (b *Base) XmlName() string {
	return b.xmlName
}

func (b *Base) Size() Size {
	return b.size
}

func (b *Base) Initialize(p *Protocol) {
	b.srcName = TypeSrcName(p, b)
}

type Enum struct {
	srcName string
	xmlName string
	Items   []*EnumItem
}

type EnumItem struct {
	srcName string
	xmlName string
	Expr    Expression
}

func (enum *Enum) SrcName() string {
	return enum.srcName
}

func (enum *Enum) XmlName() string {
	return enum.xmlName
}

func (enum *Enum) Size() Size {
	panic("Cannot take size of enum")
}

func (enum *Enum) Initialize(p *Protocol) {
	enum.srcName = TypeSrcName(p, enum)
	for _, item := range enum.Items {
		item.srcName = SrcName(p, item.xmlName)
		if item.Expr != nil {
			item.Expr.Initialize(p)
		}
	}
}

type Resource struct {
	srcName string
	xmlName string
}

func (r *Resource) SrcName() string {
	return r.srcName
}

func (r *Resource) XmlName() string {
	return r.xmlName
}

func (r *Resource) Size() Size {
	return newFixedSize(BaseTypeSizes["Id"], true)
}

func (r *Resource) Initialize(p *Protocol) {
	r.srcName = TypeSrcName(p, r)
}

type TypeDef struct {
	srcName string
	xmlName string
	Old     Type
}

func (t *TypeDef) SrcName() string {
	return t.srcName
}

func (t *TypeDef) XmlName() string {
	return t.xmlName
}

func (t *TypeDef) Size() Size {
	return t.Old.Size()
}

func (t *TypeDef) Initialize(p *Protocol) {
	t.Old = t.Old.(*Translation).RealType(p)
	t.srcName = TypeSrcName(p, t)
}

type Event struct {
	srcName    string
	xmlName    string
	Number     int
	NoSequence bool
	Fields     []Field
}

func (e *Event) SrcName() string {
	return e.srcName
}

func (e *Event) XmlName() string {
	return e.xmlName
}

func (e *Event) Size() Size {
	return newExpressionSize(&Value{v: 32}, true)
}

func (e *Event) Initialize(p *Protocol) {
	e.srcName = TypeSrcName(p, e)
	for _, field := range e.Fields {
		field.Initialize(p)
	}
}

func (e *Event) EvType() string {
	return fmt.Sprintf("%sEvent", e.srcName)
}

type EventCopy struct {
	srcName string
	xmlName string
	Old     Type
	Number  int
}

func (e *EventCopy) SrcName() string {
	return e.srcName
}

func (e *EventCopy) XmlName() string {
	return e.xmlName
}

func (e *EventCopy) Size() Size {
	return newExpressionSize(&Value{v: 32}, true)
}

func (e *EventCopy) Initialize(p *Protocol) {
	e.srcName = TypeSrcName(p, e)
	e.Old = e.Old.(*Translation).RealType(p)
	if _, ok := e.Old.(*Event); !ok {
		panic("an EventCopy's old type *must* be *Event")
	}
}

func (e *EventCopy) EvType() string {
	return fmt.Sprintf("%sEvent", e.srcName)
}

type Error struct {
	srcName string
	xmlName string
	Number  int
	Fields  []Field
}

func (e *Error) SrcName() string {
	return e.srcName
}

func (e *Error) XmlName() string {
	return e.xmlName
}

func (e *Error) Size() Size {
	return newExpressionSize(&Value{v: 32}, true)
}

func (e *Error) Initialize(p *Protocol) {
	e.srcName = TypeSrcName(p, e)
	for _, field := range e.Fields {
		field.Initialize(p)
	}
}

func (e *Error) ErrConst() string {
	return fmt.Sprintf("Bad%s", e.srcName)
}

func (e *Error) ErrType() string {
	return fmt.Sprintf("%sError", e.srcName)
}

type ErrorCopy struct {
	srcName string
	xmlName string
	Old     Type
	Number  int
}

func (e *ErrorCopy) SrcName() string {
	return e.srcName
}

func (e *ErrorCopy) XmlName() string {
	return e.xmlName
}

func (e *ErrorCopy) Size() Size {
	return newExpressionSize(&Value{v: 32}, true)
}

func (e *ErrorCopy) Initialize(p *Protocol) {
	e.srcName = TypeSrcName(p, e)
	e.Old = e.Old.(*Translation).RealType(p)
	if _, ok := e.Old.(*Error); !ok {
		panic("an ErrorCopy's old type *must* be *Event")
	}
}

func (e *ErrorCopy) ErrConst() string {
	return fmt.Sprintf("Bad%s", e.srcName)
}

func (e *ErrorCopy) ErrType() string {
	return fmt.Sprintf("%sError", e.srcName)
}

type Struct struct {
	srcName string
	xmlName string
	Fields  []Field
}

func (s *Struct) SrcName() string {
	return s.srcName
}

func (s *Struct) XmlName() string {
	return s.xmlName
}

func (s *Struct) Size() Size {
	size := newFixedSize(0, true)
	for _, field := range s.Fields {
		size = size.Add(field.Size())
	}
	return size
}

func (s *Struct) Initialize(p *Protocol) {
	s.srcName = TypeSrcName(p, s)
	for _, field := range s.Fields {
		field.Initialize(p)
	}
}

// HasList returns whether there is a field in this struct that is a list.
// When true, a more involved calculation is necessary to compute this struct's
// size.
func (s *Struct) HasList() bool {
	for _, field := range s.Fields {
		if _, ok := field.(*ListField); ok {
			return true
		}
	}
	return false
}

type Union struct {
	srcName string
	xmlName string
	Fields  []Field
}

func (u *Union) SrcName() string {
	return u.srcName
}

func (u *Union) XmlName() string {
	return u.xmlName
}

// Size for Union is broken. At least, it's broken for XKB.
// It *looks* like the protocol inherently relies on some amount of
// memory unsafety, since some members of unions in XKB are *variable* in
// length! The only thing I can come up with, maybe, is when a union has
// variable size, simply return the raw bytes. Then it's up to the user to
// pass those raw bytes into the appropriate New* constructor. GROSS!
// As of now, just pluck out the first field and return that size. This
// should work for union elements in randr.xml and xproto.xml.
func (u *Union) Size() Size {
	return u.Fields[0].Size()
}

func (u *Union) Initialize(p *Protocol) {
	u.srcName = fmt.Sprintf("%sUnion", TypeSrcName(p, u))
	for _, field := range u.Fields {
		field.Initialize(p)
	}
}
