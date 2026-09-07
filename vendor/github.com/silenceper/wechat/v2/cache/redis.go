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
	Host         string `json:"host"            yaml:"host"`
	Username     string `json:"username"        yaml:"username"`
	Password     string `json:"password"        yaml:"password"`
	Database     int    `json:"database"        yaml:"database"`
	MinIdleConns int    `json:"min_idle_conns"  yaml:"min_idle_conns"` // 最小空闲连接数
	PoolSize     int    `json:"pool_size"       yaml:"pool_size"`      // 连接池大小，0 表示使用默认值（即 CPU 核心数 * 10）
	MaxRetries   int    `json:"max_retries"     yaml:"max_retries"`    // 最大重试次数，-1 表示不重试，0 表示使用默认值（即 3 次）
	DialTimeout  int    `json:"dial_timeout"    yaml:"dial_timeout"`   // 连接超时时间（秒），0 表示使用默认值（即 5 秒）
	ReadTimeout  int    `json:"read_timeout"    yaml:"read_timeout"`   // 读取超时时间（秒），-1 表示不超时，0 表示使用默认值（即 3 秒）
	WriteTimeout int    `json:"write_timeout"   yaml:"write_timeout"`  // 写入超时时间（秒），-1 表示不超时，0 表示使用默认值（即 ReadTimeout）
	PoolTimeout  int    `json:"pool_timeout"    yaml:"pool_timeout"`   // 连接池获取连接超时时间（秒），0 表示使用默认值（即 ReadTimeout + 1 秒）
	IdleTimeout  int    `json:"idle_timeout"    yaml:"idle_timeout"`   // 空闲连接超时时间（秒），-1 表示禁用空闲连接超时检查，0 表示使用默认值（即 5 分钟）
	UseTLS       bool   `json:"use_tls"         yaml:"use_tls"`        // 是否使用 TLS

	// Deprecated: 应使用 MinIdleConns 代替
	MaxIdle int `json:"max_idle" yaml:"max_idle"`
	// Deprecated: 应使用 PoolSize 代替
	MaxActive int `json:"max_active" yaml:"max_active"`
}

// NewRedis 实例化
func NewRedis(ctx context.Context, opts *RedisOpts) *Redis {
	uniOpt := &redis.UniversalOptions{
		Addrs:        []string{opts.Host},
		DB:           opts.Database,
		Username:     opts.Username,
		Password:     opts.Password,
		MinIdleConns: opts.MinIdleConns,
		PoolSize:     opts.PoolSize,
		MaxRetries:   opts.MaxRetries,
	}

	// 兼容旧的 MaxIdle 参数，仅在未显式设置 MinIdleConns 时生效
	if opts.MaxIdle > 0 && opts.MinIdleConns == 0 {
		uniOpt.MinIdleConns = opts.MaxIdle
	}

	// 兼容旧的 MaxActive 参数，仅在未显式设置 PoolSize 时生效
	if opts.MaxActive > 0 && opts.PoolSize == 0 {
		uniOpt.PoolSize = opts.MaxActive
	}

	applyTimeout := func(seconds int, target *time.Duration) {
		if seconds > 0 {
			*target = time.Duration(seconds) * time.Second
		} else if seconds == -1 {
			// 当 seconds 为 -1 时，表示禁用超时：按 go-redis 约定，将超时时间设置为负值（如 -1ns）代表「无超时」
			*target = -1
		}
		// 当 seconds 为 0 时，使用 go-redis 的默认超时配置：
		// 不修改 target，保持其零值（0），由 go-redis 解释为“使用默认值”
	}

	applyTimeout(opts.DialTimeout, &uniOpt.DialTimeout)
	applyTimeout(opts.ReadTimeout, &uniOpt.ReadTimeout)
	applyTimeout(opts.WriteTimeout, &uniOpt.WriteTimeout)
	applyTimeout(opts.PoolTimeout, &uniOpt.PoolTimeout)
	applyTimeout(opts.IdleTimeout, &uniOpt.IdleTimeout)

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
