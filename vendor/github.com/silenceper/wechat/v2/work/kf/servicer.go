package kf

import (
	"encoding/json"
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// 添加接待人员
	receptionistAddAddr = "https://qyapi.weixin.qq.com/cgi-bin/kf/servicer/add?access_token=%s"
	// 删除接待人员
	receptionistDelAddr = "https://qyapi.weixin.qq.com/cgi-bin/kf/servicer/del?access_token=%s"
	// 获取接待人员列表
	receptionistListAddr = "https://qyapi.weixin.qq.com/cgi-bin/kf/servicer/list?access_token=%s&open_kfid=%s"
)

// ReceptionistOptions 添加接待人员请求参数
type ReceptionistOptions struct {
	OpenKFID         string   `json:"open_kfid"`          // 客服帐号ID
	UserIDList       []string `json:"userid_list"`        // 接待人员userid列表。第三方应用填密文userid，即open_userid 可填充个数：1 ~ 100。超过100个需分批调用。
	DepartmentIDList []int    `json:"department_id_list"` // 接待人员部门id列表 可填充个数：0 ~ 100。超过100个需分批调用。
}

// ReceptionistSchema 添加接待人员响应内容
type ReceptionistSchema struct {
	util.CommonError
	ResultList []struct {
		UserID       string `json:"userid"`
		DepartmentID int    `json:"department_id"`
		util.CommonError
	} `json:"result_list"`
}

// ReceptionistAdd 添加接待人员
// @see https://developer.work.weixin.qq.com/document/path/94646
func (r *Client) ReceptionistAdd(options ReceptionistOptions) (info ReceptionistSchema, err error) {
	var (
		accessToken string
		data        []byte
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return
	}
	if data, err = util.PostJSON(fmt.Sprintf(receptionistAddAddr, accessToken), options); err != nil {
		return
	}
	if err = json.Unmarshal(data, &info); err != nil {
		return
	}
	if info.ErrCode != 0 {
		return info, NewSDKErr(info.ErrCode, info.ErrMsg)
	}
	return
}

// ReceptionistDel 删除接待人员
// @see https://developer.work.weixin.qq.com/document/path/94647
func (r *Client) ReceptionistDel(options ReceptionistOptions) (info ReceptionistSchema, err error) {
	var (
		accessToken string
		data        []byte
	)
	accessToken, err = r.ctx.GetAccessToken()
	if err != nil {
		return
	}
	data, err = util.PostJSON(fmt.Sprintf(receptionistDelAddr, accessToken), options)
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &info); err != nil {
		return
	}
	if info.ErrCode != 0 {
		return info, NewSDKErr(info.ErrCode, info.ErrMsg)
	}
	return
}

// ReceptionistListSchema 获取接待人员列表响应内容
type ReceptionistListSchema struct {
	util.CommonError
	ReceptionistList []struct {
		UserID       string `json:"userid"`        // 接待人员的userid。第三方应用获取到的为密文userid，即open_userid
		Status       int    `json:"status"`        // 接待人员的接待状态。0:接待中,1:停止接待。第三方应用需具有“管理帐号、分配会话和收发消息”权限才可获取
		DepartmentID int    `json:"department_id"` // 接待人员部门的id
		StopType     int    `json:"stop_type"`     // 接待人员的接待状态为「停止接待」的子类型。0:停止接待,1:暂时挂起
	} `json:"servicer_list"`
}

// ReceptionistList 获取接待人员列表
// @see https://developer.work.weixin.qq.com/document/path/94645
func (r *Client) ReceptionistList(kfID string) (info ReceptionistListSchema, err error) {
	var (
		accessToken string
		data        []byte
	)
	accessToken, err = r.ctx.GetAccessToken()
	if err != nil {
		return
	}
	data, err = util.HTTPGet(fmt.Sprintf(receptionistListAddr, accessToken, kfID))
	if err != nil {
		return
	}
	if err = json.Unmarshal(data, &info); err != nil {
		return
	}
	if info.ErrCode != 0 {
		return info, NewSDKErr(info.ErrCode, info.ErrMsg)
	}
	return
}
