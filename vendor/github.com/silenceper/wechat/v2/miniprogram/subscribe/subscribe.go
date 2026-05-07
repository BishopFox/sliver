package subscribe

import (
	"encoding/json"
	"fmt"

	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	// 发送订阅消息
	// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/subscribe-message/subscribeMessage.send.html
	subscribeSendURL = "https://api.weixin.qq.com/cgi-bin/message/subscribe/send"
	// 获取当前帐号下的个人模板列表
	// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/subscribe-message/subscribeMessage.getTemplateList.html
	getTemplateURL = "https://api.weixin.qq.com/wxaapi/newtmpl/gettemplate"
	// 添加订阅模板
	// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/subscribe-message/subscribeMessage.addTemplate.html
	addTemplateURL = "https://api.weixin.qq.com/wxaapi/newtmpl/addtemplate"
	// 删除私有模板
	// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/subscribe-message/subscribeMessage.deleteTemplate.html
	delTemplateURL = "https://api.weixin.qq.com/wxaapi/newtmpl/deltemplate"
	// 统一服务消息
	// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/uniform-message/uniformMessage.send.html
	uniformMessageSend = "https://api.weixin.qq.com/cgi-bin/message/wxopen/template/uniform_send"
	// getCategoryURL 获取类目
	getCategoryURL = "https://api.weixin.qq.com/wxaapi/newtmpl/getcategory?access_token=%s"
	// getPubTemplateKeyWordsByIDURL 获取关键词列表
	getPubTemplateKeyWordsByIDURL = "https://api.weixin.qq.com/wxaapi/newtmpl/getpubtemplatekeywords?access_token=%s&tid=%s"
	// getPubTemplateTitleListURL 获取所属类目下的公共模板
	getPubTemplateTitleListURL = "https://api.weixin.qq.com/wxaapi/newtmpl/getpubtemplatetitles?access_token=%s&ids=%s&start=%d&limit=%d"
	// setUserNotifyURL 激活与更新服务卡片
	setUserNotifyURL = "https://api.weixin.qq.com/wxa/set_user_notify?access_token=%s"
	// setUserNotifyExtURL 更新服务卡片扩展信息
	setUserNotifyExtURL = "https://api.weixin.qq.com/wxa/set_user_notifyext?access_token=%s"
	// getUserNotifyURL 查询服务卡片状态
	getUserNotifyURL = "https://api.weixin.qq.com/wxa/get_user_notify?access_token=%s"
)

// Subscribe 订阅消息
type Subscribe struct {
	*context.Context
}

// NewSubscribe 实例化
func NewSubscribe(ctx *context.Context) *Subscribe {
	return &Subscribe{Context: ctx}
}

// Message 订阅消息请求参数
type Message struct {
	ToUser           string               `json:"touser"`            // 必选，接收者（用户）的 openid
	TemplateID       string               `json:"template_id"`       // 必选，所需下发的订阅模板id
	Page             string               `json:"page"`              // 可选，点击模板卡片后的跳转页面，仅限本小程序内的页面。支持带参数,（示例index?foo=bar）。该字段不填则模板无跳转。
	Data             map[string]*DataItem `json:"data"`              // 必选, 模板内容
	MiniprogramState string               `json:"miniprogram_state"` // 可选，跳转小程序类型：developer为开发版；trial为体验版；formal为正式版；默认为正式版
	Lang             string               `json:"lang"`              // 入小程序查看”的语言类型，支持zh_CN(简体中文)、en_US(英文)、zh_HK(繁体中文)、zh_TW(繁体中文)，默认为zh_CN
}

// DataItem 模版内某个 .DATA 的值
type DataItem struct {
	Value interface{} `json:"value"`
	Color string      `json:"color"`
}

// TemplateItem template item
type TemplateItem struct {
	PriTmplID            string             `json:"priTmplId"`
	Title                string             `json:"title"`
	Content              string             `json:"content"`
	Example              string             `json:"example"`
	Type                 int64              `json:"type"`
	KeywordEnumValueList []KeywordEnumValue `json:"keywordEnumValueList"`
}

// KeywordEnumValue 枚举参数值范围
type KeywordEnumValue struct {
	EnumValueList []string `json:"enumValueList"`
	KeywordCode   string   `json:"keywordCode"`
}

// TemplateList template list
type TemplateList struct {
	util.CommonError
	Data []TemplateItem `json:"data"`
}

// resTemplateSend 发送获取 msg id
type resTemplateSend struct {
	util.CommonError

	MsgID int64 `json:"msgid"`
}

// Send 发送订阅消息
func (s *Subscribe) Send(msg *Message) (err error) {
	var accessToken string
	accessToken, err = s.GetAccessToken()
	if err != nil {
		return
	}
	uri := fmt.Sprintf("%s?access_token=%s", subscribeSendURL, accessToken)
	response, err := util.PostJSON(uri, msg)
	if err != nil {
		return
	}
	return util.DecodeWithCommonError(response, "Send")
}

