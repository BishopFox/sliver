package config

import "github.com/silenceper/wechat/v2/cache"

const defaultBaseURL = "https://openaiapi.weixin.qq.com"

// Config for 微信智能对话.
type Config struct {
	AppID   string `json:"app_id"`
	Token   string `json:"token"`
	AESKey  string `json:"aes_key"`
	Account string `json:"account"`
	BaseURL string `json:"base_url"`
	Cache   cache.Cache
}

// GetBaseURL returns the API base URL.
func (cfg *Config) GetBaseURL() string {
	if cfg.BaseURL == "" {
		return defaultBaseURL
	}
	return cfg.BaseURL
}
