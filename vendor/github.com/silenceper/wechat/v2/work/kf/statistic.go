package kf

import (
	"fmt"

	"github.com/silenceper/wechat/v2/util"
)

const (
	// getCorpStatisticURL 获取「客户数据统计」企业汇总数据
	getCorpStatisticURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/get_corp_statistic?access_token=%s"
	// getServicerStatisticURL 获取「客户数据统计」接待人员明细数据
	getServicerStatisticURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/get_servicer_statistic?access_token=%s"
)

// GetCorpStatisticRequest 获取「客户数据统计」企业汇总数据请求
type GetCorpStatisticRequest struct {
	OpenKfID  string `json:"open_kfid"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

// GetCorpStatisticResponse 获取「客户数据统计」企业汇总数据响应
type GetCorpStatisticResponse struct {
	util.CommonError
	StatisticList []CorpStatisticList `json:"statistic_list"`
}

// CorpStatisticList 企业汇总统计数据列表
type CorpStatisticList struct {
	StatTime  int64         `json:"stat_time"`
	Statistic CorpStatistic `json:"statistic"`
}

// CorpStatistic 企业汇总统计一天的统计数据
type CorpStatistic struct {
	SessionCnt                int64   `json:"session_cnt"`
	CustomerCnt               int64   `json:"customer_cnt"`
	CustomerMsgCnt            int64   `json:"customer_msg_cnt"`
	UpgradeServiceCustomerCnt int64   `json:"upgrade_service_customer_cnt"`
	AiSessionReplyCnt         int64   `json:"ai_session_reply_cnt"`
	AiTransferRate            float64 `json:"ai_transfer_rate"`
	AiKnowledgeHitRate        float64 `json:"ai_knowledge_hit_rate"`
	MsgRejectedCustomerCnt    int64   `json:"msg_rejected_customer_cnt"`
}

// GetCorpStatistic 获取「客户数据统计」企业汇总数据
// see https://developer.work.weixin.qq.com/document/path/95489
func (r *Client) GetCorpStatistic(req *GetCorpStatisticRequest) (*GetCorpStatisticResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getCorpStatisticURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetCorpStatisticResponse{}
	err = util.DecodeWithError(response, result, "GetCorpStatistic")
	return result, err
}

// GetServicerStatisticRequest 获取「客户数据统计」接待人员明细数据请求
type GetServicerStatisticRequest struct {
	OpenKfID       string `json:"open_kfid"`
	ServicerUserID string `json:"servicer_userid"`
	StartTime      int64  `json:"start_time"`
	EndTime        int64  `json:"end_time"`
}

// GetServicerStatisticResponse 获取「客户数据统计」接待人员明细数据响应
type GetServicerStatisticResponse struct {
	util.CommonError
	StatisticList []ServicerStatisticList `json:"statistic_list"`
}

// ServicerStatisticList 接待人员明细统计数据列表
type ServicerStatisticList struct {
	StatTime  int64             `json:"stat_time"`
	Statistic ServicerStatistic `json:"statistic"`
}

// ServicerStatistic 接待人员明细统计一天的统计数据
type ServicerStatistic struct {
	SessionCnt                         int64   `json:"session_cnt"`
	CustomerCnt                        int64   `json:"customer_cnt"`
	CustomerMsgCnt                     int64   `json:"customer_msg_cnt"`
	ReplyRate                          float64 `json:"reply_rate"`
	FirstReplyAverageSec               float64 `json:"first_reply_average_sec"`
	SatisfactionInvestgateCnt          int64   `json:"satisfaction_investgate_cnt"`
	SatisfactionParticipationRate      float64 `json:"satisfaction_participation_rate"`
	SatisfiedRate                      float64 `json:"satisfied_rate"`
	MiddlingRate                       float64 `json:"middling_rate"`
	DissatisfiedRate                   float64 `json:"dissatisfied_rate"`
	UpgradeServiceCustomerCnt          int64   `json:"upgrade_service_customer_cnt"`
	UpgradeServiceMemberInviteCnt      int64   `json:"upgrade_service_member_invite_cnt"`
	UpgradeServiceMemberCustomerCnt    int64   `json:"upgrade_service_member_customer_cnt"`
	UpgradeServiceGroupChatInviteCnt   int64   `json:"upgrade_service_groupchat_invite_cnt"`
	UpgradeServiceGroupChatCustomerCnt int64   `json:"upgrade_service_groupchat_customer_cnt"`
	MsgRejectedCustomerCnt             int64   `json:"msg_rejected_customer_cnt"`
}

// GetServicerStatistic 获取「客户数据统计」接待人员明细数据
// see https://developer.work.weixin.qq.com/document/path/95490
func (r *Client) GetServicerStatistic(req *GetServicerStatisticRequest) (*GetServicerStatisticResponse, error) {
	var (
		accessToken string
		err         error
	)
	if accessToken, err = r.ctx.GetAccessToken(); err != nil {
		return nil, err
	}
	var response []byte
	if response, err = util.PostJSON(fmt.Sprintf(getServicerStatisticURL, accessToken), req); err != nil {
		return nil, err
	}
	result := &GetServicerStatisticResponse{}
	err = util.DecodeWithError(response, result, "GetServicerStatistic")
	return result, err
}
