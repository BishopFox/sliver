package dialog

import (
	stdcontext "context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/silenceper/wechat/v2/aispeech/config"
	"github.com/silenceper/wechat/v2/cache"
)

const accessTokenCacheKeyPrefix = "gowechat_aispeech_"

var errEmptyAccessToken = errors.New("aispeech access_token is empty")

// AccessToken 微信智能对话 access token.
type AccessToken struct {
	cfg             *config.Config
	accessTokenLock *sync.Mutex
}

// NewAccessToken new AccessToken.
func NewAccessToken(cfg *config.Config) *AccessToken {
	if cfg.Cache == nil {
		panic("cache is needed")
	}
	return &AccessToken{
		cfg:             cfg,
		accessTokenLock: new(sync.Mutex),
	}
}

// GetAccessToken 获取 access token.
func (ak *AccessToken) GetAccessToken() (string, error) {
	return ak.GetAccessTokenContext(stdcontext.Background())
}

// GetAccessTokenContext 获取 access token.
func (ak *AccessToken) GetAccessTokenContext(ctx stdcontext.Context) (string, error) {
	cacheKey := fmt.Sprintf("%s_access_token_%s_%s", accessTokenCacheKeyPrefix, ak.cfg.AppID, ak.cfg.Account)
	if val := cache.GetContext(ctx, ak.cfg.Cache, cacheKey); val != nil {
		if accessToken, ok := val.(string); ok && accessToken != "" {
			return accessToken, nil
		}
	}

	ak.accessTokenLock.Lock()
	defer ak.accessTokenLock.Unlock()

	if val := cache.GetContext(ctx, ak.cfg.Cache, cacheKey); val != nil {
		if accessToken, ok := val.(string); ok && accessToken != "" {
			return accessToken, nil
		}
	}

	accessToken, err := ak.getAccessTokenFromServer(ctx)
	if err != nil {
		return "", err
	}
	if err = cache.SetContext(ctx, ak.cfg.Cache, cacheKey, accessToken, 110*time.Minute); err != nil {
		return "", err
	}
	return accessToken, nil
}

func (ak *AccessToken) getAccessTokenFromServer(ctx stdcontext.Context) (string, error) {
	req := &AccessTokenRequest{Account: ak.cfg.Account}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	response, err := post(ctx, ak.cfg, tokenPath, body, "application/json", "", ak.cfg.AppID)
	if err != nil {
		return "", err
	}

	var res accessTokenData
	if _, err = decodeResponse(response, &res, "AISpeechGetAccessToken"); err != nil {
		return "", err
	}
	if res.AccessToken == "" {
		return "", errEmptyAccessToken
	}
	return res.AccessToken, nil
}
