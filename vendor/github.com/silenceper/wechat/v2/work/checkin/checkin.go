package checkin

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// setScheduleListURL 为打卡人员排班
	setScheduleListURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/setcheckinschedulist?access_token=%s"
	// punchCorrectionURL 为打卡人员补卡
	punchCorrectionURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/punch_correction?access_token=%s"
	// addUserFaceURL 录入打卡人员人脸信息
	addUserFaceURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/addcheckinuserface?access_token=%s"
	// addOptionURL 创建打卡规则
	addOptionURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/add_checkin_option?access_token=%s"
	// updateOptionURL 修改打卡规则
	updateOptionURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/update_checkin_option?access_token=%s"
	// clearOptionURL 清空打卡规则数组元素
	clearOptionURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/clear_checkin_option_array_field?access_token=%s"
	// delOptionURL 删除打卡规则
	delOptionURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/del_checkin_option?access_token=%s"
	// addRecordURL 添加打卡记录
	addRecordURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/add_checkin_record?access_token=%s"
)

// SetScheduleListRequest 为打卡人员排班请求
type SetScheduleListRequest struct {
	GroupID   int64                 `json:"groupid"`
	Items     []SetScheduleListItem `json:"items"`
	YearMonth int64                 `json:"yearmonth"`
}

// SetScheduleListItem 排班表信息
type SetScheduleListItem struct {
	UserID     string `json:"userid"`
	Day        int64  `json:"day"`
	ScheduleID int64  `json:"schedule_id"`
}

// SetScheduleList 为打卡人员排班
// see https://developer.work.weixin.qq.com/document/path/93385
func (r *Client) SetScheduleList(req *SetScheduleListRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(setScheduleListURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "SetScheduleList")
}

// PunchCorrectionRequest 为打卡人员补卡请求
type PunchCorrectionRequest struct {
	UserID              string `json:"userid"`
	ScheduleDateTime    int64  `json:"schedule_date_time"`
	ScheduleCheckinTime int64  `json:"schedule_checkin_time"`
	CheckinTime         int64  `json:"checkin_time"`
	Remark              string `json:"remark"`
}

// PunchCorrection 为打卡人员补卡
// see https://developer.work.weixin.qq.com/document/path/95803
func (r *Client) PunchCorrection(req *PunchCorrectionRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(punchCorrectionURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "PunchCorrection")
}

// AddUserFaceRequest 录入打卡人员人脸信息请求
type AddUserFaceRequest struct {
	UserID   string `json:"userid"`
	UserFace string `json:"userface"`
}

// AddUserFace 录入打卡人员人脸信息
// see https://developer.work.weixin.qq.com/document/path/93378
func (r *Client) AddUserFace(req *AddUserFaceRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(addUserFaceURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "AddUserFace")
}

// AddOptionRequest 创建打卡规则请求
type AddOptionRequest struct {
	EffectiveNow bool            `json:"effective_now,omitempty"`
	Group        OptionGroupRule `json:"group,omitempty"`
}

