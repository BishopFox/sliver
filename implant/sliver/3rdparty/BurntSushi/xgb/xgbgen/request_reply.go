package main

import (
	"fmt"
	"log"
	"unicode"
)

// Request represents all XML 'request' nodes.
// If the request doesn't have a reply, Reply is nil.
type Request struct {
	srcName string // The Go name of this request.
	xmlName string // The XML name of this request.
	Opcode  int
	Combine bool    // Not currently used.
	Fields  []Field // All fields in the request.
	Reply   *Reply  // A reply, if one exists for this request.
}

type Requests []*Request

func (rs Requests) Len() int           { return len(rs) }
func (rs Requests) Swap(i, j int)      { rs[i], rs[j] = rs[j], rs[i] }
func (rs Requests) Less(i, j int) bool { return rs[i].xmlName < rs[j].xmlName }

// Initialize creates the proper Go source name for this request.
// It also initializes the reply if one exists, and all fields in this request.
func (r *Request) Initialize(p *Protocol) {
	r.srcName = SrcName(p, r.xmlName)
	if p.isExt() {
		r.srcName = r.srcName
	}

	if r.Reply != nil {
		r.Reply.Initialize(p)
	}
	for _, field := range r.Fields {
		field.Initialize(p)
	}
}

func (r *Request) SrcName() string {
	return r.srcName
}

func (r *Request) XmlName() string {
	return r.xmlName
}

// ReplyName gets the Go source name of the function that generates a
// reply type from a slice of bytes.
// The generated function is not currently exported.
func (r *Request) ReplyName() string {
	if r.Reply == nil {
		log.Panicf("Cannot call 'ReplyName' on request %s, which has no reply.",
			r.SrcName())
	}
	name := r.SrcName()
	lower := string(unicode.ToLower(rune(name[0]))) + name[1:]
	return fmt.Sprintf("%sReply", lower)
}

// ReplyTypeName gets the Go source name of the type holding all reply data
// for this request.
func (r *Request) ReplyTypeName() string {
	if r.Reply == nil {
		log.Panicf("Cannot call 'ReplyName' on request %s, which has no reply.",
			r.SrcName())
	}
	return fmt.Sprintf("%sReply", r.SrcName())
}

// ReqName gets the Go source name of the function that generates a byte
// slice from a list of parameters.
// The generated function is not currently exported.
func (r *Request) ReqName() string {
	name := r.SrcName()
	lower := string(unicode.ToLower(rune(name[0]))) + name[1:]
	return fmt.Sprintf("%sRequest", lower)
}

// CookieName gets the Go source name of the type that holds cookies for
// this request.
func (r *Request) CookieName() string {
	return fmt.Sprintf("%sCookie", r.SrcName())
}

// Size for Request needs a context.
// Namely, if this is an extension, we need to account for *four* bytes
// of a header (extension opcode, request opcode, and the sequence number).
// If it's a core protocol request, then we only account for *three*
// bytes of the header (remove the extension opcode).
func (r *Request) Size(c *Context) Size {
	size := newFixedSize(0, true)

	// If this is a core protocol request, we squeeze in an extra byte of
	// data (from the fields below) between the opcode and the size of the
	// request. In an extension request, this byte is always occupied
	// by the opcode of the request (while the first byte is always occupied
	// by the opcode of the extension).
	if !c.protocol.isExt() {
		size = size.Add(newFixedSize(3, true))
	} else {
		size = size.Add(newFixedSize(4, true))
	}

	for _, field := range r.Fields {
		switch field.(type) {
		case *LocalField: // local fields don't go over the wire
			continue
		case *SingleField:
			// mofos!!!
			if r.SrcName() == "ConfigureWindow" &&
				field.SrcName() == "ValueMask" {

				continue
			}
			size = size.Add(field.Size())
		default:
			size = size.Add(field.Size())
		}
	}
	return newExpressionSize(&Padding{
		Expr: size.Expression,
	}, size.exact)
}

// Reply encapsulates the fields associated with a 'reply' element.
type Reply struct {
	Fields []Field
}

// Size gets the number of bytes in this request's reply.
// A reply always has at least 7 bytes:
// 1 byte: A reply discriminant (first byte set to 1)
// 2 bytes: A sequence number
// 4 bytes: Number of additional bytes in 4-byte units past initial 32 bytes.
func (r *Reply) Size() Size {
	size := newFixedSize(0, true)

	// Account for reply discriminant, sequence number and reply length
	size = size.Add(newFixedSize(7, true))

	for _, field := range r.Fields {
		size = size.Add(field.Size())
	}
	return size
}

func (r *Reply) Initialize(p *Protocol) {
	for _, field := range r.Fields {
		field.Initialize(p)
	}
}
