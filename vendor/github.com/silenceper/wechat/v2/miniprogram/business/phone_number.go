package business

import (
	"context"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	getPhoneNumberURL = "https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s"
)

// GetPhoneNumberRequest 获取手机号请求
type GetPhoneNumberRequest struct {
	Code string `json:"code"` // 手机号获取凭证
}

// PhoneInfo 手机号信息
type PhoneInfo struct {
	PhoneNumber     string `json:"phoneNumber"`     // 用户绑定的手机号（国外手机号会有区号）
	PurePhoneNumber string `json:"purePhoneNumber"` // 没有区号的手机号
	CountryCode     string `json:"countryCode"`     // 区号
	Watermark       struct {
		AppID     string `json:"appid"`     // 小程序appid
		Timestamp int64  `json:"timestamp"` // 用户获取手机号操作的时间戳
	} `json:"watermark"`
}

// GetPhoneNumber code换取用户手机号。 每个code只能使用一次，code的有效期为5min
func (business *Business) GetPhoneNumber(in *GetPhoneNumberRequest) (info PhoneInfo, err error) {
	return business.GetPhoneNumberWithContext(context.Background(), in)
}

// GetPhoneNumberWithContext 利用context将code换取用户手机号。 每个code只能使用一次，code的有效期为5min
func (business *Business) GetPhoneNumberWithContext(ctx context.Context, in *GetPhoneNumberRequest) (info PhoneInfo, err error) {
	accessToken, err := business.GetAccessTokenContext(ctx)
	if err != nil {
		return
	}

	uri := fmt.Sprintf(getPhoneNumberURL, accessToken)
	response, err := util.PostJSONContext(ctx, uri, in)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	var resp struct {
		util.CommonError
		PhoneInfo PhoneInfo `json:"phone_info"`
	}
	err = util.DecodeWithError(response, &resp, "business.GetPhoneNumber")
	return resp.PhoneInfo, err
}
