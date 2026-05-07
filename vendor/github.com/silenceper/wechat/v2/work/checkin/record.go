package checkin

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// getCheckinDataURL 获取打卡记录数据
	getCheckinDataURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/getcheckindata?access_token=%s"
	// getDayDataURL 获取打卡日报数据
	getDayDataURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/getcheckin_daydata?access_token=%s"
	// getMonthDataURL 获取打卡月报数据
	getMonthDataURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/getcheckin_monthdata?access_token=%s"
	// getCorpOptionURL 获取企业所有打卡规则
	getCorpOptionURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/getcorpcheckinoption?access_token=%s"
	// getOptionURL 获取员工打卡规则
	getOptionURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/getcheckinoption?access_token=%s"
	// getScheduleListURL 获取打卡人员排班信息
	getScheduleListURL = "https://qyapi.weixin.qq.com/cgi-bin/checkin/getcheckinschedulist?access_token=%s"
	// getHardwareDataURL获取设备打卡数据
	getHardwareDataURL = "https://qyapi.weixin.qq.com/cgi-bin/hardware/get_hardware_checkin_data?access_token=%s"
)

type (
	// GetCheckinDataRequest 获取打卡记录数据请求
	GetCheckinDataRequest struct {
		OpenCheckinDataType int64    `json:"opencheckindatatype"`
		StartTime           int64    `json:"starttime"`
		EndTime             int64    `json:"endtime"`
		UserIDList          []string `json:"useridlist"`
	}
	// GetCheckinDataResponse 获取打卡记录数据响应
	GetCheckinDataResponse struct {
		util.CommonError
		CheckinData []*GetCheckinDataItem `json:"checkindata"`
	}
	// GetCheckinDataItem 打卡记录数据
	GetCheckinDataItem struct {
		UserID         string   `json:"userid"`
		GroupName      string   `json:"groupname"`
		CheckinType    string   `json:"checkin_type"`
		ExceptionType  string   `json:"exception_type"`
		CheckinTime    int64    `json:"checkin_time"`
		LocationTitle  string   `json:"location_title"`
		LocationDetail string   `json:"location_detail"`
		WifiName       string   `json:"wifiname"`
		Notes          string   `json:"notes"`
		WifiMac        string   `json:"wifimac"`
		MediaIDs       []string `json:"mediaids"`
		SchCheckinTime int64    `json:"sch_checkin_time"`
		GroupID        int64    `json:"groupid"`
		ScheduleID     int64    `json:"schedule_id"`
		TimelineID     int64    `json:"timeline_id"`
		Lat            int64    `json:"lat,omitempty"`
		Lng            int64    `json:"lng,omitempty"`
		DeviceID       string   `json:"deviceid,omitempty"`
	}
)

// GetCheckinData 获取打卡记录数据
// @see https://developer.work.weixin.qq.com/document/path/90262
func (r *Client) GetCheckinData(req *GetCheckinDataRequest) (*GetCheckinDataResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getCheckinDataURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetCheckinDataResponse{}
	err = util.DecodeWithError(response, result, "GetCheckinData")
	return result, err
}

