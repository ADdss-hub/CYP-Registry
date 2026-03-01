package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cyp-registry/registry/src/modules/storage"
	"github.com/cyp-registry/registry/src/pkg/cache"
	"github.com/cyp-registry/registry/src/pkg/database"
)

// cleanupDatabase 清理数据库数据
// 删除所有表的数据，但保留表结构
func cleanupDatabase() error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 获取所有表名
	var tables []string
	if err := db.Raw(`
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public'
	`).Scan(&tables).Error; err != nil {
		return fmt.Errorf("获取表列表失败: %w", err)
	}

	// 禁用外键约束检查（PostgreSQL使用TRUNCATE CASCADE）
	// 注意：这会删除所有表的数据，包括关联数据
	for _, table := range tables {
		// 使用TRUNCATE CASCADE删除所有数据并重置自增序列
		if err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			log.Printf("警告: 清理表 %s 失败: %v", table, err)
			// 继续清理其他表
		}
	}

	return nil
}

// cleanupStorage 清理文件存储
func cleanupStorage(store storage.Storage) error {
	if store == nil {
		return fmt.Errorf("存储未初始化")
	}

	ctx := context.Background()

	// 列出所有文件并删除
	// 注意：这里假设存储根目录是空字符串或"/"
	paths, err := store.List(ctx, "")
	if err != nil {
		// 如果List失败，尝试删除已知的路径
		log.Printf("警告: 列出存储文件失败: %v，尝试清理已知路径", err)
		// 清理常见的存储路径
		commonPaths := []string{"blobs", "manifests", "repositories"}
		for _, path := range commonPaths {
			if exists, _ := store.Exists(ctx, path); exists {
				if err := store.Delete(ctx, path); err != nil {
					log.Printf("警告: 删除存储路径 %s 失败: %v", path, err)
				}
			}
		}
		return nil
	}

	// 删除所有列出的文件
	for _, path := range paths {
		if err := store.Delete(ctx, path); err != nil {
			log.Printf("警告: 删除存储文件 %s 失败: %v", path, err)
			// 继续删除其他文件
		}
	}

	return nil
}

// cleanupUploads 清理上传文件（头像等）
func cleanupUploads(uploadsDir string) error {
	if uploadsDir == "" {
		return nil // 没有配置上传目录，跳过
	}

	// 检查目录是否存在
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		return nil // 目录不存在，无需清理
	}

	// 删除整个上传目录
	if err := os.RemoveAll(uploadsDir); err != nil {
		return fmt.Errorf("删除上传目录失败: %w", err)
	}

	// 重新创建空目录（可选，保持目录结构）
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Printf("警告: 重新创建上传目录失败: %v", err)
	}

	return nil
}

// cleanupCache 清理缓存数据
func cleanupCache() error {
	if cache.Cache == nil {
		return nil // 缓存未初始化，跳过
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 获取所有键（使用通配符匹配）
	// 注意：Redis的KEYS命令在生产环境中可能影响性能
	// 这里使用SCAN来迭代所有键
	iter := cache.Cache.Scan(ctx, 0, "*", 0).Iterator()
	keys := []string{}
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("扫描缓存键失败: %w", err)
	}

	// 删除所有键
	if len(keys) > 0 {
		if err := cache.Cache.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("删除缓存键失败: %w", err)
		}
	}

	return nil
}

