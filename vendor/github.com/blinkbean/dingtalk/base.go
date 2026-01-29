package dingtalk

import "time"

type msgTypeType string

const (
	TEXT        msgTypeType = "text"
	LINK        msgTypeType = "link"
	MARKDOWN    msgTypeType = "markdown"
	ACTION_CARD msgTypeType = "actionCard"
	FEED_CARD   msgTypeType = "feedCard"
)

type initModel struct {
	InitSendTimeout time.Duration
}

func (i initModel) GetSendTimeout() time.Duration {
	if i.InitSendTimeout > 0 {
		return i.InitSendTimeout
	}
	return time.Second * 2
}

type initOption interface {
	applyInit(model *initModel)
}

type funcInitOption struct {
	f func(model *initModel)
}

func (fdo *funcInitOption) applyInit(do *initModel) {
	fdo.f(do)
}

func newFuncInitOption(f func(model *initModel)) *funcInitOption {
	return &funcInitOption{f: f}
}

func WithInitSendTimeout(v time.Duration) initOption {
	return newFuncInitOption(func(o *initModel) {
		o.InitSendTimeout = v
	})
}

type DingTalk struct {
	robotToken []string
	secret     string
	keyWord    string
	InitModel  initModel
}

type textModel struct {
	Content string `json:"content,omitempty"`
}

type atModel struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

type linkModel struct {
	Text       string `json:"text,omitempty"`
	Title      string `json:"title,omitempty"`
	PicUrl     string `json:"picUrl,omitempty"`
	MessageUrl string `json:"messageUrl,omitempty"`
}

type markDownModel struct {
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

type actionCardBtnOrientationType string

const (
	horizontal actionCardBtnOrientationType = "0" // 横向
	vertical   actionCardBtnOrientationType = "1" // 竖向
)

type actionCardModel struct {
	Title          string                       `json:"title,omitempty"`
	Text           string                       `json:"text,omitempty"`
	BtnOrientation actionCardBtnOrientationType `json:"btnOrientation,omitempty"`
	SingleTitle    string                       `json:"singleTitle,omitempty"`
	SingleURL      string                       `json:"singleURL,omitempty"`
	Btns           []ActionCardMultiBtnModel    `json:"btns,omitempty"`
}

type ActionCardMultiBtnModel struct {
	Title     string `json:"title,omitempty"`
	ActionURL string `json:"actionURL,omitempty"`
}

type feedCardModel struct {
	Links []FeedCardLinkModel `json:"links,omitempty"`
}

type FeedCardLinkModel struct {
	Title      string `json:"title,omitempty"`
	MessageURL string `json:"messageURL,omitempty"`
	PicURL     string `json:"picURL,omitempty"`
}

type outGoingModel struct {
	AtUsers []struct {
		DingtalkID string `json:"dingtalkId"`
	} `json:"atUsers"`
	ChatbotUserID             string `json:"chatbotUserId"`
	ConversationID            string `json:"conversationId"`
	ConversationTitle         string `json:"conversationTitle"`
	ConversationType          string `json:"conversationType"`
	CreateAt                  int64  `json:"createAt"`
	IsAdmin                   bool   `json:"isAdmin"`
	IsInAtList                bool   `json:"isInAtList"`
	MsgID                     string `json:"msgId"`
	Msgtype                   string `json:"msgtype"`
	SceneGroupCode            string `json:"sceneGroupCode"`
	SenderID                  string `json:"senderId"`
	SenderNick                string `json:"senderNick"`
	SessionWebhook            string `json:"sessionWebhook"`
	SessionWebhookExpiredTime int64  `json:"sessionWebhookExpiredTime"`
	Text                      struct {
		Content string `json:"content"`
	} `json:"text"`
}

type ExecFunc func(args []string) []byte
