package externalcontact

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// transferCustomerURL 分配在职成员的客户
	transferCustomerURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/transfer_customer?access_token=%s"
	// transferResultURL 查询客户接替状态
	transferResultURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/transfer_result?access_token=%s"
	// groupChatOnJobTransferURL 分配在职成员的客户群
	groupChatOnJobTransferURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/groupchat/onjob_transfer?access_token=%s"
	// getUnassignedListURL 获取待分配的离职成员列表
	getUnassignedListURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_unassigned_list?access_token=%s"
	// resignedTransferCustomerURL 分配离职成员的客户
	resignedTransferCustomerURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/resigned/transfer_customer?access_token=%s"
	// resignedTransferResultURL 查询离职客户接替状态
	resignedTransferResultURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/resigned/transfer_result?access_token=%s"
	// groupChatTransferURL 分配离职成员的客户群
	groupChatTransferURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/groupchat/transfer?access_token=%s"
)

// TransferCustomerRequest 分配在职成员的客户请求
type TransferCustomerRequest struct {
	HandoverUserID     string   `json:"handover_userid"`
	TakeoverUserID     string   `json:"takeover_userid"`
	ExternalUserID     []string `json:"external_userid"`
	TransferSuccessMsg string   `json:"transfer_success_msg"`
}

// TransferCustomerResponse 分配在职成员的客户请求响应
type TransferCustomerResponse struct {
	util.CommonError
	Customer []TransferCustomerItem `json:"customer"`
}

// TransferCustomerItem 客户分配结果
type TransferCustomerItem struct {
	ExternalUserID string `json:"external_userid"`
	ErrCode        int    `json:"errcode"`
}

