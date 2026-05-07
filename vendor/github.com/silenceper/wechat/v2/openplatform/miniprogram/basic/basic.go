package basic

import (
	"fmt"

	openContext "github.com/silenceper/wechat/v2/openplatform/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	getAccountBasicInfoURL = "https://api.weixin.qq.com/cgi-bin/account/getaccountbasicinfo"
	checkNickNameURL       = "https://api.weixin.qq.com/cgi-bin/wxverify/checkwxverifynickname"
	setNickNameURL         = "https://api.weixin.qq.com/wxa/setnickname"
	setSignatureURL        = "https://api.weixin.qq.com/cgi-bin/account/modifysignature"
	setHeadImageURL        = "https://api.weixin.qq.com/cgi-bin/account/modifyheadimage"
	getSearchStatusURL     = "https://api.weixin.qq.com/wxa/getwxasearchstatus"
	setSearchStatusURL     = "https://api.weixin.qq.com/wxa/changewxasearchstatus"
)

// Basic 基础信息设置
type Basic struct {
	*openContext.Context
	appID string
}

// NewBasic new
func NewBasic(opContext *openContext.Context, appID string) *Basic {
	return &Basic{Context: opContext, appID: appID}
}

// AccountBasicInfo 基础信息
type AccountBasicInfo struct {
	util.CommonError
}

// GetAccountBasicInfo 获取小程序基础信息
//
//reference:https://developers.weixin.qq.com/doc/oplatform/Third-party_Platforms/Mini_Programs/Mini_Program_Information_Settings.html
func (basic *Basic) GetAccountBasicInfo() (*AccountBasicInfo, error) {
	ak, err := basic.GetAuthrAccessToken(basic.AppID)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s?access_token=%s", getAccountBasicInfoURL, ak)
	data, err := util.HTTPGet(url)
	if err != nil {
		return nil, err
	}
	result := &AccountBasicInfo{}
	if err := util.DecodeWithError(data, result, "account/getaccountbasicinfo"); err != nil {
		return nil, err
	}
	return result, nil
}

// modify_domain设置服务器域名
// TODO
// func (encryptor *Basic) modifyDomain() {
// }

// CheckNickNameResp 小程序名称检测结果
type CheckNickNameResp struct {
	util.CommonError
	HitCondition bool   `json:"hit_condition"` // 是否命中关键字策略。若命中，可以选填关键字材料
	Wording      string `json:"wording"`       // 命中关键字的说明描述
}

// CheckNickName 检测微信认证的名称是否符合规则
// ref: https://developers.weixin.qq.com/doc/oplatform/openApi/OpenApiDoc/miniprogram-management/basic-info-management/checkNickName.html
func (basic *Basic) CheckNickName(nickname string) (*CheckNickNameResp, error) {
	ak, err := basic.GetAuthrAccessToken(basic.AppID)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s?access_token=%s", checkNickNameURL, ak)
	data, err := util.PostJSON(url, map[string]string{
		"nick_name": nickname,
	})
	if err != nil {
		return nil, err
	}
	res := &CheckNickNameResp{}
	err = util.DecodeWithError(data, res, "CheckNickName")
	return res, err
}

// SetNickNameResp 设置小程序名称结果
type SetNickNameResp struct {
	util.CommonError
	AuditID int64  `json:"audit_id"` // 审核单Id，通过用于查询改名审核状态
	Wording string `json:"wording"`  // 材料说明
}

// SetNickNameParam 设置小程序名称参数
type SetNickNameParam struct {
	NickName           string `json:"nick_name"`                      // 昵称，不支持包含“小程序”关键字的昵称
	IDCard             string `json:"id_card,omitempty"`              // 身份证照片 mediaid，个人号必填
	License            string `json:"license,omitempty"`              // 组织机构代码证或营业执照 mediaid，组织号必填
	NameingOtherStuff1 string `json:"naming_other_stuff_1,omitempty"` // 其他证明材料 mediaid，选填
	NameingOtherStuff2 string `json:"naming_other_stuff_2,omitempty"` // 其他证明材料 mediaid，选填
	NameingOtherStuff3 string `json:"naming_other_stuff_3,omitempty"` // 其他证明材料 mediaid，选填
	NameingOtherStuff4 string `json:"naming_other_stuff_4,omitempty"` // 其他证明材料 mediaid，选填
	NameingOtherStuff5 string `json:"naming_other_stuff_5,omitempty"` // 其他证明材料 mediaid，选填
}

// SetNickName 设置小程序名称
func (basic *Basic) SetNickName(nickname string) (*SetNickNameResp, error) {
	return basic.SetNickNameFull(&SetNickNameParam{
		NickName: nickname,
	})
}

