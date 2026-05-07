package auth

import (
	context2 "context"
	"encoding/json"
	"fmt"

	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	// code2SessionURL 小程序登录
	code2SessionURL = "https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code"
	// checkEncryptedDataURL 检查加密信息
	checkEncryptedDataURL = "https://api.weixin.qq.com/wxa/business/checkencryptedmsg?access_token=%s"
	// getPhoneNumber 获取手机号
	getPhoneNumber = "https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s"
	// checkSessionURL 检验登录态
	checkSessionURL = "https://api.weixin.qq.com/wxa/checksession?access_token=%s&signature=%s&openid=%s&sig_method=hmac_sha256"
	// resetUserSessionKeyURL 重置登录态
	resetUserSessionKeyURL = "https://api.weixin.qq.com/wxa/resetusersessionkey?access_token=%s&signature=%s&openid=%s&sig_method=hmac_sha256"
	// getPluginOpenPIDURL 获取插件用户openPID
	getPluginOpenPIDURL = "https://api.weixin.qq.com/wxa/getpluginopenpid?access_token=%s"
	// getPaidUnionIDURL 支付后获取 UnionID
	getPaidUnionIDURL = "https://api.weixin.qq.com/wxa/getpaidunionid"
	// getUserEncryptKeyURL 获取用户encryptKey
	getUserEncryptKeyURL = "https://api.weixin.qq.com/wxa/business/getuserencryptkey?access_token=%s&signature=%s&openid=%s&sig_method=hmac_sha256"
)

// Auth 登录/用户信息
type Auth struct {
	*context.Context
}

// NewAuth new auth
func NewAuth(ctx *context.Context) *Auth {
	return &Auth{ctx}
}

// ResCode2Session 登录凭证校验的返回结果
type ResCode2Session struct {
	util.CommonError

	OpenID     string `json:"openid"`      // 用户唯一标识
	SessionKey string `json:"session_key"` // 会话密钥
	UnionID    string `json:"unionid"`     // 用户在开放平台的唯一标识符，在满足UnionID下发条件的情况下会返回
}

// RspCheckEncryptedData .
type RspCheckEncryptedData struct {
	util.CommonError

	Vaild      bool   `json:"vaild"`       // 是否是合法的数据
	CreateTime uint64 `json:"create_time"` // 加密数据生成的时间戳
}

// Code2Session 登录凭证校验。
func (auth *Auth) Code2Session(jsCode string) (result ResCode2Session, err error) {
	return auth.Code2SessionContext(context2.Background(), jsCode)
}

// Code2SessionContext 登录凭证校验。
func (auth *Auth) Code2SessionContext(ctx context2.Context, jsCode string) (result ResCode2Session, err error) {
	var response []byte
	if response, err = util.HTTPGetContext(ctx, fmt.Sprintf(code2SessionURL, auth.AppID, auth.AppSecret, jsCode)); err != nil {
		return
	}
	if err = json.Unmarshal(response, &result); err != nil {
		return
	}
	if result.ErrCode != 0 {
		err = fmt.Errorf("Code2Session error : errcode=%v , errmsg=%v", result.ErrCode, result.ErrMsg)
		return
	}
	return
}

type (
	// GetPaidUnionIDRequest 支付后获取UnionID请求
	GetPaidUnionIDRequest struct {
		OpenID        string `json:"openid"`
		TransactionID string `json:"transaction_id,omitempty"`
		MchID         string `json:"mch_id,omitempty"`
		OutTradeNo    string `json:"out_trade_no,omitempty"`
	}

	// GetPaidUnionIDResponse 支付后获取UnionID响应
	GetPaidUnionIDResponse struct {
		util.CommonError
		UnionID string `json:"unionid"`
	}
)

// GetPaidUnionID 用户支付完成后，获取该用户的 UnionId，无需用户授权
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-info/basic-info/getPaidUnionid.html
func (auth *Auth) GetPaidUnionID(req *GetPaidUnionIDRequest) (string, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = auth.GetAccessToken(); err != nil {
		return "", err
	}
	var url string
	if req.TransactionID != "" {
		url = fmt.Sprintf("%s?access_token=%s&openid=%s&transaction_id=%s", getPaidUnionIDURL, accessToken, req.OpenID, req.TransactionID)
	} else {
		url = fmt.Sprintf("%s?access_token=%s&openid=%s&mch_id=%s&out_trade_no=%s", getPaidUnionIDURL, accessToken, req.OpenID, req.MchID, req.OutTradeNo)
	}
	var response []byte
	if response, err = util.HTTPGet(url); err != nil {
		return "", err
	}
	result := &GetPaidUnionIDResponse{}
	err = util.DecodeWithError(response, result, "GetPaidUnionID")
	return result.UnionID, err
}

// CheckEncryptedData .检查加密信息是否由微信生成（当前只支持手机号加密数据），只能检测最近3天生成的加密数据
func (auth *Auth) CheckEncryptedData(encryptedMsgHash string) (result RspCheckEncryptedData, err error) {
	return auth.CheckEncryptedDataContext(context2.Background(), encryptedMsgHash)
}

