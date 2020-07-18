package main

import (
	"fmt"
	"strings"
)

func (r *Request) Define(c *Context) {
	c.Putln("// %s is a cookie used only for %s requests.",
		r.CookieName(), r.SrcName())
	c.Putln("type %s struct {", r.CookieName())
	c.Putln("*xgb.Cookie")
	c.Putln("}")
	c.Putln("")
	if r.Reply != nil {
		c.Putln("// %s sends a checked request.", r.SrcName())
		c.Putln("// If an error occurs, it will be returned with the reply "+
			"by calling %s.Reply()", r.CookieName())
		c.Putln("func %s(c *xgb.Conn, %s) %s {",
			r.SrcName(), r.ParamNameTypes(), r.CookieName())
		r.CheckExt(c)
		c.Putln("cookie := c.NewCookie(true, true)")
		c.Putln("c.NewRequest(%s(c, %s), cookie)", r.ReqName(), r.ParamNames())
		c.Putln("return %s{cookie}", r.CookieName())
		c.Putln("}")
		c.Putln("")

		c.Putln("// %sUnchecked sends an unchecked request.", r.SrcName())
		c.Putln("// If an error occurs, it can only be retrieved using " +
			"xgb.WaitForEvent or xgb.PollForEvent.")
		c.Putln("func %sUnchecked(c *xgb.Conn, %s) %s {",
			r.SrcName(), r.ParamNameTypes(), r.CookieName())
		r.CheckExt(c)
		c.Putln("cookie := c.NewCookie(false, true)")
		c.Putln("c.NewRequest(%s(c, %s), cookie)", r.ReqName(), r.ParamNames())
		c.Putln("return %s{cookie}", r.CookieName())
		c.Putln("}")
		c.Putln("")

		r.ReadReply(c)
	} else {
		c.Putln("// %s sends an unchecked request.", r.SrcName())
		c.Putln("// If an error occurs, it can only be retrieved using " +
			"xgb.WaitForEvent or xgb.PollForEvent.")
		c.Putln("func %s(c *xgb.Conn, %s) %s {",
			r.SrcName(), r.ParamNameTypes(), r.CookieName())
		r.CheckExt(c)
		c.Putln("cookie := c.NewCookie(false, false)")
		c.Putln("c.NewRequest(%s(c, %s), cookie)", r.ReqName(), r.ParamNames())
		c.Putln("return %s{cookie}", r.CookieName())
		c.Putln("}")
		c.Putln("")

		c.Putln("// %sChecked sends a checked request.", r.SrcName())
		c.Putln("// If an error occurs, it can be retrieved using "+
			"%s.Check()", r.CookieName())
		c.Putln("func %sChecked(c *xgb.Conn, %s) %s {",
			r.SrcName(), r.ParamNameTypes(), r.CookieName())
		r.CheckExt(c)
		c.Putln("cookie := c.NewCookie(true, false)")
		c.Putln("c.NewRequest(%s(c, %s), cookie)", r.ReqName(), r.ParamNames())
		c.Putln("return %s{cookie}", r.CookieName())
		c.Putln("}")
		c.Putln("")

		c.Putln("// Check returns an error if one occurred for checked " +
			"requests that are not expecting a reply.")
		c.Putln("// This cannot be called for requests expecting a reply, " +
			"nor for unchecked requests.")
		c.Putln("func (cook %s) Check() error {", r.CookieName())
		c.Putln("return cook.Cookie.Check()")
		c.Putln("}")
		c.Putln("")
	}
	r.WriteRequest(c)
}

func (r *Request) CheckExt(c *Context) {
	if !c.protocol.isExt() {
		return
	}
	c.Putln("c.ExtLock.RLock()")
	c.Putln("defer c.ExtLock.RUnlock()")
	c.Putln("if _, ok := c.Extensions[\"%s\"]; !ok {", c.protocol.ExtXName)
	c.Putln("panic(\"Cannot issue request '%s' using the uninitialized "+
		"extension '%s'. %s.Init(connObj) must be called first.\")",
		r.SrcName(), c.protocol.ExtXName, c.protocol.PkgName())
	c.Putln("}")
}

