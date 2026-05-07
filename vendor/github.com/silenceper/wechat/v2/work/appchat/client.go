// Package appchat 应用发送消息到群聊会话,企业微信接口：https://developer.work.weixin.qq.com/document/path/90248
package appchat

import (
	"github.com/silenceper/wechat/v2/work/context"
)

// Client 接口实例
type Client struct {
	*context.Context
}

// NewClient 初始化实例
func NewClient(ctx *context.Context) *Client {
	return &Client{ctx}
}