// TransferCustomer 分配在职成员的客户
// see https://developer.work.weixin.qq.com/document/path/92125
func (r *Client) TransferCustomer(req *TransferCustomerRequest) (*TransferCustomerResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(transferCustomerURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &TransferCustomerResponse{}
	err = util.DecodeWithError(response, result, "TransferCustomer")
	return result, err
}

// TransferResultRequest 查询客户接替状态请求
type TransferResultRequest struct {
	HandoverUserID string `json:"handover_userid"`
	TakeoverUserID string `json:"takeover_userid"`
	Cursor         string `json:"cursor"`
}

// TransferResultResponse 查询客户接替状态响应
type TransferResultResponse struct {
	util.CommonError
	Customer   []TransferResultItem `json:"customer"`
	NextCursor string               `json:"next_cursor"`
}

// TransferResultItem 客户接替状态
type TransferResultItem struct {
	ExternalUserID string `json:"external_userid"`
	Status         int    `json:"status"`
	TakeoverTime   int64  `json:"takeover_time"`
}

// TransferResult 查询客户接替状态
// see https://developer.work.weixin.qq.com/document/path/94088
func (r *Client) TransferResult(req *TransferResultRequest) (*TransferResultResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(transferResultURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &TransferResultResponse{}
	err = util.DecodeWithError(response, result, "TransferResult")
	return result, err
}

// GroupChatOnJobTransferRequest 分配在职成员的客户群请求
type GroupChatOnJobTransferRequest struct {
	ChatIDList []string `json:"chat_id_list"`
	NewOwner   string   `json:"new_owner"`
}

// GroupChatOnJobTransferResponse 分配在职成员的客户群响应
type GroupChatOnJobTransferResponse struct {
	util.CommonError
	FailedChatList []FailedChat `json:"failed_chat_list"`
}

// FailedChat 没能成功继承的群
type FailedChat struct {
	ChatID  string `json:"chat_id"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// GroupChatOnJobTransfer 分配在职成员的客户群
// see https://developer.work.weixin.qq.com/document/path/95703
func (r *Client) GroupChatOnJobTransfer(req *GroupChatOnJobTransferRequest) (*GroupChatOnJobTransferResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(groupChatOnJobTransferURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GroupChatOnJobTransferResponse{}
	err = util.DecodeWithError(response, result, "GroupChatOnJobTransfer")
	return result, err
}

// GetUnassignedListRequest 获取待分配的离职成员列表请求
type GetUnassignedListRequest struct {
	Cursor   string `json:"cursor"`
	PageSize int    `json:"page_size"`
}

// GetUnassignedListResponse 获取待分配的离职成员列表响应
type GetUnassignedListResponse struct {
	util.CommonError
	Info       []UnassignedListInfo `json:"info"`
	IsLast     bool                 `json:"is_last"`
	NextCursor string               `json:"next_cursor"`
}

// UnassignedListInfo 离职成员信息
type UnassignedListInfo struct {
	HandoverUserID string `json:"handover_userid"`
	ExternalUserID string `json:"external_userid"`
	DimissionTime  int64  `json:"dimission_time"`
}

// GetUnassignedList 获取待分配的离职成员列表
// see https://developer.work.weixin.qq.com/document/path/92124
func (r *Client) GetUnassignedList(req *GetUnassignedListRequest) (*GetUnassignedListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getUnassignedListURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetUnassignedListResponse{}
	err = util.DecodeWithError(response, result, "GetUnassignedList")
	return result, err
}

// ResignedTransferCustomerRequest 分配离职成员的客户请求
type ResignedTransferCustomerRequest struct {
	HandoverUserID string   `json:"handover_userid"`
	TakeoverUserID string   `json:"takeover_userid"`
	ExternalUserID []string `json:"external_userid"`
}

// ResignedTransferCustomerResponse 分配离职成员的客户响应
type ResignedTransferCustomerResponse struct {
	util.CommonError
	Customer []TransferCustomerItem `json:"customer"`
}

// ResignedTransferCustomer 分配离职成员的客户
// see https://developer.work.weixin.qq.com/document/path/94081
func (r *Client) ResignedTransferCustomer(req *ResignedTransferCustomerRequest) (*ResignedTransferCustomerResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(resignedTransferCustomerURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &ResignedTransferCustomerResponse{}
	err = util.DecodeWithError(response, result, "ResignedTransferCustomer")
	return result, err
}

// ResignedTransferResultRequest 查询离职客户接替状态请求
type ResignedTransferResultRequest struct {
	HandoverUserID string `json:"handover_userid"`
	TakeoverUserID string `json:"takeover_userid"`
	Cursor         string `json:"cursor"`
}

// ResignedTransferResultResponse 查询离职客户接替状态响应
type ResignedTransferResultResponse struct {
	util.CommonError
	Customer   []TransferResultItem `json:"customer"`
	NextCursor string               `json:"next_cursor"`
}

// ResignedTransferResult 查询离职客户接替状态
// see https://developer.work.weixin.qq.com/document/path/94082
func (r *Client) ResignedTransferResult(req *ResignedTransferResultRequest) (*ResignedTransferResultResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(resignedTransferResultURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &ResignedTransferResultResponse{}
	err = util.DecodeWithError(response, result, "ResignedTransferResult")
	return result, err
}

// GroupChatTransferRequest 分配离职成员的客户群请求
type GroupChatTransferRequest struct {
	ChatIDList []string `json:"chat_id_list"`
	NewOwner   string   `json:"new_owner"`
}

// GroupChatTransferResponse 分配离职成员的客户群响应
type GroupChatTransferResponse struct {
	util.CommonError
	FailedChatList []FailedChat `json:"failed_chat_list"`
}

// GroupChatTransfer 分配离职成员的客户群
// see https://developer.work.weixin.qq.com/document/path/92127
func (r *Client) GroupChatTransfer(req *GroupChatTransferRequest) (*GroupChatTransferResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(groupChatTransferURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GroupChatTransferResponse{}
	err = util.DecodeWithError(response, result, "GroupChatTransfer")
	return result, err
}
