package lark

// PostContent .
type PostContent map[string]PostBody

// PostBody .
type PostBody struct {
	Title   string       `json:"title"`
	Content [][]PostElem `json:"content"`
}

// PostElem .
type PostElem struct {
	Tag string `json:"tag"`
	// For Text
	UnEscape *bool   `json:"un_escape,omitempty"`
	Text     *string `json:"text,omitempty"`
	Lines    *int    `json:"lines,omitempty"`
	// For Link
	Href *string `json:"href,omitempty"`
	// For At
	UserID *string `json:"user_id,omitempty"`
	// For Image
	ImageKey    *string `json:"image_key,omitempty"`
	ImageWidth  *int    `json:"width,omitempty"`
	ImageHeight *int    `json:"height,omitempty"`
}

const (
	msgPostText  = "text"
	msgPostLink  = "a"
	msgPostAt    = "at"
	msgPostImage = "img"
)

// PostBuf .
type PostBuf struct {
	Title   string     `json:"title"`
	Content []PostElem `json:"content"`
}

// MsgPostBuilder for build text buf
type MsgPostBuilder struct {
	buf       map[string]*PostBuf
	curLocale string
}

const defaultLocale = LocaleZhCN

// NewPostBuilder creates a text builder
func NewPostBuilder() *MsgPostBuilder {
	return &MsgPostBuilder{
		buf:       make(map[string]*PostBuf),
		curLocale: defaultLocale,
	}
}

// Locale renamed to WithLocale but still available
func (pb *MsgPostBuilder) Locale(locale string) *MsgPostBuilder {
	return pb.WithLocale(locale)
}

// WithLocale switches to locale and returns self
func (pb *MsgPostBuilder) WithLocale(locale string) *MsgPostBuilder {
	if _, ok := pb.buf[locale]; !ok {
		pb.buf[locale] = &PostBuf{}
	}

	pb.curLocale = locale
	return pb
}

// CurLocale switches to locale and returns the buffer of that locale
func (pb *MsgPostBuilder) CurLocale() *PostBuf {
	return pb.WithLocale(pb.curLocale).buf[pb.curLocale]
}

// Title sets title
func (pb *MsgPostBuilder) Title(title string) *MsgPostBuilder {
	pb.CurLocale().Title = title
	return pb
}

// TextTag creates a text tag
func (pb *MsgPostBuilder) TextTag(text string, lines int, unescape bool) *MsgPostBuilder {
	pe := PostElem{
		Tag:      msgPostText,
		Text:     &text,
		Lines:    &lines,
		UnEscape: &unescape,
	}
	pb.CurLocale().Content = append(pb.CurLocale().Content, pe)
	return pb
}

// LinkTag creates a link tag
func (pb *MsgPostBuilder) LinkTag(text, href string) *MsgPostBuilder {
	pe := PostElem{
		Tag:  msgPostLink,
		Text: &text,
		Href: &href,
	}
	pb.CurLocale().Content = append(pb.CurLocale().Content, pe)
	return pb
}

// AtTag creates an at tag
func (pb *MsgPostBuilder) AtTag(text, userID string) *MsgPostBuilder {
	pe := PostElem{
		Tag:    msgPostAt,
		Text:   &text,
		UserID: &userID,
	}
	pb.CurLocale().Content = append(pb.CurLocale().Content, pe)
	return pb
}

// ImageTag creates an image tag
func (pb *MsgPostBuilder) ImageTag(imageKey string, imageWidth, imageHeight int) *MsgPostBuilder {
	pe := PostElem{
		Tag:         msgPostImage,
		ImageKey:    &imageKey,
		ImageWidth:  &imageWidth,
		ImageHeight: &imageHeight,
	}
	pb.CurLocale().Content = append(pb.CurLocale().Content, pe)
	return pb
}

// Clear all message
func (pb *MsgPostBuilder) Clear() {
	pb.curLocale = defaultLocale
	pb.buf = make(map[string]*PostBuf)
}

// Render message
func (pb *MsgPostBuilder) Render() *PostContent {
	content := make(PostContent)
	for locale, buf := range pb.buf {
		content[locale] = PostBody{
			Title:   buf.Title,
			Content: [][]PostElem{buf.Content},
		}
	}
	return &content
}

// Len returns buf len
func (pb MsgPostBuilder) Len() int {
	return len(pb.CurLocale().Content)
}
