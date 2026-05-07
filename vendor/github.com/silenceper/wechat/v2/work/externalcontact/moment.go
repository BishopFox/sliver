package externalcontact

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// addMomentTaskURL 创建发表任务
	addMomentTaskURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/add_moment_task?access_token=%s"
	// getMomentTaskResultURL 获取任务创建结果
	getMomentTaskResultURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_moment_task_result?access_token=%s&jobid=%s"
	// cancelMomentTaskURL 停止发表企业朋友圈
	cancelMomentTaskURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/cancel_moment_task?access_token=%s"
	// getMomentListURL 获取企业全部的发表列表
	getMomentListURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_moment_list?access_token=%s"
	// getMomentTaskURL 获取客户朋友圈企业发表的列表
	getMomentTaskURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_moment_task?access_token=%s"
	// getMomentCustomerListURL 获取客户朋友圈发表时选择的可见范围
	getMomentCustomerListURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_moment_customer_list?access_token=%s"
	// getMomentSendResultURL 获取客户朋友圈发表后的可见客户列表
	getMomentSendResultURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_moment_send_result?access_token=%s"
	// getMomentCommentsURL 获取客户朋友圈的互动数据
	getMomentCommentsURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_moment_comments?access_token=%s"
	// listMomentStrategyURL 获取规则组列表
	listMomentStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/moment_strategy/list?access_token=%s"
	// getMomentStrategyURL 获取规则组详情
	getMomentStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/moment_strategy/get?access_token=%s"
	// getRangeMomentStrategyURL 获取规则组管理范围
	getRangeMomentStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/moment_strategy/get_range?access_token=%s"
	// createMomentStrategyURL 创建新的规则组
	createMomentStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/moment_strategy/create?access_token=%s"
	// editMomentStrategyURL 编辑规则组及其管理范围
	editMomentStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/moment_strategy/edit?access_token=%s"
	// delMomentStrategyURL 删除规则组
	delMomentStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/moment_strategy/del?access_token=%s"
)

// AddMomentTaskRequest 创建发表任务请求
type AddMomentTaskRequest struct {
	Text         MomentTaskText         `json:"text"`
	Attachments  []MomentTaskAttachment `json:"attachments"`
	VisibleRange MomentVisibleRange     `json:"visible_range"`
}

// MomentTaskText 发表任务文本消息
type MomentTaskText struct {
	Content string `json:"content"`
}

// MomentTaskImage 发表任务图片消息
type MomentTaskImage struct {
	MediaID string `json:"media_id"`
}

// MomentTaskVideo 发表任务视频消息
type MomentTaskVideo struct {
	MediaID string `json:"media_id"`
}

// MomentTaskLink 发表任务图文消息
type MomentTaskLink struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	MediaID string `json:"media_id"`
}

// MomentTaskAttachment 发表任务附件
type MomentTaskAttachment struct {
	MsgType string          `json:"msgtype"`
	Image   MomentTaskImage `json:"image,omitempty"`
	Video   MomentTaskVideo `json:"video,omitempty"`
	Link    MomentTaskLink  `json:"link,omitempty"`
}

// MomentVisibleRange 朋友圈指定的发表范围
type MomentVisibleRange struct {
	SenderList          MomentSenderList          `json:"sender_list"`
	ExternalContactList MomentExternalContactList `json:"external_contact_list"`
}

// MomentSenderList 发表任务的执行者列表
type MomentSenderList struct {
	UserList       []string `json:"user_list"`
	DepartmentList []int    `json:"department_list"`
}

// MomentExternalContactList 可见到该朋友圈的客户列表
type MomentExternalContactList struct {
	TagList []string `json:"tag_list"`
}

// AddMomentTaskResponse 创建发表任务响应
type AddMomentTaskResponse struct {
	util.CommonError
	JobID string `json:"jobid"`
}