type (
	// GetDayDataResponse 获取打卡日报数据
	GetDayDataResponse struct {
		util.CommonError
		Datas []DayDataItem `json:"datas"`
	}

	// DayDataItem 日报
	DayDataItem struct {
		BaseInfo       DayBaseInfo     `json:"base_info"`
		SummaryInfo    DaySummaryInfo  `json:"summary_info"`
		HolidayInfos   []HolidayInfo   `json:"holiday_infos"`
		ExceptionInfos []ExceptionInfo `json:"exception_infos"`
		OtInfo         OtInfo          `json:"ot_info"`
		SpItems        []SpItem        `json:"sp_items"`
	}

	// DayBaseInfo 基础信息
	DayBaseInfo struct {
		Date        int64       `json:"date"`
		RecordType  int64       `json:"record_type"`
		Name        string      `json:"name"`
		NameEx      string      `json:"name_ex"`
		DepartsName string      `json:"departs_name"`
		AcctID      string      `json:"acctid"`
		DayType     int64       `json:"day_type"`
		RuleInfo    DayRuleInfo `json:"rule_info"`
	}

	// DayCheckInTime 当日打卡时间
	DayCheckInTime struct {
		WorkSec    int64 `json:"work_sec"`
		OffWorkSec int64 `json:"off_work_sec"`
	}

	// DayRuleInfo 打卡人员所属规则信息
	DayRuleInfo struct {
		GroupID      int64            `json:"groupid"`
		GroupName    string           `json:"groupname"`
		ScheduleID   int64            `json:"scheduleid"`
		ScheduleName string           `json:"schedulename"`
		CheckInTimes []DayCheckInTime `json:"checkintime"`
	}

	// DaySummaryInfo 汇总信息
	DaySummaryInfo struct {
		CheckinCount    int64 `json:"checkin_count"`
		RegularWorkSec  int64 `json:"regular_work_sec"`
		StandardWorkSec int64 `json:"standard_work_sec"`
		EarliestTime    int64 `json:"earliest_time"`
		LastestTime     int64 `json:"lastest_time"`
	}

	// HolidayInfo 假勤相关信息
	HolidayInfo struct {
		SpNumber      string        `json:"sp_number"`
		SpTitle       SpTitle       `json:"sp_title"`
		SpDescription SpDescription `json:"sp_description"`
	}

	// SpTitle 假勤信息摘要-标题信息
	SpTitle struct {
		Data []SpData `json:"data"`
	}

	// SpDescription 假勤信息摘要-描述信息
	SpDescription struct {
		Data []SpData `json:"data"`
	}

	// SpData 假勤信息(多种语言描述，目前只有中文一种)
	SpData struct {
		Lang string `json:"lang"`
		Text string `json:"text"`
	}

	// SpItem 假勤统计信息
	SpItem struct {
		Count      int64  `json:"count"`
		Duration   int64  `json:"duration"`
		TimeType   int64  `json:"time_type"`
		Type       int64  `json:"type"`
		VacationID int64  `json:"vacation_id"`
		Name       string `json:"name"`
	}

	// ExceptionInfo 校准状态信息
	ExceptionInfo struct {
		Count     int64 `json:"count"`
		Duration  int64 `json:"duration"`
		Exception int64 `json:"exception"`
	}

	// OtInfo 加班信息
	OtInfo struct {
		OtStatus              int64    `json:"ot_status"`
		OtDuration            int64    `json:"ot_duration"`
		ExceptionDuration     []uint64 `json:"exception_duration"`
		WorkdayOverAsVacation int64    `json:"workday_over_as_vacation"`
		WorkdayOverAsMoney    int64    `json:"workday_over_as_money"`
		RestdayOverAsVacation int64    `json:"restday_over_as_vacation"`
		RestdayOverAsMoney    int64    `json:"restday_over_as_money"`
		HolidayOverAsVacation int64    `json:"holiday_over_as_vacation"`
		HolidayOverAsMoney    int64    `json:"holiday_over_as_money"`
	}
)

// GetDayData 获取打卡日报数据
// @see https://developer.work.weixin.qq.com/document/path/96498
func (r *Client) GetDayData(req *GetCheckinDataRequest) (result *GetDayDataResponse, err error) {
	var (
		response    []byte
		accessToken string
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return
	}
	if response, err = util.PostJSON(fmt.Sprintf(getDayDataURL, accessToken), req); err != nil {
		return
	}

	result = new(GetDayDataResponse)
	err = util.DecodeWithError(response, result, "GetDayData")
	return
}

type (
	// GetMonthDataResponse 获取打卡月报数据
	GetMonthDataResponse struct {
		util.CommonError
		Datas []MonthDataItem `json:"datas"`
	}

	// MonthDataItem 月报数据
	MonthDataItem struct {
		BaseInfo       MonthBaseInfo    `json:"base_info"`
		SummaryInfo    MonthSummaryInfo `json:"summary_info"`
		ExceptionInfos []ExceptionInfo  `json:"exception_infos"`
		SpItems        []SpItem         `json:"sp_items"`
		OverWorkInfo   OverWorkInfo     `json:"overwork_info"`
	}

	// MonthBaseInfo 基础信息
	MonthBaseInfo struct {
		RecordType  int64         `json:"record_type"`
		Name        string        `json:"name"`
		NameEx      string        `json:"name_ex"`
		DepartsName string        `json:"departs_name"`
		AcctID      string        `json:"acctid"`
		RuleInfo    MonthRuleInfo `json:"rule_info"`
	}

	// MonthRuleInfo 打卡人员所属规则信息
	MonthRuleInfo struct {
		GroupID   int64  `json:"groupid"`
		GroupName string `json:"groupname"`
	}

	// MonthSummaryInfo 汇总信息
	MonthSummaryInfo struct {
		WorkDays        int64 `json:"work_days"`
		ExceptDays      int64 `json:"except_days"`
		RegularDays     int64 `json:"regular_days"`
		RegularWorkSec  int64 `json:"regular_work_sec"`
		StandardWorkSec int64 `json:"standard_work_sec"`
		RestDays        int64 `json:"rest_days"`
	}

	// OverWorkInfo 加班情况
	OverWorkInfo struct {
		WorkdayOverSec         int64 `json:"workday_over_sec"`
		HolidayOverSec         int64 `json:"holidays_over_sec"`
		RestDayOverSec         int64 `json:"restdays_over_sec"`
		WorkdaysOverAsVacation int64 `json:"workdays_over_as_vacation"`
		WorkdaysOverAsMoney    int64 `json:"workdays_over_as_money"`
		RestdaysOverAsVacation int64 `json:"restdays_over_as_vacation"`
		RestdaysOverAsMoney    int64 `json:"restdays_over_as_money"`
		HolidaysOverAsVacation int64 `json:"holidays_over_as_vacation"`
		HolidaysOverAsMoney    int64 `json:"holidays_over_as_money"`
	}
)

