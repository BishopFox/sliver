package urlscheme

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	querySchemeURL = "https://api.weixin.qq.com/wxa/queryscheme?access_token=%s"
)

// QueryScheme 获取小程序访问scheme
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/url-scheme/urlscheme.query.html#参数
type QueryScheme struct {
	// 小程序 scheme 码
	Scheme    string `json:"scheme"`
	QueryType int    `json:"query_type"`
}

// SchemeInfo scheme 配置
type SchemeInfo struct {
	// 小程序 appid。
	AppID string `json:"appid"`
	// 小程序页面路径。
	Path string `json:"path"`
	// 小程序页面query。
	Query string `json:"query"`
	// 创建时间，为 Unix 时间戳。
	CreateTime int64 `json:"create_time"`
	// 到期失效时间，为 Unix 时间戳，0 表示永久生效
	ExpireTime int64 `json:"expire_time"`
	// 要打开的小程序版本。正式版为"release"，体验版为"trial"，开发版为"develop"。
	EnvVersion EnvVersion `json:"env_version"`
}

// QuotaInfo quota 配置
type QuotaInfo struct {
	RemainVisitQuota int64 `json:"remain_visit_quota"`
}

// ResQueryScheme 返回结构体
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/url-scheme/urlscheme.query.html#参数
type ResQueryScheme struct {
	// 通用错误
	util.CommonError
	// scheme 配置
	SchemeInfo SchemeInfo `json:"scheme_info"`
	// 访问该链接的openid，没有用户访问过则为空字符串
	VisitOpenid string    `json:"visit_openid"`
	QuotaInfo   QuotaInfo `json:"quota_info"`
}

// QueryScheme 查询小程序 scheme 码
func (u *URLScheme) QueryScheme(querySchemeParams QueryScheme) (schemeInfo SchemeInfo, visitOpenid string, err error) {
	res, err := u.QuerySchemeWithRes(querySchemeParams)
	if err != nil {
		return
	}
	return res.SchemeInfo, res.VisitOpenid, err
}

// QuerySchemeWithRes 查询scheme码
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/qrcode-link/url-scheme/queryScheme.html
func (u *URLScheme) QuerySchemeWithRes(req QueryScheme) (*ResQueryScheme, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = u.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(querySchemeURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &ResQueryScheme{}
	err = util.DecodeWithError(response, result, "QueryScheme")
	return result, err
}
