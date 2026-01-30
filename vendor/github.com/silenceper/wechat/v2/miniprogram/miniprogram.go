package miniprogram

import (
	"github.com/silenceper/wechat/v2/credential"
	"github.com/silenceper/wechat/v2/internal/openapi"
	"github.com/silenceper/wechat/v2/miniprogram/analysis"
	"github.com/silenceper/wechat/v2/miniprogram/auth"
	"github.com/silenceper/wechat/v2/miniprogram/business"
	"github.com/silenceper/wechat/v2/miniprogram/config"
	"github.com/silenceper/wechat/v2/miniprogram/content"
	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/miniprogram/encryptor"
	"github.com/silenceper/wechat/v2/miniprogram/express"
	"github.com/silenceper/wechat/v2/miniprogram/message"
	"github.com/silenceper/wechat/v2/miniprogram/minidrama"
	"github.com/silenceper/wechat/v2/miniprogram/ocr"
	"github.com/silenceper/wechat/v2/miniprogram/operation"
	"github.com/silenceper/wechat/v2/miniprogram/order"
	"github.com/silenceper/wechat/v2/miniprogram/privacy"
	"github.com/silenceper/wechat/v2/miniprogram/qrcode"
	"github.com/silenceper/wechat/v2/miniprogram/redpacketcover"
	"github.com/silenceper/wechat/v2/miniprogram/riskcontrol"
	"github.com/silenceper/wechat/v2/miniprogram/security"
	"github.com/silenceper/wechat/v2/miniprogram/shortlink"
	"github.com/silenceper/wechat/v2/miniprogram/subscribe"
	"github.com/silenceper/wechat/v2/miniprogram/tcb"
	"github.com/silenceper/wechat/v2/miniprogram/urllink"
	"github.com/silenceper/wechat/v2/miniprogram/urlscheme"
	"github.com/silenceper/wechat/v2/miniprogram/virtualpayment"
	"github.com/silenceper/wechat/v2/miniprogram/werun"
)

// MiniProgram 微信小程序相关 API
type MiniProgram struct {
	ctx *context.Context
}

// NewMiniProgram 实例化小程序 API
func NewMiniProgram(cfg *config.Config) *MiniProgram {
	var defaultAkHandle credential.AccessTokenContextHandle
	const cacheKeyPrefix = credential.CacheKeyMiniProgramPrefix
	if cfg.UseStableAK {
		defaultAkHandle = credential.NewStableAccessToken(cfg.AppID, cfg.AppSecret, cacheKeyPrefix, cfg.Cache)
	} else {
		defaultAkHandle = credential.NewDefaultAccessToken(cfg.AppID, cfg.AppSecret, cacheKeyPrefix, cfg.Cache)
	}
	ctx := &context.Context{
		Config:                   cfg,
		AccessTokenContextHandle: defaultAkHandle,
	}
	return &MiniProgram{ctx}
}

// SetAccessTokenHandle 自定义 access_token 获取方式
func (miniProgram *MiniProgram) SetAccessTokenHandle(accessTokenHandle credential.AccessTokenHandle) {
	miniProgram.ctx.AccessTokenContextHandle = credential.AccessTokenCompatibleHandle{
		AccessTokenHandle: accessTokenHandle,
	}
}

// SetAccessTokenContextHandle 自定义 access_token 获取方式
func (miniProgram *MiniProgram) SetAccessTokenContextHandle(accessTokenContextHandle credential.AccessTokenContextHandle) {
	miniProgram.ctx.AccessTokenContextHandle = accessTokenContextHandle
}

// GetContext get Context
func (miniProgram *MiniProgram) GetContext() *context.Context {
	return miniProgram.ctx
}

// GetEncryptor  小程序加解密
func (miniProgram *MiniProgram) GetEncryptor() *encryptor.Encryptor {
	return encryptor.NewEncryptor(miniProgram.ctx)
}

// GetAuth 登录/用户信息相关接口
func (miniProgram *MiniProgram) GetAuth() *auth.Auth {
	return auth.NewAuth(miniProgram.ctx)
}

// GetAnalysis 数据分析
func (miniProgram *MiniProgram) GetAnalysis() *analysis.Analysis {
	return analysis.NewAnalysis(miniProgram.ctx)
}

// GetBusiness 业务接口
func (miniProgram *MiniProgram) GetBusiness() *business.Business {
	return business.NewBusiness(miniProgram.ctx)
}

