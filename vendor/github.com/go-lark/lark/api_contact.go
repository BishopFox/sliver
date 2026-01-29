package lark

import (
	"fmt"
	"net/url"
)

const (
	getUserInfoURL      = "/open-apis/contact/v3/users/%s?user_id_type=%s"
	batchGetUserInfoURL = "/open-apis/contact/v3/users/batch?%s"
)

// GetUserInfoResponse .
type GetUserInfoResponse struct {
	BaseResponse
	Data struct {
		User UserInfo
	}
}

// BatchGetUserInfoResponse .
type BatchGetUserInfoResponse struct {
	BaseResponse
	Data struct {
		Items []UserInfo
	}
}

// UserInfo .
type UserInfo struct {
	OpenID          string     `json:"open_id,omitempty"`
	Email           string     `json:"email,omitempty"`
	UserID          string     `json:"user_id,omitempty"`
	ChatID          string     `json:"chat_id,omitempty"`
	UnionID         string     `json:"union_id,omitempty"`
	Name            string     `json:"name,omitempty"`
	EnglishName     string     `json:"en_name,omitempty"`
	NickName        string     `json:"nickname,omitempty"`
	Mobile          string     `json:"mobile,omitempty"`
	MobileVisible   bool       `json:"mobile_visible,omitempty"`
	Gender          int        `json:"gender,omitempty"`
	Avatar          UserAvatar `json:"avatar,omitempty"`
	Status          UserStatus `json:"status,omitempty"`
	City            string     `json:"city,omitempty"`
	Country         string     `json:"country,omitempty"`
	WorkStation     string     `json:"work_station,omitempty"`
	JoinTime        int        `json:"join_time,omitempty"`
	EmployeeNo      string     `json:"employee_no,omitempty"`
	EmployeeType    int        `json:"employee_type,omitempty"`
	EnterpriseEmail string     `json:"enterprise_email,omitempty"`
	Geo             string     `json:"geo,omitempty"`
	JobTitle        string     `json:"job_title,omitempty"`
	JobLevelID      string     `json:"job_level_id,omitempty"`
	JobFamilyID     string     `json:"job_family_id,omitempty"`
	DepartmentIDs   []string   `json:"department_ids,omitempty"`
	LeaderUserID    string     `json:"leader_user_id,omitempty"`
	IsTenantManager bool       `json:"is_tenant_manager,omitempty"`
}

// UserAvatar .
type UserAvatar struct {
	Avatar72     string `json:"avatar_72,omitempty"`
	Avatar240    string `json:"avatar_240,omitempty"`
	Avatar640    string `json:"avatar_640,omitempty"`
	AvatarOrigin string `json:"avatar_origin,omitempty"`
}

// UserStatus .
type UserStatus struct {
	IsFrozen    bool
	IsResigned  bool
	IsActivated bool
	IsExited    bool
	IsUnjoin    bool
}

// GetUserInfo gets contact info
func (bot Bot) GetUserInfo(userID *OptionalUserID) (*GetUserInfoResponse, error) {
	url := fmt.Sprintf(getUserInfoURL, userID.RealID, userID.UIDType)
	var respData GetUserInfoResponse
	err := bot.GetAPIRequest("GetUserInfo", url, true, nil, &respData)
	return &respData, err
}

// BatchGetUserInfo gets contact info in batch
func (bot Bot) BatchGetUserInfo(userIDType string, userIDs ...string) (*BatchGetUserInfoResponse, error) {
	if len(userIDs) == 0 || len(userIDs) > 50 {
		return nil, ErrParamExceedInputLimit
	}
	v := url.Values{}
	v.Set("user_id_type", userIDType)
	for _, userID := range userIDs {
		v.Add("user_ids", userID)
	}
	url := fmt.Sprintf(batchGetUserInfoURL, v.Encode())
	var respData BatchGetUserInfoResponse
	err := bot.GetAPIRequest("GetUserInfo", url, true, nil, &respData)
	return &respData, err
}