// SendGetMsgID 发送订阅消息返回 msgid
func (s *Subscribe) SendGetMsgID(msg *Message) (msgID int64, err error) {
	var accessToken string
	accessToken, err = s.GetAccessToken()
	if err != nil {
		return
	}
	uri := fmt.Sprintf("%s?access_token=%s", subscribeSendURL, accessToken)
	response, err := util.PostJSON(uri, msg)
	if err != nil {
		return
	}

	var result resTemplateSend
	if err = json.Unmarshal(response, &result); err != nil {
		return
	}
	if result.ErrCode != 0 {
		err = fmt.Errorf("template msg send error : errcode=%v , errmsg=%v", result.ErrCode, result.ErrMsg)
		return
	}

	msgID = result.MsgID

	return
}

// ListTemplates 获取当前帐号下的个人模板列表
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/subscribe-message/subscribeMessage.getTemplateList.html
func (s *Subscribe) ListTemplates() (*TemplateList, error) {
	accessToken, err := s.GetAccessToken()
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("%s?access_token=%s", getTemplateURL, accessToken)
	response, err := util.HTTPGet(uri)
	if err != nil {
		return nil, err
	}
	templateList := TemplateList{}
	err = util.DecodeWithError(response, &templateList, "ListTemplates")
	if err != nil {
		return nil, err
	}
	return &templateList, nil
}

// UniformMessage 统一服务消息
type UniformMessage struct {
	ToUser           string `json:"touser"`
	WeappTemplateMsg struct {
		TemplateID      string               `json:"template_id"`
		Page            string               `json:"page"`
		FormID          string               `json:"form_id"`
		Data            map[string]*DataItem `json:"data"`
		EmphasisKeyword string               `json:"emphasis_keyword"`
	} `json:"weapp_template_msg"`
	MpTemplateMsg struct {
		Appid       string `json:"appid"`
		TemplateID  string `json:"template_id"`
		URL         string `json:"url"`
		Miniprogram struct {
			Appid    string `json:"appid"`
			Pagepath string `json:"pagepath"`
		} `json:"miniprogram"`
		Data map[string]*DataItem `json:"data"`
	} `json:"mp_template_msg"`
}

// UniformSend 发送统一服务消息
func (s *Subscribe) UniformSend(msg *UniformMessage) (err error) {
	var accessToken string
	accessToken, err = s.GetAccessToken()
	if err != nil {
		return
	}
	uri := fmt.Sprintf("%s?access_token=%s", uniformMessageSend, accessToken)
	response, err := util.PostJSON(uri, msg)
	if err != nil {
		return
	}
	return util.DecodeWithCommonError(response, "UniformSend")
}

type resSubscribeAdd struct {
	util.CommonError

	TemplateID string `json:"priTmplId"`
}

// Add 添加订阅消息模板
func (s *Subscribe) Add(ShortID string, kidList []int, sceneDesc string) (templateID string, err error) {
	var accessToken string
	accessToken, err = s.GetAccessToken()
	if err != nil {
		return
	}
	var msg = struct {
		TemplateIDShort string `json:"tid"`
		SceneDesc       string `json:"sceneDesc"`
		KidList         []int  `json:"kidList"`
	}{TemplateIDShort: ShortID, SceneDesc: sceneDesc, KidList: kidList}
	uri := fmt.Sprintf("%s?access_token=%s", addTemplateURL, accessToken)
	var response []byte
	response, err = util.PostJSON(uri, msg)
	if err != nil {
		return
	}
	var result resSubscribeAdd
	err = util.DecodeWithError(response, &result, "AddSubscribe")
	return result.TemplateID, err
}

// Delete 删除私有模板
func (s *Subscribe) Delete(templateID string) (err error) {
	var accessToken string
	accessToken, err = s.GetAccessToken()
	if err != nil {
		return
	}
	var msg = struct {
		TemplateID string `json:"priTmplId"`
	}{TemplateID: templateID}
	uri := fmt.Sprintf("%s?access_token=%s", delTemplateURL, accessToken)
	var response []byte
	response, err = util.PostJSON(uri, msg)
	if err != nil {
		return
	}
	return util.DecodeWithCommonError(response, "DeleteSubscribe")
}

// GetCategoryResponse 获取类目响应
type GetCategoryResponse struct {
	util.CommonError
	Data []Category `json:"data"`
}

// Category 类目
type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GetCategory 获取类目
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-message-management/subscribe-message/getCategory.html
func (s *Subscribe) GetCategory() ([]Category, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = s.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getCategoryURL, accessToken)); err != nil {
		return nil, err
	}
	result := &GetCategoryResponse{}
	err = util.DecodeWithError(response, result, "GetCategory")
	return result.Data, err
}

