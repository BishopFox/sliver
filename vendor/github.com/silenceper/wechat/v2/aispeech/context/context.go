package context

import (
	"github.com/silenceper/wechat/v2/aispeech/config"
	"github.com/silenceper/wechat/v2/credential"
)

// Context struct
type Context struct {
	*config.Config
	credential.AccessTokenContextHandle
}
