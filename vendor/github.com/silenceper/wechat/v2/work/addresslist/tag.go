package addresslist

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// createTagURL 创建标签
	createTagURL = "https://qyapi.weixin.qq.com/cgi-bin/tag/create?access_token=%s"
	// updateTagURL 更新标签名字
	updateTagURL = "https://qyapi.weixin.qq.com/cgi-bin/tag/update?access_token=%s"
	// deleteTagURL 删除标签
	deleteTagURL = "https://qyapi.weixin.qq.com/cgi-bin/tag/delete?access_token=%s&tagid=%d"
	// getTagURL 获取标签成员
	getTagURL = "https://qyapi.weixin.qq.com/cgi-bin/tag/get?access_token=%s&tagid=%d"
	// addTagUsersURL 增加标签成员
	addTagUsersURL = "https://qyapi.weixin.qq.com/cgi-bin/tag/addtagusers?access_token=%s"
	// delTagUsersURL 删除标签成员
	delTagUsersURL = "https://qyapi.weixin.qq.com/cgi-bin/tag/deltagusers?access_token=%s"
	// listTagURL 获取标签列表
	listTagURL = "https://qyapi.weixin.qq.com/cgi-bin/tag/list?access_token=%s"
)

type (
	// CreateTagRequest 创建标签请求
	CreateTagRequest struct {
		TagName string `json:"tagname"`
		TagID   int    `json:"tagid,omitempty"`
	}
	// CreateTagResponse 创建标签响应
	CreateTagResponse struct {
		util.CommonError
		TagID int `json:"tagid"`
	}
)

// CreateTag 创建标签
// see https://developer.work.weixin.qq.com/document/path/90210
func (r *Client) CreateTag(req *CreateTagRequest) (*CreateTagResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(createTagURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &CreateTagResponse{}
	err = util.DecodeWithError(response, result, "CreateTag")
	return result, err
}

type (
	// UpdateTagRequest 更新标签名字请求
	UpdateTagRequest struct {
		TagID   int    `json:"tagid"`
		TagName string `json:"tagname"`
	}
)

// UpdateTag 更新标签名字
// see https://developer.work.weixin.qq.com/document/path/90211
func (r *Client) UpdateTag(req *UpdateTagRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(updateTagURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "UpdateTag")
}

// DeleteTag 删除标签
// @see https://developer.work.weixin.qq.com/document/path/90212
func (r *Client) DeleteTag(tagID int) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(deleteTagURL, accessToken, tagID)); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DeleteTag")
}

type (
	// GetTagResponse 获取标签成员响应
	GetTagResponse struct {
		util.CommonError
		TagName   string           `json:"tagname"`
		UserList  []GetTagUserList `json:"userlist"`
		PartyList []int            `json:"partylist"`
	}
	// GetTagUserList 标签中包含的成员列表
	GetTagUserList struct {
		UserID string `json:"userid"`
		Name   string `json:"name"`
	}
)

// GetTag 获取标签成员
// @see https://developer.work.weixin.qq.com/document/path/90213
func (r *Client) GetTag(tagID int) (*GetTagResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getTagURL, accessToken, tagID)); err != nil {
		return nil, err
	}
	result := &GetTagResponse{}
	err = util.DecodeWithError(response, result, "GetTag")
	return result, err
}

type (
	// AddTagUsersRequest 增加标签成员请求
	AddTagUsersRequest struct {
		TagID     int      `json:"tagid"`
		UserList  []string `json:"userlist"`
		PartyList []int    `json:"partylist"`
	}
	// AddTagUsersResponse 增加标签成员响应
	AddTagUsersResponse struct {
		util.CommonError
		InvalidList  string `json:"invalidlist"`
		InvalidParty []int  `json:"invalidparty"`
	}
)

// AddTagUsers 增加标签成员
// see https://developer.work.weixin.qq.com/document/path/90214
func (r *Client) AddTagUsers(req *AddTagUsersRequest) (*AddTagUsersResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(addTagUsersURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &AddTagUsersResponse{}
	err = util.DecodeWithError(response, result, "AddTagUsers")
	return result, err
}

type (
	// DelTagUsersRequest 删除标签成员请求
	DelTagUsersRequest struct {
		TagID     int      `json:"tagid"`
		UserList  []string `json:"userlist"`
		PartyList []int    `json:"partylist"`
	}
	// DelTagUsersResponse 删除标签成员响应
	DelTagUsersResponse struct {
		util.CommonError
		InvalidList  string `json:"invalidlist"`
		InvalidParty []int  `json:"invalidparty"`
	}
)

// DelTagUsers 删除标签成员
// see https://developer.work.weixin.qq.com/document/path/90215
func (r *Client) DelTagUsers(req *DelTagUsersRequest) (*DelTagUsersResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(delTagUsersURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &DelTagUsersResponse{}
	err = util.DecodeWithError(response, result, "DelTagUsers")
	return result, err
}

type (
	// ListTagResponse 获取标签列表响应
	ListTagResponse struct {
		util.CommonError
		TagList []Tag `json:"taglist"`
	}
	// Tag 标签
	Tag struct {
		TagID   int    `json:"tagid"`
		TagName string `json:"tagname"`
	}
)

// ListTag 获取标签列表
// @see https://developer.work.weixin.qq.com/document/path/90216
func (r *Client) ListTag() (*ListTagResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(listTagURL, accessToken)); err != nil {
		return nil, err
	}
	result := &ListTagResponse{}
	err = util.DecodeWithError(response, result, "ListTag")
	return result, err
}
