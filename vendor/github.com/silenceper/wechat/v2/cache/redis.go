package cache

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/go-redis/redis/v8"
)

// Redis .redis cache
type Redis struct {
	ctx  context.Context
	conn redis.UniversalClient
}

// RedisOpts redis 连接属性
type RedisOpts struct {
	Host        string `json:"host"         yml:"host"`
	Username    string `json:"username"                        yaml:"username"`
	Password    string `json:"password"     yml:"password"`
	Database    int    `json:"database"     yml:"database"`
	MaxIdle     int    `json:"max_idle"     yml:"max_idle"`
	MaxActive   int    `json:"max_active"   yml:"max_active"`
	IdleTimeout int    `json:"idle_timeout" yml:"idle_timeout"` // second
	UseTLS      bool   `json:"use_tls"      yml:"use_tls"`      // 是否使用TLS
}

// NewRedis 实例化
func NewRedis(ctx context.Context, opts *RedisOpts) *Redis {
	uniOpt := &redis.UniversalOptions{
		Addrs:        []string{opts.Host},
		DB:           opts.Database,
		Username:     opts.Username,
		Password:     opts.Password,
		IdleTimeout:  time.Second * time.Duration(opts.IdleTimeout),
		MinIdleConns: opts.MaxIdle,
	}

	if opts.UseTLS {
		h, _, err := net.SplitHostPort(opts.Host)
		if err != nil {
			h = opts.Host
		}
		uniOpt.TLSConfig = &tls.Config{
			ServerName: h,
		}
	}

	conn := redis.NewUniversalClient(uniOpt)
	return &Redis{ctx: ctx, conn: conn}
}

// SetConn 设置conn
func (r *Redis) SetConn(conn redis.UniversalClient) {
	r.conn = conn
}

// SetRedisCtx 设置redis ctx 参数
func (r *Redis) SetRedisCtx(ctx context.Context) {
	r.ctx = ctx
}

// Get 获取一个值
func (r *Redis) Get(key string) interface{} {
	return r.GetContext(r.ctx, key)
}

// GetContext 获取一个值
func (r *Redis) GetContext(ctx context.Context, key string) interface{} {
	result, err := r.conn.Do(ctx, "GET", key).Result()
	if err != nil {
		return nil
	}
	return result
}

// Set 设置一个值
func (r *Redis) Set(key string, val interface{}, timeout time.Duration) error {
	return r.SetContext(r.ctx, key, val, timeout)
}

// SetContext 设置一个值
func (r *Redis) SetContext(ctx context.Context, key string, val interface{}, timeout time.Duration) error {
	return r.conn.SetEX(ctx, key, val, timeout).Err()
}

// IsExist 判断key是否存在
func (r *Redis) IsExist(key string) bool {
	return r.IsExistContext(r.ctx, key)
}

// IsExistContext 判断key是否存在
func (r *Redis) IsExistContext(ctx context.Context, key string) bool {
	result, _ := r.conn.Exists(ctx, key).Result()

	return result > 0
}

// Delete 删除
func (r *Redis) Delete(key string) error {
	return r.DeleteContext(r.ctx, key)
}

// DeleteContext 删除
func (r *Redis) DeleteContext(ctx context.Context, key string) error {
	return r.conn.Del(ctx, key).Err()
}
