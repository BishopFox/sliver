package dialog

import (
	"encoding/json"
	"fmt"
)

type apiResponse struct {
	RequestID string          `json:"request_id"`
	Code      int64           `json:"code"`
	Msg       string          `json:"msg"`
	Data      json.RawMessage `json:"data"`
}

// APIError 智能对话接口错误.
type APIError struct {
	APIName   string
	Code      int64
	Msg       string
	RequestID string
	Data      json.RawMessage
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s Error , code=%d , msg=%s , request_id=%s", e.APIName, e.Code, e.Msg, e.RequestID)
}

type accessTokenData struct {
	AccessToken string `json:"access_token"`
}

// ImportJSONResponse 简单问答导入响应.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/import.html
type ImportJSONResponse struct {
	RequestID string `json:"-"`
	TaskID    string `json:"task_id"`
}

// PublishResponse 发布响应.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/publish.html
type PublishResponse struct {
	RequestID string `json:"-"`
	TaskID    string `json:"task_id"`
}

// EffectiveProgressResponse 发布进度查询响应.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/progress.html
type EffectiveProgressResponse struct {
	RequestID string `json:"-"`
	EndTime   string `json:"end_time"`
	Progress  int    `json:"progress"`
	Status    int    `json:"status"`
}

// FetchAsyncResponse 异步任务查询响应.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/fetch.html
type FetchAsyncResponse struct {
	RequestID          string      `json:"-"`
	State              int         `json:"state"`
	Msg                string      `json:"msg"`
	Progress           int         `json:"progress"`
	Start              int64       `json:"start"`
	End                int64       `json:"end"`
	URL                string      `json:"url"`
	TotalCount         int         `json:"total_count"`
	SuccessCount       int         `json:"success_count"`
	FailCount          int         `json:"fail_count"`
	ReplaceCount       int         `json:"replace_count"`
	SuccessUploadCount int         `json:"success_upload_count"`
	SuccessSkillInfo   []SkillInfo `json:"success_skill_info"`
	FailSkillInfo      []SkillInfo `json:"fail_skill_info"`
}

// SkillInfo 技能信息.
type SkillInfo struct {
	ID      int64        `json:"id"`
	Name    string       `json:"name"`
	Intents []IntentInfo `json:"intents"`
}

// IntentInfo 意图信息.
type IntentInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// QueryResponse 调用智能对话响应.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/query.html
type QueryResponse struct {
	RequestID  string          `json:"-"`
	Answer     string          `json:"answer"`
	RawAnswer  json.RawMessage `json:"-"`
	AnswerType string          `json:"answer_type"`
	SkillName  string          `json:"skill_name"`
	IntentName string          `json:"intent_name"`
	MsgID      string          `json:"msg_id"`
	Options    []Option        `json:"options"`
	Status     string          `json:"status"`
	Slots      []SlotDetail    `json:"slots"`
}

// Option 推荐问题.
type Option struct {
	AnsNodeName string  `json:"ans_node_name"`
	Title       string  `json:"title"`
	Answer      string  `json:"answer"`
	Confidence  float64 `json:"confidence"`
}

// SlotDetail 槽位数据.
type SlotDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Norm  string `json:"norm"`
}

func decodeResponse(response []byte, data interface{}, apiName string) (string, error) {
	var res apiResponse
	if err := json.Unmarshal(response, &res); err != nil {
		return "", fmt.Errorf("json Unmarshal Error, err=%v", err)
	}
	if res.Code != 0 {
		return "", &APIError{
			APIName:   apiName,
			Code:      res.Code,
			Msg:       res.Msg,
			RequestID: res.RequestID,
			Data:      res.Data,
		}
	}
	if data != nil && len(res.Data) > 0 && string(res.Data) != "null" {
		if err := json.Unmarshal(res.Data, data); err != nil {
			return "", fmt.Errorf("json Unmarshal Data Error, err=%v", err)
		}
	}
	return res.RequestID, nil
}