// AddMomentTask 创建发表任务
// see https://developer.work.weixin.qq.com/document/path/95094#%E5%88%9B%E5%BB%BA%E5%8F%91%E8%A1%A8%E4%BB%BB%E5%8A%A1
func (r *Client) AddMomentTask(req *AddMomentTaskRequest) (*AddMomentTaskResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(addMomentTaskURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &AddMomentTaskResponse{}
	err = util.DecodeWithError(response, result, "AddMomentTask")
	return result, err
}

// GetMomentTaskResultResponse 获取任务创建结果响应
type GetMomentTaskResultResponse struct {
	util.CommonError
	Status int              `json:"status"`
	Type   string           `json:"type"`
	Result MomentTaskResult `json:"result"`
}

// MomentTaskResult 任务创建结果
type MomentTaskResult struct {
	ErrCode                    int64                            `json:"errcode"`
	ErrMsg                     string                           `json:"errmsg"`
	MomentID                   string                           `json:"moment_id"`
	InvalidSenderList          MomentInvalidSenderList          `json:"invalid_sender_list"`
	InvalidExternalContactList MomentInvalidExternalContactList `json:"invalid_external_contact_list"`
}

// MomentInvalidSenderList 不合法的执行者列表
type MomentInvalidSenderList struct {
	UserList       []string `json:"user_list"`
	DepartmentList []int    `json:"department_list"`
}

// MomentInvalidExternalContactList 不合法的可见到该朋友圈的客户列表
type MomentInvalidExternalContactList struct {
	TagList []string `json:"tag_list"`
}

// GetMomentTaskResult 获取任务创建结果
// see https://developer.work.weixin.qq.com/document/path/95094#%E8%8E%B7%E5%8F%96%E4%BB%BB%E5%8A%A1%E5%88%9B%E5%BB%BA%E7%BB%93%E6%9E%9C
func (r *Client) GetMomentTaskResult(jobID string) (*GetMomentTaskResultResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getMomentTaskResultURL, accessToken, jobID)); err != nil {
		return nil, err
	}
	result := &GetMomentTaskResultResponse{}
	err = util.DecodeWithError(response, result, "GetMomentTaskResult")
	return result, err
}

// CancelMomentTaskRequest 停止发表企业朋友圈请求
type CancelMomentTaskRequest struct {
	MomentID string `json:"moment_id"`
}

// CancelMomentTask 停止发表企业朋友圈
// see https://developer.work.weixin.qq.com/document/path/97612
func (r *Client) CancelMomentTask(req *CancelMomentTaskRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(cancelMomentTaskURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "CancelMomentTask")
}

// GetMomentListRequest 获取企业全部的发表列表请求
type GetMomentListRequest struct {
	StartTime  int64  `json:"start_time"`
	EndTime    int64  `json:"end_time"`
	Creator    string `json:"creator"`
	FilterType int    `json:"filter_type"`
	Cursor     string `json:"cursor"`
	Limit      int    `json:"limit"`
}

// GetMomentListResponse 获取企业全部的发表列表响应
type GetMomentListResponse struct {
	util.CommonError
	NextCursor string       `json:"next_cursor"`
	MomentList []MomentItem `json:"moment_list"`
}

// MomentItem 朋友圈
type MomentItem struct {
	MomentID    string         `json:"moment_id"`
	Creator     string         `json:"creator"`
	CreateTime  int64          `json:"create_time"`
	CreateType  int            `json:"create_type"`
	VisibleType int            `json:"visible_type"`
	Text        MomentText     `json:"text"`
	Image       []MomentImage  `json:"image"`
	Video       MomentVideo    `json:"video"`
	Link        MomentLink     `json:"link"`
	Location    MomentLocation `json:"location"`
}

// MomentText 朋友圈文本消息
type MomentText struct {
	Content string `json:"content"`
}

// MomentImage 朋友圈图片
type MomentImage struct {
	MediaID string `json:"media_id"`
}

// MomentVideo 朋友圈视频
type MomentVideo struct {
	MediaID      string `json:"media_id"`
	ThumbMediaID string `json:"thumb_media_id"`
}

// MomentLink 朋友圈网页链接
type MomentLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// MomentLocation 朋友圈地理位置
type MomentLocation struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
	Name      string `json:"name"`
}

// GetMomentList 获取企业全部的发表列表
// see https://developer.work.weixin.qq.com/document/path/93333#%E8%8E%B7%E5%8F%96%E4%BC%81%E4%B8%9A%E5%85%A8%E9%83%A8%E7%9A%84%E5%8F%91%E8%A1%A8%E5%88%97%E8%A1%A8
func (r *Client) GetMomentList(req *GetMomentListRequest) (*GetMomentListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getMomentListURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetMomentListResponse{}
	err = util.DecodeWithError(response, result, "GetMomentList")
	return result, err
}

