package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/silenceper/wechat/v2/officialaccount/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	userInfoURL      = "https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN"
	userInfoBatchURL = "https://api.weixin.qq.com/cgi-bin/user/info/batchget"
	updateRemarkURL  = "https://api.weixin.qq.com/cgi-bin/user/info/updateremark?access_token=%s"
	userListURL      = "https://api.weixin.qq.com/cgi-bin/user/get"
)

// User 用户管理
type User struct {
	*context.Context
}

// NewUser 实例化
func NewUser(context *context.Context) *User {
	user := new(User)
	user.Context = context
	return user
}

// Info 用户基本信息
type Info struct {
	util.CommonError
	userInfo
}

// 用户基本信息
type userInfo struct {
	Subscribe      int32   `json:"subscribe"`
	OpenID         string  `json:"openid"`
	Nickname       string  `json:"nickname"`
	Sex            int32   `json:"sex"`
	City           string  `json:"city"`
	Country        string  `json:"country"`
	Province       string  `json:"province"`
	Language       string  `json:"language"`
	Headimgurl     string  `json:"headimgurl"`
	SubscribeTime  int32   `json:"subscribe_time"`
	UnionID        string  `json:"unionid"`
	Remark         string  `json:"remark"`
	GroupID        int32   `json:"groupid"`
	TagIDList      []int32 `json:"tagid_list"`
	SubscribeScene string  `json:"subscribe_scene"`
	QrScene        int     `json:"qr_scene"`
	QrSceneStr     string  `json:"qr_scene_str"`
}

// OpenidList 用户列表
type OpenidList struct {
	util.CommonError

	Total int `json:"total"`
	Count int `json:"count"`
	Data  struct {
		OpenIDs []string `json:"openid"`
	} `json:"data"`
	NextOpenID string `json:"next_openid"`
}

// GetUserInfo 获取用户基本信息
func (user *User) GetUserInfo(openID string) (userInfo *Info, err error) {
	var accessToken string
	accessToken, err = user.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(userInfoURL, accessToken, openID)
	var response []byte
	response, err = util.HTTPGet(uri)
	if err != nil {
		return
	}
	userInfo = new(Info)
	err = json.Unmarshal(response, userInfo)
	if err != nil {
		return
	}
	if userInfo.ErrCode != 0 {
		err = fmt.Errorf("GetUserInfo Error , errcode=%d , errmsg=%s", userInfo.ErrCode, userInfo.ErrMsg)
		return
	}
	return
}

// BatchGetUserInfoParams 批量获取用户基本信息参数
type BatchGetUserInfoParams struct {
	UserList []BatchGetUserListItem `json:"user_list"` // 需要批量获取基本信息的用户列表
}

// BatchGetUserListItem 需要获取基本信息的用户
type BatchGetUserListItem struct {
	OpenID string `json:"openid"` // 用户的标识，对当前公众号唯一
	Lang   string `json:"lang"`   // 国家地区语言版本，zh_CN 简体，zh_TW 繁体，en 英语，默认为zh-CN
}

// InfoList 用户基本信息列表
type InfoList struct {
	util.CommonError
	UserInfoList []userInfo `json:"user_info_list"`
}

// BatchGetUserInfo 批量获取用户基本信息
func (user *User) BatchGetUserInfo(params BatchGetUserInfoParams) (*InfoList, error) {
	if len(params.UserList) > 100 {
		return nil, errors.New("params length must be less than or equal to 100")
	}

	ak, err := user.GetAccessToken()
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("%s?access_token=%s", userInfoBatchURL, ak)
	res, err := util.PostJSON(uri, params)
	if err != nil {
		return nil, err
	}

	var data InfoList
	err = util.DecodeWithError(res, &data, "BatchGetUserInfo")
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// UpdateRemark 设置用户备注名
func (user *User) UpdateRemark(openID, remark string) (err error) {
	var accessToken string
	accessToken, err = user.GetAccessToken()
	if err != nil {
		return
	}

	uri := fmt.Sprintf(updateRemarkURL, accessToken)
	var response []byte
	response, err = util.PostJSON(uri, map[string]string{"openid": openID, "remark": remark})
	if err != nil {
		return
	}

	return util.DecodeWithCommonError(response, "UpdateRemark")
}

// ListUserOpenIDs 返回用户列表
func (user *User) ListUserOpenIDs(nextOpenid ...string) (*OpenidList, error) {
	accessToken, err := user.GetAccessToken()
	if err != nil {
		return nil, err
	}

	uri, _ := url.Parse(userListURL)
	q := uri.Query()
	q.Set("access_token", accessToken)
	if len(nextOpenid) > 0 && nextOpenid[0] != "" {
		q.Set("next_openid", nextOpenid[0])
	}
	uri.RawQuery = q.Encode()

	response, err := util.HTTPGet(uri.String())
	if err != nil {
		return nil, err
	}

	userlist := OpenidList{}

	err = util.DecodeWithError(response, &userlist, "ListUserOpenIDs")
	if err != nil {
		return nil, err
	}

	return &userlist, nil
}

// ListAllUserOpenIDs 返回所有用户OpenID列表
func (user *User) ListAllUserOpenIDs() ([]string, error) {
	nextOpenid := ""
	openids := make([]string, 0)
	count := 0
	for {
		ul, err := user.ListUserOpenIDs(nextOpenid)
		if err != nil {
			return nil, err
		}
		openids = append(openids, ul.Data.OpenIDs...)
		count += ul.Count
		if ul.Total > count {
			nextOpenid = ul.NextOpenID
		} else {
			return openids, nil
		}
	}
}