// OptionGroupRule 打卡规则字段
type OptionGroupRule struct {
	GroupID                int64                        `json:"groupid,omitempty"`
	GroupType              int64                        `json:"grouptype"`
	GroupName              string                       `json:"groupname"`
	CheckinDate            []OptionGroupRuleCheckinDate `json:"checkindate,omitempty"`
	SpeWorkdays            []OptionGroupSpeWorkdays     `json:"spe_workdays,omitempty"`
	SpeOffDays             []OptionGroupSpeOffDays      `json:"spe_offdays,omitempty"`
	SyncHolidays           bool                         `json:"sync_holidays,omitempty"`
	NeedPhoto              bool                         `json:"need_photo,omitempty"`
	NoteCanUseLocalPic     bool                         `json:"note_can_use_local_pic,omitempty"`
	WifiMacInfos           []OptionGroupWifiMacInfos    `json:"wifimac_infos,omitempty"`
	LocInfos               []OptionGroupLocInfos        `json:"loc_infos,omitempty"`
	AllowCheckinOffWorkday bool                         `json:"allow_checkin_offworkday,omitempty"`
	AllowApplyOffWorkday   bool                         `json:"allow_apply_offworkday,omitempty"`
	Range                  []OptionGroupRange           `json:"range"`
	WhiteUsers             []string                     `json:"white_users,omitempty"`
	Type                   int64                        `json:"type,omitempty"`
	ReporterInfo           OptionGroupReporterInfo      `json:"reporterinfo,omitempty"`
	AllowApplyBkCnt        int64                        `json:"allow_apply_bk_cnt,omitempty"`
	AllowApplyBkDayLimit   int64                        `json:"allow_apply_bk_day_limit,omitempty"`
	BukaLimitNextMonth     int64                        `json:"buka_limit_next_month,omitempty"`
	OptionOutRange         int64                        `json:"option_out_range,omitempty"`
	ScheduleList           []OptionGroupScheduleList    `json:"schedulelist,omitempty"`
	OffWorkIntervalTime    int64                        `json:"offwork_interval_time,omitempty"`
	UseFaceDetect          bool                         `json:"use_face_detect,omitempty"`
	OpenFaceLiveDetect     bool                         `json:"open_face_live_detect,omitempty"`
	OtInfoV2               OptionGroupOtInfoV2          `json:"ot_info_v2,omitempty"`
	SyncOutCheckin         bool                         `json:"sync_out_checkin,omitempty"`
	BukaRemind             OptionGroupBukaRemind        `json:"buka_remind,omitempty"`
	BukaRestriction        int64                        `json:"buka_restriction,omitempty"`
	CheckinMethodType      int64                        `json:"checkin_method_type,omitempty"`
	SpanDayTime            int64                        `json:"span_day_time,omitempty"`
	StandardWorkDuration   int64                        `json:"standard_work_duration,omitempty"`
}

// OptionGroupRuleCheckinDate 固定时间上下班打卡时间
type OptionGroupRuleCheckinDate struct {
	Workdays            []int64                      `json:"workdays"`
	CheckinTime         []OptionGroupRuleCheckinTime `json:"checkintime"`
	FlexTime            int64                        `json:"flex_time"`
	AllowFlex           bool                         `json:"allow_flex"`
	FlexOnDutyTime      int64                        `json:"flex_on_duty_time"`
	FlexOffDutyTime     int64                        `json:"flex_off_duty_time"`
	MaxAllowArriveEarly int64                        `json:"max_allow_arrive_early"`
	MaxAllowArriveLate  int64                        `json:"max_allow_arrive_late"`
	LateRule            OptionGroupLateRule          `json:"late_rule"`
}

// OptionGroupRuleCheckinTime 工作日上下班打卡时间信息
type OptionGroupRuleCheckinTime struct {
	TimeID             int64 `json:"time_id"`
	WorkSec            int64 `json:"work_sec"`
	OffWorkSec         int64 `json:"off_work_sec"`
	RemindWorkSec      int64 `json:"remind_work_sec"`
	RemindOffWorkSec   int64 `json:"remind_off_work_sec"`
	AllowRest          bool  `json:"allow_rest"`
	RestBeginTime      int64 `json:"rest_begin_time"`
	RestEndTime        int64 `json:"rest_end_time"`
	EarliestWorkSec    int64 `json:"earliest_work_sec"`
	LatestWorkSec      int64 `json:"latest_work_sec"`
	EarliestOffWorkSec int64 `json:"earliest_off_work_sec"`
	LatestOffWorkSec   int64 `json:"latest_off_work_sec"`
	NoNeedCheckOn      bool  `json:"no_need_checkon"`
	NoNeedCheckOff     bool  `json:"no_need_checkoff"`
}

// OptionGroupLateRule 晚走晚到时间规则信息
type OptionGroupLateRule struct {
	OffWorkAfterTime      int64                 `json:"offwork_after_time"`
	OnWorkFlexTime        int64                 `json:"onwork_flex_time"`
	AllowOffWorkAfterTime int64                 `json:"allow_offwork_after_time"`
	TimeRules             []OptionGroupTimeRule `json:"timerules"`
}

