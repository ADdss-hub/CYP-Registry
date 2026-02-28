// Package storage 存储后端模块
// 提供统一的存储抽象接口，支持本地文件系统和MinIO S3兼容存储
package storage

import (
	"context"
	"errors"
	"io"
)

// ErrNotFound 资源不存在
var ErrNotFound = errors.New("storage: resource not found")

// ErrQuotaExceeded 存储配额超限
var ErrQuotaExceeded = errors.New("storage: storage quota exceeded")

// ErrInvalidPath 无效路径
var ErrInvalidPath = errors.New("storage: invalid path")

// Storage 存储接口
// 定义统一的存储操作，所有存储驱动必须实现此接口
type Storage interface {
	// Put 上传文件
	// ctx: 上下文
	// path: 存储路径（相对路径）
	// reader: 数据源
	// size: 数据大小（-1表示未知）
	Put(ctx context.Context, path string, reader io.Reader, size int64) error

	// Get 获取文件
	// ctx: 上下文
	// path: 存储路径
	// 返回文件内容和长度
	Get(ctx context.Context, path string) (io.Reader, int64, error)

	// Delete 删除文件
	// ctx: 上下文
	// path: 存储路径
	Delete(ctx context.Context, path string) error

	// Exists 检查文件是否存在
	// ctx: 上下文
	// path: 存储路径
	Exists(ctx context.Context, path string) (bool, error)

	// Stat 获取文件信息
	// ctx: 上下文
	// path: 存储路径
	// 返回文件大小和修改时间
	Stat(ctx context.Context, path string) (size int64, modTime string, err error)

	// List 列出目录下的文件
	// ctx: 上下文
	// path: 目录路径
	// 返回文件路径列表
	List(ctx context.Context, path string) ([]string, error)

	// GetUsage 获取存储使用量
	// ctx: 上下文
	// path: 目录路径
	// 返回总大小和文件数
	GetUsage(ctx context.Context, path string) (totalSize int64, fileCount int64, err error)

	// Name 返回存储驱动名称
	Name() string

	// Close 关闭存储连接
	Close() error
}

// BlobInfo Blob信息
type BlobInfo struct {
	Digest    string `json:"digest"`    // SHA256摘要
	Size      int64  `json:"size"`      // 大小
	CreatedAt string `json:"created_at"` // 创建时间
}

// ManifestInfo Manifest信息
type ManifestInfo struct {
	SchemaVersion int      `json:"schema_version"` // 架构版本
	MediaType     string   `json:"media_type"`     // 媒体类型
	Digest        string   `json:"digest"`         // SHA256摘要
	Size          int64    `json:"size"`           // 大小
	Referrers     []string `json:"referrers"`      // 引用此manifest的列表
}