// GetPrivacy 小程序隐私协议相关 API
func (miniProgram *MiniProgram) GetPrivacy() *privacy.Privacy {
	return privacy.NewPrivacy(miniProgram.ctx)
}

// GetQRCode 小程序码相关 API
func (miniProgram *MiniProgram) GetQRCode() *qrcode.QRCode {
	return qrcode.NewQRCode(miniProgram.ctx)
}

// GetTcb 小程序云开发 API
func (miniProgram *MiniProgram) GetTcb() *tcb.Tcb {
	return tcb.NewTcb(miniProgram.ctx)
}

// GetSubscribe 小程序订阅消息
func (miniProgram *MiniProgram) GetSubscribe() *subscribe.Subscribe {
	return subscribe.NewSubscribe(miniProgram.ctx)
}

// GetCustomerMessage 客服消息接口
func (miniProgram *MiniProgram) GetCustomerMessage() *message.Manager {
	return message.NewCustomerMessageManager(miniProgram.ctx)
}

// GetWeRun 微信运动接口
func (miniProgram *MiniProgram) GetWeRun() *werun.WeRun {
	return werun.NewWeRun(miniProgram.ctx)
}

// GetContentSecurity 内容安全接口
func (miniProgram *MiniProgram) GetContentSecurity() *content.Content {
	return content.NewContent(miniProgram.ctx)
}

// GetURLLink 小程序 URL Link 接口
func (miniProgram *MiniProgram) GetURLLink() *urllink.URLLink {
	return urllink.NewURLLink(miniProgram.ctx)
}

// GetRiskControl 安全风控接口
func (miniProgram *MiniProgram) GetRiskControl() *riskcontrol.RiskControl {
	return riskcontrol.NewRiskControl(miniProgram.ctx)
}

// GetSecurity 内容安全接口
func (miniProgram *MiniProgram) GetSecurity() *security.Security {
	return security.NewSecurity(miniProgram.ctx)
}

// GetShortLink 小程序短链接口
func (miniProgram *MiniProgram) GetShortLink() *shortlink.ShortLink {
	return shortlink.NewShortLink(miniProgram.ctx)
}

// GetSURLScheme 小程序 URL Scheme 接口
func (miniProgram *MiniProgram) GetSURLScheme() *urlscheme.URLScheme {
	return urlscheme.NewURLScheme(miniProgram.ctx)
}

// GetOpenAPI openApi 管理接口
func (miniProgram *MiniProgram) GetOpenAPI() *openapi.OpenAPI {
	return openapi.NewOpenAPI(miniProgram.ctx)
}

// GetVirtualPayment 小程序虚拟支付
func (miniProgram *MiniProgram) GetVirtualPayment() *virtualpayment.VirtualPayment {
	return virtualpayment.NewVirtualPayment(miniProgram.ctx)
}

// GetMessageReceiver 获取消息推送接收器
func (miniProgram *MiniProgram) GetMessageReceiver() *message.PushReceiver {
	return message.NewPushReceiver(miniProgram.ctx)
}

// GetShipping 小程序发货信息管理服务
func (miniProgram *MiniProgram) GetShipping() *order.Shipping {
	return order.NewShipping(miniProgram.ctx)
}

// GetMiniDrama 小程序娱乐微短剧
func (miniProgram *MiniProgram) GetMiniDrama() *minidrama.MiniDrama {
	return minidrama.NewMiniDrama(miniProgram.ctx)
}

// GetRedPacketCover 小程序微信红包封面 API
func (miniProgram *MiniProgram) GetRedPacketCover() *redpacketcover.RedPacketCover {
	return redpacketcover.NewRedPacketCover(miniProgram.ctx)
}

// GetUpdatableMessage 小程序动态消息
func (miniProgram *MiniProgram) GetUpdatableMessage() *message.UpdatableMessage {
	return message.NewUpdatableMessage(miniProgram.ctx)
}

// GetOperation 小程序运维中心
func (miniProgram *MiniProgram) GetOperation() *operation.Operation {
	return operation.NewOperation(miniProgram.ctx)
}

// GetExpress 微信物流服务
func (miniProgram *MiniProgram) GetExpress() *express.Express {
	return express.NewExpress(miniProgram.ctx)
}

// GetOCR OCR接口
func (miniProgram *MiniProgram) GetOCR() *ocr.OCR {
	return ocr.NewOCR(miniProgram.ctx)
}