// CheckEncryptedDataContext .检查加密信息是否由微信生成（当前只支持手机号加密数据），只能检测最近3天生成的加密数据
func (auth *Auth) CheckEncryptedDataContext(ctx context2.Context, encryptedMsgHash string) (result RspCheckEncryptedData, err error) {
	var response []byte
	var (
		at string
	)
	if at, err = auth.GetAccessTokenContext(ctx); err != nil {
		return
	}

	// 由于GetPhoneNumberContext需要传入JSON，所以HTTPPostContext入参改为[]byte
	if response, err = util.HTTPPostContext(ctx, fmt.Sprintf(checkEncryptedDataURL, at), []byte("encrypted_msg_hash="+encryptedMsgHash), nil); err != nil {
		return
	}
	if err = util.DecodeWithError(response, &result, "CheckEncryptedDataAuth"); err != nil {
		return
	}
	return
}

// GetPhoneNumberResponse 新版获取用户手机号响应结构体
type GetPhoneNumberResponse struct {
	util.CommonError

	PhoneInfo PhoneInfo `json:"phone_info"`
}

// PhoneInfo 获取用户手机号内容
type PhoneInfo struct {
	PhoneNumber     string `json:"phoneNumber"`     // 用户绑定的手机号
	PurePhoneNumber string `json:"purePhoneNumber"` // 没有区号的手机号
	CountryCode     string `json:"countryCode"`     // 区号
	WaterMark       struct {
		Timestamp int64  `json:"timestamp"`
		AppID     string `json:"appid"`
	} `json:"watermark"` // 数据水印
}

// GetPhoneNumberContext 小程序通过code获取用户手机号
func (auth *Auth) GetPhoneNumberContext(ctx context2.Context, code string) (*GetPhoneNumberResponse, error) {
	var response []byte
	var (
		at  string
		err error
	)
	if at, err = auth.GetAccessTokenContext(ctx); err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"code": code,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	header := map[string]string{"Content-Type": "application/json;charset=utf-8"}
	if response, err = util.HTTPPostContext(ctx, fmt.Sprintf(getPhoneNumber, at), bodyBytes, header); err != nil {
		return nil, err
	}

	var result GetPhoneNumberResponse
	err = util.DecodeWithError(response, &result, "phonenumber.getPhoneNumber")
	return &result, err
}

// GetPhoneNumber 小程序通过code获取用户手机号
func (auth *Auth) GetPhoneNumber(code string) (*GetPhoneNumberResponse, error) {
	return auth.GetPhoneNumberContext(context2.Background(), code)
}

// CheckSession 检验登录态
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/checkSessionKey.html
func (auth *Auth) CheckSession(signature, openID string) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = auth.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(checkSessionURL, accessToken, signature, openID)); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "CheckSession")
}

// ResetUserSessionKeyResponse 重置登录态响应
type ResetUserSessionKeyResponse struct {
	util.CommonError
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
}

// ResetUserSessionKey 重置登录态
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/ResetUserSessionKey.html
func (auth *Auth) ResetUserSessionKey(signature, openID string) (*ResetUserSessionKeyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = auth.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(resetUserSessionKeyURL, accessToken, signature, openID)); err != nil {
		return nil, err
	}
	result := &ResetUserSessionKeyResponse{}
	err = util.DecodeWithError(response, result, "ResetUserSessionKey")
	return result, err
}

type (
	// GetPluginOpenPIDRequest 获取插件用户openPID请求
	GetPluginOpenPIDRequest struct {
		Code string `json:"code"`
	}

	// GetPluginOpenPIDResponse 获取插件用户openPID响应
	GetPluginOpenPIDResponse struct {
		util.CommonError
		OpenPID string `json:"openpid"`
	}
)

// GetPluginOpenPID 获取插件用户openPID
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-info/basic-info/getPluginOpenPId.html
func (auth *Auth) GetPluginOpenPID(code string) (string, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = auth.GetAccessToken(); err != nil {
		return "", err
	}
	req := &GetPluginOpenPIDRequest{
		Code: code,
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getPluginOpenPIDURL, accessToken), req); err != nil {
		return "", err
	}
	result := &GetPluginOpenPIDResponse{}
	err = util.DecodeWithError(response, result, "GetPluginOpenPID")
	return result.OpenPID, err
}

// GetUserEncryptKeyResponse 获取用户encryptKey响应
type GetUserEncryptKeyResponse struct {
	util.CommonError
	KeyInfoList []KeyInfo `json:"key_info_list"`
}

// KeyInfo 用户最近三次的加密key
type KeyInfo struct {
	EncryptKey string `json:"encrypt_key"`
	Version    int64  `json:"version"`
	ExpireIn   int64  `json:"expire_in"`
	Iv         string `json:"iv"`
	CreateTime int64  `json:"create_time"`
}

// GetUserEncryptKey 获取用户encryptKey
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-info/internet/getUserEncryptKey.html
func (auth *Auth) GetUserEncryptKey(signature, openID string) (*GetUserEncryptKeyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = auth.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getUserEncryptKeyURL, accessToken, signature, openID)); err != nil {
		return nil, err
	}
	result := &GetUserEncryptKeyResponse{}
	err = util.DecodeWithError(response, result, "GetUserEncryptKey")
	return result, err
}
