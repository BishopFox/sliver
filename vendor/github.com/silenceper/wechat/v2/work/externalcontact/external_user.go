package externalcontact

import (
	"encoding/json"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// fetchExternalContactUserListURL 获取客户列表
	fetchExternalContactUserListURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/list"
	// fetchExternalContactUserDetailURL 获取客户详情
	fetchExternalContactUserDetailURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get"
	// fetchBatchExternalContactUserDetailURL 批量获取客户详情
	fetchBatchExternalContactUserDetailURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/batch/get_by_user"
	// updateUserRemarkURL 更新客户备注信息
	updateUserRemarkURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/remark"
	// listCustomerStrategyURL 获取规则组列表
	listCustomerStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/customer_strategy/list?access_token=%s"
	// getCustomerStrategyURL 获取规则组详情
	getCustomerStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/customer_strategy/get?access_token=%s"
	// getRangeCustomerStrategyURL 获取规则组管理范围
	getRangeCustomerStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/customer_strategy/get_range?access_token=%s"
	// createCustomerStrategyURL 创建新的规则组
	createCustomerStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/customer_strategy/create?access_token=%s"
	// editCustomerStrategyURL 编辑规则组及其管理范围
	editCustomerStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/customer_strategy/edit?access_token=%s"
	// delCustomerStrategyURL 删除规则组
	delCustomerStrategyURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/customer_strategy/del?access_token=%s"
)

// ExternalUserListResponse 外部联系人列表响应
type ExternalUserListResponse struct {
	util.CommonError
	ExternalUserID []string `json:"external_userid"`
}

// GetExternalUserList 获取客户列表
// @see https://developer.work.weixin.qq.com/document/path/92113
func (r *Client) GetExternalUserList(userID string) ([]string, error) {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}
	var response []byte
	response, err = util.HTTPGet(fmt.Sprintf("%s?access_token=%v&userid=%v", fetchExternalContactUserListURL, accessToken, userID))
	if err != nil {
		return nil, err
	}
	var result ExternalUserListResponse
	err = util.DecodeWithError(response, &result, "GetExternalUserList")
	return result.ExternalUserID, err
}

// ExternalUserDetailResponse 外部联系人详情响应
type ExternalUserDetailResponse struct {
	util.CommonError
	ExternalContact ExternalUser `json:"external_contact"`
	FollowUser      []FollowUser `json:"follow_user"`
	NextCursor      string       `json:"next_cursor"`
}

// ExternalUser 外部联系人
type ExternalUser struct {
	ExternalUserID  string           `json:"external_userid"`
	Name            string           `json:"name"`
	Avatar          string           `json:"avatar"`
	Type            int64            `json:"type"`
	Gender          int64            `json:"gender"`
	UnionID         string           `json:"unionid"`
	Position        string           `json:"position"`
	CorpName        string           `json:"corp_name"`
	CorpFullName    string           `json:"corp_full_name"`
	ExternalProfile *ExternalProfile `json:"external_profile,omitempty"`
}

// FollowUser 跟进用户（指企业内部用户）
type FollowUser struct {
	UserID         string        `json:"userid"`
	Remark         string        `json:"remark"`
	Description    string        `json:"description"`
	CreateTime     int64         `json:"createtime"`
	Tags           []Tag         `json:"tags"`
	RemarkCorpName string        `json:"remark_corp_name"`
	RemarkMobiles  []string      `json:"remark_mobiles"`
	OperUserID     string        `json:"oper_userid"`
	AddWay         int64         `json:"add_way"`
	WeChatChannels WechatChannel `json:"wechat_channels"`
	State          string        `json:"state"`
}

// Tag 已绑定在外部联系人的标签
type Tag struct {
	GroupName string `json:"group_name"`
	TagName   string `json:"tag_name"`
	Type      int64  `json:"type"`
	TagID     string `json:"tag_id"`
}

// WechatChannel 视频号添加的场景
type WechatChannel struct {
	NickName string `json:"nickname"`
	Source   int    `json:"source"`
}

// ExternalProfile 外部联系人的自定义展示信息,可以有多个字段和多种类型，包括文本，网页和小程序
type ExternalProfile struct {
	ExternalCorpName string         `json:"external_corp_name"`
	WechatChannels   WechatChannels `json:"wechat_channels"`
	ExternalAttr     []ExternalAttr `json:"external_attr"`
}

