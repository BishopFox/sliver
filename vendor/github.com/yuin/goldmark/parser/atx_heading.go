package parser

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// A HeadingConfig struct is a data structure that holds configuration of the renderers related to headings.
type HeadingConfig struct {
	AutoHeadingID bool
	Attribute     bool
}

// SetOption implements SetOptioner.
func (b *HeadingConfig) SetOption(name OptionName, _ any) {
	switch name {
	case optAutoHeadingID:
		b.AutoHeadingID = true
	case optAttribute:
		b.Attribute = true
	}
}

// A HeadingOption interface sets options for heading parsers.
type HeadingOption interface {
	Option
	SetHeadingOption(*HeadingConfig)
}

// AutoHeadingID is an option name that enables auto IDs for headings.
const optAutoHeadingID OptionName = "AutoHeadingID"

type withAutoHeadingID struct {
}

func (o *withAutoHeadingID) SetParserOption(c *Config) {
	c.Options[optAutoHeadingID] = true
}

func (o *withAutoHeadingID) SetHeadingOption(p *HeadingConfig) {
	p.AutoHeadingID = true
}

// WithAutoHeadingID is a functional option that enables custom heading ids and
// auto generated heading ids.
func WithAutoHeadingID() HeadingOption {
	return &withAutoHeadingID{}
}

type withHeadingAttribute struct {
	Option
}

func (o *withHeadingAttribute) SetHeadingOption(p *HeadingConfig) {
	p.Attribute = true
}

// WithHeadingAttribute is a functional option that enables custom heading attributes.
func WithHeadingAttribute() HeadingOption {
	return &withHeadingAttribute{WithAttribute()}
}

type atxHeadingParser struct {
	HeadingConfig
}

// NewATXHeadingParser return a new BlockParser that can parse ATX headings.
func NewATXHeadingParser(opts ...HeadingOption) BlockParser {
	p := &atxHeadingParser{}
	for _, o := range opts {
		o.SetHeadingOption(&p.HeadingConfig)
	}
	return p
}

func (b *atxHeadingParser) Trigger() []byte {
	return []byte{'#'}
}

func (b *atxHeadingParser) Open(parent ast.Node, reader text.Reader, pc Context) (ast.Node, State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 {
		return nil, NoChildren
	}
	i := pos
	for ; i < len(line) && line[i] == '#'; i++ {
	}
	level := i - pos
	if i == pos || level > 6 {
		return nil, NoChildren
	}
	if i == len(line) { // alone '#' (without a new line character)
		return ast.NewHeading(level), NoChildren
	}
	l := util.TrimLeftSpaceLength(line[i:])
	if l == 0 {
		return nil, NoChildren
	}

	start := min(i+l, len(line)-1)
	node := ast.NewHeading(level)
	hl := text.NewSegment(
		segment.Start+start-segment.Padding,
		segment.Start+len(line)-segment.Padding)
	hl = hl.TrimRightSpace(reader.Source())
	if hl.Len() == 0 {
		reader.AdvanceToEOL()
		return node, NoChildren
	}

	if b.Attribute {
		node.Lines().Append(hl)
		parseLastLineAttributes(node, reader, pc)
		hl = node.Lines().At(0)
		node.Lines().Clear()
	}

	// handle closing sequence of '#' characters
	line = hl.Value(reader.Source())
	stop := len(line)
	if stop == 0 { // empty headings like '##[space]'
		stop = 0
	} else {
		i = stop - 1
		for ; line[i] == '#' && i > 0; i-- {
		}
		if i == 0 && line[0] == '#' { // empty headings like '### ###'
			reader.AdvanceToEOL()
			return node, NoChildren
		}
		if i != stop-1 && util.IsSpace(line[i]) {
			stop = i
			stop -= util.TrimRightSpaceLength(line[0:stop])
		}
	}
	hl.Stop = hl.Start + stop
	node.Lines().Append(hl)
	reader.AdvanceToEOL()

	return node, NoChildren
}

func (b *atxHeadingParser) Continue(node ast.Node, reader text.Reader, pc Context) State {
	return Close
}

func (b *atxHeadingParser) Close(node ast.Node, reader text.Reader, pc Context) {
	if b.AutoHeadingID {
		id, ok := node.AttributeString("id")
		if !ok {
			generateAutoHeadingID(node.(*ast.Heading), reader, pc)
		} else {
			pc.IDs().Put(id.([]byte))
		}
	}
}

func (b *atxHeadingParser) CanInterruptParagraph() bool {
	return true
}

func (b *atxHeadingParser) CanAcceptIndentedLine() bool {
	return false
}

func generateAutoHeadingID(node *ast.Heading, reader text.Reader, pc Context) {
	var line []byte
	lastIndex := node.Lines().Len() - 1
	if lastIndex > -1 {
		lastLine := node.Lines().At(lastIndex)
		line = lastLine.Value(reader.Source())
	}
	headingID := pc.IDs().Generate(line, ast.KindHeading)
	node.SetAttribute(attrNameID, headingID)
}

func parseLastLineAttributes(node ast.Node, reader text.Reader, _ Context) {
	lastIndex := node.Lines().Len() - 1
	if lastIndex < 0 { // empty headings
		return
	}
	lastLine := node.Lines().At(lastIndex)
	line := lastLine.Value(reader.Source())
	lr := text.NewReader(line)
	var start text.Segment
	var sl int
	for {
		c := lr.Peek()
		if c == text.EOF || c == '\n' {
			break
		}
		if c == '\\' {
			lr.Advance(1)
			if util.IsPunct(lr.Peek()) {
				lr.Advance(1)
			}
			continue
		}
		if c == '{' {
			sl, start = lr.Position()
			attrs, ok := ParseAttributes(lr)
			if ok {
				if nl, _ := lr.PeekLine(); nl == nil || util.IsBlank(nl) {
					for _, attr := range attrs {
						node.SetAttribute(attr.Name, attr.Value)
					}
					lastLine.Stop = lastLine.Start + start.Start
					lastLine = lastLine.TrimRightSpace(reader.Source())
					node.Lines().Set(lastIndex, lastLine)
					return
				}
			}
			lr.SetPosition(sl, start)
		}
		lr.Advance(1)
	}
}
