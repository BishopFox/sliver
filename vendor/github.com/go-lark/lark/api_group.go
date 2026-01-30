package lark

import (
	"fmt"
)

const (
	groupListURL          = "/open-apis/chat/v3/list/"
	groupInfoURL          = "/open-apis/chat/v4/info/"
	createGroupURL        = "/open-apis/chat/v3/create/"
	addGroupMemberURL     = "/open-apis/chat/v4/chatter/add/"
	deleteGroupMemberURL  = "/open-apis/chat/v3/chatter/delete/"
	updateGroupURL        = "/open-apis/chat/v3/update/"
	addBotToGroupURL      = "/open-apis/bot/v4/add"
	removeBotFromGroupURL = "/open-apis/bot/v4/remove"
	disbandGroupURL       = "/open-apis/chat/v4/disband"
)

// GroupListResponse .
type GroupListResponse struct {
	BaseResponse
	HasMore bool `json:"has_more"`
	Chats   []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		OwnerID string `json:"owner_id"`
	} `json:"chats"`
}

// GroupInfoResponse .
type GroupInfoResponse struct {
	BaseResponse
	Data struct {
		AddMemberVerify        bool   `json:"add_member_verify"`
		Avatar                 string `json:"avatar"`
		ChatID                 string `json:"chat_id"`
		Description            string `json:"description"`
		GroupEmailEnabled      bool   `json:"group_email_enabled"`
		JoinMessageVisibility  string `json:"join_message_visibility"`
		LeaveMessageVisibility string `json:"leave_message_visibility"`
		Members                []struct {
			OpenID string `json:"open_id"`
		} `json:"members"`
		Name                     string `json:"name"`
		OnlyOwnerAdd             bool   `json:"only_owner_add"`
		OnlyOwnerAtAll           bool   `json:"only_owner_at_all"`
		OnlyOwnerEdit            bool   `json:"only_owner_edit"`
		OwnerOpenID              string `json:"owner_open_id"`
		SendGroupEmailPermission string `json:"send_group_email_permission"`
		SendMessagePermission    string `json:"send_message_permission"`
		ShareAllowed             bool   `json:"share_allowed"`
		Type                     string `json:"type"`
	} `json:"data"`
}

// CreateGroupResponse .
type CreateGroupResponse struct {
	BaseResponse
	OpenChatID    string   `json:"open_chat_id"`
	InvalidOpenID []string `json:"invalid_open_ids"`
}

// AddGroupMemberResponse .
type AddGroupMemberResponse struct {
	BaseResponse
	InvalidOpenID []string `json:"invalid_open_ids"`
}

// DeleteGroupMemberResponse .
type DeleteGroupMemberResponse AddGroupMemberResponse

// UpdateGroupInfoReq .
type UpdateGroupInfoReq struct {
	OpenChatID      string            `json:"open_chat_id"`
	OwnerID         string            `json:"owner_id,omitempty"`
	OwnerEmployeeID string            `json:"owner_employee_id,omitempty"`
	Name            string            `json:"name,omitempty"`
	I18nNames       map[string]string `json:"i18n_names,omitempty"`
}

// UpdateGroupInfoResponse .
type UpdateGroupInfoResponse struct {
	BaseResponse
	OpenChatID string `json:"open_chat_id"`
}

// AddBotToGroupResponse .
type AddBotToGroupResponse = BaseResponse

// RemoveBotFromGroupResponse .
type RemoveBotFromGroupResponse = BaseResponse

// DisbandGroupResponse .
type DisbandGroupResponse = BaseResponse

// GetGroupList returns group list
func (bot *Bot) GetGroupList(pageNum, pageSize int) (*GroupListResponse, error) {
	params := map[string]interface{}{
		"page":      pageNum,
		"page_size": pageSize,
	}
	var respData GroupListResponse
	err := bot.PostAPIRequest("GetGroupList", groupListURL, true, params, &respData)
	return &respData, err
}

// GetGroupInfo returns group info
func (bot *Bot) GetGroupInfo(openChatID string) (*GroupInfoResponse, error) {
	params := map[string]interface{}{
		"chat_id": openChatID,
	}
	var respData GroupInfoResponse
	err := bot.PostAPIRequest("GetGroupInfo", groupInfoURL, true, params, &respData)
	return &respData, err
}

// CreateGroup creates a group
func (bot *Bot) CreateGroup(name, description string, openID []string) (*CreateGroupResponse, error) {
	params := map[string]interface{}{
		"name":        name,
		"description": description,
		"open_ids":    openID,
	}
	var respData CreateGroupResponse
	err := bot.PostAPIRequest("CreateGroup", createGroupURL, true, params, &respData)
	return &respData, err
}

// AddGroupMember adds a group member
func (bot *Bot) AddGroupMember(openChatID string, openID []string) (*AddGroupMemberResponse, error) {
	params := map[string]interface{}{
		"chat_id":  openChatID,
		"open_ids": openID,
	}
	var respData AddGroupMemberResponse
	err := bot.PostAPIRequest("AddGroupMember", addGroupMemberURL, true, params, &respData)
	return &respData, err
}

// AddGroupMemberByUserID adds a group member
func (bot *Bot) AddGroupMemberByUserID(openChatID string, userID []string) (*AddGroupMemberResponse, error) {
	params := map[string]interface{}{
		"chat_id":  openChatID,
		"user_ids": userID,
	}
	var respData AddGroupMemberResponse
	err := bot.PostAPIRequest("AddGroupMemberByUserID", addGroupMemberURL, true, params, &respData)
	return &respData, err
}

// DeleteGroupMember deletes a group member
func (bot *Bot) DeleteGroupMember(openChatID string, openID []string) (*DeleteGroupMemberResponse, error) {
	params := map[string]interface{}{
		"open_chat_id": openChatID,
		"open_ids":     openID,
	}
	var respData DeleteGroupMemberResponse
	err := bot.PostAPIRequest("DeleteGroupMember", deleteGroupMemberURL, true, params, &respData)
	return &respData, err
}

// UpdateGroupInfo update lark group info
func (bot *Bot) UpdateGroupInfo(params *UpdateGroupInfoReq) (*UpdateGroupInfoResponse, error) {
	if params.OpenChatID == "" {
		return nil, fmt.Errorf("open chat id is empty in parameters")
	}
	var respData UpdateGroupInfoResponse
	err := bot.PostAPIRequest("UpdateGroupInfo", updateGroupURL, true, params, &respData)
	return &respData, err
}

// AddBotToGroup .
func (bot *Bot) AddBotToGroup(openChatID string) (*AddBotToGroupResponse, error) {
	params := map[string]interface{}{
		"chat_id": openChatID,
	}
	var respData AddBotToGroupResponse
	err := bot.PostAPIRequest("AddBotToGroup", addBotToGroupURL, true, params, &respData)
	return &respData, err
}

// RemoveBotFromGroup .
func (bot *Bot) RemoveBotFromGroup(openChatID string) (*RemoveBotFromGroupResponse, error) {
	params := map[string]interface{}{
		"chat_id": openChatID,
	}
	var respData RemoveBotFromGroupResponse
	err := bot.PostAPIRequest("RemoveBotFromGroup", removeBotFromGroupURL, true, params, &respData)
	return &respData, err
}

// DisbandGroup .
func (bot *Bot) DisbandGroup(openChatID string) (*DisbandGroupResponse, error) {
	params := map[string]interface{}{
		"chat_id": openChatID,
	}
	var respData DisbandGroupResponse
	err := bot.PostAPIRequest("DisbandGroup", disbandGroupURL, true, params, &respData)
	return &respData, err
}
