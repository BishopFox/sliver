package main

/*
	translation.go provides a 'Translate' method on every XML type that converts
	the XML type into our "better" representation.

	i.e., the representation of Fields and Expressions is just too general.
	We end up losing a lot of the advantages of static typing if we keep
	the types that encoding/xml forces us into.

	Please see 'representation.go' for the type definitions that we're
	translating to.
*/

import (
	"log"
	"strconv"
	"strings"
)

func (xml *XML) Translate(parent *Protocol) *Protocol {
	protocol := &Protocol{
		Parent:       parent,
		Name:         xml.Header,
		ExtXName:     xml.ExtensionXName,
		ExtName:      xml.ExtensionName,
		MajorVersion: xml.MajorVersion,
		MinorVersion: xml.MinorVersion,

		Imports:  make([]*Protocol, 0),
		Types:    make([]Type, 0),
		Requests: make([]*Request, len(xml.Requests)),
	}

	for _, imp := range xml.Imports {
		if imp.xml != nil {
			protocol.Imports = append(protocol.Imports,
				imp.xml.Translate(protocol))
		}
	}

	for xmlName, srcName := range BaseTypeMap {
		newBaseType := &Base{
			srcName: srcName,
			xmlName: xmlName,
			size:    newFixedSize(BaseTypeSizes[xmlName], true),
		}
		protocol.Types = append(protocol.Types, newBaseType)
	}
	for _, enum := range xml.Enums {
		protocol.Types = append(protocol.Types, enum.Translate())
	}
	for _, xid := range xml.Xids {
		protocol.Types = append(protocol.Types, xid.Translate())
	}
	for _, xidunion := range xml.XidUnions {
		protocol.Types = append(protocol.Types, xidunion.Translate())
	}
	for _, typedef := range xml.TypeDefs {
		protocol.Types = append(protocol.Types, typedef.Translate())
	}
	for _, s := range xml.Structs {
		protocol.Types = append(protocol.Types, s.Translate())
	}
	for _, union := range xml.Unions {
		protocol.Types = append(protocol.Types, union.Translate())
	}
	for _, ev := range xml.Events {
		protocol.Types = append(protocol.Types, ev.Translate())
	}
	for _, evcopy := range xml.EventCopies {
		protocol.Types = append(protocol.Types, evcopy.Translate())
	}
	for _, err := range xml.Errors {
		protocol.Types = append(protocol.Types, err.Translate())
	}
	for _, errcopy := range xml.ErrorCopies {
		protocol.Types = append(protocol.Types, errcopy.Translate())
	}

	for i, request := range xml.Requests {
		protocol.Requests[i] = request.Translate()
	}

	// Now load all of the type and source name information.
	protocol.Initialize()

	// Make sure all enums have concrete values.
	for _, typ := range protocol.Types {
		enum, ok := typ.(*Enum)
		if !ok {
			continue
		}
		nextValue := 0
		for _, item := range enum.Items {
			if item.Expr == nil {
				item.Expr = &Value{v: nextValue}
				nextValue++
			} else {
				nextValue = item.Expr.Eval() + 1
			}
		}
	}

	return protocol
}

func (x *XMLEnum) Translate() *Enum {
	enum := &Enum{
		xmlName: x.Name,
		Items:   make([]*EnumItem, len(x.Items)),
	}
	for i, item := range x.Items {
		enum.Items[i] = &EnumItem{
			xmlName: item.Name,
			Expr:    item.Expr.Translate(),
		}
	}
	return enum
}

func (x *XMLXid) Translate() *Resource {
	return &Resource{
		xmlName: x.Name,
	}
}

func (x *XMLTypeDef) Translate() *TypeDef {
	return &TypeDef{
		xmlName: x.New,
		Old:     newTranslation(x.Old),
	}
}

func (x *XMLEvent) Translate() *Event {
	ev := &Event{
		xmlName:    x.Name,
		Number:     x.Number,
		NoSequence: x.NoSequence,
		Fields:     make([]Field, 0, len(x.Fields)),
	}
	for _, field := range x.Fields {
		if field.XMLName.Local == "doc" {
			continue
		}
		ev.Fields = append(ev.Fields, field.Translate(ev))
	}
	return ev
}

