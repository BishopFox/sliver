package externalcontact

import (
	"encoding/json"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// getCropTagURL 获取标签列表
	getCropTagURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_corp_tag_list"
	// addCropTagURL 添加标签
	addCropTagURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/add_corp_tag"
	// editCropTagURL 修改标签
	editCropTagURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/edit_corp_tag"
	// delCropTagURL 删除标签
	delCropTagURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/del_corp_tag"
	// markCropTagURL 为客户打上、删除标签
	markCropTagURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/mark_tag"
	// getStrategyTagListURL 获取指定规则组下的企业客户标签
	getStrategyTagListURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/get_strategy_tag_list?access_token=%s"
	// addStrategyTagURL 为指定规则组创建企业客户标签
	addStrategyTagURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/add_strategy_tag?access_token=%s"
	// editStrategyTagURL 编辑指定规则组下的企业客户标签
	editStrategyTagURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/edit_strategy_tag?access_token=%s"
	// delStrategyTagURL 删除指定规则组下的企业客户标签
	delStrategyTagURL = "https://qyapi.weixin.qq.com/cgi-bin/externalcontact/del_strategy_tag?access_token=%s"
)

// GetCropTagRequest 获取企业标签请求
type GetCropTagRequest struct {
	TagID   []string `json:"tag_id"`
	GroupID []string `json:"group_id"`
}

// GetCropTagListResponse 获取企业标签列表响应
type GetCropTagListResponse struct {
	util.CommonError
	TagGroup []TagGroup `json:"tag_group"`
}

// TagGroup 企业标签组
type TagGroup struct {
	GroupID    string            `json:"group_id"`
	GroupName  string            `json:"group_name"`
	CreateTime int64             `json:"create_time"`
	GroupOrder int               `json:"group_order"`
	Deleted    bool              `json:"deleted"`
	Tag        []TagGroupTagItem `json:"tag"`
}

// TagGroupTagItem 企业标签内的子项
type TagGroupTagItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CreateTime int64  `json:"create_time"`
	Order      int    `json:"order"`
	Deleted    bool   `json:"deleted"`
}

// GetCropTagList 获取企业标签库
// @see https://developer.work.weixin.qq.com/document/path/92117
func (r *Client) GetCropTagList(req GetCropTagRequest) ([]TagGroup, error) {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}
	var response []byte
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", getCropTagURL, accessToken), string(jsonData))
	if err != nil {
		return nil, err
	}
	var result GetCropTagListResponse
	err = util.DecodeWithError(response, &result, "GetCropTagList")
	return result.TagGroup, err
}

// AddCropTagRequest 添加企业标签请求
type AddCropTagRequest struct {
	GroupID   string           `json:"group_id,omitempty"`
	GroupName string           `json:"group_name"`
	Order     int              `json:"order"`
	Tag       []AddCropTagItem `json:"tag"`
	AgentID   int              `json:"agentid"`
}

// AddCropTagItem 添加企业标签子项
type AddCropTagItem struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
}

// AddCropTagResponse 添加企业标签响应
type AddCropTagResponse struct {
	util.CommonError
	TagGroup TagGroup `json:"tag_group"`
}

// AddCropTag 添加企业客户标签
// @see https://developer.work.weixin.qq.com/document/path/92117
func (r *Client) AddCropTag(req AddCropTagRequest) (*TagGroup, error) {
	var accessToken string
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}
	var response []byte
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", addCropTagURL, accessToken), string(jsonData))
	if err != nil {
		return nil, err
	}
	var result AddCropTagResponse
	err = util.DecodeWithError(response, &result, "AddCropTag")
	return &result.TagGroup, err
}

// EditCropTagRequest 编辑客户企业标签请求
type EditCropTagRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Order   int    `json:"order"`
	AgentID string `json:"agent_id"`
}

// EditCropTag 修改企业客户标签
// @see https://developer.work.weixin.qq.com/document/path/92117
func (r *Client) EditCropTag(req EditCropTagRequest) error {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return err
	}
	var response []byte
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", editCropTagURL, accessToken), string(jsonData))
	if err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "EditCropTag")
}

// DeleteCropTagRequest 删除企业标签请求
type DeleteCropTagRequest struct {
	TagID   []string `json:"tag_id"`
	GroupID []string `json:"group_id"`
	AgentID string   `json:"agent_id"`
}

// DeleteCropTag 删除企业客户标签
// @see https://developer.work.weixin.qq.com/document/path/92117
func (r *Client) DeleteCropTag(req DeleteCropTagRequest) error {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return err
	}
	var response []byte
	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", delCropTagURL, accessToken), string(jsonData))
	if err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DeleteCropTag")
}

// MarkTagRequest 给客户打标签请求
// 相关文档地址：https://developer.work.weixin.qq.com/document/path/92118
type MarkTagRequest struct {
	UserID         string   `json:"userid"`
	ExternalUserID string   `json:"external_userid"`
	AddTag         []string `json:"add_tag"`
	RemoveTag      []string `json:"remove_tag"`
}

// MarkTag 为客户打上标签
// @see https://developer.work.weixin.qq.com/document/path/92118
func (r *Client) MarkTag(request MarkTagRequest) error {
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return err
	}
	var response []byte
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}
	response, err = util.HTTPPost(fmt.Sprintf("%s?access_token=%v", markCropTagURL, accessToken), string(jsonData))
	if err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "MarkTag")
}

