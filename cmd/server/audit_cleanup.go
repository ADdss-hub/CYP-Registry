// Package main 审计日志清理任务
package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/cyp-registry/registry/src/pkg/audit"
)

// startAuditLogCleanupTask 启动审计日志清理定时任务
func startAuditLogCleanupTask() {
	// 从环境变量读取保留天数，默认15天
	retentionDays := 15
	if v := os.Getenv("AUDIT_LOG_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			retentionDays = n
		}
	}

	// 从环境变量读取清理间隔，默认每天凌晨2点执行
	cleanupInterval := 24 * time.Hour
	if v := os.Getenv("AUDIT_LOG_CLEANUP_INTERVAL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cleanupInterval = time.Duration(n) * time.Hour
		}
	}

	// 首次清理延迟：等待到下一个整点小时
	now := time.Now()
	nextHour := now.Truncate(time.Hour).Add(time.Hour)
	initialDelay := nextHour.Sub(now)
	if initialDelay < 0 {
		initialDelay = 0
	}

	log.Printf("审计日志清理任务已启动: 保留天数=%d天, 清理间隔=%v, 首次执行延迟=%v", retentionDays, cleanupInterval, initialDelay)

	// 首次延迟执行
	time.Sleep(initialDelay)

	// 执行首次清理
	performCleanup(retentionDays)

	// 定时执行清理
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		performCleanup(retentionDays)
	}
}

// performCleanup 执行清理操作
func performCleanup(retentionDays int) {
	log.Printf("开始清理审计日志（保留%d天）...", retentionDays)

	// 获取清理前的日志数量
	oldCount, err := audit.GetOldLogCount(retentionDays)
	if err != nil {
		log.Printf("警告: 获取待清理日志数量失败: %v", err)
		return
	}

	if oldCount == 0 {
		log.Printf("没有需要清理的日志")
		return
	}

	// 执行清理
	if err := audit.CleanupOldLogs(retentionDays); err != nil {
		log.Printf("错误: 清理审计日志失败: %v", err)
		return
	}

	// 获取清理后的日志数量
	totalCount, err := audit.GetLogCount()
	if err != nil {
		log.Printf("警告: 获取日志总数失败: %v", err)
	} else {
		log.Printf("审计日志清理完成: 删除了 %d 条日志，当前剩余 %d 条日志", oldCount, totalCount)
	}
}