func (x *XMLEventCopy) Translate() *EventCopy {
	return &EventCopy{
		xmlName: x.Name,
		Number:  x.Number,
		Old:     newTranslation(x.Ref),
	}
}

func (x *XMLError) Translate() *Error {
	err := &Error{
		xmlName: x.Name,
		Number:  x.Number,
		Fields:  make([]Field, len(x.Fields)),
	}
	for i, field := range x.Fields {
		err.Fields[i] = field.Translate(err)
	}
	return err
}

func (x *XMLErrorCopy) Translate() *ErrorCopy {
	return &ErrorCopy{
		xmlName: x.Name,
		Number:  x.Number,
		Old:     newTranslation(x.Ref),
	}
}

func (x *XMLStruct) Translate() *Struct {
	s := &Struct{
		xmlName: x.Name,
		Fields:  make([]Field, len(x.Fields)),
	}
	for i, field := range x.Fields {
		s.Fields[i] = field.Translate(s)
	}
	return s
}

func (x *XMLUnion) Translate() *Union {
	u := &Union{
		xmlName: x.Name,
		Fields:  make([]Field, len(x.Fields)),
	}
	for i, field := range x.Fields {
		u.Fields[i] = field.Translate(u)
	}
	return u
}

func (x *XMLRequest) Translate() *Request {
	r := &Request{
		xmlName: x.Name,
		Opcode:  x.Opcode,
		Combine: x.Combine,
		Fields:  make([]Field, 0, len(x.Fields)),
		Reply:   x.Reply.Translate(),
	}
	for _, field := range x.Fields {
		if field.XMLName.Local == "doc" || field.XMLName.Local == "fd" {
			continue
		}
		r.Fields = append(r.Fields, field.Translate(r))
	}

	// Address bug (or legacy code) in QueryTextExtents.
	// The XML protocol description references 'string_len' in the
	// computation of the 'odd_length' field. However, 'string_len' is not
	// defined. Therefore, let's forcefully add it as a 'local field'.
	// (i.e., a parameter in the caller but does not get sent over the wire.)
	if x.Name == "QueryTextExtents" {
		stringLenLocal := &LocalField{&SingleField{
			xmlName: "string_len",
			Type:    newTranslation("CARD16"),
		}}
		r.Fields = append(r.Fields, stringLenLocal)
	}

	return r
}

func (x *XMLReply) Translate() *Reply {
	if x == nil {
		return nil
	}

	r := &Reply{
		Fields: make([]Field, 0, len(x.Fields)),
	}
	for _, field := range x.Fields {
		if field.XMLName.Local == "doc" || field.XMLName.Local == "fd" {
			continue
		}
		r.Fields = append(r.Fields, field.Translate(r))
	}
	return r
}

func (x *XMLExpression) Translate() Expression {
	if x == nil {
		return nil
	}

	switch x.XMLName.Local {
	case "op":
		if len(x.Exprs) != 2 {
			log.Panicf("'op' found %d expressions; expected 2.", len(x.Exprs))
		}
		return &BinaryOp{
			Op:    x.Op,
			Expr1: x.Exprs[0].Translate(),
			Expr2: x.Exprs[1].Translate(),
		}
	case "unop":
		if len(x.Exprs) != 1 {
			log.Panicf("'unop' found %d expressions; expected 1.", len(x.Exprs))
		}
		return &UnaryOp{
			Op:   x.Op,
			Expr: x.Exprs[0].Translate(),
		}
	case "popcount":
		if len(x.Exprs) != 1 {
			log.Panicf("'popcount' found %d expressions; expected 1.",
				len(x.Exprs))
		}
		return &PopCount{
			Expr: x.Exprs[0].Translate(),
		}
	case "value":
		val, err := strconv.Atoi(strings.TrimSpace(x.Data))
		if err != nil {
			log.Panicf("Could not convert '%s' in 'value' expression to int.",
				x.Data)
		}
		return &Value{
			v: val,
		}
	case "bit":
		bit, err := strconv.Atoi(strings.TrimSpace(x.Data))
		if err != nil {
			log.Panicf("Could not convert '%s' in 'bit' expression to int.",
				x.Data)
		}
		if bit < 0 || bit > 31 {
			log.Panicf("A 'bit' literal must be in the range [0, 31], but "+
				" is %d", bit)
		}
		return &Bit{
			b: bit,
		}
	case "fieldref":
		return &FieldRef{
			Name: x.Data,
		}
	case "enumref":
		return &EnumRef{
			EnumKind: newTranslation(x.Ref),
			EnumItem: x.Data,
		}
	case "sumof":
		return &SumOf{
			Name: x.Ref,
		}
	}

	log.Panicf("Unrecognized tag '%s' in expression context. Expected one of "+
		"op, fieldref, value, bit, enumref, unop, sumof or popcount.",
		x.XMLName.Local)
	panic("unreachable")
}