// GetStrategyTagListRequest 获取指定规则组下的企业客户标签请求
type GetStrategyTagListRequest struct {
	StrategyID int      `json:"strategy_id"`
	TagID      []string `json:"tag_id"`
	GroupID    []string `json:"group_id"`
}

// GetStrategyTagListResponse 获取指定规则组下的企业客户标签响应
type GetStrategyTagListResponse struct {
	util.CommonError
	TagGroup []StrategyTagGroup `json:"tag_group"`
}

// StrategyTagGroup 规则组下的企业标签组
type StrategyTagGroup struct {
	GroupID    string        `json:"group_id"`
	GroupName  string        `json:"group_name"`
	CreateTime int64         `json:"create_time"`
	Order      int           `json:"order"`
	StrategyID int           `json:"strategy_id"`
	Tag        []StrategyTag `json:"tag"`
}

// StrategyTag 规则组下的企业标签
type StrategyTag struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CreateTime int64  `json:"create_time"`
	Order      int    `json:"order"`
}

// GetStrategyTagList 获取指定规则组下的企业客户标签
// @see https://developer.work.weixin.qq.com/document/path/94882#%E8%8E%B7%E5%8F%96%E6%8C%87%E5%AE%9A%E8%A7%84%E5%88%99%E7%BB%84%E4%B8%8B%E7%9A%84%E4%BC%81%E4%B8%9A%E5%AE%A2%E6%88%B7%E6%A0%87%E7%AD%BE
func (r *Client) GetStrategyTagList(req *GetStrategyTagListRequest) (*GetStrategyTagListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getStrategyTagListURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetStrategyTagListResponse{}
	err = util.DecodeWithError(response, result, "GetStrategyTagList")
	return result, err
}

// AddStrategyTagRequest 为指定规则组创建企业客户标签请求
type AddStrategyTagRequest struct {
	StrategyID int                         `json:"strategy_id"`
	GroupID    string                      `json:"group_id"`
	GroupName  string                      `json:"group_name"`
	Order      int                         `json:"order"`
	Tag        []AddStrategyTagRequestItem `json:"tag"`
}

// AddStrategyTagRequestItem 为指定规则组创建企业客户标签请求条目
type AddStrategyTagRequestItem struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
}

// AddStrategyTagResponse 为指定规则组创建企业客户标签响应
type AddStrategyTagResponse struct {
	util.CommonError
	TagGroup AddStrategyTagResponseTagGroup `json:"tag_group"`
}

// AddStrategyTagResponseTagGroup 为指定规则组创建企业客户标签响应标签组
type AddStrategyTagResponseTagGroup struct {
	GroupID    string                       `json:"group_id"`
	GroupName  string                       `json:"group_name"`
	CreateTime int64                        `json:"create_time"`
	Order      int                          `json:"order"`
	Tag        []AddStrategyTagResponseItem `json:"tag"`
}

// AddStrategyTagResponseItem 标签组内的标签列表
type AddStrategyTagResponseItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CreateTime int64  `json:"create_time"`
	Order      int    `json:"order"`
}

// AddStrategyTag 为指定规则组创建企业客户标签
// @see https://developer.work.weixin.qq.com/document/path/94882#%E4%B8%BA%E6%8C%87%E5%AE%9A%E8%A7%84%E5%88%99%E7%BB%84%E5%88%9B%E5%BB%BA%E4%BC%81%E4%B8%9A%E5%AE%A2%E6%88%B7%E6%A0%87%E7%AD%BE
func (r *Client) AddStrategyTag(req *AddStrategyTagRequest) (*AddStrategyTagResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(addStrategyTagURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &AddStrategyTagResponse{}
	err = util.DecodeWithError(response, result, "AddStrategyTag")
	return result, err
}

// EditStrategyTagRequest 编辑指定规则组下的企业客户标签请求
type EditStrategyTagRequest struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
}

// EditStrategyTag 编辑指定规则组下的企业客户标签
// see https://developer.work.weixin.qq.com/document/path/94882#%E7%BC%96%E8%BE%91%E6%8C%87%E5%AE%9A%E8%A7%84%E5%88%99%E7%BB%84%E4%B8%8B%E7%9A%84%E4%BC%81%E4%B8%9A%E5%AE%A2%E6%88%B7%E6%A0%87%E7%AD%BE
func (r *Client) EditStrategyTag(req *EditStrategyTagRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(editStrategyTagURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "EditStrategyTag")
}

// DelStrategyTagRequest 删除指定规则组下的企业客户标签请求
type DelStrategyTagRequest struct {
	TagID   []string `json:"tag_id"`
	GroupID []string `json:"group_id"`
}

// DelStrategyTag 删除指定规则组下的企业客户标签
// see https://developer.work.weixin.qq.com/document/path/94882#%E5%88%A0%E9%99%A4%E6%8C%87%E5%AE%9A%E8%A7%84%E5%88%99%E7%BB%84%E4%B8%8B%E7%9A%84%E4%BC%81%E4%B8%9A%E5%AE%A2%E6%88%B7%E6%A0%87%E7%AD%BE
func (r *Client) DelStrategyTag(req *DelStrategyTagRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(delStrategyTagURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DelStrategyTag")
}
