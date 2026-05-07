package kf

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// addKnowledgeGroupURL 知识库分组添加
	addKnowledgeGroupURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/knowledge/add_group?access_token=%s"
	// delKnowledgeGroupURL 知识库分组删除
	delKnowledgeGroupURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/knowledge/del_group?access_token=%s"
	// modKnowledgeGroupURL 知识库分组修改
	modKnowledgeGroupURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/knowledge/mod_group?access_token=%s"
	// listKnowledgeGroupURL 知识库分组列表
	listKnowledgeGroupURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/knowledge/list_group?access_token=%s"
	// addKnowledgeIntentURL 知识库问答添加
	addKnowledgeIntentURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/knowledge/add_intent?access_token=%s"
	// delKnowledgeIntentURL 知识库问答删除
	delKnowledgeIntentURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/knowledge/del_intent?access_token=%s"
	// modKnowledgeIntentURL 知识库问答修改
	modKnowledgeIntentURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/knowledge/mod_intent?access_token=%s"
	// listKnowledgeIntentURL 知识库问答列表
	listKnowledgeIntentURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/knowledge/list_intent?access_token=%s"
)

// AddKnowledgeGroupRequest 知识库分组添加请求
type AddKnowledgeGroupRequest struct {
	Name string `json:"name"`
}

// AddKnowledgeGroupResponse 知识库分组添加响应
type AddKnowledgeGroupResponse struct {
	util.CommonError
	GroupID string `json:"group_id"`
}