// GetMonthData 获取打卡月报数据
// @see https://developer.work.weixin.qq.com/document/path/96499
func (r *Client) GetMonthData(req *GetCheckinDataRequest) (result *GetMonthDataResponse, err error) {
	var (
		response    []byte
		accessToken string
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return
	}
	if response, err = util.PostJSON(fmt.Sprintf(getMonthDataURL, accessToken), req); err != nil {
		return
	}

	result = new(GetMonthDataResponse)
	err = util.DecodeWithError(response, result, "GetMonthData")
	return
}

// GetCorpOptionResponse 获取企业所有打卡规则响应
type GetCorpOptionResponse struct {
	util.CommonError
	Group []CorpOptionGroup `json:"group"`
}

// CorpOptionGroup 企业规则信息列表
type CorpOptionGroup struct {
	GroupType              int64              `json:"grouptype"`
	GroupID                int64              `json:"groupid"`
	GroupName              string             `json:"groupname"`
	CheckinDate            []GroupCheckinDate `json:"checkindate"`
	SpeWorkdays            []SpeWorkdays      `json:"spe_workdays"`
	SpeOffDays             []SpeOffDays       `json:"spe_offdays"`
	SyncHolidays           bool               `json:"sync_holidays"`
	NeedPhoto              bool               `json:"need_photo"`
	NoteCanUseLocalPic     bool               `json:"note_can_use_local_pic"`
	AllowCheckinOffWorkday bool               `json:"allow_checkin_offworkday"`
	AllowApplyOffWorkday   bool               `json:"allow_apply_offworkday"`
	WifiMacInfos           []WifiMacInfos     `json:"wifimac_infos"`
	LocInfos               []LocInfos         `json:"loc_infos"`
	Range                  []Range            `json:"range"`
	CreateTime             int64              `json:"create_time"`
	WhiteUsers             []string           `json:"white_users"`
	Type                   int64              `json:"type"`
	ReporterInfo           ReporterInfo       `json:"reporterinfo"`
	OtInfo                 GroupOtInfo        `json:"ot_info"`
	OtApplyInfo            OtApplyInfo        `json:"otapplyinfo"`
	Uptime                 int64              `json:"uptime"`
	AllowApplyBkCnt        int64              `json:"allow_apply_bk_cnt"`
	OptionOutRange         int64              `json:"option_out_range"`
	CreateUserID           string             `json:"create_userid"`
	UseFaceDetect          bool               `json:"use_face_detect"`
	AllowApplyBkDayLimit   int64              `json:"allow_apply_bk_day_limit"`
	UpdateUserID           string             `json:"update_userid"`
	BukaRestriction        int64              `json:"buka_restriction"`
	ScheduleList           []ScheduleList     `json:"schedulelist"`
	OffWorkIntervalTime    int64              `json:"offwork_interval_time"`
	SpanDayTime            int64              `json:"span_day_time"`
	StandardWorkDuration   int64              `json:"standard_work_duration"`
	OpenSpCheckin          bool               `json:"open_sp_checkin"`
	CheckinMethodType      int64              `json:"checkin_method_type"`
}

