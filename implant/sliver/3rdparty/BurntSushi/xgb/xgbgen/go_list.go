package main

import (
	"fmt"
	"log"
	"strings"
)

// List fields
func (f *ListField) Define(c *Context) {
	c.Putln("%s %s // size: %s",
		f.SrcName(), f.SrcType(), f.Size())
}

func (f *ListField) Read(c *Context, prefix string) {
	switch t := f.Type.(type) {
	case *Resource:
		length := f.LengthExpr.Reduce(prefix)
		c.Putln("%s%s = make([]%s, %s)",
			prefix, f.SrcName(), t.SrcName(), length)
		c.Putln("for i := 0; i < int(%s); i++ {", length)
		ReadSimpleSingleField(c, fmt.Sprintf("%s%s[i]", prefix, f.SrcName()), t)
		c.Putln("}")
	case *Base:
		length := f.LengthExpr.Reduce(prefix)
		if strings.ToLower(t.XmlName()) == "char" {
			c.Putln("{")
			c.Putln("byteString := make([]%s, %s)", t.SrcName(), length)
			c.Putln("copy(byteString[:%s], buf[b:])", length)
			c.Putln("%s%s = string(byteString)", prefix, f.SrcName())
			// This is apparently a special case. The "Str" type itself
			// doesn't specify any padding. I suppose it's up to the
			// request/reply spec that uses it to get the padding right?
			c.Putln("b += int(%s)", length)
			c.Putln("}")
		} else if t.SrcName() == "byte" {
			c.Putln("%s%s = make([]%s, %s)",
				prefix, f.SrcName(), t.SrcName(), length)
			c.Putln("copy(%s%s[:%s], buf[b:])", prefix, f.SrcName(), length)
			c.Putln("b += int(%s)", length)
		} else {
			c.Putln("%s%s = make([]%s, %s)",
				prefix, f.SrcName(), t.SrcName(), length)
			c.Putln("for i := 0; i < int(%s); i++ {", length)
			ReadSimpleSingleField(c,
				fmt.Sprintf("%s%s[i]", prefix, f.SrcName()), t)
			c.Putln("}")
		}
	case *TypeDef:
		length := f.LengthExpr.Reduce(prefix)
		c.Putln("%s%s = make([]%s, %s)",
			prefix, f.SrcName(), t.SrcName(), length)
		c.Putln("for i := 0; i < int(%s); i++ {", length)
		ReadSimpleSingleField(c, fmt.Sprintf("%s%s[i]", prefix, f.SrcName()), t)
		c.Putln("}")
	case *Union:
		c.Putln("%s%s = make([]%s, %s)",
			prefix, f.SrcName(), t.SrcName(), f.LengthExpr.Reduce(prefix))
		c.Putln("b += %sReadList(buf[b:], %s%s)",
			t.SrcName(), prefix, f.SrcName())
	case *Struct:
		c.Putln("%s%s = make([]%s, %s)",
			prefix, f.SrcName(), t.SrcName(), f.LengthExpr.Reduce(prefix))
		c.Putln("b += %sReadList(buf[b:], %s%s)",
			t.SrcName(), prefix, f.SrcName())
	default:
		log.Panicf("Cannot read list field '%s' with %T type.",
			f.XmlName(), f.Type)
	}
}

func (f *ListField) Write(c *Context, prefix string) {
	switch t := f.Type.(type) {
	case *Resource:
		length := f.Length().Reduce(prefix)
		c.Putln("for i := 0; i < int(%s); i++ {", length)
		WriteSimpleSingleField(c,
			fmt.Sprintf("%s%s[i]", prefix, f.SrcName()), t)
		c.Putln("}")
	case *Base:
		length := f.Length().Reduce(prefix)
		if t.SrcName() == "byte" {
			c.Putln("copy(buf[b:], %s%s[:%s])", prefix, f.SrcName(), length)
			c.Putln("b += int(%s)", length)
		} else {
			c.Putln("for i := 0; i < int(%s); i++ {", length)
			WriteSimpleSingleField(c,
				fmt.Sprintf("%s%s[i]", prefix, f.SrcName()), t)
			c.Putln("}")
		}
	case *TypeDef:
		length := f.Length().Reduce(prefix)
		c.Putln("for i := 0; i < int(%s); i++ {", length)
		WriteSimpleSingleField(c,
			fmt.Sprintf("%s%s[i]", prefix, f.SrcName()), t)
		c.Putln("}")
	case *Union:
		c.Putln("b += %sListBytes(buf[b:], %s%s)",
			t.SrcName(), prefix, f.SrcName())
	case *Struct:
		c.Putln("b += %sListBytes(buf[b:], %s%s)",
			t.SrcName(), prefix, f.SrcName())
	default:
		log.Panicf("Cannot write list field '%s' with %T type.",
			f.XmlName(), f.Type)
	}
}
