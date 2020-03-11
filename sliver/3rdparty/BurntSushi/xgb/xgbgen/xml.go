package main

import (
	"encoding/xml"
	"io/ioutil"
	"log"
)

type XML struct {
	// Root 'xcb' element properties.
	XMLName        xml.Name `xml:"xcb"`
	Header         string   `xml:"header,attr"`
	ExtensionXName string   `xml:"extension-xname,attr"`
	ExtensionName  string   `xml:"extension-name,attr"`
	MajorVersion   string   `xml:"major-version,attr"`
	MinorVersion   string   `xml:"minor-version,attr"`

	// Types for all top-level elements.
	// First are the simple ones.
	Imports     XMLImports      `xml:"import"`
	Enums       []*XMLEnum      `xml:"enum"`
	Xids        []*XMLXid       `xml:"xidtype"`
	XidUnions   []*XMLXid       `xml:"xidunion"`
	TypeDefs    []*XMLTypeDef   `xml:"typedef"`
	EventCopies []*XMLEventCopy `xml:"eventcopy"`
	ErrorCopies []*XMLErrorCopy `xml:"errorcopy"`

	// Here are the complex ones, i.e., anything with "structure contents"
	Structs  []*XMLStruct  `xml:"struct"`
	Unions   []*XMLUnion   `xml:"union"`
	Requests []*XMLRequest `xml:"request"`
	Events   []*XMLEvent   `xml:"event"`
	Errors   []*XMLError   `xml:"error"`
}

type XMLImports []*XMLImport

func (imports XMLImports) Eval() {
	for _, imp := range imports {
		xmlBytes, err := ioutil.ReadFile(*protoPath + "/" + imp.Name + ".xml")
		if err != nil {
			log.Fatalf("Could not read X protocol description for import "+
				"'%s' because: %s", imp.Name, err)
		}

		imp.xml = &XML{}
		err = xml.Unmarshal(xmlBytes, imp.xml)
		if err != nil {
			log.Fatal("Could not parse X protocol description for import "+
				"'%s' because: %s", imp.Name, err)
		}

		// recursive imports...
		imp.xml.Imports.Eval()
	}
}

type XMLImport struct {
	Name string `xml:",chardata"`
	xml  *XML   `xml:"-"`
}

type XMLEnum struct {
	Name  string         `xml:"name,attr"`
	Items []*XMLEnumItem `xml:"item"`
}

type XMLEnumItem struct {
	Name string         `xml:"name,attr"`
	Expr *XMLExpression `xml:",any"`
}

type XMLXid struct {
	XMLName xml.Name
	Name    string `xml:"name,attr"`
}

type XMLTypeDef struct {
	Old string `xml:"oldname,attr"`
	New string `xml:"newname,attr"`
}

type XMLEventCopy struct {
	Name   string `xml:"name,attr"`
	Number int    `xml:"number,attr"`
	Ref    string `xml:"ref,attr"`
}

type XMLErrorCopy struct {
	Name   string `xml:"name,attr"`
	Number int    `xml:"number,attr"`
	Ref    string `xml:"ref,attr"`
}

type XMLStruct struct {
	Name   string      `xml:"name,attr"`
	Fields []*XMLField `xml:",any"`
}

type XMLUnion struct {
	Name   string      `xml:"name,attr"`
	Fields []*XMLField `xml:",any"`
}

type XMLRequest struct {
	Name    string      `xml:"name,attr"`
	Opcode  int         `xml:"opcode,attr"`
	Combine bool        `xml:"combine-adjacent,attr"`
	Fields  []*XMLField `xml:",any"`
	Reply   *XMLReply   `xml:"reply"`
}

type XMLReply struct {
	Fields []*XMLField `xml:",any"`
}

type XMLEvent struct {
	Name       string      `xml:"name,attr"`
	Number     int         `xml:"number,attr"`
	NoSequence bool        `xml:"no-sequence-number,attr"`
	Fields     []*XMLField `xml:",any"`
}

type XMLError struct {
	Name   string      `xml:"name,attr"`
	Number int         `xml:"number,attr"`
	Fields []*XMLField `xml:",any"`
}

type XMLExpression struct {
	XMLName xml.Name

	Exprs []*XMLExpression `xml:",any"`

	Data string `xml:",chardata"`
	Op   string `xml:"op,attr"`
	Ref  string `xml:"ref,attr"`
}