// GetPubTemplateKeywordsByIDResponse 获取关键词列表响应
type GetPubTemplateKeywordsByIDResponse struct {
	util.CommonError
	Count int64                 `json:"count"`
	Data  []PubTemplateKeywords `json:"data"`
}

// PubTemplateKeywords 关键词
type PubTemplateKeywords struct {
	KID     int64  `json:"kid"`
	Name    string `json:"name"`
	Example string `json:"example"`
	Rule    string `json:"rule"`
}

// GetPubTemplateKeywordsByID 获取关键词列表
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-message-management/subscribe-message/getPubTemplateKeyWordsById.html
func (s *Subscribe) GetPubTemplateKeywordsByID(tid string) (*GetPubTemplateKeywordsByIDResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = s.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getPubTemplateKeyWordsByIDURL, accessToken, tid)); err != nil {
		return nil, err
	}
	result := &GetPubTemplateKeywordsByIDResponse{}
	err = util.DecodeWithError(response, result, "GetPubTemplateKeywordsByID")
	return result, err
}

// GetPubTemplateTitleListRequest 获取所属类目下的公共模板请求
type GetPubTemplateTitleListRequest struct {
	Start int64
	Limit int64
	IDs   string
}

// GetPubTemplateTitleListResponse 获取所属类目下的公共模板响应
type GetPubTemplateTitleListResponse struct {
	util.CommonError
	Count int64              `json:"count"`
	Data  []PubTemplateTitle `json:"data"`
}

// PubTemplateTitle 模板标题
type PubTemplateTitle struct {
	Type       int64  `json:"type"`
	TID        string `json:"tid"`
	Title      string `json:"title"`
	CategoryID string `json:"categoryId"`
}

// GetPubTemplateTitleList 获取所属类目下的公共模板
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-message-management/subscribe-message/getPubTemplateTitleList.html
func (s *Subscribe) GetPubTemplateTitleList(req *GetPubTemplateTitleListRequest) (*GetPubTemplateTitleListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = s.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getPubTemplateTitleListURL, accessToken, req.IDs, req.Start, req.Limit)); err != nil {
		return nil, err
	}
	result := &GetPubTemplateTitleListResponse{}
	err = util.DecodeWithError(response, result, "GetPubTemplateTitleList")
	return result, err
}

// SetUserNotifyRequest 激活与更新服务卡片请求
type SetUserNotifyRequest struct {
	OpenID      string `json:"openid"`
	NotifyType  int64  `json:"notify_type"`
	NotifyCode  string `json:"notify_code"`
	ContentJSON string `json:"content_json"`
	CheckJSON   string `json:"check_json,omitempty"`
}

// SetUserNotify 激活与更新服务卡片
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-message-management/subscribe-message/setUserNotify.html
func (s *Subscribe) SetUserNotify(req *SetUserNotifyRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = s.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(setUserNotifyURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "SetUserNotify")
}

// SetUserNotifyExtRequest 更新服务卡片扩展信息请求
type SetUserNotifyExtRequest struct {
	OpenID     string `json:"openid"`
	NotifyType int64  `json:"notify_type"`
	NotifyCode string `json:"notify_code"`
	ExtJSON    string `json:"ext_json"`
}

// SetUserNotifyExt 更新服务卡片扩展信息
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-message-management/subscribe-message/setUserNotifyExt.html
func (s *Subscribe) SetUserNotifyExt(req *SetUserNotifyExtRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = s.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(setUserNotifyExtURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "SetUserNotifyExt")
}

// GetUserNotifyRequest 查询服务卡片状态请求
type GetUserNotifyRequest struct {
	OpenID     string `json:"openid"`
	NotifyType int64  `json:"notify_type"`
	NotifyCode string `json:"notify_code"`
}

// GetUserNotifyResponse 查询服务卡片状态响应
type GetUserNotifyResponse struct {
	util.CommonError
	NotifyInfo NotifyInfo `json:"notify_info"`
}

// NotifyInfo 卡片状态
type NotifyInfo struct {
	NotifyType     int64  `json:"notify_type"`
	ContentJSON    string `json:"content_json"`
	CodeState      int64  `json:"code_state"`
	CodeExpireTime int64  `json:"code_expire_time"`
}

// GetUserNotify 查询服务卡片状态
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-message-management/subscribe-message/getUserNotify.html
func (s *Subscribe) GetUserNotify(req *GetUserNotifyRequest) (*GetUserNotifyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = s.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getUserNotifyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetUserNotifyResponse{}
	err = util.DecodeWithError(response, result, "GetUserNotify")
	return result, err
}