// OptionGroupTimeRule 晚走晚到时间规则
type OptionGroupTimeRule struct {
	OffWorkAfterTime int64 `json:"offwork_after_time"`
	OnWorkFlexTime   int64 `json:"onwork_flex_time"`
}

// OptionGroupSpeWorkdays 特殊工作日
type OptionGroupSpeWorkdays struct {
	Timestamp   int64                    `json:"timestamp"`
	Notes       string                   `json:"notes"`
	CheckinTime []OptionGroupCheckinTime `json:"checkintime"`
	Type        int64                    `json:"type"`
	BegTime     int64                    `json:"begtime"`
	EndTime     int64                    `json:"endtime"`
}

// OptionGroupCheckinTime 特殊工作日的上下班打卡时间配置
type OptionGroupCheckinTime struct {
	TimeID           int64 `json:"time_id"`
	WorkSec          int64 `json:"work_sec"`
	OffWorkSec       int64 `json:"off_work_sec"`
	RemindWorkSec    int64 `json:"remind_work_sec"`
	RemindOffWorkSec int64 `json:"remind_off_work_sec"`
}

// OptionGroupSpeOffDays 特殊非工作日
type OptionGroupSpeOffDays struct {
	Timestamp int64  `json:"timestamp"`
	Notes     string `json:"notes"`
	Type      int64  `json:"type"`
	BegTime   int64  `json:"begtime"`
	EndTime   int64  `json:"endtime"`
}

// OptionGroupWifiMacInfos WIFI信息
type OptionGroupWifiMacInfos struct {
	WifiName string `json:"wifiname"`
	WifiMac  string `json:"wifimac"`
}

// OptionGroupLocInfos 地点信息
type OptionGroupLocInfos struct {
	Lat       int64  `json:"lat"`
	Lng       int64  `json:"lng"`
	LocTitle  string `json:"loc_title"`
	LocDetail string `json:"loc_detail"`
	Distance  int64  `json:"distance"`
}

// OptionGroupRange 人员信息
type OptionGroupRange struct {
	PartyID []string `json:"party_id"`
	UserID  []string `json:"userid"`
	TagID   []int64  `json:"tagid"`
}

// OptionGroupReporterInfo 汇报人
type OptionGroupReporterInfo struct {
	Reporters []OptionGroupReporters `json:"reporters"`
}

// OptionGroupReporters 汇报对象
type OptionGroupReporters struct {
	UserID string `json:"userid"`
	TagID  int64  `json:"tagid"`
}

// OptionGroupScheduleList 自定义排班规则所有排班
type OptionGroupScheduleList struct {
	ScheduleID          int64                    `json:"schedule_id"`
	ScheduleName        string                   `json:"schedule_name"`
	TimeSection         []OptionGroupTimeSection `json:"time_section"`
	AllowFlex           bool                     `json:"allow_flex"`
	FlexOnDutyTime      int64                    `json:"flex_on_duty_time"`
	FlexOffDutyTime     int64                    `json:"flex_off_duty_time"`
	LateRule            OptionGroupLateRule      `json:"late_rule"`
	MaxAllowArriveEarly int64                    `json:"max_allow_arrive_early"`
	MaxAllowArriveLate  int64                    `json:"max_allow_arrive_late"`
}

// OptionGroupTimeSection 班次上下班时段信息
type OptionGroupTimeSection struct {
	TimeID             int64 `json:"time_id"`
	WorkSec            int64 `json:"work_sec"`
	OffWorkSec         int64 `json:"off_work_sec"`
	RemindWorkSec      int64 `json:"remind_work_sec"`
	RemindOffWorkSec   int64 `json:"remind_off_work_sec"`
	RestBeginTime      int64 `json:"rest_begin_time"`
	RestEndTime        int64 `json:"rest_end_time"`
	AllowRest          bool  `json:"allow_rest"`
	EarliestWorkSec    int64 `json:"earliest_work_sec"`
	LatestWorkSec      int64 `json:"latest_work_sec"`
	EarliestOffWorkSec int64 `json:"earliest_off_work_sec"`
	LatestOffWorkSec   int64 `json:"latest_off_work_sec"`
	NoNeedCheckOn      bool  `json:"no_need_checkon"`
	NoNeedCheckOff     bool  `json:"no_need_checkoff"`
}

