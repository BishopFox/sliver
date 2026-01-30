package redpacketcover

import (
	"fmt"

	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	getRedPacketCoverURL = "https://api.weixin.qq.com/redpacketcover/wxapp/cover_url/get_by_token?access_token=%s"
)

// RedPacketCover struct
type RedPacketCover struct {
	*context.Context
}

// NewRedPacketCover 实例
func NewRedPacketCover(context *context.Context) *RedPacketCover {
	redPacketCover := new(RedPacketCover)
	redPacketCover.Context = context
	return redPacketCover
}

// GetRedPacketCoverRequest 获取微信红包封面参数
type GetRedPacketCoverRequest struct {
	// openid 可领取用户的openid
	OpenID string `json:"openid"`
	// ctoken 在红包封面平台获取发放ctoken（需要指定可以发放的appid）
	CToken string `json:"ctoken"`
}

// GetRedPacketCoverResp 获取微信红包封面
type GetRedPacketCoverResp struct {
	util.CommonError
	Data struct {
		URL string `json:"url"`
	} `json:"data"` // 唯一请求标识
}

// GetRedPacketCoverURL 获得指定用户可以领取的红包封面链接。获取参数ctoken参考微信红包封面开放平台
// 文档地址： https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/red-packet-cover/getRedPacketCoverUrl.html
func (cover *RedPacketCover) GetRedPacketCoverURL(coderParams GetRedPacketCoverRequest) (res GetRedPacketCoverResp, err error) {
	accessToken, err := cover.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(getRedPacketCoverURL, accessToken)
	response, err := util.PostJSON(uri, coderParams)
	if err != nil {
		return
	}

	// 使用通用方法返回错误
	err = util.DecodeWithError(response, &res, "GetRedPacketCoverURL")
	return
}