// WechatChannels 视频号属性。须从企业绑定到企业微信的视频号中选择，可在“我的企业”页中查看绑定的视频号
type WechatChannels struct {
	Nickname string `json:"nickname"`
	Status   int    `json:"status"`
}

// ExternalAttr 属性列表，目前支持文本、网页、小程序三种类型
type ExternalAttr struct {
	Type        int          `json:"type"`
	Name        string       `json:"name"`
	Text        *Text        `json:"text,omitempty"`
	Web         *Web         `json:"web,omitempty"`
	MiniProgram *MiniProgram `json:"miniprogram,omitempty"`
}

// Text 文本
type Text struct {
	Value string `json:"value"`
}

// Web 网页
type Web struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// MiniProgram 小程序
type MiniProgram struct {
	AppID    string `json:"appid"`
	Pagepath string `json:"pagepath"`
	Title    string `json:"title"`
}

// GetExternalUserDetail 获取外部联系人详情
// @see https://developer.work.weixin.qq.com/document/path/92114
func (r *Client) GetExternalUserDetail(externalUserID string, nextCursor ...string) (*ExternalUserDetailResponse, error) {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}
	var response []byte
	var cursor string
	if len(nextCursor) > 0 {
		cursor = nextCursor[0]
	}
	response, err = util.HTTPGet(fmt.Sprintf("%s?access_token=%v&external_userid=%v&cursor=%v", fetchExternalContactUserDetailURL, accessToken, externalUserID, cursor))
	if err != nil {
		return nil, err
	}
	result := &ExternalUserDetailResponse{}
	err = util.DecodeWithError(response, result, "get_external_user_detail")
	return result, err
}

// BatchGetExternalUserDetailsRequest 批量获取外部联系人详情请求
type BatchGetExternalUserDetailsRequest struct {
	UserIDList []string `json:"userid_list"`
	Cursor     string   `json:"cursor"`
	Limit      int      `json:"limit,omitempty"`
}

// ExternalUserDetailListResponse 批量获取外部联系人详情响应
type ExternalUserDetailListResponse struct {
	util.CommonError
	ExternalContactList []ExternalUserForBatch `json:"external_contact_list"`
	NextCursor          string                 `json:"next_cursor"`
}

// ExternalUserForBatch 批量获取外部联系人客户列表
type ExternalUserForBatch struct {
	ExternalContact ExternalContact `json:"external_contact"`
	FollowInfo      FollowInfo      `json:"follow_info"`
}

// ExternalContact 批量获取外部联系人用户信息
type ExternalContact struct {
	ExternalUserID  string `json:"external_userid"`
	Name            string `json:"name"`
	Position        string `json:"position"`
	Avatar          string `json:"avatar"`
	CorpName        string `json:"corp_name"`
	CorpFullName    string `json:"corp_full_name"`
	Type            int64  `json:"type"`
	Gender          int64  `json:"gender"`
	UnionID         string `json:"unionid"`
	ExternalProfile string `json:"external_profile"`
}

// FollowInfo 批量获取外部联系人跟进人信息
type FollowInfo struct {
	UserID         string        `json:"userid"`
	Remark         string        `json:"remark"`
	Description    string        `json:"description"`
	CreateTime     int64         `json:"createtime"`
	TagID          []string      `json:"tag_id"`
	RemarkCorpName string        `json:"remark_corp_name"`
	RemarkMobiles  []string      `json:"remark_mobiles"`
	OperUserID     string        `json:"oper_userid"`
	AddWay         int64         `json:"add_way"`
	WeChatChannels WechatChannel `json:"wechat_channels"`
}

// BatchGetExternalUserDetails 批量获取外部联系人详情
// @see https://developer.work.weixin.qq.com/document/path/92994
func (r *Client) BatchGetExternalUserDetails(request BatchGetExternalUserDetailsRequest) ([]ExternalUserForBatch, string, error) {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, "", err
	}
	var response []byte
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, "", err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", fetchBatchExternalContactUserDetailURL, accessToken), string(jsonData))
	if err != nil {
		return nil, "", err
	}
	var result ExternalUserDetailListResponse
	err = util.DecodeWithError(response, &result, "BatchGetExternalUserDetails")
	return result.ExternalContactList, result.NextCursor, err
}