func (x *XMLField) Translate(parent interface{}) Field {
	switch x.XMLName.Local {
	case "pad":
		return &PadField{
			Bytes: x.Bytes,
		}
	case "field":
		return &SingleField{
			xmlName: x.Name,
			Type:    newTranslation(x.Type),
		}
	case "list":
		return &ListField{
			xmlName:    x.Name,
			Type:       newTranslation(x.Type),
			LengthExpr: x.Expr.Translate(),
		}
	case "localfield":
		return &LocalField{&SingleField{
			xmlName: x.Name,
			Type:    newTranslation(x.Type),
		}}
	case "exprfield":
		return &ExprField{
			xmlName: x.Name,
			Type:    newTranslation(x.Type),
			Expr:    x.Expr.Translate(),
		}
	case "valueparam":
		return &ValueField{
			Parent:   parent,
			MaskType: newTranslation(x.ValueMaskType),
			MaskName: x.ValueMaskName,
			ListName: x.ValueListName,
		}
	case "switch":
		swtch := &SwitchField{
			Name:     x.Name,
			Expr:     x.Expr.Translate(),
			Bitcases: make([]*Bitcase, len(x.Bitcases)),
		}
		for i, bitcase := range x.Bitcases {
			swtch.Bitcases[i] = bitcase.Translate()
		}
		return swtch
	}

	log.Panicf("Unrecognized field element: %s", x.XMLName.Local)
	panic("unreachable")
}

func (x *XMLBitcase) Translate() *Bitcase {
	b := &Bitcase{
		Expr:   x.Expr().Translate(),
		Fields: make([]Field, len(x.Fields)),
	}
	for i, field := range x.Fields {
		b.Fields[i] = field.Translate(b)
	}
	return b
}

// SrcName is used to translate any identifier into a Go name.
// Mostly used for fields, but used in a couple other places too (enum items).
func SrcName(p *Protocol, name string) string {
	// If it's in the name map, use that translation.
	if newn, ok := NameMap[name]; ok {
		return newn
	}
	return splitAndTitle(name)
}

func TypeSrcName(p *Protocol, typ Type) string {
	t := typ.XmlName()

	// If this is a base type, then write the raw Go type.
	if baseType, ok := typ.(*Base); ok {
		return baseType.SrcName()
	}

	// If it's in the type map, use that translation.
	if newt, ok := TypeMap[t]; ok {
		return newt
	}

	// If there's a namespace to this type, just use it and be done.
	if colon := strings.Index(t, ":"); colon > -1 {
		namespace := t[:colon]
		rest := t[colon+1:]
		return p.ProtocolFind(namespace).PkgName() + "." + splitAndTitle(rest)
	}

	// Since there's no namespace, we're left with the raw type name.
	// If the type is part of the source we're generating (i.e., there is
	// no parent protocol), then just return that type name.
	// Otherwise, we must qualify it with a package name.
	if p.Parent == nil {
		return splitAndTitle(t)
	}
	return p.PkgName() + "." + splitAndTitle(t)
}
