package externalcontact

import (
	"encoding/json"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// getUserBehaviorDataURL 获取「联系客户统计」数据
	getUserBehaviorDataURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_user_behavior_data"
	// getGroupChatStatURL 获取「群聊数据统计」数据 按群主聚合的方式
	getGroupChatStatURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/groupchat/statistic"
	// getGroupChatStatByDayURL 获取「群聊数据统计」数据 按自然日聚合的方式
	getGroupChatStatByDayURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/groupchat/statistic_group_by_day"
)

type (
	// GetUserBehaviorRequest 获取「联系客户统计」数据请求
	GetUserBehaviorRequest struct {
		UserID    []string `json:"userid"`
		PartyID   []int    `json:"partyid"`
		StartTime int      `json:"start_time"`
		EndTime   int      `json:"end_time"`
	}
	// GetUserBehaviorResponse 获取「联系客户统计」数据响应
	GetUserBehaviorResponse struct {
		util.CommonError
		BehaviorData []BehaviorData `json:"behavior_data"`
	}
	// BehaviorData 联系客户统计数据
	BehaviorData struct {
		StatTime            int     `json:"stat_time"`
		ChatCnt             int     `json:"chat_cnt"`
		MessageCnt          int     `json:"message_cnt"`
		ReplyPercentage     float64 `json:"reply_percentage"`
		AvgReplyTime        int     `json:"avg_reply_time"`
		NegativeFeedbackCnt int     `json:"negative_feedback_cnt"`
		NewApplyCnt         int     `json:"new_apply_cnt"`
		NewContactCnt       int     `json:"new_contact_cnt"`
	}
)

// GetUserBehaviorData 获取「联系客户统计」数据
// @see https://developer.work.weixin.qq.com/document/path/92132
func (r *Client) GetUserBehaviorData(req *GetUserBehaviorRequest) ([]BehaviorData, error) {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}
	var response []byte
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", getUserBehaviorDataURL, accessToken), string(jsonData))
	if err != nil {
		return nil, err
	}
	var result GetUserBehaviorResponse
	err = util.DecodeWithError(response, &result, "GetUserBehaviorData")
	return result.BehaviorData, err
}

type (
	// GetGroupChatStatRequest 获取「群聊数据统计」数据 按群主聚合的方式 请求
	GetGroupChatStatRequest struct {
		DayBeginTime int         `json:"day_begin_time"`
		DayEndTime   int         `json:"day_end_time"`
		OwnerFilter  OwnerFilter `json:"owner_filter"`
		OrderBy      int         `json:"order_by"`
		OrderAsc     int         `json:"order_asc"`
		Offset       int         `json:"offset"`
		Limit        int         `json:"limit"`
	}
	// GetGroupChatStatResponse 获取「群聊数据统计」数据 按群主聚合的方式 响应
	GetGroupChatStatResponse struct {
		util.CommonError
		Total      int                 `json:"total"`
		NextOffset int                 `json:"next_offset"`
		Items      []GroupChatStatItem `json:"items"`
	}
	// GroupChatStatItem 群聊数据统计(按群主聚合)条目
	GroupChatStatItem struct {
		Owner string                `json:"owner"`
		Data  GroupChatStatItemData `json:"data"`
	}
)

// OwnerFilter 群主过滤
type OwnerFilter struct {
	UseridList []string `json:"userid_list"`
}

// GroupChatStatItemData 群聊数据统计条目数据
type GroupChatStatItemData struct {
	NewChatCnt            int `json:"new_chat_cnt"`
	ChatTotal             int `json:"chat_total"`
	ChatHasMsg            int `json:"chat_has_msg"`
	NewMemberCnt          int `json:"new_member_cnt"`
	MemberTotal           int `json:"member_total"`
	MemberHasMsg          int `json:"member_has_msg"`
	MsgTotal              int `json:"msg_total"`
	MigrateTraineeChatCnt int `json:"migrate_trainee_chat_cnt"`
}

// GetGroupChatStat 获取「群聊数据统计」数据 按群主聚合的方式
// @see https://developer.work.weixin.qq.com/document/path/92133
func (r *Client) GetGroupChatStat(req *GetGroupChatStatRequest) (*GetGroupChatStatResponse, error) {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}
	var response []byte
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", getGroupChatStatURL, accessToken), string(jsonData))
	if err != nil {
		return nil, err
	}
	result := &GetGroupChatStatResponse{}
	err = util.DecodeWithError(response, result, "GetGroupChatStat")
	return result, err
}

type (
	// GetGroupChatStatByDayRequest 获取「群聊数据统计」数据 按自然日聚合的方式 请求
	GetGroupChatStatByDayRequest struct {
		DayBeginTime int         `json:"day_begin_time"`
		DayEndTime   int         `json:"day_end_time"`
		OwnerFilter  OwnerFilter `json:"owner_filter"`
	}
	// GetGroupChatStatByDayResponse 获取「群聊数据统计」数据 按自然日聚合的方式 响应
	GetGroupChatStatByDayResponse struct {
		util.CommonError
		Items []GetGroupChatStatByDayItem `json:"items"`
	}
	// GetGroupChatStatByDayItem 群聊数据统计(按自然日聚合)条目
	GetGroupChatStatByDayItem struct {
		StatTime int                   `json:"stat_time"`
		Data     GroupChatStatItemData `json:"data"`
	}
)

// GetGroupChatStatByDay 获取「群聊数据统计」数据 按自然日聚合的方式
// @see https://developer.work.weixin.qq.com/document/path/92133
func (r *Client) GetGroupChatStatByDay(req *GetGroupChatStatByDayRequest) ([]GetGroupChatStatByDayItem, error) {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}
	var response []byte
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", getGroupChatStatByDayURL, accessToken), string(jsonData))
	if err != nil {
		return nil, err
	}
	var result GetGroupChatStatByDayResponse
	err = util.DecodeWithError(response, &result, "GetGroupChatStatByDay")
	return result.Items, err
}
