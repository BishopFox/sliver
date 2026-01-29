package urlscheme

import (
	"fmt"

	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

// URLScheme 小程序 URL Scheme
type URLScheme struct {
	*context.Context
}

// NewURLScheme 实例化
func NewURLScheme(ctx *context.Context) *URLScheme {
	return &URLScheme{Context: ctx}
}

const (
	// generateURL 获取加密scheme码
	generateURL = "https://api.weixin.qq.com/wxa/generatescheme"
	// generateNFCURL 获取 NFC 的小程序 scheme
	generateNFCURL = "https://api.weixin.qq.com/wxa/generatenfcscheme?access_token=%s"
)

// TExpireType 失效类型 (指定时间戳/指定间隔)
type TExpireType int

// EnvVersion 要打开的小程序版本
type EnvVersion string

const (
	// ExpireTypeTime 指定时间戳后失效
	ExpireTypeTime TExpireType = 0
	// ExpireTypeInterval 间隔指定天数后失效
	ExpireTypeInterval TExpireType = 1

	// EnvVersionRelease 正式版为"release"
	EnvVersionRelease EnvVersion = "release"
	// EnvVersionTrial 体验版为"trial"
	EnvVersionTrial EnvVersion = "trial"
	// EnvVersionDevelop 开发版为"develop"
	EnvVersionDevelop EnvVersion = "develop"
)

// JumpWxa 跳转到的目标小程序信息
type JumpWxa struct {
	Path  string `json:"path"`
	Query string `json:"query"`
	// envVersion 要打开的小程序版本。正式版为 "release"，体验版为 "trial"，开发版为 "develop"
	EnvVersion EnvVersion `json:"env_version,omitempty"`
}

// USParams 请求参数
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/url-scheme/urlscheme.generate.html#请求参数
type USParams struct {
	JumpWxa        *JumpWxa    `json:"jump_wxa,omitempty"`
	ExpireType     TExpireType `json:"expire_type,omitempty"`
	ExpireTime     int64       `json:"expire_time,omitempty"`
	ExpireInterval int         `json:"expire_interval,omitempty"`
	IsExpire       bool        `json:"is_expire,omitempty"`
	ModelID        string      `json:"model_id,omitempty"`
	Sn             string      `json:"sn,omitempty"`
}

// USResult 返回的结果
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/url-scheme/urlscheme.generate.html#返回值
type USResult struct {
	util.CommonError

	OpenLink string `json:"openlink"`
}

// Generate 生成url link
func (u *URLScheme) Generate(params *USParams) (string, error) {
	accessToken, err := u.GetAccessToken()
	if err != nil {
		return "", err
	}

	uri := fmt.Sprintf("%s?access_token=%s", generateURL, accessToken)
	response, err := util.PostJSON(uri, params)
	if err != nil {
		return "", err
	}
	var resp USResult
	err = util.DecodeWithError(response, &resp, "URLScheme.Generate")
	return resp.OpenLink, err
}

// GenerateNFC 获取 NFC 的小程序 scheme
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/qrcode-link/url-scheme/generateNFCScheme.html
func (u *URLScheme) GenerateNFC(params *USParams) (string, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = u.GetAccessToken(); err != nil {
		return "", err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(generateNFCURL, accessToken), params); err != nil {
		return "", err
	}
	result := &USResult{}
	err = util.DecodeWithError(response, result, "URLScheme.GenerateNFC")
	return result.OpenLink, err
}
