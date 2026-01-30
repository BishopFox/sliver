package addresslist

import (
	"fmt"
	"strings"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// userSimpleListURL 获取部门成员
	userSimpleListURL = "https://qyapi.weixin.qq.com/cgi-bin/user/simplelist"
	// userCreateURL 创建成员
	userCreateURL = "https://qyapi.weixin.qq.com/cgi-bin/user/create?access_token=%s"
	// userUpdateURL 更新成员
	userUpdateURL = "https://qyapi.weixin.qq.com/cgi-bin/user/update?access_token=%s"
	// userGetURL 读取成员
	userGetURL = "https://qyapi.weixin.qq.com/cgi-bin/user/get"
	// userDeleteURL 删除成员
	userDeleteURL = "https://qyapi.weixin.qq.com/cgi-bin/user/delete"
	// userListIDURL 获取成员ID列表
	userListIDURL = "https://qyapi.weixin.qq.com/cgi-bin/user/list_id"
	// convertToOpenIDURL userID转openID
	convertToOpenIDURL = "https://qyapi.weixin.qq.com/cgi-bin/user/convert_to_openid"
	// convertToUserIDURL openID转userID
	convertToUserIDURL = "https://qyapi.weixin.qq.com/cgi-bin/user/convert_to_userid"
)

type (
	// UserSimpleListResponse 获取部门成员响应
	UserSimpleListResponse struct {
		util.CommonError
		UserList []*UserList
	}
	// UserList 部门成员
	UserList struct {
		UserID     string `json:"userid"`
		Name       string `json:"name"`
		Department []int  `json:"department"`
		OpenUserID string `json:"open_userid"`
	}
)

// UserSimpleList 获取部门成员
// @see https://developer.work.weixin.qq.com/document/path/90200
func (r *Client) UserSimpleList(departmentID int) ([]*UserList, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(strings.Join([]string{
		userSimpleListURL,
		util.Query(map[string]interface{}{
			"access_token":  accessToken,
			"department_id": departmentID,
		}),
	}, "?")); err != nil {
		return nil, err
	}
	result := &UserSimpleListResponse{}
	err = util.DecodeWithError(response, result, "UserSimpleList")
	return result.UserList, err
}

type (
	// UserCreateRequest 创建成员数据请求
	UserCreateRequest struct {
		UserID         string   `json:"userid"`
		Name           string   `json:"name"`
		Alias          string   `json:"alias"`
		Mobile         string   `json:"mobile"`
		Department     []int    `json:"department"`
		Order          []int    `json:"order"`
		Position       string   `json:"position"`
		Gender         int      `json:"gender"`
		Email          string   `json:"email"`
		BizMail        string   `json:"biz_mail"`
		IsLeaderInDept []int    `json:"is_leader_in_dept"`
		DirectLeader   []string `json:"direct_leader"`
		Enable         int      `json:"enable"`
		AvatarMediaid  string   `json:"avatar_mediaid"`
		Telephone      string   `json:"telephone"`
		Address        string   `json:"address"`
		MainDepartment int      `json:"main_department"`
		Extattr        struct {
			Attrs []ExtraAttr `json:"attrs"`
		} `json:"extattr"`
		ToInvite         bool            `json:"to_invite"`
		ExternalPosition string          `json:"external_position"`
		ExternalProfile  ExternalProfile `json:"external_profile"`
	}
	// ExtraAttr 扩展属性
	ExtraAttr struct {
		Type int    `json:"type"`
		Name string `json:"name"`
		Text struct {
			Value string `json:"value"`
		} `json:"text,omitempty"`
		Web struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		} `json:"web,omitempty"`
	}
	// ExternalProfile 成员对外信息
	ExternalProfile struct {
		ExternalCorpName string `json:"external_corp_name"`
		WechatChannels   struct {
			Nickname string `json:"nickname"`
			Status   int    `json:"status"`
		} `json:"wechat_channels"`
		ExternalAttr []ExternalProfileAttr `json:"external_attr"`
	}
	// ExternalProfileAttr 成员对外信息属性
	ExternalProfileAttr struct {
		Type int    `json:"type"`
		Name string `json:"name"`
		Text struct {
			Value string `json:"value"`
		} `json:"text,omitempty"`
		Web struct {
			URL   string `json:"url"`
			Title string `json:"title"`
		} `json:"web,omitempty"`
		Miniprogram struct {
			Appid    string `json:"appid"`
			Pagepath string `json:"pagepath"`
			Title    string `json:"title"`
		} `json:"miniprogram,omitempty"`
	}
	// UserCreateResponse 创建成员数据响应
	UserCreateResponse struct {
		util.CommonError
	}
)

