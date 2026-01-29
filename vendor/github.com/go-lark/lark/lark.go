// Package lark is an easy-to-use SDK for Feishu and Lark Open Platform,
// which implements messaging APIs, with full-fledged supports on building Chat Bot and Notification Bot.
package lark

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	// ChatBot should be created with NewChatBot
	// Create from https://open.feishu.cn/ or https://open.larksuite.com/
	ChatBot = iota
	// NotificationBot for webhook, behave as a simpler notification bot
	// Create from Lark group
	NotificationBot
)

// Bot definition
type Bot struct {
	// bot type
	botType int
	// Auth info
	appID             string
	appSecret         string
	accessToken       atomic.Value
	tenantAccessToken atomic.Value

	// user id type for api chat
	userIDType string
	// webhook for NotificationBot
	webhook string
	// API Domain
	domain string
	// http client
	client *http.Client
	// custom http client
	useCustomClient bool
	customClient    HTTPWrapper
	// auth heartbeat
	heartbeat chan bool
	// auth heartbeat counter (for testing)
	heartbeatCounter int64

	ctx    context.Context
	logger LogWrapper
	debug  bool
}

// Domains
const (
	DomainFeishu = "https://open.feishu.cn"
	DomainLark   = "https://open.larksuite.com"
)

// NewChatBot with appID and appSecret
func NewChatBot(appID, appSecret string) *Bot {
	bot := &Bot{
		botType:   ChatBot,
		appID:     appID,
		appSecret: appSecret,
		client:    initClient(),
		domain:    DomainFeishu,
		ctx:       context.Background(),
		logger:    initDefaultLogger(),
	}
	bot.accessToken.Store("")
	bot.tenantAccessToken.Store("")

	return bot
}

// NewNotificationBot with URL
func NewNotificationBot(hookURL string) *Bot {
	bot := &Bot{
		botType: NotificationBot,
		webhook: hookURL,
		client:  initClient(),
		ctx:     context.Background(),
		logger:  initDefaultLogger(),
	}
	bot.accessToken.Store("")
	bot.tenantAccessToken.Store("")

	return bot
}

// requireType checks whether the action is allowed in a list of bot types
func (bot Bot) requireType(botType ...int) bool {
	for _, iterType := range botType {
		if bot.botType == iterType {
			return true
		}
	}
	return false
}

// SetClient assigns a new client to bot.client
func (bot *Bot) SetClient(c *http.Client) {
	bot.client = c
}

func initClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

// SetCustomClient .
func (bot *Bot) SetCustomClient(c HTTPWrapper) {
	bot.useCustomClient = true
	bot.customClient = c
}

// UnsetCustomClient .
func (bot *Bot) UnsetCustomClient() {
	bot.useCustomClient = false
	bot.customClient = nil
}

// SetDomain set domain of endpoint, so we could call Feishu/Lark
// go-lark does not check your host, just use the right one or fail.
func (bot *Bot) SetDomain(domain string) {
	bot.domain = domain
}

// Domain returns current domain
func (bot Bot) Domain() string {
	return bot.domain
}

// AppID returns bot.appID for external use
func (bot Bot) AppID() string {
	return bot.appID
}

// BotType returns bot.botType for external use
func (bot Bot) BotType() int {
	return bot.botType
}

// AccessToken returns bot.accessToken for external use
func (bot Bot) AccessToken() string {
	return bot.accessToken.Load().(string)
}

// TenantAccessToken returns bot.tenantAccessToken for external use
func (bot Bot) TenantAccessToken() string {
	return bot.tenantAccessToken.Load().(string)
}

// SetWebhook updates webhook URL
func (bot *Bot) SetWebhook(url string) {
	bot.webhook = url
}
