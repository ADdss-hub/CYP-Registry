// Package driver 存储驱动模块
// 提供多种存储后端实现：本地文件系统和MinIO S3
package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// ErrNotFound 资源不存在
var ErrNotFound = errors.New("driver: resource not found")

// ErrPermissionDenied 权限拒绝
var ErrPermissionDenied = errors.New("driver: permission denied")

// LocalStorage 本地文件系统存储驱动
// 用于开发环境和小型部署
type LocalStorage struct {
	basePath string
}

// NewLocalStorage 创建本地存储驱动
// config: 配置对象，包含storage.local.path
func NewLocalStorage(cfg *config.Config) (*LocalStorage, error) {
	path := cfg.GetString("storage.local.path")
	if path == "" {
		path = "./storage"
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve storage directory path: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 确保目录权限正确（即使目录已存在）
	if err := os.Chmod(absPath, 0755); err != nil {
		// 记录警告但不失败，允许继续运行
		log.Printf(`{"timestamp":"%s","level":"warn","module":"storage","driver":"local","operation":"chmod_base","path":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), absPath, err)
		// 如果权限设置失败，后续的文件操作会失败并返回明确的错误
	}

	return &LocalStorage{basePath: absPath}, nil
}

// validatePath 验证路径安全
// 防止路径遍历攻击
func (s *LocalStorage) validatePath(path string) error {
	// 检查是否以绝对路径字符开头（Unix: /, Windows: C:\ 等）
	if len(path) > 0 {
		// 检查 Unix 风格绝对路径
		if path[0] == '/' {
			return ErrPermissionDenied
		}
		// 检查 Windows 风格绝对路径（如 C:\）
		if len(path) >= 2 && path[1] == ':' {
			return ErrPermissionDenied
		}
	}

	// 检查原始路径中是否包含父目录引用
	if strings.Contains(path, "..") {
		return ErrPermissionDenied
	}

	// 清理路径
	cleanPath := filepath.Clean(path)

	// 检查是否超出基础目录
	absPath := filepath.Clean(filepath.Join(s.basePath, cleanPath))
	basePath := filepath.Clean(s.basePath)

	// 使用 filepath.Rel 来检查路径是否在基础目录下
	rel, err := filepath.Rel(basePath, absPath)
	if err != nil {
		return ErrPermissionDenied
	}

	// 如果相对路径以 ".." 开头，说明路径越界了
	if strings.HasPrefix(rel, "..") {
		return ErrPermissionDenied
	}

	return nil
}

// getFullPath 获取完整路径
func (s *LocalStorage) getFullPath(path string) string {
	return filepath.Join(s.basePath, path)
}

// Put 上传文件
func (s *LocalStorage) Put(ctx context.Context, path string, reader io.Reader, size int64) error {
	// 验证路径
	if err := s.validatePath(path); err != nil {
		return err
	}

	fullPath := s.getFullPath(path)

	// 确保父目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 确保目录权限正确（即使目录已存在）
	if err := os.Chmod(dir, 0755); err != nil {
		// 记录警告但不失败，允许继续运行
		log.Printf(`{"timestamp":"%s","level":"warn","module":"storage","driver":"local","operation":"chmod_dir","path":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), dir, err)
		// 如果权限设置失败，后续的文件操作会失败并返回明确的错误
	}

	// 创建文件
	file, err := os.Create(fullPath)
	if err != nil {
		// 检查是否是权限问题
		if os.IsPermission(err) {
			return fmt.Errorf("failed to create file: permission denied: %w", err)
		}
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 确保文件权限正确（0644: 所有者可读写，组和其他用户可读）
	if err := os.Chmod(fullPath, 0644); err != nil {
		// 记录警告但不失败，允许继续运行
		log.Printf(`{"timestamp":"%s","level":"warn","module":"storage","driver":"local","operation":"chmod_file","path":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), fullPath, err)
		// 如果权限设置失败，文件已经创建，可以继续写入
	}

	// 写入数据
	if size > 0 {
		// 已知大小，使用buffer优化
		_, err = io.Copy(file, reader)
	} else {
		// 未知大小，逐块复制
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, readErr := reader.Read(buf)
			if n > 0 {
				if _, writeErr := file.Write(buf[:n]); writeErr != nil {
					return fmt.Errorf("failed to write data: %w", writeErr)
				}
			}
			if readErr != nil {
				if readErr == io.EOF {
					break
				}
				return fmt.Errorf("failed to read data: %w", readErr)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Get 获取文件
func (s *LocalStorage) Get(ctx context.Context, path string) (io.Reader, int64, error) {
	// 验证路径
	if err := s.validatePath(path); err != nil {
		return nil, 0, err
	}

	fullPath := s.getFullPath(path)

	// 打开文件
	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 统一使用全局 NotFound 错误，便于上层（如 registry）按规范返回 404 而不是 500
			return nil, 0, response.ErrNotFound
		}
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}

	// 获取文件信息
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("failed to stat file: %w", err)
	}

	return file, stat.Size(), nil
}

// Delete 删除文件
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	// 验证路径
	if err := s.validatePath(path); err != nil {
		return err
	}

	fullPath := s.getFullPath(path)

	// 删除文件
	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return response.ErrNotFound
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists 检查文件是否存在
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	// 验证路径
	if err := s.validatePath(path); err != nil {
		return false, err
	}

	fullPath := s.getFullPath(path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file: %w", err)
	}

	return true, nil
}

// Stat 获取文件信息
func (s *LocalStorage) Stat(ctx context.Context, path string) (int64, string, error) {
	// 验证路径
	if err := s.validatePath(path); err != nil {
		return 0, "", err
	}

	fullPath := s.getFullPath(path)

	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, "", response.ErrNotFound
		}
		return 0, "", fmt.Errorf("failed to stat file: %w", err)
	}

	return stat.Size(), stat.ModTime().Format(time.RFC3339), nil
}

// List 列出目录下的文件
func (s *LocalStorage) List(ctx context.Context, path string) ([]string, error) {
	// 验证路径
	if err := s.validatePath(path); err != nil {
		return nil, err
	}

	fullPath := s.getFullPath(path)

	// 检查目录是否存在
	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, response.ErrNotFound
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	// 列出文件
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			files = append(files, path+"/"+entry.Name()+"/")
		} else {
			files = append(files, path+"/"+entry.Name())
		}
	}

	return files, nil
}

// GetUsage 获取存储使用量
func (s *LocalStorage) GetUsage(ctx context.Context, path string) (int64, int64, error) {
	// 验证路径
	if err := s.validatePath(path); err != nil {
		return 0, 0, err
	}

	fullPath := s.getFullPath(path)

	var totalSize int64
	var fileCount int64

	// 递归计算目录大小
	err := filepath.Walk(fullPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("failed to walk directory: %w", err)
	}

	return totalSize, fileCount, nil
}

// Name 返回驱动名称
func (s *LocalStorage) Name() string {
	return "local"
}

// Close 关闭连接（本地存储无需关闭）
func (s *LocalStorage) Close() error {
	return nil
}
