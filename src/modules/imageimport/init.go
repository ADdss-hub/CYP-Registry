// Package imageimport 提供镜像导入模块的初始化入口
// 主要负责数据库表结构初始化（AutoMigrate）
package imageimport

import (
	"fmt"

	"github.com/cyp-registry/registry/src/modules/imageimport/models"
	"github.com/cyp-registry/registry/src/pkg/database"
)

// InitDatabase 初始化镜像导入相关的数据库表
// 在 cmd/server/main.go 中调用；失败时不会阻止主进程启动，而是以警告形式输出
func InitDatabase() error {
	if database.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := database.DB.AutoMigrate(&models.ImportTask{}); err != nil {
		return fmt.Errorf("auto migrate image_import_tasks failed: %w", err)
	}
	return nil
}