// AddKnowledgeGroup 知识库分组添加
// see https://developer.work.weixin.qq.com/document/path/95971#%E6%B7%BB%E5%8A%A0%E5%88%86%E7%BB%84
func (r *Client) AddKnowledgeGroup(req *AddKnowledgeGroupRequest) (*AddKnowledgeGroupResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(addKnowledgeGroupURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &AddKnowledgeGroupResponse{}
	err = util.DecodeWithError(response, result, "AddKnowledgeGroup")
	return result, err
}

// DelKnowledgeGroupRequest 知识库分组删除请求
type DelKnowledgeGroupRequest struct {
	GroupID string `json:"group_id"`
}

// DelKnowledgeGroup 知识库分组删除
// see https://developer.work.weixin.qq.com/document/path/95971#%E5%88%A0%E9%99%A4%E5%88%86%E7%BB%84
func (r *Client) DelKnowledgeGroup(req *DelKnowledgeGroupRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(delKnowledgeGroupURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DelKnowledgeGroup")
}

// ModKnowledgeGroupRequest 知识库分组修改请求
type ModKnowledgeGroupRequest struct {
	GroupID string `json:"group_id"`
	Name    string `json:"name"`
}

// ModKnowledgeGroup 知识库分组修改
// see https://developer.work.weixin.qq.com/document/path/95971#%E4%BF%AE%E6%94%B9%E5%88%86%E7%BB%84
func (r *Client) ModKnowledgeGroup(req *ModKnowledgeGroupRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(modKnowledgeGroupURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "ModKnowledgeGroup")
}

// ListKnowledgeGroupRequest 知识库分组列表请求
type ListKnowledgeGroupRequest struct {
	Cursor  string `json:"cursor"`
	Limit   int    `json:"limit"`
	GroupID string `json:"group_id"`
}

// ListKnowledgeGroupResponse 知识库分组列表响应
type ListKnowledgeGroupResponse struct {
	util.CommonError
	NextCursor string           `json:"next_cursor"`
	HasMore    int              `json:"has_more"`
	GroupList  []KnowledgeGroup `json:"group_list"`
}

// KnowledgeGroup 知识库分组
type KnowledgeGroup struct {
	GroupID   string `json:"group_id"`
	Name      string `json:"name"`
	IsDefault int    `json:"is_default"`
}

// ListKnowledgeGroup 知识库分组列表
// see https://developer.work.weixin.qq.com/document/path/95971#%E8%8E%B7%E5%8F%96%E5%88%86%E7%BB%84%E5%88%97%E8%A1%A8
func (r *Client) ListKnowledgeGroup(req *ListKnowledgeGroupRequest) (*ListKnowledgeGroupResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(listKnowledgeGroupURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &ListKnowledgeGroupResponse{}
	err = util.DecodeWithError(response, result, "ListKnowledgeGroup")
	return result, err
}

// AddKnowledgeIntentRequest 知识库问答添加请求
type AddKnowledgeIntentRequest struct {
	GroupID          string                 `json:"group_id"`
	Question         IntentQuestion         `json:"question"`
	SimilarQuestions IntentSimilarQuestions `json:"similar_questions"`
	Answers          []IntentAnswerReq      `json:"answers"`
}

// IntentQuestion 主问题
type IntentQuestion struct {
	Text IntentQuestionText `json:"text"`
}

// IntentQuestionText 问题文本
type IntentQuestionText struct {
	Content string `json:"content"`
}

// IntentSimilarQuestions 相似问题
type IntentSimilarQuestions struct {
	Items []IntentQuestion `json:"items"`
}

// IntentAnswerReq 回答请求
type IntentAnswerReq struct {
	Text        IntentAnswerText            `json:"text"`
	Attachments []IntentAnswerAttachmentReq `json:"attachments"`
}

// IntentAnswerText 回答文本
type IntentAnswerText struct {
	Content string `json:"content"`
}

// IntentAnswerAttachmentReq 回答附件请求
type IntentAnswerAttachmentReq struct {
	MsgType     string                               `json:"msgtype"`
	Image       IntentAnswerAttachmentImgReq         `json:"image,omitempty"`
	Video       IntentAnswerAttachmentVideoReq       `json:"video,omitempty"`
	Link        IntentAnswerAttachmentLink           `json:"link,omitempty"`
	MiniProgram IntentAnswerAttachmentMiniProgramReq `json:"miniprogram,omitempty"`
}

// IntentAnswerAttachmentImgReq 图片类型回答附件请求
type IntentAnswerAttachmentImgReq struct {
	MediaID string `json:"media_id"`
}

// IntentAnswerAttachmentVideoReq 视频类型回答附件请求
type IntentAnswerAttachmentVideoReq struct {
	MediaID string `json:"media_id"`
}

// IntentAnswerAttachmentLink 链接类型回答附件
type IntentAnswerAttachmentLink struct {
	Title  string `json:"title"`
	PicURL string `json:"picurl"`
	Desc   string `json:"desc"`
	URL    string `json:"url"`
}

// IntentAnswerAttachmentMiniProgramReq 小程序类型回答附件请求
type IntentAnswerAttachmentMiniProgramReq struct {
	Title        string `json:"title"`
	ThumbMediaID string `json:"thumb_media_id"`
	AppID        string `json:"appid"`
	PagePath     string `json:"pagepath"`
}

// AddKnowledgeIntentResponse 知识库问答添加响应
type AddKnowledgeIntentResponse struct {
	util.CommonError
	IntentID string `json:"intent_id"`
}

// AddKnowledgeIntent 知识库问答添加
// see https://developer.work.weixin.qq.com/document/path/95972#%E6%B7%BB%E5%8A%A0%E9%97%AE%E7%AD%94
func (r *Client) AddKnowledgeIntent(req *AddKnowledgeIntentRequest) (*AddKnowledgeIntentResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(addKnowledgeIntentURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &AddKnowledgeIntentResponse{}
	err = util.DecodeWithError(response, result, "AddKnowledgeIntent")
	return result, err
}

// DelKnowledgeIntentRequest 知识库问答删除请求
type DelKnowledgeIntentRequest struct {
	IntentID string `json:"intent_id"`
}

// DelKnowledgeIntent 知识库问答删除
// see https://developer.work.weixin.qq.com/document/path/95972#%E5%88%A0%E9%99%A4%E9%97%AE%E7%AD%94
func (r *Client) DelKnowledgeIntent(req *DelKnowledgeIntentRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(delKnowledgeIntentURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DelKnowledgeIntent")
}

// ModKnowledgeIntentRequest 知识库问答修改请求
type ModKnowledgeIntentRequest struct {
	IntentID         string                 `json:"intent_id"`
	Question         IntentQuestion         `json:"question"`
	SimilarQuestions IntentSimilarQuestions `json:"similar_questions"`
	Answers          []IntentAnswerReq      `json:"answers"`
}

// ModKnowledgeIntent 知识库问答修改
// see https://developer.work.weixin.qq.com/document/path/95972#%E4%BF%AE%E6%94%B9%E9%97%AE%E7%AD%94
func (r *Client) ModKnowledgeIntent(req *ModKnowledgeIntentRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(modKnowledgeIntentURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "ModKnowledgeIntent")
}

// ListKnowledgeIntentRequest 知识库问答列表请求
type ListKnowledgeIntentRequest struct {
	Cursor   string `json:"cursor"`
	Limit    int    `json:"limit"`
	GroupID  string `json:"group_id"`
	IntentID string `json:"intent_id"`
}

// ListKnowledgeIntentResponse 知识库问答列表响应
type ListKnowledgeIntentResponse struct {
	util.CommonError
	NextCursor string            `json:"next_cursor"`
	HasMore    int               `json:"has_more"`
	IntentList []KnowledgeIntent `json:"intent_list"`
}

// KnowledgeIntent 问答摘要
type KnowledgeIntent struct {
	GroupID          string                 `json:"group_id"`
	IntentID         string                 `json:"intent_id"`
	Question         IntentQuestion         `json:"question"`
	SimilarQuestions IntentSimilarQuestions `json:"similar_questions"`
	Answers          []IntentAnswerRes      `json:"answers"`
}

// IntentAnswerRes 回答返回
type IntentAnswerRes struct {
	Text        IntentAnswerText            `json:"text"`
	Attachments []IntentAnswerAttachmentRes `json:"attachments"`
}

// IntentAnswerAttachmentRes 回答附件返回
type IntentAnswerAttachmentRes struct {
	MsgType     string                               `json:"msgtype"`
	Image       IntentAnswerAttachmentImgRes         `json:"image,omitempty"`
	Video       IntentAnswerAttachmentVideoRes       `json:"video,omitempty"`
	Link        IntentAnswerAttachmentLink           `json:"link,omitempty"`
	MiniProgram IntentAnswerAttachmentMiniProgramRes `json:"miniprogram,omitempty"`
}

// IntentAnswerAttachmentImgRes 图片类型回答附件返回
type IntentAnswerAttachmentImgRes struct {
	Name string `json:"name"`
}

// IntentAnswerAttachmentVideoRes 视频类型回答附件返回
type IntentAnswerAttachmentVideoRes struct {
	Name string `json:"name"`
}

// IntentAnswerAttachmentMiniProgramRes 小程序类型回答附件返回
type IntentAnswerAttachmentMiniProgramRes struct {
	Title    string `json:"title"`
	AppID    string `json:"appid"`
	PagePath string `json:"pagepath"`
}

// ListKnowledgeIntent 知识库问答列表
// see https://developer.work.weixin.qq.com/document/path/95972#%E8%8E%B7%E5%8F%96%E9%97%AE%E7%AD%94%E5%88%97%E8%A1%A8
func (r *Client) ListKnowledgeIntent(req *ListKnowledgeIntentRequest) (*ListKnowledgeIntentResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(listKnowledgeIntentURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &ListKnowledgeIntentResponse{}
	err = util.DecodeWithError(response, result, "ListKnowledgeIntent")
	return result, err
}