// GetMomentTaskRequest 获取客户朋友圈企业发表的列表请求
type GetMomentTaskRequest struct {
	MomentID string `json:"moment_id"`
	Cursor   string `json:"cursor"`
	Limit    int    `json:"limit"`
}

// GetMomentTaskResponse 获取客户朋友圈企业发表的列表响应
type GetMomentTaskResponse struct {
	util.CommonError
	NextCursor string       `json:"next_cursor"`
	TaskList   []MomentTask `json:"task_list"`
}

// MomentTask 发表任务
type MomentTask struct {
	UserID        string `json:"userid"`
	PublishStatus int    `json:"publish_status"`
}

// GetMomentTask 获取客户朋友圈企业发表的列表
// see https://developer.work.weixin.qq.com/document/path/93333#%E8%8E%B7%E5%8F%96%E5%AE%A2%E6%88%B7%E6%9C%8B%E5%8F%8B%E5%9C%88%E4%BC%81%E4%B8%9A%E5%8F%91%E8%A1%A8%E7%9A%84%E5%88%97%E8%A1%A8
func (r *Client) GetMomentTask(req *GetMomentTaskRequest) (*GetMomentTaskResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getMomentTaskURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetMomentTaskResponse{}
	err = util.DecodeWithError(response, result, "GetMomentTask")
	return result, err
}

// GetMomentCustomerListRequest 获取客户朋友圈发表时选择的可见范围请求
type GetMomentCustomerListRequest struct {
	MomentID string `json:"moment_id"`
	UserID   string `json:"userid"`
	Cursor   string `json:"cursor"`
	Limit    int    `json:"limit"`
}

// GetMomentCustomerListResponse 获取客户朋友圈发表时选择的可见范围响应
type GetMomentCustomerListResponse struct {
	util.CommonError
	NextCursor   string           `json:"next_cursor"`
	CustomerList []MomentCustomer `json:"customer_list"`
}

// MomentCustomer 成员可见客户列表
type MomentCustomer struct {
	UserID         string `json:"userid"`
	ExternalUserID string `json:"external_userid"`
}

// GetMomentCustomerList 获取客户朋友圈发表时选择的可见范围
// see https://developer.work.weixin.qq.com/document/path/93333#%E8%8E%B7%E5%8F%96%E5%AE%A2%E6%88%B7%E6%9C%8B%E5%8F%8B%E5%9C%88%E5%8F%91%E8%A1%A8%E6%97%B6%E9%80%89%E6%8B%A9%E7%9A%84%E5%8F%AF%E8%A7%81%E8%8C%83%E5%9B%B4
func (r *Client) GetMomentCustomerList(req *GetMomentCustomerListRequest) (*GetMomentCustomerListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getMomentCustomerListURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetMomentCustomerListResponse{}
	err = util.DecodeWithError(response, result, "GetMomentCustomerList")
	return result, err
}

// GetMomentSendResultRequest 获取客户朋友圈发表后的可见客户列表请求
type GetMomentSendResultRequest struct {
	MomentID string `json:"moment_id"`
	UserID   string `json:"userid"`
	Cursor   string `json:"cursor"`
	Limit    int    `json:"limit"`
}

// GetMomentSendResultResponse 获取客户朋友圈发表后的可见客户列表响应
type GetMomentSendResultResponse struct {
	util.CommonError
	NextCursor   string               `json:"next_cursor"`
	CustomerList []MomentSendCustomer `json:"customer_list"`
}

// MomentSendCustomer 成员发送成功客户
type MomentSendCustomer struct {
	ExternalUserID string `json:"external_userid"`
}