// UpdateUserRemarkRequest 修改客户备注信息请求体
type UpdateUserRemarkRequest struct {
	UserID           string   `json:"userid"`
	ExternalUserID   string   `json:"external_userid"`
	Remark           string   `json:"remark"`
	Description      string   `json:"description"`
	RemarkCompany    string   `json:"remark_company"`
	RemarkMobiles    []string `json:"remark_mobiles"`
	RemarkPicMediaID string   `json:"remark_pic_mediaid"`
}

// UpdateUserRemark 修改客户备注信息
// @see https://developer.work.weixin.qq.com/document/path/92115
func (r *Client) UpdateUserRemark(request UpdateUserRemarkRequest) error {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return err
	}
	var response []byte
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", updateUserRemarkURL, accessToken), string(jsonData))
	if err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "UpdateUserRemark")
}

// ListCustomerStrategyRequest 获取规则组列表请求
type ListCustomerStrategyRequest struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

// ListCustomerStrategyResponse 获取规则组列表响应
type ListCustomerStrategyResponse struct {
	util.CommonError
	Strategy   []StrategyID `json:"strategy"`
	NextCursor string       `json:"next_cursor"`
}

// StrategyID 规则组ID
type StrategyID struct {
	StrategyID int `json:"strategy_id"`
}

