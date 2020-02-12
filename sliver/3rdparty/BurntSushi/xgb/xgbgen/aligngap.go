package main

import (
	"fmt"
	"os"
)

func (p *Protocol) AddAlignGaps() {
	for i := range p.Imports {
		p.Imports[i].AddAlignGaps()
	}
	for i := range p.Types {
		switch t := p.Types[i].(type) {
		case *Struct:
			t.Fields = addAlignGapsToFields(t.xmlName, t.Fields)
		case *Event:
			t.Fields = addAlignGapsToFields(t.xmlName, t.Fields)
		case *Error:
			t.Fields = addAlignGapsToFields(t.xmlName, t.Fields)
		}
	}
	for i := range p.Requests {
		p.Requests[i].Fields = addAlignGapsToFields(
			p.Requests[i].xmlName, p.Requests[i].Fields)
		if p.Requests[i].Reply != nil {
			p.Requests[i].Reply.Fields = addAlignGapsToFields(
				p.Requests[i].xmlName, p.Requests[i].Reply.Fields)
		}
	}
}

func addAlignGapsToFields(name string, fields []Field) []Field {
	var i int
	for i = 0; i < len(fields); i++ {
		if _, ok := fields[i].(*ListField); ok {
			break
		}
	}
	if i >= len(fields) {
		return fields
	}

	r := make([]Field, 0, len(fields)+2)
	r = append(r, fields[:i]...)

	r = append(r, fields[i])
	for i = i + 1; i < len(fields); i++ {
		switch f := fields[i].(type) {
		case *ListField:
			// ok, add padding
			sz := xcbSizeOfType(f.Type)
			switch {
			case sz == 1:
				// nothing
			case sz == 2:
				r = append(r, &PadField{0, 2})
			case sz == 3:
				panic(fmt.Errorf("Alignment is not a power of 2"))
			case sz >= 4:
				r = append(r, &PadField{0, 4})
			}
		case *LocalField:
			// nothing
		default:
			fmt.Fprintf(os.Stderr,
				"Can't add alignment gaps, mix of list and non-list "+
					"fields: %s\n", name)
			return fields
		}
		r = append(r, fields[i])
	}
	return r
}

func xcbSizeOfField(fld Field) int {
	switch f := fld.(type) {
	case *PadField:
		return int(f.Bytes)
	case *SingleField:
		return xcbSizeOfType(f.Type)
	case *ListField:
		return 0
	case *ExprField:
		return xcbSizeOfType(f.Type)
	case *ValueField:
		return xcbSizeOfType(f.MaskType)
	case *SwitchField:
		return 0
	default:
		return 0
	}
}

func xcbSizeOfType(typ Type) int {
	switch t := typ.(type) {
	case *Resource:
		return 4
	case *TypeDef:
		return t.Size().Eval()
	case *Base:
		return t.Size().Eval()
	case *Struct:
		sz := 0
		for i := range t.Fields {
			sz += xcbSizeOfField(t.Fields[i])
		}
		return sz
	case *Union:
		sz := 0
		for i := range t.Fields {
			csz := xcbSizeOfField(t.Fields[i])
			if csz > sz {
				sz = csz
			}
		}
		return sz
	default:
		return 0
	}
}
