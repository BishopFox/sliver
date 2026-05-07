package dingtalk

import (
	"encoding/json"
	"fmt"
	"strings"
)

type iDingMsg interface {
	Marshaler() []byte
}

type atOption interface {
	apply(model *atModel)
}

type funcAtOption struct {
	f func(model *atModel)
}

func (fdo *funcAtOption) apply(do *atModel) {
	fdo.f(do)
}

func newFuncAtOption(f func(model *atModel)) *funcAtOption {
	return &funcAtOption{f: f}
}

func WithAtAll() atOption {
	return newFuncAtOption(func(o *atModel) {
		o.IsAtAll = true
	})
}

func WithAtMobiles(mobiles []string) atOption {
	return newFuncAtOption(func(o *atModel) {
		o.AtMobiles = mobiles
	})
}

type textMsg struct {
	MsgType msgTypeType `json:"msgtype,omitempty"`
	Text    textModel   `json:"text,omitempty"`
	At      atModel     `json:"at,omitempty"`
}

func (t textMsg) Marshaler() []byte {
	b, _ := json.Marshal(t)
	return b
}

func NewTextMsg(content string, opts ...atOption) *textMsg {
	msg := &textMsg{MsgType: TEXT, Text: textModel{Content: content}}
	for _, opt := range opts {
		opt.apply(&msg.At)
	}
	return msg
}

type linkMsg struct {
	MsgType msgTypeType `json:"msgtype,omitempty"`
	Link    linkModel   `json:"link,omitempty"`
}

func (l linkMsg) Marshaler() []byte {
	b, _ := json.Marshal(l)
	return b
}

func NewLinkMsg(title, text, picUrl, msgUrl string) *linkMsg {
	return &linkMsg{MsgType: LINK, Link: linkModel{
		Text:       text,
		Title:      title,
		PicUrl:     picUrl,
		MessageUrl: msgUrl,
	}}
}

type markDownMsg struct {
	MsgType  msgTypeType   `json:"msgtype,omitempty"`
	Markdown markDownModel `json:"markdown,omitempty"`
	At       atModel       `json:"at,omitempty"`
}

func (m markDownMsg) Marshaler() []byte {
	b, _ := json.Marshal(m)
	return b
}

func NewDTMDMsg(title string, dtmdMap *dingMap, opts ...atOption) *markDownMsg {
	text := ""
	for _, v := range dtmdMap.l {
		text = text + "\n - " + fmt.Sprintf(dtmdFormat, v, dtmdMap.m[v])
	}
	return NewMarkDownMsg(title, text, opts...)
}

func NewMarkDownMsg(title string, text interface{}, opts ...atOption) *markDownMsg {

	msg := &markDownMsg{MsgType: MARKDOWN, Markdown: markDownModel{Title: title, Text: text.(string)}}
	for _, opt := range opts {
		opt.apply(&msg.At)
	}
	// markdown格式需要在文本内写入被at的人
	if len(msg.At.AtMobiles) > 0 {
		var atStr = "\n -"
		for _, mobile := range msg.At.AtMobiles {
			atStr = atStr + " @" + mobile
		}
		msg.Markdown.Text = msg.Markdown.Text + atStr
	}
	return msg
}

type actionCardOption interface {
	apply(model *actionCardModel)
}

type funcActionCardOption struct {
	f func(model *actionCardModel)
}

func (fdo *funcActionCardOption) apply(do *actionCardModel) {
	fdo.f(do)
}

func newFuncActionCardOption(f func(model *actionCardModel)) *funcActionCardOption {
	return &funcActionCardOption{f: f}
}

func WithCardBtnVertical() actionCardOption {
	return newFuncActionCardOption(func(o *actionCardModel) {
		o.BtnOrientation = vertical
	})
}

func WithCardSingleTitle(title string) actionCardOption {
	return newFuncActionCardOption(func(o *actionCardModel) {
		o.SingleTitle = title
	})
}

func WithCardSingleURL(url string) actionCardOption {
	return newFuncActionCardOption(func(o *actionCardModel) {
		o.SingleURL = url
	})
}

func WithCardBtns(btns []ActionCardMultiBtnModel) actionCardOption {
	return newFuncActionCardOption(func(o *actionCardModel) {
		o.Btns = btns
	})
}

type actionCardMsg struct {
	MsgType    msgTypeType     `json:"msgtype,omitempty"`
	ActionCard actionCardModel `json:"actionCard,omitempty"`
}

func (a actionCardMsg) Marshaler() []byte {
	b, _ := json.Marshal(a)
	return b
}

func NewActionCardMsg(title, text string, opts ...actionCardOption) *actionCardMsg {
	card := &actionCardMsg{MsgType: ACTION_CARD, ActionCard: actionCardModel{
		Title:          title,
		Text:           text,
		BtnOrientation: horizontal,
	}}
	for _, opt := range opts {
		opt.apply(&card.ActionCard)
	}
	return card
}

type feedCardMsg struct {
	MsgType  msgTypeType   `json:"msgtype,omitempty"`
	FeedCard feedCardModel `json:"feedCard,omitempty"`
}

func (f feedCardMsg) Marshaler() []byte {
	b, _ := json.Marshal(f)
	return b
}

func NewFeedCardMsg(feedCard []FeedCardLinkModel) *feedCardMsg {
	return &feedCardMsg{MsgType: FEED_CARD, FeedCard: feedCardModel{Links: feedCard}}
}

type MarkType string

// 有序map
type dingMap struct {
	m map[string]MarkType
	l []string
}

func DingMap() *dingMap {
	return &dingMap{m: make(map[string]MarkType), l: make([]string, 0, 0)}
}

func (d *dingMap) Set(val string, t MarkType) *dingMap {
	d.l = append(d.l, val)
	d.m[val] = t
	return d
}

func (d *dingMap) Remove(val string) {
	if _, ok := d.m[val]; ok {
		for i, v := range d.l {
			if v == val {
				d.l = append(d.l[:i], d.l[i+1:]...)
				break
			}
		}
		delete(d.m, val)
	}
}

func (d *dingMap) Slice() []string {
	resList := make([]string, 0, len(d.l))
	for _, val := range d.l {
		content := d.formatVal(val, d.m[val])
		resList = append(resList, content)
	}
	return resList
}

func (d *dingMap) formatVal(val string, t MarkType) (res string) {
	var ok bool
	if res, ok = hMap[t]; ok {
		vl := strings.Split(val, formatSpliter)
		if len(vl) == 3 {
			res = fmt.Sprintf(res, vl[1])
			res = vl[0] + res + vl[2]
		} else {
			res = fmt.Sprintf(res, val)
		}
	} else {
		res = val
	}
	if !strings.HasPrefix(res, "- ") && !strings.HasPrefix(res, "#") {
		res = "- " + res
	}
	return
}

type responseMsg struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}
