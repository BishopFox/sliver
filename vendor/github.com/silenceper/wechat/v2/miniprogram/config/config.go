// Package config 小程序 config 配置
package config

import (
	"github.com/silenceper/wechat/v2/cache"
)

// Config .config for 小程序
type Config struct {
	AppID          string `json:"app_id"`           // appid
	AppSecret      string `json:"app_secret"`       // appSecret
	AppKey         string `json:"app_key"`          // appKey
	OfferID        string `json:"offer_id"`         // offerId
	Token          string `json:"token"`            // token
	EncodingAESKey string `json:"encoding_aes_key"` // EncodingAESKey
	Cache          cache.Cache
	UseStableAK    bool // use the stable access_token
}