// ListCustomerStrategy 获取规则组列表
// @see https://developer.work.weixin.qq.com/document/path/94883#%E8%8E%B7%E5%8F%96%E8%A7%84%E5%88%99%E7%BB%84%E5%88%97%E8%A1%A8
func (r *Client) ListCustomerStrategy(req *ListCustomerStrategyRequest) (*ListCustomerStrategyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(listCustomerStrategyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &ListCustomerStrategyResponse{}
	err = util.DecodeWithError(response, result, "ListCustomerStrategy")
	return result, err
}

// GetCustomerStrategyRequest 获取规则组详情请求
type GetCustomerStrategyRequest struct {
	StrategyID int `json:"strategy_id"`
}

// GetCustomerStrategyResponse 获取规则组详情响应
type GetCustomerStrategyResponse struct {
	util.CommonError
	Strategy Strategy `json:"strategy"`
}

// Strategy 规则组
type Strategy struct {
	StrategyID   int       `json:"strategy_id"`
	ParentID     int       `json:"parent_id"`
	StrategyName string    `json:"strategy_name"`
	CreateTime   int64     `json:"create_time"`
	AdminList    []string  `json:"admin_list"`
	Privilege    Privilege `json:"privilege"`
}

// Privilege 权限
type Privilege struct {
	ViewCustomerList        bool `json:"view_customer_list"`
	ViewCustomerData        bool `json:"view_customer_data"`
	ViewRoomList            bool `json:"view_room_list"`
	ContactMe               bool `json:"contact_me"`
	JoinRoom                bool `json:"join_room"`
	ShareCustomer           bool `json:"share_customer"`
	OperResignCustomer      bool `json:"oper_resign_customer"`
	OperResignGroup         bool `json:"oper_resign_group"`
	SendCustomerMsg         bool `json:"send_customer_msg"`
	EditWelcomeMsg          bool `json:"edit_welcome_msg"`
	ViewBehaviorData        bool `json:"view_behavior_data"`
	ViewRoomData            bool `json:"view_room_data"`
	SendGroupMsg            bool `json:"send_group_msg"`
	RoomDeduplication       bool `json:"room_deduplication"`
	RapidReply              bool `json:"rapid_reply"`
	OnjobCustomerTransfer   bool `json:"onjob_customer_transfer"`
	EditAntiSpamRule        bool `json:"edit_anti_spam_rule"`
	ExportCustomerList      bool `json:"export_customer_list"`
	ExportCustomerData      bool `json:"export_customer_data"`
	ExportCustomerGroupList bool `json:"export_customer_group_list"`
	ManageCustomerTag       bool `json:"manage_customer_tag"`
}

// GetCustomerStrategy 获取规则组详情
// @see https://developer.work.weixin.qq.com/document/path/94883#%E8%8E%B7%E5%8F%96%E8%A7%84%E5%88%99%E7%BB%84%E8%AF%A6%E6%83%85
func (r *Client) GetCustomerStrategy(req *GetCustomerStrategyRequest) (*GetCustomerStrategyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getCustomerStrategyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetCustomerStrategyResponse{}
	err = util.DecodeWithError(response, result, "GetCustomerStrategy")
	return result, err
}

// GetRangeCustomerStrategyRequest 获取规则组管理范围请求
type GetRangeCustomerStrategyRequest struct {
	StrategyID int    `json:"strategy_id"`
	Cursor     string `json:"cursor"`
	Limit      int    `json:"limit"`
}

// GetRangeCustomerStrategyResponse 获取规则组管理范围响应
type GetRangeCustomerStrategyResponse struct {
	util.CommonError
	Range      []Range `json:"range"`
	NextCursor string  `json:"next_cursor"`
}

// Range 管理范围节点
type Range struct {
	Type    int    `json:"type"`
	UserID  string `json:"userid,omitempty"`
	PartyID int    `json:"partyid,omitempty"`
}

// GetRangeCustomerStrategy 获取规则组管理范围
// @see https://developer.work.weixin.qq.com/document/path/94883#%E8%8E%B7%E5%8F%96%E8%A7%84%E5%88%99%E7%BB%84%E7%AE%A1%E7%90%86%E8%8C%83%E5%9B%B4
func (r *Client) GetRangeCustomerStrategy(req *GetRangeCustomerStrategyRequest) (*GetRangeCustomerStrategyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getRangeCustomerStrategyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetRangeCustomerStrategyResponse{}
	err = util.DecodeWithError(response, result, "GetRangeCustomerStrategy")
	return result, err
}

// CreateCustomerStrategyRequest 创建新的规则组请求
type CreateCustomerStrategyRequest struct {
	ParentID     int       `json:"parent_id"`
	StrategyName string    `json:"strategy_name"`
	AdminList    []string  `json:"admin_list"`
	Privilege    Privilege `json:"privilege"`
	Range        []Range   `json:"range"`
}

// CreateCustomerStrategyResponse 创建新的规则组响应
type CreateCustomerStrategyResponse struct {
	util.CommonError
	StrategyID int `json:"strategy_id"`
}

// CreateCustomerStrategy 创建新的规则组
// @see https://developer.work.weixin.qq.com/document/path/94883#%E5%88%9B%E5%BB%BA%E6%96%B0%E7%9A%84%E8%A7%84%E5%88%99%E7%BB%84
func (r *Client) CreateCustomerStrategy(req *CreateCustomerStrategyRequest) (*CreateCustomerStrategyResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(createCustomerStrategyURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &CreateCustomerStrategyResponse{}
	err = util.DecodeWithError(response, result, "CreateCustomerStrategy")
	return result, err
}

// EditCustomerStrategyRequest 编辑规则组及其管理范围请求
type EditCustomerStrategyRequest struct {
	StrategyID   int       `json:"strategy_id"`
	StrategyName string    `json:"strategy_name"`
	AdminList    []string  `json:"admin_list"`
	Privilege    Privilege `json:"privilege"`
	RangeAdd     []Range   `json:"range_add"`
	RangeDel     []Range   `json:"range_del"`
}

// EditCustomerStrategy 编辑规则组及其管理范围
// see https://developer.work.weixin.qq.com/document/path/94883#%E7%BC%96%E8%BE%91%E8%A7%84%E5%88%99%E7%BB%84%E5%8F%8A%E5%85%B6%E7%AE%A1%E7%90%86%E8%8C%83%E5%9B%B4
func (r *Client) EditCustomerStrategy(req *EditCustomerStrategyRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(editCustomerStrategyURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "EditCustomerStrategy")
}

// DelCustomerStrategyRequest 删除规则组请求
type DelCustomerStrategyRequest struct {
	StrategyID int `json:"strategy_id"`
}

// DelCustomerStrategy 删除规则组
// see https://developer.work.weixin.qq.com/document/path/94883#%E5%88%A0%E9%99%A4%E8%A7%84%E5%88%99%E7%BB%84
func (r *Client) DelCustomerStrategy(req *DelCustomerStrategyRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(delCustomerStrategyURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DelCustomerStrategy")
}