// GetMomentSendResult 获取客户朋友圈发表后的可见客户列表
// see https://developer.work.weixin.qq.com/document/path/93333#%E8%8E%B7%E5%8F%96%E5%AE%A2%E6%88%B7%E6%9C%8B%E5%8F%8B%E5%9C%88%E5%8F%91%E8%A1%A8%E5%90%8E%E7%9A%84%E5%8F%AF%E8%A7%81%E5%AE%A2%E6%88%B7%E5%88%97%E8%A1%A8
func (r *Client) GetMomentSendResult(req *GetMomentSendResultRequest) (*GetMomentSendResultResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getMomentSendResultURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetMomentSendResultResponse{}
	err = util.DecodeWithError(response, result, "GetMomentSendResult")
	return result, err
}

// GetMomentCommentsRequest 获取客户朋友圈的互动数据请求
type GetMomentCommentsRequest struct {
	MomentID string `json:"moment_id"`
	UserID   string `json:"userid"`
}

// GetMomentCommentsResponse 获取客户朋友圈的互动数据响应
type GetMomentCommentsResponse struct {
	util.CommonError
	CommentList []MomentComment `json:"comment_list"`
	LikeList    []MomentLike    `json:"like_list"`
}

// MomentComment 朋友圈评论
type MomentComment struct {
	ExternalUserID string `json:"external_userid,omitempty"`
	UserID         string `json:"userid,omitempty"`
	CreateTime     int64  `json:"create_time"`
}

// MomentLike 朋友圈点赞
type MomentLike struct {
	ExternalUserID string `json:"external_userid,omitempty"`
	UserID         string `json:"userid,omitempty"`
	CreateTime     int64  `json:"create_time"`
}

// GetMomentComments 获取客户朋友圈的互动数据
// see https://developer.work.weixin.qq.com/document/path/93333#%E8%8E%B7%E5%8F%96%E5%AE%A2%E6%88%B7%E6%9C%8B%E5%8F%8B%E5%9C%88%E7%9A%84%E4%BA%92%E5%8A%A8%E6%95%B0%E6%8D%AE
func (r *Client) GetMomentComments(req *GetMomentCommentsRequest) (*GetMomentCommentsResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getMomentCommentsURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetMomentCommentsResponse{}
	err = util.DecodeWithError(response, result, "GetMomentComments")
	return result, err
}

// ListMomentStrategyRequest 获取规则组列表请求
type ListMomentStrategyRequest struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

// ListMomentStrategyResponse 获取规则组列表响应
type ListMomentStrategyResponse struct {
	util.CommonError
	Strategy   []MomentStrategyID `json:"strategy"`
	NextCursor string             `json:"next_cursor"`
}

// MomentStrategyID 规则组ID
type MomentStrategyID struct {
	StrategyID int `json:"strategy_id"`
}

