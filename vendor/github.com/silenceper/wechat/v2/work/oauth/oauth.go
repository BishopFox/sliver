package oauth

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/silenceper/wechat/v2/util"
	"github.com/silenceper/wechat/v2/work/context"
)

// Oauth auth
type Oauth struct {
	*context.Context
}

var (
	// oauthTargetURL 企业微信内跳转地址
	oauthTargetURL = "https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_base&state=STATE#wechat_redirect"
	// oauthTargetURL 企业微信内跳转地址(获取成员的详细信息)
	oauthTargetPrivateURL = "https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_privateinfo&agentid=%s&state=STATE#wechat_redirect"
	// oauthUserInfoURL 获取用户信息地址
	oauthUserInfoURL = "https://qyapi.weixin.qq.com/cgi-bin/user/getuserinfo?access_token=%s&code=%s"
	// oauthQrContentTargetURL 构造独立窗口登录二维码
	oauthQrContentTargetURL = "https://open.work.weixin.qq.com/wwopen/sso/qrConnect?appid=%s&agentid=%s&redirect_uri=%s&state=%s"
	// getUserInfoURL 获取访问用户身份&获取用户登录身份
	getUserInfoURL = "https://qyapi.weixin.qq.com/cgi-bin/auth/getuserinfo?access_token=%s&code=%s"
	// getUserDetailURL 获取访问用户敏感信息
	getUserDetailURL = "https://qyapi.weixin.qq.com/cgi-bin/auth/getuserdetail?access_token=%s"
)

// NewOauth new init oauth
func NewOauth(ctx *context.Context) *Oauth {
	return &Oauth{
		ctx,
	}
}

// GetTargetURL 获取授权地址
func (ctr *Oauth) GetTargetURL(callbackURL string) string {
	// url encode
	return fmt.Sprintf(
		oauthTargetURL,
		ctr.CorpID,
		url.QueryEscape(callbackURL),
	)
}

// GetTargetPrivateURL 获取个人信息授权地址
func (ctr *Oauth) GetTargetPrivateURL(callbackURL string, agentID string) string {
	// url encode
	return fmt.Sprintf(
		oauthTargetPrivateURL,
		ctr.CorpID,
		url.QueryEscape(callbackURL),
		agentID,
	)
}

// GetQrContentTargetURL 构造独立窗口登录二维码
func (ctr *Oauth) GetQrContentTargetURL(callbackURL string) string {
	// url encode
	return fmt.Sprintf(
		oauthQrContentTargetURL,
		ctr.CorpID,
		ctr.AgentID,
		url.QueryEscape(callbackURL),
		util.RandomStr(16),
	)
}

// ResUserInfo 返回得用户信息
type ResUserInfo struct {
	util.CommonError
	// 当用户为企业成员时返回
	UserID   string `json:"UserId"`
	DeviceID string `json:"DeviceId"`
	// 非企业成员授权时返回
	OpenID         string `json:"OpenId"`
	ExternalUserID string `json:"external_userid"`
}

// UserFromCode 根据code获取用户信息
func (ctr *Oauth) UserFromCode(code string) (result ResUserInfo, err error) {
	var accessToken string
	if accessToken, err = ctr.GetAccessToken(); err != nil {
		return
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(oauthUserInfoURL, accessToken, code)); err != nil {
		return
	}
	err = json.Unmarshal(response, &result)
	if result.ErrCode != 0 {
		err = fmt.Errorf("GetUserAccessToken error : errcode=%v , errmsg=%v", result.ErrCode, result.ErrMsg)
		return
	}
	return
}

// GetUserInfoResponse 获取访问用户身份&获取用户登录身份响应
type GetUserInfoResponse struct {
	util.CommonError
	UserID         string `json:"userid"`
	UserTicket     string `json:"user_ticket"`
	OpenID         string `json:"openid"`
	ExternalUserID string `json:"external_userid"`
}

// GetUserInfo 获取访问用户身份&获取用户登录身份
// @see https://developer.work.weixin.qq.com/document/path/90213 获取访问用户身份
// @see https://developer.work.weixin.qq.com/document/path/98176 获取用户登录身份
func (ctr *Oauth) GetUserInfo(code string) (*GetUserInfoResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = ctr.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getUserInfoURL, accessToken, code)); err != nil {
		return nil, err
	}
	result := &GetUserInfoResponse{}
	err = util.DecodeWithError(response, result, "GetUserInfo")
	return result, err
}

// GetUserDetailRequest 获取访问用户敏感信息请求
type GetUserDetailRequest struct {
	UserTicket string `json:"user_ticket"`
}

// GetUserDetailResponse 获取访问用户敏感信息响应
type GetUserDetailResponse struct {
	util.CommonError
	UserID  string `json:"userid"`
	Gender  string `json:"gender"`
	Avatar  string `json:"avatar"`
	QrCode  string `json:"qr_code"`
	Mobile  string `json:"mobile"`
	Email   string `json:"email"`
	BizMail string `json:"biz_mail"`
	Address string `json:"address"`
}

// GetUserDetail 获取访问用户敏感信息
// @see https://developer.work.weixin.qq.com/document/path/95833
func (ctr *Oauth) GetUserDetail(req *GetUserDetailRequest) (*GetUserDetailResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = ctr.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getUserDetailURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetUserDetailResponse{}
	err = util.DecodeWithError(response, result, "GetUserDetail")
	return result, err
}
