package aispeech

import (
	stdcontext "context"

	"github.com/silenceper/wechat/v2/aispeech/config"
	"github.com/silenceper/wechat/v2/aispeech/context"
	"github.com/silenceper/wechat/v2/aispeech/dialog"
	"github.com/silenceper/wechat/v2/credential"
)

// AISpeech 微信智能对话相关 API.
type AISpeech struct {
	ctx    *context.Context
	dialog *dialog.Dialog
}

// NewAISpeech 实例化智能对话 API.
func NewAISpeech(cfg *config.Config) *AISpeech {
	ctx := &context.Context{
		Config:                   cfg,
		AccessTokenContextHandle: dialog.NewAccessToken(cfg),
	}
	return &AISpeech{ctx: ctx}
}

// GetContext get Context.
func (a *AISpeech) GetContext() *context.Context {
	return a.ctx
}

// SetAccessTokenHandle 自定义 access_token 获取方式.
func (a *AISpeech) SetAccessTokenHandle(accessTokenHandle credential.AccessTokenHandle) {
	a.ctx.AccessTokenContextHandle = credential.AccessTokenCompatibleHandle{
		AccessTokenHandle: accessTokenHandle,
	}
}

// SetAccessTokenContextHandle 自定义 access_token 获取方式.
func (a *AISpeech) SetAccessTokenContextHandle(accessTokenContextHandle credential.AccessTokenContextHandle) {
	a.ctx.AccessTokenContextHandle = accessTokenContextHandle
}

// GetAccessToken 获取 access token.
func (a *AISpeech) GetAccessToken() (string, error) {
	return a.ctx.GetAccessToken()
}

// GetAccessTokenContext 获取 access token.
func (a *AISpeech) GetAccessTokenContext(ctx stdcontext.Context) (string, error) {
	return a.ctx.GetAccessTokenContext(ctx)
}

// GetDialog 获取对话平台 API.
func (a *AISpeech) GetDialog() *dialog.Dialog {
	if a.dialog == nil {
		a.dialog = dialog.NewDialog(a.ctx)
	}
	return a.dialog
}
