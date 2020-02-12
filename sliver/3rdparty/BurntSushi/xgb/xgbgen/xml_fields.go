package main

import (
	"encoding/xml"
	"log"
)

type XMLField struct {
	XMLName xml.Name

	// For 'pad' element
	Bytes uint `xml:"bytes,attr"`

	// For 'field', 'list', 'localfield', 'exprfield' and 'switch' elements.
	Name string `xml:"name,attr"`

	// For 'field', 'list', 'localfield', and 'exprfield' elements.
	Type string `xml:"type,attr"`

	// For 'list', 'exprfield' and 'switch' elements.
	Expr *XMLExpression `xml:",any"`

	// For 'valueparm' element.
	ValueMaskType string `xml:"value-mask-type,attr"`
	ValueMaskName string `xml:"value-mask-name,attr"`
	ValueListName string `xml:"value-list-name,attr"`

	// For 'switch' element.
	Bitcases []*XMLBitcase `xml:"bitcase"`

	// I don't know which elements these are for. The documentation is vague.
	// They also seem to be completely optional.
	OptEnum    string `xml:"enum,attr"`
	OptMask    string `xml:"mask,attr"`
	OptAltEnum string `xml:"altenum,attr"`
}

// Bitcase represents a single expression followed by any number of fields.
// Namely, if the switch's expression (all bitcases are inside a switch),
// and'd with the bitcase's expression is equal to the bitcase expression,
// then the fields should be included in its parent structure.
// Note that since a bitcase is unique in that expressions and fields are
// siblings, we must exhaustively search for one of them. Essentially,
// it's the closest thing to a Union I can get to in Go without interfaces.
// Would an '<expression>' tag have been too much to ask? :-(
type XMLBitcase struct {
	Fields []*XMLField `xml:",any"`

	// All the different expressions.
	// When it comes time to choose one, use the 'Expr' method.
	ExprOp    *XMLExpression `xml:"op"`
	ExprUnOp  *XMLExpression `xml:"unop"`
	ExprField *XMLExpression `xml:"fieldref"`
	ExprValue *XMLExpression `xml:"value"`
	ExprBit   *XMLExpression `xml:"bit"`
	ExprEnum  *XMLExpression `xml:"enumref"`
	ExprSum   *XMLExpression `xml:"sumof"`
	ExprPop   *XMLExpression `xml:"popcount"`
}

// Expr chooses the only non-nil Expr* field from Bitcase.
// Panic if there is more than one non-nil expression.
func (b *XMLBitcase) Expr() *XMLExpression {
	choices := []*XMLExpression{
		b.ExprOp, b.ExprUnOp, b.ExprField, b.ExprValue,
		b.ExprBit, b.ExprEnum, b.ExprSum, b.ExprPop,
	}

	var choice *XMLExpression = nil
	numNonNil := 0
	for _, c := range choices {
		if c != nil {
			numNonNil++
			choice = c
		}
	}

	if choice == nil {
		log.Panicf("No top level expression found in a bitcase.")
	}
	if numNonNil > 1 {
		log.Panicf("More than one top-level expression was found in a bitcase.")
	}
	return choice
}
