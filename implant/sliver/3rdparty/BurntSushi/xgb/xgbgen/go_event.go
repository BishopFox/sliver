package main

import (
	"fmt"
)

// Event types
func (e *Event) Define(c *Context) {
	c.Putln("// %s is the event number for a %s.", e.SrcName(), e.EvType())
	c.Putln("const %s = %d", e.SrcName(), e.Number)
	c.Putln("")
	c.Putln("type %s struct {", e.EvType())
	if !e.NoSequence {
		c.Putln("Sequence uint16")
	}
	for _, field := range e.Fields {
		field.Define(c)
	}
	c.Putln("}")
	c.Putln("")

	// Read defines a function that transforms a byte slice into this
	// event struct.
	e.Read(c)

	// Write defines a function that transforms this event struct into
	// a byte slice.
	e.Write(c)

	// Makes sure that this event type is an Event interface.
	c.Putln("// SequenceId returns the sequence id attached to the %s event.",
		e.SrcName())
	c.Putln("// Events without a sequence number (KeymapNotify) return 0.")
	c.Putln("// This is mostly used internally.")
	c.Putln("func (v %s) SequenceId() uint16 {", e.EvType())
	if e.NoSequence {
		c.Putln("return uint16(0)")
	} else {
		c.Putln("return v.Sequence")
	}
	c.Putln("}")
	c.Putln("")
	c.Putln("// String is a rudimentary string representation of %s.",
		e.EvType())
	c.Putln("func (v %s) String() string {", e.EvType())
	EventFieldString(c, e.Fields, e.SrcName())
	c.Putln("}")
	c.Putln("")

	// Let's the XGB event loop read this event.
	c.Putln("func init() {")
	if c.protocol.isExt() {
		c.Putln("xgb.NewExtEventFuncs[\"%s\"][%d] = %sNew",
			c.protocol.ExtXName, e.Number, e.EvType())
	} else {
		c.Putln("xgb.NewEventFuncs[%d] = %sNew", e.Number, e.EvType())
	}
	c.Putln("}")
	c.Putln("")
}

func (e *Event) Read(c *Context) {
	c.Putln("// %sNew constructs a %s value that implements xgb.Event from "+
		"a byte slice.", e.EvType(), e.EvType())
	c.Putln("func %sNew(buf []byte) xgb.Event {", e.EvType())
	c.Putln("v := %s{}", e.EvType())
	c.Putln("b := 1 // don't read event number")
	c.Putln("")
	for i, field := range e.Fields {
		if i == 1 && !e.NoSequence {
			c.Putln("v.Sequence = xgb.Get16(buf[b:])")
			c.Putln("b += 2")
			c.Putln("")
		}
		field.Read(c, "v.")
		c.Putln("")
	}
	c.Putln("return v")
	c.Putln("}")
	c.Putln("")
}

func (e *Event) Write(c *Context) {
	c.Putln("// Bytes writes a %s value to a byte slice.", e.EvType())
	c.Putln("func (v %s) Bytes() []byte {", e.EvType())
	c.Putln("buf := make([]byte, %s)", e.Size())
	c.Putln("b := 0")
	c.Putln("")
	c.Putln("// write event number")
	c.Putln("buf[b] = %d", e.Number)
	c.Putln("b += 1")
	c.Putln("")
	for i, field := range e.Fields {
		if i == 1 && !e.NoSequence {
			c.Putln("b += 2 // skip sequence number")
			c.Putln("")
		}
		field.Write(c, "v.")
		c.Putln("")
	}
	c.Putln("return buf")
	c.Putln("}")
	c.Putln("")
}

// EventCopy types
func (e *EventCopy) Define(c *Context) {
	c.Putln("// %s is the event number for a %s.", e.SrcName(), e.EvType())
	c.Putln("const %s = %d", e.SrcName(), e.Number)
	c.Putln("")
	c.Putln("type %s %s", e.EvType(), e.Old.(*Event).EvType())
	c.Putln("")

	// Read defines a function that transforms a byte slice into this
	// event struct.
	e.Read(c)

	// Write defines a function that transoforms this event struct into
	// a byte slice.
	e.Write(c)

	// Makes sure that this event type is an Event interface.
	c.Putln("// SequenceId returns the sequence id attached to the %s event.",
		e.SrcName())
	c.Putln("// Events without a sequence number (KeymapNotify) return 0.")
	c.Putln("// This is mostly used internally.")
	c.Putln("func (v %s) SequenceId() uint16 {", e.EvType())
	if e.Old.(*Event).NoSequence {
		c.Putln("return uint16(0)")
	} else {
		c.Putln("return v.Sequence")
	}
	c.Putln("}")
	c.Putln("")
	c.Putln("func (v %s) String() string {", e.EvType())
	EventFieldString(c, e.Old.(*Event).Fields, e.SrcName())
	c.Putln("}")
	c.Putln("")

	// Let's the XGB event loop read this event.
	c.Putln("func init() {")
	if c.protocol.isExt() {
		c.Putln("xgb.NewExtEventFuncs[\"%s\"][%d] = %sNew",
			c.protocol.ExtXName, e.Number, e.EvType())
	} else {
		c.Putln("xgb.NewEventFuncs[%d] = %sNew", e.Number, e.EvType())
	}
	c.Putln("}")
	c.Putln("")
}

func (e *EventCopy) Read(c *Context) {
	c.Putln("// %sNew constructs a %s value that implements xgb.Event from "+
		"a byte slice.", e.EvType(), e.EvType())
	c.Putln("func %sNew(buf []byte) xgb.Event {", e.EvType())
	c.Putln("return %s(%sNew(buf).(%s))",
		e.EvType(), e.Old.(*Event).EvType(), e.Old.(*Event).EvType())
	c.Putln("}")
	c.Putln("")
}

func (e *EventCopy) Write(c *Context) {
	c.Putln("// Bytes writes a %s value to a byte slice.", e.EvType())
	c.Putln("func (v %s) Bytes() []byte {", e.EvType())
	c.Putln("return %s(v).Bytes()", e.Old.(*Event).EvType())
	c.Putln("}")
	c.Putln("")
}

// EventFieldString works for both Event and EventCopy. It assembles all of the
// fields in an event and formats them into a single string.
func EventFieldString(c *Context, fields []Field, evName string) {
	c.Putln("fieldVals := make([]string, 0, %d)", len(fields))
	if evName != "KeymapNotify" {
		c.Putln("fieldVals = append(fieldVals, "+
			"xgb.Sprintf(\"Sequence: %s\", v.Sequence))", "%d")
	}
	for _, field := range fields {
		switch f := field.(type) {
		case *PadField:
			continue
		case *SingleField:
			switch f.Type.(type) {
			case *Base:
			case *Resource:
			case *TypeDef:
			default:
				continue
			}

			switch field.SrcType() {
			case "string":
				format := fmt.Sprintf("xgb.Sprintf(\"%s: %s\", v.%s)",
					field.SrcName(), "%s", field.SrcName())
				c.Putln("fieldVals = append(fieldVals, %s)", format)
			case "bool":
				format := fmt.Sprintf("xgb.Sprintf(\"%s: %s\", v.%s)",
					field.SrcName(), "%t", field.SrcName())
				c.Putln("fieldVals = append(fieldVals, %s)", format)
			default:
				format := fmt.Sprintf("xgb.Sprintf(\"%s: %s\", v.%s)",
					field.SrcName(), "%d", field.SrcName())
				c.Putln("fieldVals = append(fieldVals, %s)", format)
			}
		}
	}
	c.Putln("return \"%s {\" + xgb.StringsJoin(fieldVals, \", \") + \"}\"",
		evName)
}
