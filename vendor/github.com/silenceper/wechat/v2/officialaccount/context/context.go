package context

import (
	"github.com/silenceper/wechat/v2/credential"
	"github.com/silenceper/wechat/v2/officialaccount/config"
)

// Context struct
type Context struct {
	*config.Config
	credential.AccessTokenHandle
}
