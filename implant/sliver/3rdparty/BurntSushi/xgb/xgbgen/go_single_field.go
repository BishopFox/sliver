package main

import (
	"fmt"
	"log"
)

func (f *SingleField) Define(c *Context) {
	c.Putln("%s %s", f.SrcName(), f.Type.SrcName())
}

func ReadSimpleSingleField(c *Context, name string, typ Type) {
	switch t := typ.(type) {
	case *Resource:
		c.Putln("%s = %s(xgb.Get32(buf[b:]))", name, t.SrcName())
	case *TypeDef:
		switch t.Size().Eval() {
		case 1:
			c.Putln("%s = %s(buf[b])", name, t.SrcName())
		case 2:
			c.Putln("%s = %s(xgb.Get16(buf[b:]))", name, t.SrcName())
		case 4:
			c.Putln("%s = %s(xgb.Get32(buf[b:]))", name, t.SrcName())
		case 8:
			c.Putln("%s = %s(xgb.Get64(buf[b:]))", name, t.SrcName())
		}
	case *Base:
		// If this is a bool, stop short and do something special.
		if t.SrcName() == "bool" {
			c.Putln("if buf[b] == 1 {")
			c.Putln("%s = true", name)
			c.Putln("} else {")
			c.Putln("%s = false", name)
			c.Putln("}")
			break
		}

		var val string
		switch t.Size().Eval() {
		case 1:
			val = fmt.Sprintf("buf[b]")
		case 2:
			val = fmt.Sprintf("xgb.Get16(buf[b:])")
		case 4:
			val = fmt.Sprintf("xgb.Get32(buf[b:])")
		case 8:
			val = fmt.Sprintf("xgb.Get64(buf[b:])")
		}

		// We need to convert base types if they aren't uintXX or byte
		ty := t.SrcName()
		if ty != "byte" && ty != "uint16" && ty != "uint32" && ty != "uint64" {
			val = fmt.Sprintf("%s(%s)", ty, val)
		}
		c.Putln("%s = %s", name, val)
	default:
		log.Panicf("Cannot read field '%s' as a simple field with %T type.",
			name, typ)
	}

	c.Putln("b += %s", typ.Size())
}

func (f *SingleField) Read(c *Context, prefix string) {
	switch t := f.Type.(type) {
	case *Resource:
		ReadSimpleSingleField(c, fmt.Sprintf("%s%s", prefix, f.SrcName()), t)
	case *TypeDef:
		ReadSimpleSingleField(c, fmt.Sprintf("%s%s", prefix, f.SrcName()), t)
	case *Base:
		ReadSimpleSingleField(c, fmt.Sprintf("%s%s", prefix, f.SrcName()), t)
	case *Struct:
		c.Putln("%s%s = %s{}", prefix, f.SrcName(), t.SrcName())
		c.Putln("b += %sRead(buf[b:], &%s%s)", t.SrcName(), prefix, f.SrcName())
	case *Union:
		c.Putln("%s%s = %s{}", prefix, f.SrcName(), t.SrcName())
		c.Putln("b += %sRead(buf[b:], &%s%s)", t.SrcName(), prefix, f.SrcName())
	default:
		log.Panicf("Cannot read field '%s' with %T type.", f.XmlName(), f.Type)
	}
}

func WriteSimpleSingleField(c *Context, name string, typ Type) {
	switch t := typ.(type) {
	case *Resource:
		c.Putln("xgb.Put32(buf[b:], uint32(%s))", name)
	case *TypeDef:
		switch t.Size().Eval() {
		case 1:
			c.Putln("buf[b] = byte(%s)", name)
		case 2:
			c.Putln("xgb.Put16(buf[b:], uint16(%s))", name)
		case 4:
			c.Putln("xgb.Put32(buf[b:], uint32(%s))", name)
		case 8:
			c.Putln("xgb.Put64(buf[b:], uint64(%s))", name)
		}
	case *Base:
		// If this is a bool, stop short and do something special.
		if t.SrcName() == "bool" {
			c.Putln("if %s {", name)
			c.Putln("buf[b] = 1")
			c.Putln("} else {")
			c.Putln("buf[b] = 0")
			c.Putln("}")
			break
		}

		switch t.Size().Eval() {
		case 1:
			if t.SrcName() != "byte" {
				c.Putln("buf[b] = byte(%s)", name)
			} else {
				c.Putln("buf[b] = %s", name)
			}
		case 2:
			if t.SrcName() != "uint16" {
				c.Putln("xgb.Put16(buf[b:], uint16(%s))", name)
			} else {
				c.Putln("xgb.Put16(buf[b:], %s)", name)
			}
		case 4:
			if t.SrcName() != "uint32" {
				c.Putln("xgb.Put32(buf[b:], uint32(%s))", name)
			} else {
				c.Putln("xgb.Put32(buf[b:], %s)", name)
			}
		case 8:
			if t.SrcName() != "uint64" {
				c.Putln("xgb.Put64(buf[b:], uint64(%s))", name)
			} else {
				c.Putln("xgb.Put64(buf[b:], %s)", name)
			}
		}
	default:
		log.Fatalf("Cannot read field '%s' as a simple field with %T type.",
			name, typ)
	}

	c.Putln("b += %s", typ.Size())
}

func (f *SingleField) Write(c *Context, prefix string) {
	switch t := f.Type.(type) {
	case *Resource:
		WriteSimpleSingleField(c, fmt.Sprintf("%s%s", prefix, f.SrcName()), t)
	case *TypeDef:
		WriteSimpleSingleField(c, fmt.Sprintf("%s%s", prefix, f.SrcName()), t)
	case *Base:
		WriteSimpleSingleField(c, fmt.Sprintf("%s%s", prefix, f.SrcName()), t)
	case *Union:
		c.Putln("{")
		c.Putln("unionBytes := %s%s.Bytes()", prefix, f.SrcName())
		c.Putln("copy(buf[b:], unionBytes)")
		c.Putln("b += len(unionBytes)")
		c.Putln("}")
	case *Struct:
		c.Putln("{")
		c.Putln("structBytes := %s%s.Bytes()", prefix, f.SrcName())
		c.Putln("copy(buf[b:], structBytes)")
		c.Putln("b += len(structBytes)")
		c.Putln("}")
	default:
		log.Fatalf("Cannot read field '%s' with %T type.", f.XmlName(), f.Type)
	}
}
