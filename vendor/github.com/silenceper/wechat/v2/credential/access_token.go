package credential

import "context"

// AccessTokenHandle AccessToken 接口
type AccessTokenHandle interface {
	GetAccessToken() (accessToken string, err error)
}

// AccessTokenCompatibleHandle 同时实现 AccessTokenHandle 和 AccessTokenContextHandle
type AccessTokenCompatibleHandle struct {
	AccessTokenHandle
}

// GetAccessTokenContext 获取access_token,先从cache中获取，没有则从服务端获取
func (c AccessTokenCompatibleHandle) GetAccessTokenContext(_ context.Context) (accessToken string, err error) {
	return c.GetAccessToken()
}

// AccessTokenContextHandle AccessToken 接口
type AccessTokenContextHandle interface {
	AccessTokenHandle
	GetAccessTokenContext(ctx context.Context) (accessToken string, err error)
}