// GroupCheckinDate 打卡时间，当规则类型为排班时没有意义
type GroupCheckinDate struct {
	Workdays        []int64            `json:"workdays"`
	CheckinTime     []GroupCheckinTime `json:"checkintime"`
	NoNeedOffWork   bool               `json:"noneed_offwork"`
	LimitAheadTime  int64              `json:"limit_aheadtime"`
	FlexOnDutyTime  int64              `json:"flex_on_duty_time"`
	FlexOffDutyTime int64              `json:"flex_off_duty_time"`
}

// GroupCheckinTime 工作日上下班打卡时间信息
type GroupCheckinTime struct {
	WorkSec          int64 `json:"work_sec"`
	OffWorkSec       int64 `json:"off_work_sec"`
	RemindWorkSec    int64 `json:"remind_work_sec"`
	RemindOffWorkSec int64 `json:"remind_off_work_sec"`
}

// SpeWorkdays 特殊日期-必须打卡日期信息
type SpeWorkdays struct {
	Timestamp   int64              `json:"timestamp"`
	Notes       string             `json:"notes"`
	CheckinTime []GroupCheckinTime `json:"checkintime"`
}

// SpeOffDays 特殊日期-不用打卡日期信息
type SpeOffDays struct {
	Timestamp int64  `json:"timestamp"`
	Notes     string `json:"notes"`
}

// WifiMacInfos 打卡地点-WiFi打卡信息
type WifiMacInfos struct {
	WifiName string `json:"wifiname"`
	WifiMac  string `json:"wifimac"`
}

// LocInfos 打卡地点-位置打卡信息
type LocInfos struct {
	Lat       int64  `json:"lat"`
	Lng       int64  `json:"lng"`
	LocTitle  string `json:"loc_title"`
	LocDetail string `json:"loc_detail"`
	Distance  int64  `json:"distance"`
}

// Range 打卡人员信息
type Range struct {
	PartyID []string `json:"partyid"`
	UserID  []string `json:"userid"`
	TagID   []int64  `json:"tagid"`
}

// ReporterInfo 汇报对象信息
type ReporterInfo struct {
	Reporters  []Reporters `json:"reporters"`
	UpdateTime int64       `json:"updatetime"`
}

// Reporters 汇报对象，每个汇报人用userid表示
type Reporters struct {
	UserID string `json:"userid"`
}

// GroupOtInfo 加班信息
type GroupOtInfo struct {
	Type                 int64       `json:"type"`
	AllowOtWorkingDay    bool        `json:"allow_ot_workingday"`
	AllowOtNonWorkingDay bool        `json:"allow_ot_nonworkingday"`
	OtCheckInfo          OtCheckInfo `json:"otcheckinfo"`
}

// OtCheckInfo 以打卡时间为准-加班时长计算规则信息
type OtCheckInfo struct {
	OtWorkingDayTimeStart      int64      `json:"ot_workingday_time_start"`
	OtWorkingDayTimeMin        int64      `json:"ot_workingday_time_min"`
	OtWorkingDayTimeMax        int64      `json:"ot_workingday_time_max"`
	OtNonworkingDayTimeMin     int64      `json:"ot_nonworkingday_time_min"`
	OtNonworkingDayTimeMax     int64      `json:"ot_nonworkingday_time_max"`
	OtNonworkingDaySpanDayTime int64      `json:"ot_nonworkingday_spanday_time"`
	OtWorkingDayRestInfo       OtRestInfo `json:"ot_workingday_restinfo"`
	OtNonWorkingDayRestInfo    OtRestInfo `json:"ot_nonworkingday_restinfo"`
}

// OtRestInfo 加班-休息扣除配置信息
type OtRestInfo struct {
	Type          int64         `json:"type"`
	FixTimeRule   FixTimeRule   `json:"fix_time_rule"`
	CalOtTimeRule CalOtTimeRule `json:"cal_ottime_rule"`
}

// FixTimeRule 工作日加班-指定休息时间配置信息
type FixTimeRule struct {
	FixTimeBeginSec int64 `json:"fix_time_begin_sec"`
	FixTimeEndSec   int64 `json:"fix_time_end_sec"`
}

// CalOtTimeRule 工作日加班-按加班时长扣除配置信息
type CalOtTimeRule struct {
	Items []CalOtTimeRuleItem `json:"items"`
}

// CalOtTimeRuleItem 工作日加班-按加班时长扣除条件信息
type CalOtTimeRuleItem struct {
	OtTime   int64 `json:"ot_time"`
	RestTime int64 `json:"rest_time"`
}

