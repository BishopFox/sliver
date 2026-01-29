package credential

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/util"
)

// TicketType ticket类型
type TicketType int

const (
	// TicketTypeCorpJs 企业jsapi ticket
	TicketTypeCorpJs TicketType = iota
	// TicketTypeAgentJs 应用jsapi ticket
	TicketTypeAgentJs
)

// 企业微信相关的 ticket URL
const (
	// 企业微信 jsapi ticket
	getWorkJsTicketURL = "https://qyapi.weixin.qq.com/cgi-bin/get_jsapi_ticket?access_token=%s"
	// 企业微信应用 jsapi ticket
	getWorkAgentJsTicketURL = "https://qyapi.weixin.qq.com/cgi-bin/ticket/get?access_token=%s&type=agent_config"
)

// WorkJsTicket 企业微信js ticket获取
type WorkJsTicket struct {
	corpID          string
	agentID         string
	cacheKeyPrefix  string
	cache           cache.Cache
	jsAPITicketLock *sync.Mutex
}

// NewWorkJsTicket new WorkJsTicket
func NewWorkJsTicket(corpID, agentID, cacheKeyPrefix string, cache cache.Cache) *WorkJsTicket {
	return &WorkJsTicket{
		corpID:          corpID,
		agentID:         agentID,
		cache:           cache,
		cacheKeyPrefix:  cacheKeyPrefix,
		jsAPITicketLock: new(sync.Mutex),
	}
}

// GetTicket 根据类型获取相应的jsapi_ticket
func (js *WorkJsTicket) GetTicket(accessToken string, ticketType TicketType) (ticketStr string, err error) {
	var cacheKey string
	switch ticketType {
	case TicketTypeCorpJs:
		cacheKey = fmt.Sprintf("%s_corp_jsapi_ticket_%s", js.cacheKeyPrefix, js.corpID)
	case TicketTypeAgentJs:
		if js.agentID == "" {
			err = fmt.Errorf("agentID is empty")
			return
		}
		cacheKey = fmt.Sprintf("%s_agent_jsapi_ticket_%s_%s", js.cacheKeyPrefix, js.corpID, js.agentID)
	default:
		err = fmt.Errorf("unsupported ticket type: %v", ticketType)
		return
	}

	if val := js.cache.Get(cacheKey); val != nil {
		return val.(string), nil
	}

	js.jsAPITicketLock.Lock()
	defer js.jsAPITicketLock.Unlock()

	// 双检，防止重复从微信服务器获取
	if val := js.cache.Get(cacheKey); val != nil {
		return val.(string), nil
	}

	var ticket ResTicket
	ticket, err = js.getTicketFromServer(accessToken, ticketType)
	if err != nil {
		return
	}
	expires := ticket.ExpiresIn - 1500
	err = js.cache.Set(cacheKey, ticket.Ticket, time.Duration(expires)*time.Second)
	ticketStr = ticket.Ticket
	return
}

// getTicketFromServer 从服务器中获取ticket
func (js *WorkJsTicket) getTicketFromServer(accessToken string, ticketType TicketType) (ticket ResTicket, err error) {
	var url string
	switch ticketType {
	case TicketTypeCorpJs:
		url = fmt.Sprintf(getWorkJsTicketURL, accessToken)
	case TicketTypeAgentJs:
		url = fmt.Sprintf(getWorkAgentJsTicketURL, accessToken)
	default:
		err = fmt.Errorf("unsupported ticket type: %v", ticketType)
		return
	}

	var response []byte
	response, err = util.HTTPGet(url)
	if err != nil {
		return
	}
	err = json.Unmarshal(response, &ticket)
	if err != nil {
		return
	}
	if ticket.ErrCode != 0 {
		err = fmt.Errorf("getTicket Error : errcode=%d , errmsg=%s", ticket.ErrCode, ticket.ErrMsg)
		return
	}
	return
}