// ListMomentStrategy 获取规则组列表
// see https://developer.work.weixin.qq.com/document/path/94890#%E8%8E%B7%E5%8F%96%E8%A7%84%E5%88%99%E7%BB%84%E5%88%97%E8%A1%A8
func (r *Client) ListMomentStrategy(req *ListMomentStrategyRequest) (*ListMomentStrategyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(listMomentStrategyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &ListMomentStrategyResponse{}
	err = util.DecodeWithError(response, result, "ListMomentStrategy")
	return result, err
}

// GetMomentStrategyRequest 获取规则组详情请求
type GetMomentStrategyRequest struct {
	StrategyID int `json:"strategy_id"`
}

// GetMomentStrategyResponse 获取规则组详情响应
type GetMomentStrategyResponse struct {
	util.CommonError
	Strategy MomentStrategy `json:"strategy"`
}

// MomentStrategy 规则组
type MomentStrategy struct {
	StrategyID   int             `json:"strategy_id"`
	ParentID     int             `json:"parent_id"`
	StrategyName string          `json:"strategy_name"`
	CreateTime   int64           `json:"create_time"`
	AdminList    []string        `json:"admin_list"`
	Privilege    MomentPrivilege `json:"privilege"`
}

// MomentPrivilege 规则组权限
type MomentPrivilege struct {
	ViewMomentList           bool `json:"view_moment_list"`
	SendMoment               bool `json:"send_moment"`
	ManageMomentCoverAndSign bool `json:"manage_moment_cover_and_sign"`
}

// GetMomentStrategy 获取规则组详情
// see https://developer.work.weixin.qq.com/document/path/94890#%E8%8E%B7%E5%8F%96%E8%A7%84%E5%88%99%E7%BB%84%E8%AF%A6%E6%83%85
func (r *Client) GetMomentStrategy(req *GetMomentStrategyRequest) (*GetMomentStrategyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getMomentStrategyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetMomentStrategyResponse{}
	err = util.DecodeWithError(response, result, "GetMomentStrategy")
	return result, err
}

// GetRangeMomentStrategyRequest 获取规则组管理范围请求
type GetRangeMomentStrategyRequest struct {
	StrategyID int    `json:"strategy_id"`
	Cursor     string `json:"cursor"`
	Limit      int    `json:"limit"`
}

// GetRangeMomentStrategyResponse 获取规则组管理范围响应
type GetRangeMomentStrategyResponse struct {
	util.CommonError
	Range      []RangeMomentStrategy `json:"range"`
	NextCursor string                `json:"next_cursor"`
}

// RangeMomentStrategy 	管理范围内配置的成员或部门
type RangeMomentStrategy struct {
	Type    int    `json:"type"`
	UserID  string `json:"userid,omitempty"`
	PartyID int    `json:"partyid,omitempty"`
}

// GetRangeMomentStrategy 获取规则组管理范围
// see https://developer.work.weixin.qq.com/document/path/94890#%E8%8E%B7%E5%8F%96%E8%A7%84%E5%88%99%E7%BB%84%E7%AE%A1%E7%90%86%E8%8C%83%E5%9B%B4
func (r *Client) GetRangeMomentStrategy(req *GetRangeMomentStrategyRequest) (*GetRangeMomentStrategyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getRangeMomentStrategyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetRangeMomentStrategyResponse{}
	err = util.DecodeWithError(response, result, "GetRangeMomentStrategy")
	return result, err
}

// CreateMomentStrategyRequest 创建新的规则组请求
type CreateMomentStrategyRequest struct {
	ParentID     int                   `json:"parent_id"`
	StrategyName string                `json:"strategy_name"`
	AdminList    []string              `json:"admin_list"`
	Privilege    MomentPrivilege       `json:"privilege"`
	Range        []RangeMomentStrategy `json:"range"`
}

// CreateMomentStrategyResponse 创建新的规则组响应
type CreateMomentStrategyResponse struct {
	util.CommonError
	StrategyID int `json:"strategy_id"`
}

// CreateMomentStrategy 创建新的规则组
// see https://developer.work.weixin.qq.com/document/path/94890#%E5%88%9B%E5%BB%BA%E6%96%B0%E7%9A%84%E8%A7%84%E5%88%99%E7%BB%84
func (r *Client) CreateMomentStrategy(req *CreateMomentStrategyRequest) (*CreateMomentStrategyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(createMomentStrategyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &CreateMomentStrategyResponse{}
	err = util.DecodeWithError(response, result, "CreateMomentStrategy")
	return result, err
}

// EditMomentStrategyRequest 编辑规则组及其管理范围请求
type EditMomentStrategyRequest struct {
	StrategyID   int                   `json:"strategy_id"`
	StrategyName string                `json:"strategy_name"`
	AdminList    []string              `json:"admin_list"`
	Privilege    MomentPrivilege       `json:"privilege"`
	RangeAdd     []RangeMomentStrategy `json:"range_add"`
	RangeDel     []RangeMomentStrategy `json:"range_del"`
}

// EditMomentStrategy 编辑规则组及其管理范围
// see https://developer.work.weixin.qq.com/document/path/94890#%E7%BC%96%E8%BE%91%E8%A7%84%E5%88%99%E7%BB%84%E5%8F%8A%E5%85%B6%E7%AE%A1%E7%90%86%E8%8C%83%E5%9B%B4
func (r *Client) EditMomentStrategy(req *EditMomentStrategyRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(editMomentStrategyURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "EditMomentStrategy")
}

// DelMomentStrategyRequest 删除规则组请求
type DelMomentStrategyRequest struct {
	StrategyID int `json:"strategy_id"`
}

// DelMomentStrategy 删除规则组
// see https://developer.work.weixin.qq.com/document/path/94890#%E5%88%A0%E9%99%A4%E8%A7%84%E5%88%99%E7%BB%84
func (r *Client) DelMomentStrategy(req *DelMomentStrategyRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(delMomentStrategyURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DelMomentStrategy")
}
