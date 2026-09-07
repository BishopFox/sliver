package dialog

// AccessTokenRequest 获取 access token 请求.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/token.html
type AccessTokenRequest struct {
	Account string `json:"account,omitempty"`
}

// ImportJSONRequest 简单问答导入请求.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/import.html
type ImportJSONRequest struct {
	Mode int         `json:"mode"`
	Data []BotIntent `json:"data"`
}

// BotIntent 简单问答意图.
type BotIntent struct {
	Skill     string   `json:"skill"`
	Intent    string   `json:"intent"`
	Threshold string   `json:"threshold,omitempty"`
	Disable   bool     `json:"disable"`
	Questions []string `json:"questions"`
	Answers   []string `json:"answers"`
}

// EffectiveProgressRequest 发布进度查询请求.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/progress.html
type EffectiveProgressRequest struct {
	Env string `json:"env"`
}

// FetchAsyncRequest 异步任务查询请求.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/fetch.html
type FetchAsyncRequest struct {
	TaskID string `json:"task_id"`
}

// QueryRequest 调用智能对话请求.
// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/query.html
type QueryRequest struct {
	Query                string   `json:"query"`
	Env                  string   `json:"env,omitempty"`
	FirstPrioritySkills  []string `json:"first_priority_skills,omitempty"`
	SecondPrioritySkills []string `json:"second_priority_skills,omitempty"`
	UserName             string   `json:"user_name,omitempty"`
	Avatar               string   `json:"avatar,omitempty"`
	UserID               string   `json:"userid,omitempty"`
}