// UserCreate 创建成员
// @see https://developer.work.weixin.qq.com/document/path/90195
func (r *Client) UserCreate(req *UserCreateRequest) (*UserCreateResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(userCreateURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &UserCreateResponse{}
	err = util.DecodeWithError(response, result, "UserCreate")
	return result, err
}

// UserUpdateRequest 更新成员请求
type UserUpdateRequest struct {
	UserID       string `json:"userid"`
	NewUserID    string `json:"new_userid"`
	Name         string `json:"name"`
	Alias        string `json:"alias"`
	Mobile       string `json:"mobile"`
	Department   []int  `json:"department"`
	Order        []int  `json:"order"`
	Position     string `json:"position"`
	Gender       int    `json:"gender"`
	Email        string `json:"email"`
	BizMail      string `json:"biz_mail"`
	BizMailAlias struct {
		Item []string `json:"item"`
	} `json:"biz_mail_alias"`
	IsLeaderInDept []int    `json:"is_leader_in_dept"`
	DirectLeader   []string `json:"direct_leader"`
	Enable         int      `json:"enable"`
	AvatarMediaid  string   `json:"avatar_mediaid"`
	Telephone      string   `json:"telephone"`
	Address        string   `json:"address"`
	MainDepartment int      `json:"main_department"`
	Extattr        struct {
		Attrs []ExtraAttr `json:"attrs"`
	} `json:"extattr"`
	ToInvite         bool            `json:"to_invite"`
	ExternalPosition string          `json:"external_position"`
	ExternalProfile  ExternalProfile `json:"external_profile"`
}

// UserUpdate 更新成员
// see https://developer.work.weixin.qq.com/document/path/90197
func (r *Client) UserUpdate(req *UserUpdateRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(userUpdateURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "UserUpdate")
}

// UserGetResponse 获取部门成员响应
type UserGetResponse struct {
	util.CommonError
	UserID         string   `json:"userid"`            // 成员UserID。对应管理端的帐号，企业内必须唯一。不区分大小写，长度为1~64个字节；第三方应用返回的值为open_userid
	Name           string   `json:"name"`              // 成员名称；第三方不可获取，调用时返回userid以代替name；代开发自建应用需要管理员授权才返回；对于非第三方创建的成员，第三方通讯录应用也不可获取；未返回name的情况需要通过通讯录展示组件来展示名字
	Department     []int    `json:"department"`        // 成员所属部门id列表，仅返回该应用有查看权限的部门id；成员授权模式下，固定返回根部门id，即固定为1。对授权了“组织架构信息”权限的第三方应用，返回成员所属的全部部门id
	Order          []int    `json:"order"`             // 部门内的排序值，默认为0。数量必须和department一致，数值越大排序越前面。值范围是[0, 2^32)。成员授权模式下不返回该字段
	Position       string   `json:"position"`          // 职务信息；代开发自建应用需要管理员授权才返回；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	Mobile         string   `json:"mobile"`            // 手机号码，代开发自建应用需要管理员授权且成员oauth2授权获取；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	Gender         string   `json:"gender"`            // 性别。0表示未定义，1表示男性，2表示女性。代开发自建应用需要管理员授权且成员oauth2授权获取；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段。注：不可获取指返回值0
	Email          string   `json:"email"`             // 邮箱，代开发自建应用需要管理员授权且成员oauth2授权获取；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	BizMail        string   `json:"biz_mail"`          // 企业邮箱，代开发自建应用需要管理员授权且成员oauth2授权获取；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	IsLeaderInDept []int    `json:"is_leader_in_dept"` // 表示在所在的部门内是否为部门负责人，数量与department一致；第三方通讯录应用或者授权了“组织架构信息-应用可获取企业的部门组织架构信息-部门负责人”权限的第三方应用可获取；对于非第三方创建的成员，第三方通讯录应用不可获取；上游企业不可获取下游企业成员该字段
	DirectLeader   []string `json:"direct_leader"`     // 直属上级UserID，返回在应用可见范围内的直属上级列表，最多有五个直属上级；第三方通讯录应用或者授权了“组织架构信息-应用可获取可见范围内成员组织架构信息-直属上级”权限的第三方应用可获取；对于非第三方创建的成员，第三方通讯录应用不可获取；上游企业不可获取下游企业成员该字段；代开发自建应用不可获取该字段
	Avatar         string   `json:"avatar"`            // 头像url。 代开发自建应用需要管理员授权且成员oauth2授权获取；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	ThumbAvatar    string   `json:"thumb_avatar"`      // 头像缩略图url。第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	Telephone      string   `json:"telephone"`         // 座机。代开发自建应用需要管理员授权才返回；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	Alias          string   `json:"alias"`             // 别名；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	Address        string   `json:"address"`           // 地址。代开发自建应用需要管理员授权且成员oauth2授权获取；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	OpenUserid     string   `json:"open_userid"`       // 全局唯一。对于同一个服务商，不同应用获取到企业内同一个成员的open_userid是相同的，最多64个字节。仅第三方应用可获取
	MainDepartment int      `json:"main_department"`   // 主部门，仅当应用对主部门有查看权限时返回。
	Extattr        struct {
		Attrs []struct {
			Type int    `json:"type"`
			Name string `json:"name"`
			Text struct {
				Value string `json:"value"`
			} `json:"text,omitempty"`
			Web struct {
				URL   string `json:"url"`
				Title string `json:"title"`
			} `json:"web,omitempty"`
		} `json:"attrs"`
	} `json:"extattr"` // 扩展属性，代开发自建应用需要管理员授权才返回；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	Status           int    `json:"status"`            // 激活状态: 1=已激活，2=已禁用，4=未激活，5=退出企业。 已激活代表已激活企业微信或已关注微信插件（原企业号）。未激活代表既未激活企业微信又未关注微信插件（原企业号）。
	QrCode           string `json:"qr_code"`           // 员工个人二维码，扫描可添加为外部联系人(注意返回的是一个url，可在浏览器上打开该url以展示二维码)；代开发自建应用需要管理员授权且成员oauth2授权获取；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	ExternalPosition string `json:"external_position"` // 对外职务，如果设置了该值，则以此作为对外展示的职务，否则以position来展示。代开发自建应用需要管理员授权才返回；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
	ExternalProfile  struct {
		ExternalCorpName string `json:"external_corp_name"`
		WechatChannels   struct {
			Nickname string `json:"nickname"`
			Status   int    `json:"status"`
		} `json:"wechat_channels"`
		ExternalAttr []struct {
			Type int    `json:"type"`
			Name string `json:"name"`
			Text struct {
				Value string `json:"value"`
			} `json:"text,omitempty"`
			Web struct {
				URL   string `json:"url"`
				Title string `json:"title"`
			} `json:"web,omitempty"`
			Miniprogram struct {
				Appid    string `json:"appid"`
				Pagepath string `json:"pagepath"`
				Title    string `json:"title"`
			} `json:"miniprogram,omitempty"`
		} `json:"external_attr"`
	} `json:"external_profile"` // 成员对外属性，字段详情见对外属性；代开发自建应用需要管理员授权才返回；第三方仅通讯录应用可获取；对于非第三方创建的成员，第三方通讯录应用也不可获取；上游企业不可获取下游企业成员该字段
}

// UserGet 读取成员
// @see https://developer.work.weixin.qq.com/document/path/90196
func (r *Client) UserGet(UserID string) (*UserGetResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte

	if response, err = util.HTTPGet(
		strings.Join([]string{
			userGetURL,
			util.Query(map[string]interface{}{
				"access_token": accessToken,
				"userid":       UserID,
			}),
		}, "?")); err != nil {
		return nil, err
	}
	result := &UserGetResponse{}
	err = util.DecodeWithError(response, result, "UserGet")
	return result, err
}

type (
	// UserDeleteResponse 删除成员数据响应
	UserDeleteResponse struct {
		util.CommonError
	}
)

// UserDelete 删除成员
// @see https://developer.work.weixin.qq.com/document/path/90334
func (r *Client) UserDelete(userID string) (*UserDeleteResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPGet(strings.Join([]string{
		userDeleteURL,
		util.Query(map[string]interface{}{
			"access_token": accessToken,
			"userid":       userID,
		}),
	}, "?")); err != nil {
		return nil, err
	}
	result := &UserDeleteResponse{}
	err = util.DecodeWithError(response, result, "UserDelete")
	return result, err
}

// UserListIDRequest 获取成员ID列表请求
type UserListIDRequest struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

// UserListIDResponse 获取成员ID列表响应
type UserListIDResponse struct {
	util.CommonError
	NextCursor string      `json:"next_cursor"`
	DeptUser   []*DeptUser `json:"dept_user"`
}

// DeptUser 用户-部门关系
type DeptUser struct {
	UserID     string `json:"userid"`
	Department int    `json:"department"`
}

// UserListID 获取成员ID列表
// see https://developer.work.weixin.qq.com/document/path/96067
func (r *Client) UserListID(req *UserListIDRequest) (*UserListIDResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(strings.Join([]string{
		userListIDURL,
		util.Query(map[string]interface{}{
			"access_token": accessToken,
		}),
	}, "?"), req); err != nil {
		return nil, err
	}
	result := &UserListIDResponse{}
	err = util.DecodeWithError(response, result, "UserListID")
	return result, err
}

type (
	// convertToOpenIDRequest userID转openID请求
	convertToOpenIDRequest struct {
		UserID string `json:"userid"`
	}

	// convertToOpenIDResponse userID转openID响应
	convertToOpenIDResponse struct {
		util.CommonError
		OpenID string `json:"openid"`
	}
)

// ConvertToOpenID userID转openID
// see https://developer.work.weixin.qq.com/document/path/90202
func (r *Client) ConvertToOpenID(userID string) (string, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return "", err
	}
	var response []byte

	if response, err = util.PostJSON(strings.Join([]string{
		convertToOpenIDURL,
		util.Query(map[string]interface{}{
			"access_token": accessToken,
		}),
	}, "?"), &convertToOpenIDRequest{
		UserID: userID,
	}); err != nil {
		return "", err
	}
	result := &convertToOpenIDResponse{}
	err = util.DecodeWithError(response, result, "ConvertToOpenID")
	return result.OpenID, err
}

type (
	// convertToUserIDRequest openID转userID请求
	convertToUserIDRequest struct {
		OpenID string `json:"openid"`
	}

	// convertToUserIDResponse openID转userID响应
	convertToUserIDResponse struct {
		util.CommonError
		UserID string `json:"userid"`
	}
)

// ConvertToUserID openID转userID
// see https://developer.work.weixin.qq.com/document/path/90202
func (r *Client) ConvertToUserID(openID string) (string, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return "", err
	}
	var response []byte

	if response, err = util.PostJSON(strings.Join([]string{
		convertToUserIDURL,
		util.Query(map[string]interface{}{
			"access_token": accessToken,
		}),
	}, "?"), &convertToUserIDRequest{
		OpenID: openID,
	}); err != nil {
		return "", err
	}
	result := &convertToUserIDResponse{}
	err = util.DecodeWithError(response, result, "ConvertToUserID")
	return result.UserID, err
}
