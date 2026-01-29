package credential

import (
	context2 "context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/util"
)

// getTicketURL 获取ticket的url
const getTicketURL = "https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi"

// DefaultJsTicket 默认获取js ticket方法
type DefaultJsTicket struct {
	appID          string
	cacheKeyPrefix string
	cache          cache.Cache
	// jsAPITicket 读写锁 同一个AppID一个
	jsAPITicketLock *sync.Mutex
}

// NewDefaultJsTicket new
func NewDefaultJsTicket(appID string, cacheKeyPrefix string, cache cache.Cache) JsTicketHandle {
	return &DefaultJsTicket{
		appID:           appID,
		cache:           cache,
		cacheKeyPrefix:  cacheKeyPrefix,
		jsAPITicketLock: new(sync.Mutex),
	}
}

// ResTicket 请求jsapi_tikcet返回结果
type ResTicket struct {
	util.CommonError

	Ticket    string `json:"ticket"`
	ExpiresIn int64  `json:"expires_in"`
}

// GetTicket 获取jsapi_ticket
func (js *DefaultJsTicket) GetTicket(accessToken string) (ticketStr string, err error) {
	return js.GetTicketContext(context2.Background(), accessToken)
}

// GetTicketFromServer 从服务器中获取ticket
func GetTicketFromServer(accessToken string) (ticket ResTicket, err error) {
	return GetTicketFromServerContext(context2.Background(), accessToken)
}

// GetTicketContext 获取jsapi_ticket
func (js *DefaultJsTicket) GetTicketContext(ctx context2.Context, accessToken string) (ticketStr string, err error) {
	// 先从cache中取
	jsAPITicketCacheKey := fmt.Sprintf("%s_jsapi_ticket_%s", js.cacheKeyPrefix, js.appID)
	if val := js.cache.Get(jsAPITicketCacheKey); val != nil {
		return val.(string), nil
	}

	js.jsAPITicketLock.Lock()
	defer js.jsAPITicketLock.Unlock()

	// 双检，防止重复从微信服务器获取
	if val := js.cache.Get(jsAPITicketCacheKey); val != nil {
		return val.(string), nil
	}

	var ticket ResTicket
	ticket, err = GetTicketFromServerContext(ctx, accessToken)
	if err != nil {
		return
	}
	expires := ticket.ExpiresIn - 1500
	err = js.cache.Set(jsAPITicketCacheKey, ticket.Ticket, time.Duration(expires)*time.Second)
	ticketStr = ticket.Ticket
	return
}

// GetTicketFromServerContext 从服务器中获取ticket
func GetTicketFromServerContext(ctx context2.Context, accessToken string) (ticket ResTicket, err error) {
	var response []byte
	url := fmt.Sprintf(getTicketURL, accessToken)
	response, err = util.HTTPGetContext(ctx, url)
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
