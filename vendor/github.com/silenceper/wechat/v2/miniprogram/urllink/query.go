package urllink

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const queryURL = "https://api.weixin.qq.com/wxa/query_urllink?access_token=%s"

// ULQueryRequest 查询加密URLLink请求
type ULQueryRequest struct {
	URLLink   string `json:"url_link"`
	QueryType int    `json:"query_type"`
}

// ULQueryResult 返回的结果
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/url-link/urllink.query.html 返回值
type ULQueryResult struct {
	util.CommonError

	URLLinkInfo struct {
		Appid      string `json:"appid"`
		Path       string `json:"path"`
		Query      string `json:"query"`
		CreateTime int64  `json:"create_time"`
		ExpireTime int64  `json:"expire_time"`
		EnvVersion string `json:"env_version"`
		CloudBase  struct {
			Env           string `json:"env"`
			Domain        string `json:"domain"`
			Path          string `json:"path"`
			Query         string `json:"query"`
			ResourceAppid string `json:"resource_appid"`
		} `json:"cloud_base"`
	} `json:"url_link_info"`
	VisitOpenid string    `json:"visit_openid"`
	QuotaInfo   QuotaInfo `json:"quota_info"`
}

// QuotaInfo quota 配置
type QuotaInfo struct {
	RemainVisitQuota int64 `json:"remain_visit_quota"`
}

// Query 查询小程序 url_link 配置。
func (u *URLLink) Query(urlLink string) (*ULQueryResult, error) {
	return u.QueryWithType(&ULQueryRequest{URLLink: urlLink})
}

// QueryWithType 查询加密URLLink
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/qrcode-link/url-link/queryUrlLink.html
func (u *URLLink) QueryWithType(req *ULQueryRequest) (*ULQueryResult, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = u.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(queryURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &ULQueryResult{}
	err = util.DecodeWithError(response, result, "URLLink.Query")
	return result, err
}
