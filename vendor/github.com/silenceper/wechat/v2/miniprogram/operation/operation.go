package operation

import (
	"fmt"

	"github.com/silenceper/wechat/v2/miniprogram/context"
	"github.com/silenceper/wechat/v2/util"
)

const (
	// getDomainInfoURL 查询域名配置
	getDomainInfoURL = "https://api.weixin.qq.com/wxa/getwxadevinfo?access_token=%s"
	// getPerformanceURL 获取性能数据
	getPerformanceURL = "https://api.weixin.qq.com/wxaapi/log/get_performance?access_token=%s"
	// getSceneListURL 获取访问来源
	getSceneListURL = "https://api.weixin.qq.com/wxaapi/log/get_scene?access_token=%s"
	// getVersionListURL 获取客户端版本
	getVersionListURL = "https://api.weixin.qq.com/wxaapi/log/get_client_version?access_token=%s"
	// realTimeLogSearchURL 查询实时日志
	realTimeLogSearchURL = "https://api.weixin.qq.com/wxaapi/userlog/userlog_search?%s"
	// getFeedbackListURL 获取用户反馈列表
	getFeedbackListURL = "https://api.weixin.qq.com/wxaapi/feedback/list?%s"
	// getJsErrDetailURL 查询js错误详情
	getJsErrDetailURL = "https://api.weixin.qq.com/wxaapi/log/jserr_detail?access_token=%s"
	// getJsErrListURL 查询错误列表
	getJsErrListURL = "https://api.weixin.qq.com/wxaapi/log/jserr_list?access_token=%s"
	// getGrayReleasePlanURL 获取分阶段发布详情
	getGrayReleasePlanURL = "https://api.weixin.qq.com/wxa/getgrayreleaseplan?access_token=%s"
)

// Operation 运维中心
type Operation struct {
	*context.Context
}

// NewOperation 实例化
func NewOperation(ctx *context.Context) *Operation {
	return &Operation{ctx}
}

// GetDomainInfoRequest 查询域名配置请求
type GetDomainInfoRequest struct {
	Action string `json:"action"`
}

// GetDomainInfoResponse 查询域名配置响应
type GetDomainInfoResponse struct {
	util.CommonError
	RequestDomain   []string `json:"requestdomain"`
	WsRequestDomain []string `json:"wsrequestdomain"`
	UploadDomain    []string `json:"uploaddomain"`
	DownloadDomain  []string `json:"downloaddomain"`
	UDPDomain       []string `json:"udpdomain"`
	BizDomain       []string `json:"bizdomain"`
}

// GetDomainInfo 查询域名配置
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/getDomainInfo.html
func (o *Operation) GetDomainInfo(req *GetDomainInfoRequest) (res GetDomainInfoResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getDomainInfoURL, accessToken), req); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "GetDomainInfo")
	return
}

// GetPerformanceRequest 获取性能数据请求
type GetPerformanceRequest struct {
	CostTimeType     int64  `json:"cost_time_type"`
	DefaultStartTime int64  `json:"default_start_time"`
	DefaultEndTime   int64  `json:"default_end_time"`
	Device           string `json:"device"`
	IsDownloadCode   string `json:"is_download_code"`
	Scene            string `json:"scene"`
	NetworkType      string `json:"networktype"`
}

// GetPerformanceResponse 获取性能数据响应
type GetPerformanceResponse struct {
	util.CommonError
	DefaultTimeData string `json:"default_time_data"`
	CompareTimeData string `json:"compare_time_data"`
}

// PerformanceDefaultTimeData 查询数据
type PerformanceDefaultTimeData struct {
	List []DefaultTimeDataItem `json:"list"`
}

// DefaultTimeDataItem 查询数据
type DefaultTimeDataItem struct {
	RefData      string `json:"ref_data"`
	CostTimeType int64  `json:"cost_time_type"`
	CostTime     int64  `json:"cost_time"`
}

// GetPerformance 获取性能数据
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/getPerformance.html
func (o *Operation) GetPerformance(req *GetPerformanceRequest) (res GetPerformanceResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getPerformanceURL, accessToken), req); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "GetPerformance")
	return
}

// GetSceneListResponse 获取访问来源响应
type GetSceneListResponse struct {
	util.CommonError
	Scene []Scene `json:"scene"`
}

// Scene 访问来源
type Scene struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GetSceneList 获取访问来源
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/getSceneList.html
func (o *Operation) GetSceneList() (res GetSceneListResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getSceneListURL, accessToken)); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "GetSceneList")
	return
}

