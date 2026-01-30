package express

import (
	"github.com/silenceper/wechat/v2/miniprogram/context"
)

// Express 微信物流服务
// https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/express/business/introduction.html
type Express struct {
	*context.Context
}

// NewExpress init
func NewExpress(ctx *context.Context) *Express {
	return &Express{ctx}
}
