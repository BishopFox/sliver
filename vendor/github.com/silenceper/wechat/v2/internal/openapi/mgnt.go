package openapi

import (
	"errors"
	"fmt"

	"github.com/silenceper/wechat/v2/domain/openapi"
	mpContext "github.com/silenceper/wechat/v2/miniprogram/context"
	ocContext "github.com/silenceper/wechat/v2/officialaccount/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	clearQuotaURL            = "https://api.weixin.qq.com/cgi-bin/clear_quota"       // 重置API调用次数
	getAPIQuotaURL           = "https://api.weixin.qq.com/cgi-bin/openapi/quota/get" // 查询API调用额度
	getRidInfoURL            = "https://api.weixin.qq.com/cgi-bin/openapi/rid/get"   // 查询rid信息
	clearQuotaByAppSecretURL = "https://api.weixin.qq.com/cgi-bin/clear_quota/v2"    // 使用AppSecret重置 API 调用次数
)

// OpenAPI openApi管理
type OpenAPI struct {
	ctx interface{}
}

// NewOpenAPI 实例化
func NewOpenAPI(ctx interface{}) *OpenAPI {
	return &OpenAPI{ctx: ctx}
}

// ClearQuota 重置API调用次数
// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/openApi-mgnt/clearQuota.html
func (o *OpenAPI) ClearQuota() error {
	appID, _, err := o.getAppIDAndSecret()
	if err != nil {
		return err
	}

	var payload = struct {
		AppID string `json:"appid"`
	}{
		AppID: appID,
	}
	res, err := o.doPostRequest(clearQuotaURL, payload)
	if err != nil {
		return err
	}

	return util.DecodeWithCommonError(res, "ClearQuota")
}

// GetAPIQuota 查询API调用额度
// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/openApi-mgnt/getApiQuota.html
func (o *OpenAPI) GetAPIQuota(params openapi.GetAPIQuotaParams) (quota openapi.APIQuota, err error) {
	res, err := o.doPostRequest(getAPIQuotaURL, params)
	if err != nil {
		return
	}

	err = util.DecodeWithError(res, &quota, "GetAPIQuota")
	return
}

// GetRidInfo 查询rid信息
// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/openApi-mgnt/getRidInfo.html
func (o *OpenAPI) GetRidInfo(params openapi.GetRidInfoParams) (r openapi.RidInfo, err error) {
	res, err := o.doPostRequest(getRidInfoURL, params)
	if err != nil {
		return
	}

	err = util.DecodeWithError(res, &r, "GetRidInfo")
	return
}

// ClearQuotaByAppSecret 使用AppSecret重置 API 调用次数
// https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/openApi-mgnt/clearQuotaByAppSecret.html
func (o *OpenAPI) ClearQuotaByAppSecret() error {
	id, secret, err := o.getAppIDAndSecret()
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("%s?appid=%s&appsecret=%s", clearQuotaByAppSecretURL, id, secret)
	res, err := util.HTTPPost(uri, "")
	if err != nil {
		return err
	}

	return util.DecodeWithCommonError(res, "ClearQuotaByAppSecret")
}

// 获取 AppID 和 AppSecret
func (o *OpenAPI) getAppIDAndSecret() (string, string, error) {
	switch o.ctx.(type) {
	case *mpContext.Context:
		c := o.ctx.(*mpContext.Context)
		return c.AppID, c.AppSecret, nil
	case *ocContext.Context:
		c := o.ctx.(*ocContext.Context)
		return c.AppID, c.AppSecret, nil
	default:
		return "", "", errors.New("invalid context type")
	}
}

// 获取 AccessToken
func (o *OpenAPI) getAccessToken() (string, error) {
	switch o.ctx.(type) {
	case *mpContext.Context:
		return o.ctx.(*mpContext.Context).GetAccessToken()
	case *ocContext.Context:
		return o.ctx.(*ocContext.Context).GetAccessToken()
	default:
		return "", errors.New("invalid context type")
	}
}

// 创建 POST 请求
func (o *OpenAPI) doPostRequest(uri string, payload interface{}) ([]byte, error) {
	ak, err := o.getAccessToken()
	if err != nil {
		return nil, err
	}

	uri = fmt.Sprintf("%s?access_token=%s", uri, ak)
	return util.PostJSON(uri, payload)
}