// SetNickNameFull 设置小程序名称
// ref: https://developers.weixin.qq.com/doc/oplatform/openApi/OpenApiDoc/miniprogram-management/basic-info-management/setNickName.html
func (basic *Basic) SetNickNameFull(param *SetNickNameParam) (*SetNickNameResp, error) {
	ak, err := basic.GetAuthrAccessToken(basic.AppID)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s?access_token=%s", setNickNameURL, ak)
	data, err := util.PostJSON(url, param)
	if err != nil {
		return nil, err
	}
	res := &SetNickNameResp{}
	err = util.DecodeWithError(data, res, "SetNickName")
	return res, err
}

// SetSignatureResp 小程序功能介绍修改结果
type SetSignatureResp struct {
	util.CommonError
}

// SetSignature 小程序修改功能介绍
// ref: https://developers.weixin.qq.com/doc/oplatform/openApi/OpenApiDoc/miniprogram-management/basic-info-management/setSignature.html
func (basic *Basic) SetSignature(signature string) error {
	ak, err := basic.GetAuthrAccessToken(basic.AppID)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s?access_token=%s", setSignatureURL, ak)
	data, err := util.PostJSON(url, map[string]string{
		"signature": signature,
	})
	if err != nil {
		return err
	}
	return util.DecodeWithError(data, &SetSignatureResp{}, "SetSignature")
}

// GetSearchStatusResp 查询小程序当前是否可被搜索
type GetSearchStatusResp struct {
	util.CommonError
	Status int `json:"status"` // 1 表示不可搜索，0 表示可搜索
}

// GetSearchStatus 查询小程序当前是否可被搜索
// ref: https://developers.weixin.qq.com/doc/oplatform/openApi/OpenApiDoc/miniprogram-management/basic-info-management/getSearchStatus.html
func (basic *Basic) GetSearchStatus(signature string) (*GetSearchStatusResp, error) {
	ak, err := basic.GetAuthrAccessToken(basic.AppID)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s?access_token=%s", getSearchStatusURL, ak)
	data, err := util.HTTPGet(url)
	if err != nil {
		return nil, err
	}
	res := &GetSearchStatusResp{}
	err = util.DecodeWithError(data, res, "GetSearchStatus")
	return res, err
}

// SetSearchStatusResp 小程序是否可被搜索修改结果
type SetSearchStatusResp struct {
	util.CommonError
}

// SetSearchStatus 修改小程序是否可被搜索
// status: 1 表示不可搜索，0 表示可搜索
// ref: https://developers.weixin.qq.com/doc/oplatform/openApi/OpenApiDoc/miniprogram-management/basic-info-management/setSearchStatus.html
func (basic *Basic) SetSearchStatus(status int) error {
	ak, err := basic.GetAuthrAccessToken(basic.AppID)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s?access_token=%s", setSearchStatusURL, ak)
	data, err := util.PostJSON(url, map[string]int{
		"status": status,
	})
	if err != nil {
		return err
	}
	return util.DecodeWithError(data, &SetSearchStatusResp{}, "SetSearchStatus")
}

// SetHeadImageResp 小程序头像修改结果
type SetHeadImageResp struct {
	util.CommonError
}

// SetHeadImageParam 小程序头像修改参数
type SetHeadImageParam struct {
	HeadImageMediaID string `json:"head_img_media_id"` // 头像素材 media_id
	X1               string `json:"x1"`                // 裁剪框左上角 x 坐标（取值范围：[0, 1]）
	Y1               string `json:"y1"`                // 裁剪框左上角 y 坐标（取值范围：[0, 1]）
	X2               string `json:"x2"`                // 裁剪框右下角 x 坐标（取值范围：[0, 1]）
	Y2               string `json:"y2"`                // 裁剪框右下角 y 坐标（取值范围：[0, 1]）
}

// SetHeadImage 修改小程序头像
func (basic *Basic) SetHeadImage(imgMediaID string) error {
	return basic.SetHeadImageFull(&SetHeadImageParam{
		HeadImageMediaID: imgMediaID,
		X1:               "0",
		Y1:               "0",
		X2:               "1",
		Y2:               "1",
	})
}

// SetHeadImageFull 修改小程序头像
// 新增临时素材: https://developers.weixin.qq.com/doc/offiaccount/Asset_Management/New_temporary_materials.html
// ref: https://developers.weixin.qq.com/doc/oplatform/openApi/OpenApiDoc/miniprogram-management/basic-info-management/setHeadImage.html
func (basic *Basic) SetHeadImageFull(param *SetHeadImageParam) error {
	ak, err := basic.GetAuthrAccessToken(basic.AppID)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s?access_token=%s", setHeadImageURL, ak)
	data, err := util.PostJSON(url, param)
	if err != nil {
		return err
	}
	return util.DecodeWithError(data, &SetHeadImageResp{}, "account/setheadimage")
}
