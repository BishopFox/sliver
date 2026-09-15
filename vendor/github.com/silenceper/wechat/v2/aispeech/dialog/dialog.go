package dialog

import (
	stdcontext "context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/silenceper/wechat/v2/aispeech/config"
	"github.com/silenceper/wechat/v2/aispeech/context"
	"github.com/silenceper/wechat/v2/aispeech/encryptor"
	"github.com/silenceper/wechat/v2/util"
)

const (
	// tokenPath 获取 AccessToken.
	// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/token.html
	tokenPath = "/v2/token"
	// importJSONPath 简单问答 JSON 导入.
	// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/import.html
	importJSONPath = "/v2/bot/import/json"
	// fetchAsyncPath 异步任务查询.
	// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/fetch.html
	fetchAsyncPath = "/v2/async/fetch"
	// publishPath 发布机器人.
	// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/publish.html
	publishPath = "/v2/bot/publish"
	// effectiveProgressPath 查询机器人发布进度.
	// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/progress.html
	effectiveProgressPath = "/v2/bot/effective_progress"
	// queryPath 调用智能对话.
	// 官方文档: https://developers.weixin.qq.com/doc/aispeech/confapi/dialog/bot/query.html
	queryPath = "/v2/bot/query"
)

// Dialog 智能对话平台.
type Dialog struct {
	*context.Context
}

// NewDialog init.
func NewDialog(ctx *context.Context) *Dialog {
	return &Dialog{ctx}
}

func (d *Dialog) postJSON(ctx stdcontext.Context, path string, req interface{}, res interface{}, apiName string) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	accessToken, err := d.GetAccessTokenContext(ctx)
	if err != nil {
		return "", err
	}
	if accessToken == "" {
		return "", errEmptyAccessToken
	}
	response, err := post(ctx, d.Config, path, body, "application/json", accessToken, "")
	if err != nil {
		return "", err
	}
	return decodeResponse(response, res, apiName)
}

func (d *Dialog) postEmpty(ctx stdcontext.Context, path string, res interface{}, apiName string) (string, error) {
	accessToken, err := d.GetAccessTokenContext(ctx)
	if err != nil {
		return "", err
	}
	if accessToken == "" {
		return "", errEmptyAccessToken
	}
	response, err := post(ctx, d.Config, path, nil, "application/json", accessToken, "")
	if err != nil {
		return "", err
	}
	return decodeResponse(response, res, apiName)
}

func post(ctx stdcontext.Context, cfg *config.Config, path string, body []byte, contentType, accessToken, appID string) ([]byte, error) {
	timestamp := time.Now().Unix()
	nonce, err := randomString(16)
	if err != nil {
		return nil, err
	}
	requestID, err := randomString(32)
	if err != nil {
		return nil, err
	}

	header := map[string]string{
		"request_id":   requestID,
		"timestamp":    fmt.Sprintf("%d", timestamp),
		"nonce":        nonce,
		"sign":         encryptor.Sign(cfg.Token, timestamp, nonce, body),
		"Content-Type": contentType,
	}
	if accessToken != "" {
		header["X-OPENAI-TOKEN"] = accessToken
	} else {
		header["X-APPID"] = appID
	}

	return util.HTTPPostContext(ctx, buildURL(cfg.GetBaseURL(), path), body, header)
}

func buildURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func randomString(length int) (string, error) {
	buf := make([]byte, (length+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf)[:length], nil
}
