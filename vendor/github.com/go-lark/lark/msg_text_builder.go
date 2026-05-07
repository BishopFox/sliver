package lark

import (
	"fmt"
)

// textElemType of a text buf
type textElemType int

type textElem struct {
	elemType textElemType
	content  string
}

const (
	// MsgText text only message
	msgText textElemType = iota
	// MsgAt @somebody
	msgAt
	// MsgAtAll @all
	msgAtAll
	// msgSpace space
	msgSpace
)

// MsgTextBuilder for build text buf
type MsgTextBuilder struct {
	buf []textElem
}

// NewTextBuilder creates a text builder
func NewTextBuilder() *MsgTextBuilder {
	return &MsgTextBuilder{
		buf: make([]textElem, 0),
	}
}

// Text add simple texts
func (tb *MsgTextBuilder) Text(text ...interface{}) *MsgTextBuilder {
	elem := textElem{
		elemType: msgText,
		content:  fmt.Sprint(text...),
	}
	tb.buf = append(tb.buf, elem)
	return tb
}

// Textln add simple texts with a newline
func (tb *MsgTextBuilder) Textln(text ...interface{}) *MsgTextBuilder {
	elem := textElem{
		elemType: msgText,
		content:  fmt.Sprintln(text...),
	}
	tb.buf = append(tb.buf, elem)
	return tb
}

// Textf add texts with format
func (tb *MsgTextBuilder) Textf(textFmt string, text ...interface{}) *MsgTextBuilder {
	elem := textElem{
		elemType: msgText,
		content:  fmt.Sprintf(textFmt, text...),
	}
	tb.buf = append(tb.buf, elem)
	return tb
}

// Mention @somebody
func (tb *MsgTextBuilder) Mention(userID string) *MsgTextBuilder {
	elem := textElem{
		elemType: msgAt,
		content:  fmt.Sprintf("<at user_id=\"%s\">@user</at>", userID),
	}
	tb.buf = append(tb.buf, elem)
	return tb
}

// MentionAll @all
func (tb *MsgTextBuilder) MentionAll() *MsgTextBuilder {
	elem := textElem{
		elemType: msgAtAll,
		content:  "<at user_id=\"all\">@all</at>",
	}
	tb.buf = append(tb.buf, elem)
	return tb
}

// Clear all message
func (tb *MsgTextBuilder) Clear() {
	tb.buf = make([]textElem, 0)
}

// Render message
func (tb *MsgTextBuilder) Render() string {
	var text string
	for _, msg := range tb.buf {
		text += msg.content
	}
	return text
}

// Len returns buf len
func (tb MsgTextBuilder) Len() int {
	return len(tb.buf)
}