// OtApplyInfo 以加班申请核算打卡记录相关信息
type OtApplyInfo struct {
	AllowOtWorkingDay          bool       `json:"allow_ot_workingday"`
	AllowOtNonWorkingDay       bool       `json:"allow_ot_nonworkingday"`
	Uiptime                    int64      `json:"uptime"`
	OtNonworkingDaySpanDayTime int64      `json:"ot_nonworkingday_spanday_time"`
	OtWorkingDayRestInfo       OtRestInfo `json:"ot_workingday_restinfo"`
	OtNonWorkingDayRestInfo    OtRestInfo `json:"ot_nonworkingday_restinfo"`
}

// ScheduleList 排班信息列表
type ScheduleList struct {
	ScheduleID          int64         `json:"schedule_id"`
	ScheduleName        string        `json:"schedule_name"`
	TimeSection         []TimeSection `json:"time_section"`
	LimitAheadTime      int64         `json:"limit_aheadtime"`
	NoNeedOffWork       bool          `json:"noneed_offwork"`
	LimitOffTime        int64         `json:"limit_offtime"`
	FlexOnDutyTime      int64         `json:"flex_on_duty_time"`
	FlexOffDutyTime     int64         `json:"flex_off_duty_time"`
	AllowFlex           bool          `json:"allow_flex"`
	LateRule            LateRule      `json:"late_rule"`
	MaxAllowArriveEarly int64         `json:"max_allow_arrive_early"`
	MaxAllowArriveLate  int64         `json:"max_allow_arrive_late"`
}

// TimeSection 班次上下班时段信息
type TimeSection struct {
	TimeID           int64 `json:"time_id"`
	WorkSec          int64 `json:"work_sec"`
	OffWorkSec       int64 `json:"off_work_sec"`
	RemindWorkSec    int64 `json:"remind_work_sec"`
	RemindOffWorkSec int64 `json:"remind_off_work_sec"`
	RestBeginTime    int64 `json:"rest_begin_time"`
	RestEndTime      int64 `json:"rest_end_time"`
	AllowRest        bool  `json:"allow_rest"`
}

// LateRule 晚走晚到时间规则信息
type LateRule struct {
	AllowOffWorkAfterTime bool       `json:"allow_offwork_after_time"`
	TimeRules             []TimeRule `json:"timerules"`
}

// TimeRule 迟到规则时间
type TimeRule struct {
	OffWorkAfterTime int64 `json:"offwork_after_time"`
	OnWorkFlexTime   int64 `json:"onwork_flex_time"`
}

// GetCorpOption 获取企业所有打卡规则
// @see https://developer.work.weixin.qq.com/document/path/93384
func (r *Client) GetCorpOption() (*GetCorpOptionResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.HTTPPost(fmt.Sprintf(getCorpOptionURL, accessToken), ""); err != nil {
		return nil, err
	}
	result := &GetCorpOptionResponse{}
	err = util.DecodeWithError(response, result, "GetCorpOption")
	return result, err
}

// GetOptionRequest 获取员工打卡规则请求
type GetOptionRequest struct {
	Datetime   int64    `json:"datetime"`
	UserIDList []string `json:"useridlist"`
}

// GetOptionResponse 获取员工打卡规则响应
type GetOptionResponse struct {
	util.CommonError
	Info []OptionInfo `json:"info"`
}

// OptionInfo 打卡规则列表
type OptionInfo struct {
	UserID string      `json:"userid"`
	Group  OptionGroup `json:"group"`
}

// OptionGroup 打卡规则相关信息
type OptionGroup struct {
	GroupType              int64               `json:"grouptype"`
	GroupID                int64               `json:"groupid"`
	OpenSpCheckin          bool                `json:"open_sp_checkin"`
	GroupName              string              `json:"groupname"`
	CheckinDate            []OptionCheckinDate `json:"checkindate"`
	SpeWorkdays            []SpeWorkdays       `json:"spe_workdays"`
	SpeOffDays             []SpeOffDays        `json:"spe_offdays"`
	SyncHolidays           bool                `json:"sync_holidays"`
	NeedPhoto              bool                `json:"need_photo"`
	WifiMacInfos           []WifiMacInfos      `json:"wifimac_infos"`
	NoteCanUseLocalPic     bool                `json:"note_can_use_local_pic"`
	AllowCheckinOffWorkday bool                `json:"allow_checkin_offworkday"`
	AllowApplyOffWorkday   bool                `json:"allow_apply_offworkday"`
	LocInfos               []LocInfos          `json:"loc_infos"`
	ScheduleList           []ScheduleList      `json:"schedulelist"`
	BukaRestriction        int64               `json:"buka_restriction"`
	SpanDayTime            int64               `json:"span_day_time"`
	StandardWorkDuration   int64               `json:"standard_work_duration"`
	OffWorkIntervalTime    int64               `json:"offwork_interval_time"`
	CheckinMethodType      int64               `json:"checkin_method_type"`
}