// GetVersionListResponse 获取客户端版本响应
type GetVersionListResponse struct {
	util.CommonError
	CvList []ClientVersion `json:"cvlist"`
}

// ClientVersion 客户端版本
type ClientVersion struct {
	Type              int64    `json:"type"`
	ClientVersionList []string `json:"client_version_list"`
}

// GetVersionList 获取客户端版本
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/getVersionList.html
func (o *Operation) GetVersionList() (res GetVersionListResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getVersionListURL, accessToken)); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "GetVersionList")
	return
}

// RealTimeLogSearchRequest 查询实时日志请求
type RealTimeLogSearchRequest struct {
	Date      string
	BeginTime int64
	EndTime   int64
	Start     int64
	Limit     int64
	Level     int64
	TraceID   string
	URL       string
	ID        string
	FilterMsg string
}

// RealTimeLogSearchResponse 查询实时日志响应
type RealTimeLogSearchResponse struct {
	util.CommonError
	Data RealTimeLogSearchData `json:"data"`
}

// RealTimeLogSearchData 日志数据和日志条数总量
type RealTimeLogSearchData struct {
	List  []RealTimeLogSearchDataList `json:"list"`
	Total int64                       `json:"total"`
}

// RealTimeLogSearchDataList 日志数据列表
type RealTimeLogSearchDataList struct {
	Level          int64                          `json:"level"`
	LibraryVersion string                         `json:"libraryVersion"`
	ClientVersion  string                         `json:"clientVersion"`
	ID             string                         `json:"id"`
	Timestamp      int64                          `json:"timestamp"`
	Platform       int64                          `json:"platform"`
	URL            string                         `json:"url"`
	TraceID        string                         `json:"traceid"`
	FilterMsg      string                         `json:"filterMsg"`
	Msg            []RealTimeLogSearchDataListMsg `json:"msg"`
}

// RealTimeLogSearchDataListMsg 日志内容数组
type RealTimeLogSearchDataListMsg struct {
	Time  int64    `json:"time"`
	Level int64    `json:"level"`
	Msg   []string `json:"msg"`
}

// RealTimeLogSearch 查询实时日志
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/realtimelogSearch.html
func (o *Operation) RealTimeLogSearch(req *RealTimeLogSearchRequest) (res RealTimeLogSearchResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}

	params := map[string]interface{}{
		"access_token": accessToken,
		"date":         req.Date,
		"begintime":    req.BeginTime,
		"endtime":      req.EndTime,
	}
	if req.Start > 0 {
		params["start"] = req.Start
	}
	if req.Limit > 0 {
		params["limit"] = req.Limit
	}
	if req.TraceID != "" {
		params["traceId"] = req.TraceID
	}
	if req.URL != "" {
		params["url"] = req.URL
	}
	if req.ID != "" {
		params["id"] = req.ID
	}
	if req.FilterMsg != "" {
		params["filterMsg"] = req.FilterMsg
	}
	if req.Level > 0 {
		params["level"] = req.Level
	}
	query := util.Query(params)

	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(realTimeLogSearchURL, query)); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "RealTimeLogSearch")
	return
}

// GetFeedbackListRequest 获取用户反馈列表请求
type GetFeedbackListRequest struct {
	Page int64
	Num  int64
	Type int64
}

// GetFeedbackListResponse 获取用户反馈列表响应
type GetFeedbackListResponse struct {
	util.CommonError
	TotalNum int64      `json:"total_num"`
	List     []Feedback `json:"list"`
}

// Feedback 反馈列表
type Feedback struct {
	RecordID   int64    `json:"record_id"`
	CreateTime int64    `json:"create_time"`
	Content    string   `json:"content"`
	Phone      string   `json:"phone"`
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	HeadURL    string   `json:"head_url"`
	Type       int64    `json:"type"`
	MediaIDS   []string `json:"mediaIds"`
	SystemInfo string   `json:"systemInfo"`
}

// GetFeedbackList 获取用户反馈列表
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/getFeedback.html
func (o *Operation) GetFeedbackList(req *GetFeedbackListRequest) (res GetFeedbackListResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}

	params := map[string]interface{}{
		"access_token": accessToken,
		"page":         req.Page,
		"num":          req.Num,
	}
	if req.Type > 0 {
		params["type"] = req.Type
	}
	query := util.Query(params)

	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getFeedbackListURL, query)); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "GetFeedbackList")
	return
}

