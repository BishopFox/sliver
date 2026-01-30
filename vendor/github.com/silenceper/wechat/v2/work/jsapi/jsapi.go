package jsapi

import (
	"fmt"

	"github.com/silenceper/wechat/v2/credential"
	"github.com/silenceper/wechat/v2/util"
	"github.com/silenceper/wechat/v2/work/context"
)

// Js struct
type Js struct {
	*context.Context
	jsTicket *credential.WorkJsTicket
}

// NewJs init
func NewJs(context *context.Context) *Js {
	js := new(Js)
	js.Context = context
	js.jsTicket = credential.NewWorkJsTicket(
		context.Config.CorpID,
		context.Config.AgentID,
		credential.CacheKeyWorkPrefix,
		context.Cache,
	)
	return js
}

// Config 返回给用户使用的配置
type Config struct {
	Timestamp int64  `json:"timestamp"`
	NonceStr  string `json:"nonce_str"`
	Signature string `json:"signature"`
}

// GetConfig 获取企业微信JS配置 https://developer.work.weixin.qq.com/document/path/90514
func (js *Js) GetConfig(uri string) (config *Config, err error) {
	config = new(Config)
	var accessToken string
	accessToken, err = js.GetAccessToken()
	if err != nil {
		return
	}
	var ticketStr string
	ticketStr, err = js.jsTicket.GetTicket(accessToken, credential.TicketTypeCorpJs)
	if err != nil {
		return
	}
	config.NonceStr = util.RandomStr(16)
	config.Timestamp = util.GetCurrTS()
	str := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", ticketStr, config.NonceStr, config.Timestamp, uri)
	config.Signature = util.Signature(str)
	return
}

// GetAgentConfig 获取企业微信应用JS配置 https://developer.work.weixin.qq.com/document/path/94313
func (js *Js) GetAgentConfig(uri string) (config *Config, err error) {
	config = new(Config)
	var accessToken string
	accessToken, err = js.GetAccessToken()
	if err != nil {
		return
	}
	var ticketStr string
	ticketStr, err = js.jsTicket.GetTicket(accessToken, credential.TicketTypeAgentJs)
	if err != nil {
		return
	}
	config.NonceStr = util.RandomStr(16)
	config.Timestamp = util.GetCurrTS()
	str := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", ticketStr, config.NonceStr, config.Timestamp, uri)
	config.Signature = util.Signature(str)
	return
}
