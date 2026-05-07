package appchat

import (
	"encoding/json"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// 应用推送消息接口地址
	sendURL = "https://qyapi.weixin.qq.com/cgi-bin/appchat/send?access_token=%s"
)

type (
	// SendRequestCommon 发送应用推送消息请求公共参数
	SendRequestCommon struct {
		// 群聊id
		ChatID string `json:"chatid"`
		// 消息类型
		MsgType string `json:"msgtype"`
		// 表示是否是保密消息，0表示否，1表示是，默认0
		Safe int `json:"safe"`
	}

	// SendResponse 发送应用消息响应参数
	SendResponse struct {
		util.CommonError
	}

	// SendTextRequest 发送文本消息的请求
	SendTextRequest struct {
		*SendRequestCommon
		Text TextField `json:"text"`
	}
	// TextField 文本消息参数
	TextField struct {
		// 消息内容，最长不超过2048个字节
		Content string `json:"content"`
	}

	// SendImageRequest 发送图片消息的请求
	SendImageRequest struct {
		*SendRequestCommon
		Image ImageField `json:"image"`
	}
	// ImageField 图片消息参数
	ImageField struct {
		// 图片媒体文件id，可以调用上传临时素材接口获取
		MediaID string `json:"media_id"`
	}

	// SendVoiceRequest 发送语音消息的请求
	SendVoiceRequest struct {
		*SendRequestCommon
		Voice VoiceField `json:"voice"`
	}
	// VoiceField 语音消息参数
	VoiceField struct {
		// 语音文件id，可以调用上传临时素材接口获取
		MediaID string `json:"media_id"`
	}
)

// Send 发送应用消息
// @desc 实现企业微信发送应用消息接口：https://developer.work.weixin.qq.com/document/path/90248
func (r *Client) Send(apiName string, request interface{}) (*SendResponse, error) {
	// 获取accessToken
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}
	// 请求参数转 JSON 格式
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	// 发起http请求
	response, err := util.HTTPPost(fmt.Sprintf(sendURL, accessToken), string(jsonData))
	if err != nil {
		return nil, err
	}
	// 按照结构体解析返回值
	result := &SendResponse{}
	err = util.DecodeWithError(response, result, apiName)
	// 返回数据
	return result, err
}

// SendText 发送文本消息
func (r *Client) SendText(request SendTextRequest) (*SendResponse, error) {
	// 发送文本消息MsgType参数固定为：text
	request.MsgType = "text"
	return r.Send("MessageSendText", request)
}

// SendImage 发送图片消息
func (r *Client) SendImage(request SendImageRequest) (*SendResponse, error) {
	// 发送图片消息MsgType参数固定为：image
	request.MsgType = "image"
	return r.Send("MessageSendImage", request)
}

// SendVoice 发送语音消息
func (r *Client) SendVoice(request SendVoiceRequest) (*SendResponse, error) {
	// 发送语音消息MsgType参数固定为：voice
	request.MsgType = "voice"
	return r.Send("MessageSendVoice", request)
}

// 以上实现了部分常用消息推送：SendText 发送文本消息、SendImage 发送图片消息、SendVoice 发送语音消息，
// 如需扩展其他消息类型，建议按照以上格式，扩展对应消息类型的参数即可
// 也可以直接使用Send方法，按照企业微信消息推送的接口文档传对应消息类型的参数来使用