// OptionGroupOtInfoV2 加班配置
type OptionGroupOtInfoV2 struct {
	WorkdayConf OptionGroupWorkdayConf `json:"workdayconf"`
}

// OptionGroupWorkdayConf 工作日加班配置
type OptionGroupWorkdayConf struct {
	AllowOt bool  `json:"allow_ot"`
	Type    int64 `json:"type"`
}

// OptionGroupBukaRemind 补卡提醒
type OptionGroupBukaRemind struct {
	OpenRemind      bool  `json:"open_remind"`
	BukaRemindDay   int64 `json:"buka_remind_day"`
	BukaRemindMonth int64 `json:"buka_remind_month"`
}

// AddOption 创建打卡规则
// see https://developer.work.weixin.qq.com/document/path/98041#%E5%88%9B%E5%BB%BA%E6%89%93%E5%8D%A1%E8%A7%84%E5%88%99
func (r *Client) AddOption(req *AddOptionRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(addOptionURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "AddOption")
}

// UpdateOptionRequest 修改打卡规则请求
type UpdateOptionRequest struct {
	EffectiveNow bool            `json:"effective_now,omitempty"`
	Group        OptionGroupRule `json:"group,omitempty"`
}

// UpdateOption 修改打卡规则
// see https://developer.work.weixin.qq.com/document/path/98041#%E4%BF%AE%E6%94%B9%E6%89%93%E5%8D%A1%E8%A7%84%E5%88%99
func (r *Client) UpdateOption(req *UpdateOptionRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(updateOptionURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "UpdateOption")
}

// ClearOptionRequest 清空打卡规则数组元素请求
type ClearOptionRequest struct {
	GroupID      int64   `json:"groupid"`
	ClearField   []int64 `json:"clear_field"`
	EffectiveNow bool    `json:"effective_now"`
}

// ClearOption 清空打卡规则数组元素
// see https://developer.work.weixin.qq.com/document/path/98041#%E6%B8%85%E7%A9%BA%E6%89%93%E5%8D%A1%E8%A7%84%E5%88%99%E6%95%B0%E7%BB%84%E5%85%83%E7%B4%A0
func (r *Client) ClearOption(req *ClearOptionRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(clearOptionURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "ClearOption")
}

// DelOptionRequest 删除打卡规则请求
type DelOptionRequest struct {
	GroupID      int64 `json:"groupid"`
	EffectiveNow bool  `json:"effective_now"`
}

// DelOption 删除打卡规则
// see https://developer.work.weixin.qq.com/document/path/98041#%E5%88%A0%E9%99%A4%E6%89%93%E5%8D%A1%E8%A7%84%E5%88%99
func (r *Client) DelOption(req *DelOptionRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(delOptionURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "DelOption")
}

// AddRecordRequest 添加打卡记录请求
type AddRecordRequest struct {
	Records []Record `json:"records"`
}

// Record 打卡记录
type Record struct {
	UserID         string   `json:"userid"`
	CheckinTime    int64    `json:"checkin_time"`
	LocationTitle  string   `json:"location_title"`
	LocationDetail string   `json:"location_detail"`
	MediaIDS       []string `json:"mediaids"`
	Notes          string   `json:"notes"`
	DeviceType     int      `json:"device_type"`
	Lat            int64    `json:"lat"`
	Lng            int64    `json:"lng"`
	DeviceDetail   string   `json:"device_detail"`
	WifiName       string   `json:"wifiname"`
	WifiMac        string   `json:"wifimac"`
}

// AddRecord 添加打卡记录
// see https://developer.work.weixin.qq.com/document/path/99647
func (r *Client) AddRecord(req *AddRecordRequest) error {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(addRecordURL, accessToken), req); err != nil {
		return err
	}
	return util.DecodeWithCommonError(response, "AddRecord")
}