// GetJsErrDetailRequest 查询js错误详情请求
type GetJsErrDetailRequest struct {
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	ErrorMsgMd5   string `json:"errorMsgMd5"`
	ErrorStackMd5 string `json:"errorStackMd5"`
	AppVersion    string `json:"appVersion"`
	SdkVersion    string `json:"sdkVersion"`
	OsName        string `json:"osName"`
	ClientVersion string `json:"clientVersion"`
	OpenID        string `json:"openid"`
	Offset        int64  `json:"offset"`
	Limit         int64  `json:"limit"`
	Desc          string `json:"desc"`
}

// GetJsErrDetailResponse 查询js错误详情响应
type GetJsErrDetailResponse struct {
	util.CommonError
	TotalCount int64             `json:"totalCount"`
	OpenID     string            `json:"openid"`
	Data       []JsErrDetailData `json:"data"`
}

// JsErrDetailData 错误列表
type JsErrDetailData struct {
	Count         string `json:"Count"`
	SdkVersion    string `json:"sdkVersion"`
	ClientVersion string `json:"ClientVersion"`
	ErrorStackMd5 string `json:"errorStackMd5"`
	TimeStamp     string `json:"TimeStamp"`
	AppVersion    string `json:"appVersion"`
	ErrorMsgMd5   string `json:"errorMsgMd5"`
	ErrorMsg      string `json:"errorMsg"`
	ErrorStack    string `json:"errorStack"`
	Ds            string `json:"Ds"`
	OsName        string `json:"OsName"`
	OpenID        string `json:"openId"`
	PluginVersion string `json:"pluginversion"`
	AppID         string `json:"appId"`
	DeviceModel   string `json:"DeviceModel"`
	Source        string `json:"source"`
	Route         string `json:"route"`
	Uin           string `json:"Uin"`
	Nickname      string `json:"nickname"`
}

// GetJsErrDetail 查询js错误详情
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/getJsErrDetail.html
func (o *Operation) GetJsErrDetail(req *GetJsErrDetailRequest) (res GetJsErrDetailResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getJsErrDetailURL, accessToken), req); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "GetJsErrDetail")
	return
}

// GetJsErrListRequest 查询错误列表请求
type GetJsErrListRequest struct {
	AppVersion string `json:"appVersion"`
	ErrType    string `json:"errType"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	Keyword    string `json:"keyword"`
	OpenID     string `json:"openid"`
	OrderBy    string `json:"orderby"`
	Desc       string `json:"desc"`
	Offset     int64  `json:"offset"`
	Limit      int64  `json:"limit"`
}

// GetJsErrListResponse 查询错误列表响应
type GetJsErrListResponse struct {
	util.CommonError
	TotalCount int64           `json:"totalCount"`
	OpenID     string          `json:"openid"`
	Data       []JsErrListData `json:"data"`
}

// JsErrListData 错误列表
type JsErrListData struct {
	ErrorMsgMd5   string `json:"errorMsgMd5"`
	ErrorMsg      string `json:"errorMsg"`
	Uv            int64  `json:"uv"`
	Pv            int64  `json:"pv"`
	ErrorStackMd5 string `json:"errorStackMd5"`
	ErrorStack    string `json:"errorStack"`
	PvPercent     string `json:"pvPercent"`
	UvPercent     string `json:"uvPercent"`
}

// GetJsErrList 查询错误列表
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/getJsErrList.html
func (o *Operation) GetJsErrList(req *GetJsErrListRequest) (res GetJsErrListResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getJsErrListURL, accessToken), req); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "GetJsErrList")
	return
}

// GetGrayReleasePlanResponse 获取分阶段发布详情响应
type GetGrayReleasePlanResponse struct {
	util.CommonError
	GrayReleasePlan GrayReleasePlanDetail `json:"gray_release_plan"`
}

// GrayReleasePlanDetail 分阶段发布计划详情
type GrayReleasePlanDetail struct {
	Status                  int64 `json:"status"`
	CreateTimestamp         int64 `json:"create_timestamp"`
	GrayPercentage          int64 `json:"gray_percentage"`
	SupportExperiencerFirst bool  `json:"support_experiencer_first"`
	SupportDebugerFirst     bool  `json:"support_debuger_first"`
}

// GetGrayReleasePlan 获取分阶段发布详情
// see https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/operation/getGrayReleasePlan.html
func (o *Operation) GetGrayReleasePlan() (res GetGrayReleasePlanResponse, err error) {
	var accessToken string
	if accessToken, err = o.GetAccessToken(); err != nil {
		return
	}
	var response []byte
	if response, err = util.HTTPGet(fmt.Sprintf(getGrayReleasePlanURL, accessToken)); err != nil {
		return
	}
	err = util.DecodeWithError(response, &res, "GetGrayReleasePlan")
	return
}