// OptionCheckinDate 打卡时间配置
type OptionCheckinDate struct {
	Workdays        []int64            `json:"workdays"`
	CheckinTime     []GroupCheckinTime `json:"checkintime"`
	FlexTime        int64              `json:"flex_time"`
	NoNeedOffWork   bool               `json:"noneed_offwork"`
	LimitAheadTime  int64              `json:"limit_aheadtime"`
	FlexOnDutyTime  int64              `json:"flex_on_duty_time"`
	FlexOffDutyTime int64              `json:"flex_off_duty_time"`
}

// GetOption 获取员工打卡规则
// see https://developer.work.weixin.qq.com/document/path/90263
func (r *Client) GetOption(req *GetOptionRequest) (*GetOptionResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getOptionURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetOptionResponse{}
	err = util.DecodeWithError(response, result, "GetOption")
	return result, err
}

// GetScheduleListRequest 获取打卡人员排班信息请求
type GetScheduleListRequest struct {
	StartTime  int64    `json:"starttime"`
	EndTime    int64    `json:"endtime"`
	UserIDList []string `json:"useridlist"`
}

// GetScheduleListResponse 获取打卡人员排班信息响应
type GetScheduleListResponse struct {
	util.CommonError
	ScheduleList []ScheduleItem `json:"schedule_list"`
}

// ScheduleItem 排班表信息
type ScheduleItem struct {
	UserID    string   `json:"userid"`
	YearMonth int64    `json:"yearmonth"`
	GroupID   int64    `json:"groupid"`
	GroupName string   `json:"groupname"`
	Schedule  Schedule `json:"schedule"`
}

// Schedule 个人排班信息
type Schedule struct {
	ScheduleList []ScheduleListItem `json:"scheduleList"`
}

// ScheduleListItem 个人排班表信息
type ScheduleListItem struct {
	Day          int64        `json:"day"`
	ScheduleInfo ScheduleInfo `json:"schedule_info"`
}

// ScheduleInfo 个人当日排班信息
type ScheduleInfo struct {
	ScheduleID   int64                 `json:"schedule_id"`
	ScheduleName string                `json:"schedule_name"`
	TimeSection  []ScheduleTimeSection `json:"time_section"`
}

// ScheduleTimeSection 班次上下班时段信息
type ScheduleTimeSection struct {
	ID               int64 `json:"id"`
	WorkSec          int64 `json:"work_sec"`
	OffWorkSec       int64 `json:"off_work_sec"`
	RemindWorkSec    int64 `json:"remind_work_sec"`
	RemindOffWorkSec int64 `json:"remind_off_work_sec"`
}

// GetScheduleList 获取打卡人员排班信息
// see https://developer.work.weixin.qq.com/document/path/93380
func (r *Client) GetScheduleList(req *GetScheduleListRequest) (*GetScheduleListResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getScheduleListURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetScheduleListResponse{}
	err = util.DecodeWithError(response, result, "GetScheduleList")
	return result, err
}

// GetHardwareDataRequest 获取设备打卡数据请求
type GetHardwareDataRequest struct {
	FilterType int64    `json:"filter_type"`
	StartTime  int64    `json:"starttime"`
	EndTime    int64    `json:"endtime"`
	UserIDList []string `json:"useridlist"`
}

// GetHardwareDataResponse 获取设备打卡数据响应
type GetHardwareDataResponse struct {
	util.CommonError
	CheckinData []HardwareCheckinData `json:"checkindata"`
}

// HardwareCheckinData 设备打卡数据
type HardwareCheckinData struct {
	UserID      string `json:"userid"`
	CheckinTime int64  `json:"checkin_time"`
	DeviceSn    string `json:"device_sn"`
	DeviceName  string `json:"device_name"`
}

// GetHardwareData 获取设备打卡数据
// see https://developer.work.weixin.qq.com/document/path/94126
func (r *Client) GetHardwareData(req *GetHardwareDataRequest) (*GetHardwareDataResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getHardwareDataURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetHardwareDataResponse{}
	err = util.DecodeWithError(response, result, "GetHardwareData")
	return result, err
}
