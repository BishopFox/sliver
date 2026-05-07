package addresslist

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// departmentCreateURL 创建部门
	departmentCreateURL = "https://qyapi.weixin.qq.com/cgi-bin/department/create?access_token=%s"
	// departmentUpdateURL 更新部门
	departmentUpdateURL = "https://qyapi.weixin.qq.com/cgi-bin/department/update?access_token=%s"
	// departmentDeleteURL 删除部门
	departmentDeleteURL = "https://qyapi.weixin.qq.com/cgi-bin/department/delete?access_token=%s&id=%d"
	// departmentSimpleListURL 获取子部门ID列表
	departmentSimpleListURL = "https://qyapi.weixin.qq.com/cgi-bin/department/simplelist?access_token=%s&id=%d"
	// departmentListURL 获取部门列表
	departmentListURL     = "https://qyapi.weixin.qq.com/cgi-bin/department/list?access_token=%s"
	departmentListByIDURL = "https://qyapi.weixin.qq.com/cgi-bin/department/list?access_token=%s&id=%d"
	// departmentGetURL 获取单个部门详情
	departmentGetURL = "https://qyapi.weixin.qq.com/cgi-bin/department/get?access_token=%s&id=%d"
)

type (
	// DepartmentCreateRequest 创建部门数据请求
	DepartmentCreateRequest struct {
		Name     string `json:"name"`
		NameEn   string `json:"name_en,omitempty"`
		ParentID int    `json:"parentid"`
		Order    int    `json:"order,omitempty"`
		ID       int    `json:"id,omitempty"`
	}
	// DepartmentCreateResponse 创建部门数据响应
	DepartmentCreateResponse struct {
		util.CommonError
		ID int `json:"id"`
	}

	// DepartmentSimpleListResponse 获取子部门ID列表响应
	DepartmentSimpleListResponse struct {
		util.CommonError
		DepartmentID []*DepartmentID `json:"department_id"`
	}
	// DepartmentID 子部门ID
	DepartmentID struct {
		ID       int `json:"id"`
		ParentID int `json:"parentid"`
		Order    int `json:"order"`
	}

	// DepartmentListResponse 获取部门列表响应
	DepartmentListResponse struct {
		util.CommonError
		Department []*Department `json:"department"`
	}
	// Department 部门列表数据
	Department struct {
		ID               int      `json:"id"`                // 创建的部门id
		Name             string   `json:"name"`              // 部门名称
		NameEn           string   `json:"name_en"`           // 英文名称
		DepartmentLeader []string `json:"department_leader"` // 部门负责人的UserID
		ParentID         int      `json:"parentid"`          // 父部门id。根部门为1
		Order            int      `json:"order"`             // 在父部门中的次序值。order值大的排序靠前
	}
	// DepartmentGetResponse 获取单个部门详情
	DepartmentGetResponse struct {
		util.CommonError
		Department Department `json:"department"`
	}
)

// DepartmentCreate 创建部门
// see https://developer.work.weixin.qq.com/document/path/90205
func (r *Client) DepartmentCreate(req *DepartmentCreateRequest) (*DepartmentCreateResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(departmentCreateURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &DepartmentCreateResponse{}
	err = util.DecodeWithError(response, result, "DepartmentCreate")
	return result, err
}

// DepartmentUpdateRequest 更新部门请求
type DepartmentUpdateRequest struct {
	ID       int    `json:"id"`
	Name     string `json:"name,omitempty"`
	NameEn   string `json:"name_en,omitempty"`
	ParentID int    `json:"parentid,omitempty"`
	Order    int    `json:"order,omitempty"`
}

// DepartmentUpdate 更新部门
// see https://developer.work.weixin.qq.com/document/path/90206
func (r *Client) DepartmentUpdate(req *DepartmentUpdateRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(departmentUpdateURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DepartmentUpdate")
}

// DepartmentDelete 删除部门
// @see https://developer.work.weixin.qq.com/document/path/90207
func (r *Client) DepartmentDelete(departmentID int) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(departmentDeleteURL, accessToken, departmentID)); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DepartmentDelete")
}

// DepartmentSimpleList 获取子部门ID列表
// see https://developer.work.weixin.qq.com/document/path/95350
func (r *Client) DepartmentSimpleList(departmentID int) ([]*DepartmentID, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(departmentSimpleListURL, accessToken, departmentID)); err != nil {
		return nil, err
	}
	result := &DepartmentSimpleListResponse{}
	err = util.DecodeWithError(response, result, "DepartmentSimpleList")
	return result.DepartmentID, err
}

// DepartmentList 获取部门列表
// @desc https://developer.work.weixin.qq.com/document/path/90208
func (r *Client) DepartmentList() ([]*Department, error) {
	return r.DepartmentListByID(0)
}

// DepartmentListByID 获取部门列表
//
// departmentID 部门id。获取指定部门及其下的子部门（以及子部门的子部门等等，递归）
//
// @desc https://developer.work.weixin.qq.com/document/path/90208
func (r *Client) DepartmentListByID(departmentID int) ([]*Department, error) {
	var formatURL string

	// 获取accessToken
	accessToken, err := r.GetAccessToken()
	if err != nil {
		return nil, err
	}

	if departmentID > 0 {
		formatURL = fmt.Sprintf(departmentListByIDURL, accessToken, departmentID)
	} else {
		formatURL = fmt.Sprintf(departmentListURL, accessToken)
	}

	// 发起http请求
	response, err := util.HTTPGet(formatURL)
	if err != nil {
		return nil, err
	}
	// 按照结构体解析返回值
	result := &DepartmentListResponse{}
	err = util.DecodeWithError(response, result, "DepartmentList")
	// 返回数据
	return result.Department, err
}

// DepartmentGet 获取单个部门详情
// see https://developer.work.weixin.qq.com/document/path/95351
func (r *Client) DepartmentGet(departmentID int) (*Department, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(departmentGetURL, accessToken, departmentID)); err != nil {
		return nil, err
	}
	result := &DepartmentGetResponse{}
	err = util.DecodeWithError(response, result, "DepartmentGet")
	return &result.Department, err
}
