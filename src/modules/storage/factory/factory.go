// Package factory 存储工厂模块
// 根据配置创建相应的存储驱动实例
package factory

import (
	"errors"
	"fmt"

	"github.com/cyp-registry/registry/src/modules/storage"
	"github.com/cyp-registry/registry/src/modules/storage/driver"
	"github.com/cyp-registry/registry/src/pkg/config"
)

// StorageType 存储类型
type StorageType string

const (
	// StorageTypeLocal 本地文件系统
	StorageTypeLocal StorageType = "local"
	// StorageTypeMinIO MinIO S3兼容存储
	StorageTypeMinIO StorageType = "minio"
)

// ErrUnsupportedStorage 不支持的存储类型
var ErrUnsupportedStorage = errors.New("factory: unsupported storage type")

// NewStorage 创建存储驱动实例
// 根据配置选择相应的存储后端
func NewStorage(cfg *config.Config) (storage.Storage, error) {
	storageType := StorageType(cfg.GetString("storage.type"))
	if storageType == "" {
		storageType = StorageTypeLocal // 默认为本地存储
	}

	switch storageType {
	case StorageTypeLocal:
		return driver.NewLocalStorage(cfg)

	case StorageTypeMinIO:
		return driver.NewMinIOStorage(cfg)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedStorage, storageType)
	}
}

// NewStorageWithType 根据类型创建存储实例
// 用于测试或显式指定存储类型
func NewStorageWithType(storageType StorageType, cfg *config.Config) (storage.Storage, error) {
	switch storageType {
	case StorageTypeLocal:
		return driver.NewLocalStorage(cfg)

	case StorageTypeMinIO:
		return driver.NewMinIOStorage(cfg)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedStorage, storageType)
	}
}
