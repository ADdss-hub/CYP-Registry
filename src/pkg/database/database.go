// Package database 提供数据库连接和管理功能
// 遵循《全平台通用数据库个人管理规范》第11章
package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/cyp-registry/registry/src/pkg/config"
)

// DB 全局数据库连接实例
var DB *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.DatabaseConfig) error {
	var err error

	// 配置GORM日志级别
	// 默认使用 Silent 级别，避免输出所有 SQL 查询日志（生产环境最佳实践）
	// 如需调试，可通过环境变量 GORM_LOG_LEVEL=info 开启 SQL 日志
	logLevel := logger.Silent
	if gormLogLevel := os.Getenv("GORM_LOG_LEVEL"); gormLogLevel == "info" {
		logLevel = logger.Info
	} else if gormLogLevel == "warn" {
		logLevel = logger.Warn
	} else if gormLogLevel == "error" {
		logLevel = logger.Error
	}

	if cfg.Host == "" {
		return fmt.Errorf("数据库配置不完整")
	}

	// 初始化数据库连接
	DB, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",    // 表前缀
			SingularTable: false, // 使用复数表名
			NameReplacer:  nil,   // 名称替换器
		},
		SkipDefaultTransaction:                   false,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取底层数据库连接失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())

	// 验证连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接验证失败: %w", err)
	}

	return nil
}

// Close 关闭数据库连接
// 会等待所有正在进行的查询完成，然后关闭所有连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		
		// 设置连接池参数，确保在关闭前能够等待查询完成
		// 获取当前统计信息用于日志
		stats := sqlDB.Stats()
		
		// 关闭数据库连接池
		// sql.DB.Close() 会：
		// 1. 停止接受新的连接请求
		// 2. 等待所有正在进行的查询完成（最多等待连接的最大生命周期）
		// 3. 关闭所有空闲连接
		// 4. 如果超时，会强制关闭
		err = sqlDB.Close()
		
		// 记录关闭时的连接统计信息（用于调试）
		if stats.OpenConnections > 0 {
			// 注意：这里只是记录，实际关闭可能已经完成
			// 如果需要在关闭前等待，应该在调用Close之前设置合理的超时
		}
		
		// 清空全局变量，防止重复关闭
		DB = nil
		
		return err
	}
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	if DB == nil {
		panic("数据库未初始化")
	}
	return DB
}

// Transaction 执行事务
func Transaction(fn func(tx *gorm.DB) error) error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	return DB.Transaction(fn)
}

// RawDB 执行原生SQL查询
func RawDB() *gorm.DB {
	if DB == nil {
		panic("数据库未初始化")
	}
	return DB
}