func (r *Request) ReadReply(c *Context) {
	c.Putln("// %s represents the data returned from a %s request.",
		r.ReplyTypeName(), r.SrcName())
	c.Putln("type %s struct {", r.ReplyTypeName())
	c.Putln("Sequence uint16 // sequence number of the request for this reply")
	c.Putln("Length uint32 // number of bytes in this reply")
	for _, field := range r.Reply.Fields {
		field.Define(c)
	}
	c.Putln("}")
	c.Putln("")

	c.Putln("// Reply blocks and returns the reply data for a %s request.",
		r.SrcName())
	c.Putln("func (cook %s) Reply() (*%s, error) {",
		r.CookieName(), r.ReplyTypeName())
	c.Putln("buf, err := cook.Cookie.Reply()")
	c.Putln("if err != nil {")
	c.Putln("return nil, err")
	c.Putln("}")
	c.Putln("if buf == nil {")
	c.Putln("return nil, nil")
	c.Putln("}")
	c.Putln("return %s(buf), nil", r.ReplyName())
	c.Putln("}")
	c.Putln("")

	c.Putln("// %s reads a byte slice into a %s value.",
		r.ReplyName(), r.ReplyTypeName())
	c.Putln("func %s(buf []byte) *%s {",
		r.ReplyName(), r.ReplyTypeName())
	c.Putln("v := new(%s)", r.ReplyTypeName())
	c.Putln("b := 1 // skip reply determinant")
	c.Putln("")
	for i, field := range r.Reply.Fields {
		field.Read(c, "v.")
		c.Putln("")
		if i == 0 {
			c.Putln("v.Sequence = xgb.Get16(buf[b:])")
			c.Putln("b += 2")
			c.Putln("")
			c.Putln("v.Length = xgb.Get32(buf[b:]) // 4-byte units")
			c.Putln("b += 4")
			c.Putln("")
		}
	}
	c.Putln("return v")
	c.Putln("}")
	c.Putln("")
}

func (r *Request) WriteRequest(c *Context) {
	sz := r.Size(c)
	writeSize1 := func() {
		if sz.exact {
			c.Putln("xgb.Put16(buf[b:], uint16(size / 4)) " +
				"// write request size in 4-byte units")
		} else {
			c.Putln("blen := b")
		}
		c.Putln("b += 2")
		c.Putln("")
	}
	writeSize2 := func() {
		if sz.exact {
			c.Putln("return buf")
			return
		}
		c.Putln("b = xgb.Pad(b)")
		c.Putln("xgb.Put16(buf[blen:], uint16(b / 4)) " +
			"// write request size in 4-byte units")
		c.Putln("return buf[:b]")
	}
	c.Putln("// Write request to wire for %s", r.SrcName())
	c.Putln("// %s writes a %s request to a byte slice.",
		r.ReqName(), r.SrcName())
	c.Putln("func %s(c *xgb.Conn, %s) []byte {",
		r.ReqName(), r.ParamNameTypes())
	c.Putln("size := %s", sz)
	c.Putln("b := 0")
	c.Putln("buf := make([]byte, size)")
	c.Putln("")
	if c.protocol.isExt() {
		c.Putln("c.ExtLock.RLock()")
		c.Putln("buf[b] = c.Extensions[\"%s\"]", c.protocol.ExtXName)
		c.Putln("c.ExtLock.RUnlock()")
		c.Putln("b += 1")
		c.Putln("")
	}
	c.Putln("buf[b] = %d // request opcode", r.Opcode)
	c.Putln("b += 1")
	c.Putln("")
	if len(r.Fields) == 0 {
		if !c.protocol.isExt() {
			c.Putln("b += 1 // padding")
		}
		writeSize1()
	} else if c.protocol.isExt() {
		writeSize1()
	}
	for i, field := range r.Fields {
		field.Write(c, "")
		c.Putln("")
		if i == 0 && !c.protocol.isExt() {
			writeSize1()
		}
	}
	writeSize2()
	c.Putln("}")
	c.Putln("")
}

func (r *Request) ParamNames() string {
	names := make([]string, 0, len(r.Fields))
	for _, field := range r.Fields {
		switch f := field.(type) {
		case *ValueField:
			// mofos...
			if r.SrcName() != "ConfigureWindow" {
				names = append(names, f.MaskName)
			}
			names = append(names, f.ListName)
		case *PadField:
			continue
		case *ExprField:
			continue
		default:
			names = append(names, fmt.Sprintf("%s", field.SrcName()))
		}
	}
	return strings.Join(names, ", ")
}

func (r *Request) ParamNameTypes() string {
	nameTypes := make([]string, 0, len(r.Fields))
	for _, field := range r.Fields {
		switch f := field.(type) {
		case *ValueField:
			// mofos...
			if r.SrcName() != "ConfigureWindow" {
				nameTypes = append(nameTypes,
					fmt.Sprintf("%s %s", f.MaskName, f.MaskType.SrcName()))
			}
			nameTypes = append(nameTypes,
				fmt.Sprintf("%s []uint32", f.ListName))
		case *PadField:
			continue
		case *ExprField:
			continue
		default:
			nameTypes = append(nameTypes,
				fmt.Sprintf("%s %s", field.SrcName(), field.SrcType()))
		}
	}
	return strings.Join(nameTypes, ", ")
}
