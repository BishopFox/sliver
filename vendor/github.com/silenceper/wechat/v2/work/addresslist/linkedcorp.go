package addresslist

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// getPermListURL 获取应用的可见范围
	getPermListURL = "https://qyapi.weixin.qq.com/cgi-bin/linkedcorp/agent/get_perm_list?access_token=%s"
	// getLinkedCorpUserURL 获取互联企业成员详细信息
	getLinkedCorpUserURL = "https://qyapi.weixin.qq.com/cgi-bin/linkedcorp/user/get?access_token=%s"
	// linkedCorpSimpleListURL 获取互联企业部门成员
	linkedCorpSimpleListURL = "https://qyapi.weixin.qq.com/cgi-bin/linkedcorp/user/simplelist?access_token=%s"
	// linkedCorpUserListURL 获取互联企业部门成员详情
	linkedCorpUserListURL = "https://qyapi.weixin.qq.com/cgi-bin/linkedcorp/user/list?access_token=%s"
	// linkedCorpDepartmentListURL 获取互联企业部门列表
	linkedCorpDepartmentListURL = "https://qyapi.weixin.qq.com/cgi-bin/linkedcorp/department/list?access_token=%s"
)

// GetPermListResponse 获取应用的可见范围响应
type GetPermListResponse struct {
	util.CommonError
	UserIDs       []string `json:"userids"`
	DepartmentIDs []string `json:"department_ids"`
}

// GetPermList 获取应用的可见范围
// see https://developer.work.weixin.qq.com/document/path/93172
func (r *Client) GetPermList() (*GetPermListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPPost(fmt.Sprintf(getPermListURL, accessToken), ""); err != nil {
		return nil, err
	}
	result := &GetPermListResponse{}
	err = util.DecodeWithError(response, result, "GetPermList")
	return result, err
}

// GetLinkedCorpUserRequest 获取互联企业成员详细信息请求
type GetLinkedCorpUserRequest struct {
	UserID string `json:"userid"`
}

// GetLinkedCorpUserResponse 获取互联企业成员详细信息响应
type GetLinkedCorpUserResponse struct {
	util.CommonError
	UserInfo LinkedCorpUserInfo `json:"user_info"`
}

// LinkedCorpUserInfo 互联企业成员详细信息
type LinkedCorpUserInfo struct {
	UserID     string   `json:"userid"`
	Name       string   `json:"name"`
	Department []string `json:"department"`
	Mobile     string   `json:"mobile"`
	Telephone  string   `json:"telephone"`
	Email      string   `json:"email"`
	Position   string   `json:"position"`
	CorpID     string   `json:"corpid"`
	Extattr    Extattr  `json:"extattr"`
}

// Extattr 互联企业成员详细信息扩展属性
type Extattr struct {
	Attrs []ExtattrItem `json:"attrs"`
}

// ExtattrItem 互联企业成员详细信息扩展属性条目
type ExtattrItem struct {
	Name  string          `json:"name"`
	Value string          `json:"value,omitempty"`
	Type  int             `json:"type"`
	Text  ExtattrItemText `json:"text,omitempty"`
	Web   ExtattrItemWeb  `json:"web,omitempty"`
}

// ExtattrItemText 互联企业成员详细信息自定义属性(文本)
type ExtattrItemText struct {
	Value string `json:"value"`
}

// ExtattrItemWeb 互联企业成员详细信息自定义属性(网页)
type ExtattrItemWeb struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// GetLinkedCorpUser 获取互联企业成员详细信息
// see https://developer.work.weixin.qq.com/document/path/93171
func (r *Client) GetLinkedCorpUser(req *GetLinkedCorpUserRequest) (*GetLinkedCorpUserResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getLinkedCorpUserURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetLinkedCorpUserResponse{}
	err = util.DecodeWithError(response, result, "GetLinkedCorpUser")
	return result, err
}

// LinkedCorpSimpleListRequest 获取互联企业部门成员请求
type LinkedCorpSimpleListRequest struct {
	DepartmentID string `json:"department_id"`
}

// LinkedCorpSimpleListResponse 获取互联企业部门成员响应
type LinkedCorpSimpleListResponse struct {
	util.CommonError
	Userlist []LinkedCorpUser `json:"userlist"`
}

// LinkedCorpUser 企业部门成员
type LinkedCorpUser struct {
	UserID     string   `json:"userid"`
	Name       string   `json:"name"`
	Department []string `json:"department"`
	CorpID     string   `json:"corpid"`
}

// LinkedCorpSimpleList 获取互联企业部门成员
// see https://developer.work.weixin.qq.com/document/path/93168
func (r *Client) LinkedCorpSimpleList(req *LinkedCorpSimpleListRequest) (*LinkedCorpSimpleListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(linkedCorpSimpleListURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &LinkedCorpSimpleListResponse{}
	err = util.DecodeWithError(response, result, "LinkedCorpSimpleList")
	return result, err
}

// LinkedCorpUserListRequest 获取互联企业部门成员详情请求
type LinkedCorpUserListRequest struct {
	DepartmentID string `json:"department_id"`
}

// LinkedCorpUserListResponse 获取互联企业部门成员详情响应
type LinkedCorpUserListResponse struct {
	util.CommonError
	UserList []LinkedCorpUserInfo `json:"userlist"`
}

// LinkedCorpUserList 获取互联企业部门成员详情
// see https://developer.work.weixin.qq.com/document/path/93169
func (r *Client) LinkedCorpUserList(req *LinkedCorpUserListRequest) (*LinkedCorpUserListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(linkedCorpUserListURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &LinkedCorpUserListResponse{}
	err = util.DecodeWithError(response, result, "LinkedCorpUserList")
	return result, err
}

// LinkedCorpDepartmentListRequest 获取互联企业部门列表请求
type LinkedCorpDepartmentListRequest struct {
	DepartmentID string `json:"department_id"`
}

// LinkedCorpDepartmentListResponse 获取互联企业部门列表响应
type LinkedCorpDepartmentListResponse struct {
	util.CommonError
	DepartmentList []LinkedCorpDepartment `json:"department_list"`
}

// LinkedCorpDepartment 互联企业部门
type LinkedCorpDepartment struct {
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	ParentID       string `json:"parentid"`
	Order          int    `json:"order"`
}

// LinkedCorpDepartmentList 获取互联企业部门列表
// see https://developer.work.weixin.qq.com/document/path/93170
func (r *Client) LinkedCorpDepartmentList(req *LinkedCorpDepartmentListRequest) (*LinkedCorpDepartmentListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(linkedCorpDepartmentListURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &LinkedCorpDepartmentListResponse{}
	err = util.DecodeWithError(response, result, "LinkedCorpDepartmentList")
	return result, err
}
