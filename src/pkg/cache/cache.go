// Package cache 提供Redis缓存层封装
// 遵循《全平台通用开发任务设计规范文档》第6章缓存规范
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/redis/go-redis/v9"
)

// Cache Redis客户端实例
var Cache *redis.Client

// Init 初始化Redis连接
func Init(cfg *config.RedisConfig) error {
	Cache = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Cache.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("连接Redis失败: %w", err)
	}

	return nil
}

// Close 关闭Redis连接
func Close() error {
	if Cache != nil {
		return Cache.Close()
	}
	return nil
}

// Get 获取缓存值
func Get(ctx context.Context, key string) (string, error) {
	if Cache == nil {
		return "", fmt.Errorf("缓存未初始化")
	}
	return Cache.Get(ctx, Key(key)).Result()
}

// Set 设置缓存值
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if Cache == nil {
		return fmt.Errorf("缓存未初始化")
	}
	return Cache.Set(ctx, Key(key), value, expiration).Err()
}

// Del 删除缓存
func Del(ctx context.Context, keys ...string) error {
	if Cache == nil {
		return fmt.Errorf("缓存未初始化")
	}
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = Key(key)
	}
	return Cache.Del(ctx, fullKeys...).Err()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	if Cache == nil {
		return false, fmt.Errorf("缓存未初始化")
	}
	n, err := Cache.Exists(ctx, Key(key)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Incr 自增
func Incr(ctx context.Context, key string) (int64, error) {
	if Cache == nil {
		return 0, fmt.Errorf("缓存未初始化")
	}
	return Cache.Incr(ctx, Key(key)).Result()
}

// IncrBy 自增指定值
func IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	if Cache == nil {
		return 0, fmt.Errorf("缓存未初始化")
	}
	return Cache.IncrBy(ctx, Key(key), value).Result()
}

// Expire 设置过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	if Cache == nil {
		return fmt.Errorf("缓存未初始化")
	}
	return Cache.Expire(ctx, Key(key), expiration).Err()
}

// TTL 获取剩余过期时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	if Cache == nil {
		return 0, fmt.Errorf("缓存未初始化")
	}
	return Cache.TTL(ctx, Key(key)).Result()
}

// GetInt 获取int类型值
func GetInt(ctx context.Context, key string) (int, error) {
	val, err := Get(ctx, key)
	if err != nil {
		return 0, err
	}
	var n int
	fmt.Sscanf(val, "%d", &n)
	return n, nil
}

// GetInt64 获取int64类型值
func GetInt64(ctx context.Context, key string) (int64, error) {
	val, err := Get(ctx, key)
	if err != nil {
		return 0, err
	}
	var n int64
	fmt.Sscanf(val, "%d", &n)
	return n, nil
}

// SetNX 设置NX（键不存在时设置）
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	if Cache == nil {
		return false, fmt.Errorf("缓存未初始化")
	}
	return Cache.SetNX(ctx, Key(key), value, expiration).Result()
}

// ZAdd 有序集合添加
func ZAdd(ctx context.Context, key string, score float64, member string) error {
	if Cache == nil {
		return fmt.Errorf("缓存未初始化")
	}
	return Cache.ZAdd(ctx, Key(key), redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// ZRemRangeByScore 按分数范围删除
func ZRemRangeByScore(ctx context.Context, key, min, max string) error {
	if Cache == nil {
		return fmt.Errorf("缓存未初始化")
	}
	return Cache.ZRemRangeByScore(ctx, Key(key), min, max).Err()
}

// ZRevRangeWithScores 获取有序集合（按分数倒序）
func ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	if Cache == nil {
		return nil, fmt.Errorf("缓存未初始化")
	}
	return Cache.ZRevRangeWithScores(ctx, Key(key), start, stop).Result()
}

// HGet 获取哈希字段值
func HGet(ctx context.Context, key, field string) (string, error) {
	if Cache == nil {
		return "", fmt.Errorf("缓存未初始化")
	}
	return Cache.HGet(ctx, Key(key), field).Result()
}

// HSet 设置哈希字段值
func HSet(ctx context.Context, key, field string, value interface{}) error {
	if Cache == nil {
		return fmt.Errorf("缓存未初始化")
	}
	return Cache.HSet(ctx, Key(key), field, value).Err()
}

// HDel 删除哈希字段
func HDel(ctx context.Context, key string, fields ...string) error {
	if Cache == nil {
		return fmt.Errorf("缓存未初始化")
	}
	if len(fields) == 0 {
		return nil
	}
	return Cache.HDel(ctx, Key(key), fields...).Err()
}

// HGetAll 获取哈希所有字段
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if Cache == nil {
		return nil, fmt.Errorf("缓存未初始化")
	}
	return Cache.HGetAll(ctx, Key(key)).Result()
}

// incrCounter 内部方法：计数器自增（用于限流等）
func incrCounter(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	if Cache == nil {
		return 0, fmt.Errorf("缓存未初始化")
	}
	// 尝试设置NX键（使用Key函数添加前缀）
	fullKey := Key(key)
	set, err := Cache.SetNX(ctx, fullKey, 1, expiration).Result()
	if err != nil {
		return 0, err
	}

	// 如果键已存在，直接使用 fullKey 自增（保持一致性）
	if !set {
		return Cache.Incr(ctx, fullKey).Result()
	}

	return 1, nil
}

// GetCounter 获取计数器当前值
func GetCounter(ctx context.Context, key string) (int64, error) {
	val, err := Get(ctx, key)
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var n int64
	fmt.Sscanf(val, "%d", &n)
	return n, nil
}

// Lock 分布式锁
func Lock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return SetNX(ctx, "lock:"+key, "locked", expiration)
}

// Unlock 释放分布式锁
func Unlock(ctx context.Context, key string) error {
	return Del(ctx, "lock:"+key)
}

// cfg 配置
var cacheCfg struct {
	KeyPrefix string
}

// InitConfig 初始化缓存配置
func InitConfig(prefix string) {
	cacheCfg.KeyPrefix = prefix
}

// Key 生成带前缀的键
func Key(key string) string {
	return cacheCfg.KeyPrefix + key
}
