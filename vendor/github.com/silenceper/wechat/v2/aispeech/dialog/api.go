package dialog

import (
	stdcontext "context"
	"encoding/json"

	"github.com/silenceper/wechat/v2/aispeech/encryptor"
)

// GetAccessToken 获取 access token.
func (d *Dialog) GetAccessToken() (string, error) {
	return d.Context.GetAccessToken()
}

// GetAccessTokenContext 获取 access token.
func (d *Dialog) GetAccessTokenContext(ctx stdcontext.Context) (string, error) {
	return d.Context.GetAccessTokenContext(ctx)
}

// ImportJSON 简单问答导入.
func (d *Dialog) ImportJSON(req *ImportJSONRequest) (*ImportJSONResponse, error) {
	return d.ImportJSONContext(stdcontext.Background(), req)
}

// ImportJSONContext 简单问答导入.
func (d *Dialog) ImportJSONContext(ctx stdcontext.Context, req *ImportJSONRequest) (*ImportJSONResponse, error) {
	var res ImportJSONResponse
	requestID, err := d.postJSON(ctx, importJSONPath, req, &res, "AISpeechImportJSON")
	res.RequestID = requestID
	return &res, err
}

// Publish 发布机器人.
func (d *Dialog) Publish() (*PublishResponse, error) {
	return d.PublishContext(stdcontext.Background())
}

// PublishContext 发布机器人.
func (d *Dialog) PublishContext(ctx stdcontext.Context) (*PublishResponse, error) {
	var res PublishResponse
	requestID, err := d.postEmpty(ctx, publishPath, &res, "AISpeechPublish")
	res.RequestID = requestID
	return &res, err
}

// GetEffectiveProgress 查询机器人发布进度.
func (d *Dialog) GetEffectiveProgress(req *EffectiveProgressRequest) (*EffectiveProgressResponse, error) {
	return d.GetEffectiveProgressContext(stdcontext.Background(), req)
}

// GetEffectiveProgressContext 查询机器人发布进度.
func (d *Dialog) GetEffectiveProgressContext(ctx stdcontext.Context, req *EffectiveProgressRequest) (*EffectiveProgressResponse, error) {
	var res EffectiveProgressResponse
	requestID, err := d.postJSON(ctx, effectiveProgressPath, req, &res, "AISpeechGetEffectiveProgress")
	res.RequestID = requestID
	return &res, err
}

// FetchAsync 查询异步任务.
func (d *Dialog) FetchAsync(req *FetchAsyncRequest) (*FetchAsyncResponse, error) {
	return d.FetchAsyncContext(stdcontext.Background(), req)
}

// FetchAsyncContext 查询异步任务.
func (d *Dialog) FetchAsyncContext(ctx stdcontext.Context, req *FetchAsyncRequest) (*FetchAsyncResponse, error) {
	var res FetchAsyncResponse
	requestID, err := d.postJSON(ctx, fetchAsyncPath, req, &res, "AISpeechFetchAsync")
	res.RequestID = requestID
	return &res, err
}

// Query 调用智能对话.
func (d *Dialog) Query(req *QueryRequest) (*QueryResponse, error) {
	return d.QueryContext(stdcontext.Background(), req)
}

// QueryContext 调用智能对话.
func (d *Dialog) QueryContext(ctx stdcontext.Context, req *QueryRequest) (*QueryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	encryptedBody, err := encryptor.Encrypt(d.AESKey, body)
	if err != nil {
		return nil, err
	}
	accessToken, err := d.GetAccessTokenContext(ctx)
	if err != nil {
		return nil, err
	}
	if accessToken == "" {
		return nil, errEmptyAccessToken
	}
	response, err := post(ctx, d.Config, queryPath, []byte(encryptedBody), "text/plain", accessToken, "")
	if err != nil {
		return nil, err
	}
	plainResponse := response
	if !json.Valid(response) {
		plainResponse, err = encryptor.Decrypt(d.AESKey, string(response))
		if err != nil {
			return nil, err
		}
	}

	var res QueryResponse
	requestID, err := decodeResponse(plainResponse, &res, "AISpeechQuery")
	if err != nil {
		return nil, err
	}
	res.RequestID = requestID
	if json.Valid([]byte(res.Answer)) {
		res.RawAnswer = json.RawMessage(res.Answer)
	}
	return &res, nil
}
