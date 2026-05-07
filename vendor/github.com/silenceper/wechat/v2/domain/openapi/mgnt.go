package openapi

import "github.com/silenceper/wechat/v2/util"

// GetAPIQuotaParams 查询 API 调用额度参数
type GetAPIQuotaParams struct {
	CgiPath string `json:"cgi_path"` // api 的请求地址，例如"/cgi-bin/message/custom/send";不要前缀“https://api.weixin.qq.com” ，也不要漏了"/",否则都会 76003 的报错
}

// APIQuota API 调用额度
type APIQuota struct {
	util.CommonError
	Quota struct {
		DailyLimit int64 `json:"daily_limit"` // 当天该账号可调用该接口的次数
		Used       int64 `json:"used"`        // 当天已经调用的次数
		Remain     int64 `json:"remain"`      // 当天剩余调用次数
	} `json:"quota"` // 详情
}

// GetRidInfoParams 查询 rid 信息参数
type GetRidInfoParams struct {
	Rid string `json:"rid"` // 调用接口报错返回的 rid
}

// RidInfo rid 信息
type RidInfo struct {
	util.CommonError
	Request struct {
		InvokeTime   int64  `json:"invoke_time"`   // 发起请求的时间戳
		CostInMs     int64  `json:"cost_in_ms"`    // 请求毫秒级耗时
		RequestURL   string `json:"request_url"`   // 请求的 URL 参数
		RequestBody  string `json:"request_body"`  // post 请求的请求参数
		ResponseBody string `json:"response_body"` // 接口请求返回参数
		ClientIP     string `json:"client_ip"`     // 接口请求的客户端 ip
	} `json:"request"` // 该 rid 对应的请求详情
}
