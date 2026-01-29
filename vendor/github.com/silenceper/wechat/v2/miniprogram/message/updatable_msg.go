package message

import (
	"fmt"

	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	// createActivityIDURL 创建activity_id
	createActivityIDURL = "https://api.weixin.qq.com/cgi-bin/message/wxopen/activityid/create?access_token=%s&unionid=%s&openid=%s"
	// SendUpdatableMsgURL 修改动态消息
	setUpdatableMsgURL = "https://api.weixin.qq.com/cgi-bin/message/wxopen/updatablemsg/send?access_token=%s"
	// setChatToolMsgURL 修改小程序聊天工具的动态卡片消息
	setChatToolMsgURL = "https://api.weixin.qq.com/cgi-bin/message/wxopen/chattoolmsg/send?access_token=%s"
)

// UpdatableTargetState 动态消息状态
type UpdatableTargetState int

const (
	// TargetStateNotStarted 未开始
	TargetStateNotStarted UpdatableTargetState = 0
	// TargetStateStarted 已开始
	TargetStateStarted UpdatableTargetState = 1
	// TargetStateFinished 已结束
	TargetStateFinished UpdatableTargetState = 2
)

// UpdatableMessage 动态消息
type UpdatableMessage struct {
	*context.Context
}

// NewUpdatableMessage 实例化
func NewUpdatableMessage(ctx *context.Context) *UpdatableMessage {
	return &UpdatableMessage{
		Context: ctx,
	}
}

// CreateActivityIDRequest 创建activity_id请求
type CreateActivityIDRequest struct {
	UnionID string
	OpenID  string
}

// CreateActivityID 创建activity_id
func (updatableMessage *UpdatableMessage) CreateActivityID() (CreateActivityIDResponse, error) {
	return updatableMessage.CreateActivityIDWithReq(&CreateActivityIDRequest{})
}

// CreateActivityIDWithReq 创建activity_id
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-message-management/updatable-message/createActivityId.html
func (updatableMessage *UpdatableMessage) CreateActivityIDWithReq(req *CreateActivityIDRequest) (res CreateActivityIDResponse, err error) {
	accessToken, err := updatableMessage.GetAccessToken()
	if err != nil {
		return
	}
	url := fmt.Sprintf(createActivityIDURL, accessToken, req.UnionID, req.OpenID)
	response, err := util.HTTPGet(url)
	if err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "CreateActivityID")
	return
}

// SetUpdatableMsg 修改动态消息
func (updatableMessage *UpdatableMessage) SetUpdatableMsg(activityID string, targetState UpdatableTargetState, template UpdatableMsgTemplate) (err error) {
	accessToken, err := updatableMessage.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(setUpdatableMsgURL, accessToken)
	data := SendUpdatableMsgReq{
		ActivityID:   activityID,
		TargetState:  targetState,
		TemplateInfo: template,
	}

	response, err := util.PostJSON(uri, data)
	if err != nil {
		return
	}
	return util.DecodeWithCommonError(response, "SendUpdatableMsg")
}

// CreateActivityIDResponse 创建activity_id 返回
type CreateActivityIDResponse struct {
	util.CommonError

	ActivityID     string `json:"activity_id"`
	ExpirationTime int64  `json:"expiration_time"`
}

// UpdatableMsgTemplate 动态消息模板
type UpdatableMsgTemplate struct {
	ParameterList []UpdatableMsgParameter `json:"parameter_list"`
}

// UpdatableMsgParameter 动态消息参数
type UpdatableMsgParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SendUpdatableMsgReq 修改动态消息参数
type SendUpdatableMsgReq struct {
	ActivityID   string               `json:"activity_id"`
	TemplateInfo UpdatableMsgTemplate `json:"template_info"`
	TargetState  UpdatableTargetState `json:"target_state"`
}

// SetChatToolMsgRequest 修改小程序聊天工具的动态卡片消息请求
type SetChatToolMsgRequest struct {
	VersionType          int64                `json:"version_type"`
	TargetState          UpdatableTargetState `json:"target_state"`
	ActivityID           string               `json:"activity_id"`
	TemplateID           string               `json:"template_id"`
	ParticipatorInfoList []ParticipatorInfo   `json:"participator_info_list,omitempty"`
}

// ParticipatorInfo 更新后的聊天室成员状态
type ParticipatorInfo struct {
	State       int    `json:"state"`
	GroupOpenID string `json:"group_openid"`
}

// SetChatToolMsg 修改小程序聊天工具的动态卡片消息
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-message-management/updatable-message/setChatToolMsg.html
func (updatableMessage *UpdatableMessage) SetChatToolMsg(req *SetChatToolMsgRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = updatableMessage.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(setChatToolMsgURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "SetChatToolMsg")
}
