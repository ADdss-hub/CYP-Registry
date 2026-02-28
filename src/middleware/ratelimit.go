// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"github.com/cyp-registry/registry/src/pkg/cache"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// Limiter 限流器接口
type Limiter interface {
	Limit(ctx context.Context, key string, limit, burst uint64) (remaining uint64, ok bool)
}

// MemoryLimiter 基于内存的限流器（使用 golang.org/x/time/rate）
type MemoryLimiter struct {
	mu     sync.RWMutex
	limits map[string]*rate.Limiter
}

// NewMemoryLimiter 创建内存限流器
func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{
		limits: make(map[string]*rate.Limiter),
	}
}

// getLimiter 获取或创建限流器
func (m *MemoryLimiter) getLimiter(key string, limit rate.Limit, burst int) *rate.Limiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if l, exists := m.limits[key]; exists {
		return l
	}

	l := rate.NewLimiter(limit, burst)
	m.limits[key] = l
	return l
}

// Limit 检查是否超过限流
func (m *MemoryLimiter) Limit(ctx context.Context, key string, limit, burst uint64) (remaining uint64, ok bool) {
	l := m.getLimiter(key, rate.Limit(limit), int(burst))
	ok = l.Allow()

	// 计算剩余 token 数（简化处理）
	if ok {
		remaining = burst
	} else {
		remaining = 0
	}

	return remaining, ok
}

// RedisLimiter 基于Redis的限流器
type RedisLimiter struct {
	client    *redis.Client
	luaScript *redis.Script
}

// NewRedisLimiter 创建Redis限流器
func NewRedisLimiter(client *redis.Client) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		luaScript: redis.NewScript(`
			local key = KEYS[1]
			local limit = tonumber(ARGV[1])
			local window = tonumber(ARGV[2])
			local now = tonumber(ARGV[3])

			redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
			local count = redis.call('ZCARD', key)

			if count < limit then
				redis.call('ZADD', key, now, now .. '-' .. math.random())
				redis.call('EXPIRE', key, window)
			end

			return {limit - count, count}
		`),
	}
}

// Limit 检查是否超过限流
func (r *RedisLimiter) Limit(ctx context.Context, key string, limit, burst uint64) (remaining uint64, ok bool) {
	window := time.Second

	result, err := r.luaScript.Run(ctx, r.client, []string{key}, limit, int64(window.Milliseconds()), time.Now().UnixNano()).Slice()
	if err != nil {
		return 0, false
	}

	remaining = uint64(result[0].(int64))
	count := result[1].(int64)

	return remaining, count < int64(limit)
}

// RateLimitMiddleware 限流中间件
type RateLimitMiddleware struct {
	config       *config.RateLimitConfig
	limiter      Limiter
	redisLimiter *RedisLimiter
}

// NewRateLimitMiddleware 创建限流中间件
func NewRateLimitMiddleware(cfg *config.RateLimitConfig, redisLimiter *RedisLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		config:       cfg,
		limiter:      NewMemoryLimiter(),
		redisLimiter: redisLimiter,
	}
}

// RateLimit 限流中间件处理函数
func (m *RateLimitMiddleware) RateLimit() gin.HandlerFunc {
	if !m.config.Enabled {
		return func(ctx *gin.Context) {
			ctx.Next()
		}
	}

	return func(ctx *gin.Context) {
		// 使用IP作为限流键
		key := "ratelimit:" + ctx.ClientIP()

		var remaining uint64
		var ok bool

		if m.redisLimiter != nil {
			remaining, ok = m.redisLimiter.Limit(
				ctx,
				key,
				uint64(m.config.RequestsPerSecond),
				uint64(m.config.Burst),
			)
		} else {
			remaining, ok = m.limiter.Limit(ctx, key, uint64(m.config.RequestsPerSecond), uint64(m.config.Burst))
		}

		// 设置限流头
		ctx.Header("X-RateLimit-Limit", strconv.Itoa(m.config.RequestsPerSecond))
		ctx.Header("X-RateLimit-Remaining", strconv.FormatUint(remaining, 10))

		if !ok {
			// 超过限流
			ctx.Header("Retry-After", "1")
			response.TooManyRequests(ctx, "请求过于频繁，请稍后重试")
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// BruteForceProtection 暴力破解防护中间件
type BruteForceProtection struct {
	config *config.BruteForceConfig
}

// NewBruteForceProtection 创建暴力破解防护中间件
func NewBruteForceProtection(cfg *config.BruteForceConfig) *BruteForceProtection {
	return &BruteForceProtection{config: cfg}
}

// Protect 暴力破解防护中间件处理函数
func (b *BruteForceProtection) Protect() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		username := ctx.PostForm("username")

		// 检查IP是否被封禁
		ipKey := "blocked:ip:" + ip
		blocked, _ := cache.Exists(ctx, ipKey)
		if blocked {
			response.Forbidden(ctx, "IP地址已被封禁")
			ctx.Abort()
			return
		}

		// 检查账户是否被锁定
		if username != "" {
			userKey := "locked:user:" + username
			locked, _ := cache.Exists(ctx, userKey)
			if locked {
				response.Forbidden(ctx, "账户已被锁定")
				ctx.Abort()
				return
			}
		}

		ctx.Next()
	}
}

// CheckBruteForce 检测并记录暴力破解尝试
func (b *BruteForceProtection) CheckBruteForce(ctx context.Context, username, ip string, success bool) {
	// 成功登录，清除失败记录
	if success {
		cache.Del(ctx, "fail:ip:"+ip, "fail:user:"+username)
		return
	}

	// 记录失败
	failKey := "fail:" + ip
	cache.Incr(ctx, failKey)
	cache.Expire(ctx, failKey, time.Minute)

	userFailKey := "fail:user:" + username
	cache.Incr(ctx, userFailKey)
	cache.Expire(ctx, userFailKey, time.Minute)

	// 检查是否达到阈值
	failCount, _ := cache.GetInt(ctx, failKey)
	if failCount >= b.config.MaxAttemptsPerIP {
		// 封禁IP
		cache.SetNX(ctx, "blocked:ip:"+ip, "1", time.Duration(b.config.LockoutDuration)*time.Second)
		// TODO: 记录安全事件
	}

	userFailCount, _ := cache.GetInt(ctx, userFailKey)
	if userFailCount >= b.config.MaxAttemptsPerMinute {
		// 锁定账户
		cache.SetNX(ctx, "locked:user:"+username, "1", time.Duration(b.config.LockoutDuration)*time.Second)
		// TODO: 记录安全事件
	}
}

// HTTPErrorHandler HTTP错误处理器
type HTTPErrorHandler struct{}

// NewHTTPErrorHandler 创建HTTP错误处理器
func NewHTTPErrorHandler() *HTTPErrorHandler {
	return &HTTPErrorHandler{}
}

// Handle 处理错误响应
func (h *HTTPErrorHandler) Handle(err error) gin.H {
	return gin.H{
		"code":    50001,
		"message": err.Error(),
		"data":    nil,
	}
}

// WriteError 写入错误响应
func (h *HTTPErrorHandler) WriteError(ctx *gin.Context, status int, err error) {
	ctx.JSON(status, h.Handle(err))
}

// GlobalRateLimiter 全局限流器（用于整个应用）
var GlobalRateLimiter *RateLimitMiddleware

// InitGlobalRateLimiter 初始化全局限流器
func InitGlobalRateLimiter(cfg *config.RateLimitConfig) {
	GlobalRateLimiter = NewRateLimitMiddleware(cfg, nil)
}
