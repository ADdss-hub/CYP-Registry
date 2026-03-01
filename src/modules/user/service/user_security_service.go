// Package service 提供用户认证相关业务逻辑
// 遵循《全平台通用用户认证设计规范》
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"golang.org/x/crypto/bcrypt"

	"github.com/cyp-registry/registry/src/pkg/cache"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// ChangePassword 修改密码
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	if database.DB == nil {
		return errors.ErrDatabaseError
	}

	var user models.User
	result := database.DB.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		return ErrUserNotFound
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return ErrPasswordIncorrect
	}

	// 加密新密码
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 更新密码
	result = database.DB.Model(&user).Update("password", string(hash))
	if result.Error != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"change_password","user_id":"%s","error":"failed to update password: %v"}`, time.Now().Format(time.RFC3339), userID.String(), result.Error)
		return result.Error
	}

	// 撤销所有RefreshToken
	if err := database.DB.Model(&models.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("revoked_at", time.Now()).Error; err != nil {
		// 记录错误但不影响密码修改流程
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"change_password","user_id":"%s","error":"failed to revoke refresh tokens: %v"}`, time.Now().Format(time.RFC3339), userID.String(), err)
	}

	log.Printf(`{"timestamp":"%s","level":"info","module":"user","operation":"change_password","user_id":"%s"}`, time.Now().Format(time.RFC3339), userID.String())
	return nil
}

// isBruteForceAttack 检查是否为暴力破解攻击
func (s *Service) isBruteForceAttack(ctx context.Context, username, ip string) bool {
	// 检查账户级别锁定
	userKey := fmt.Sprintf("login:fail:user:%s", username)
	failCount, _ := cache.GetCounter(ctx, userKey)
	if failCount >= 10 {
		return true
	}

	// 检查IP级别锁定
	ipKey := fmt.Sprintf("login:fail:ip:%s", ip)
	ipFailCount, _ := cache.GetCounter(ctx, ipKey)
	return ipFailCount >= 50
}

// recordLoginFailure 记录登录失败
func (s *Service) recordLoginFailure(ctx context.Context, username, ip string) {
	userKey := fmt.Sprintf("login:fail:user:%s", username)
	ipKey := fmt.Sprintf("login:fail:ip:%s", ip)

	// 忽略计数失败错误，不影响主流程
	if _, err := cache.Incr(ctx, userKey); err != nil {
		_ = err
	}
	_ = cache.Expire(ctx, userKey, time.Minute) // 1分钟后重置

	if _, err := cache.Incr(ctx, ipKey); err != nil {
		_ = err
	}
	_ = cache.Expire(ctx, ipKey, time.Hour) // 1小时后重置

	// 如果达到锁定阈值，记录安全事件
	userFailCount, _ := cache.GetCounter(ctx, userKey)
	if userFailCount >= 10 {
		s.recordSecurityEvent(ctx, username, ip, "brute_force_account", "HIGH")
	}

	ipFailCount, _ := cache.GetCounter(ctx, ipKey)
	if ipFailCount >= 50 {
		s.recordSecurityEvent(ctx, "", ip, "brute_force_ip", "CRITICAL")
	}
}

// clearLoginFailure 清除登录失败记录
func (s *Service) clearLoginFailure(ctx context.Context, username, ip string) {
	userKey := fmt.Sprintf("login:fail:user:%s", username)
	ipKey := fmt.Sprintf("login:fail:ip:%s", ip)

	_ = cache.Del(ctx, userKey, ipKey)
}

// recordSecurityEvent 记录安全事件
func (s *Service) recordSecurityEvent(_ context.Context, username, ip, eventType, severity string) {
	event := &models.SecurityEvent{
		EventType: eventType,
		Severity:  severity,
		IP:        ip,
		Details:   fmt.Sprintf("用户名: %s", username),
	}

	// 记录安全事件，失败时仅记录错误，不影响主流程
	if err := database.DB.Create(event).Error; err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"record_security_event","event_type":"%s","severity":"%s","username":"%s","ip":"%s","error":"failed to save security event: %v"}`, time.Now().Format(time.RFC3339), eventType, severity, username, ip, err)
	} else {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"security_event","event_type":"%s","severity":"%s","username":"%s","ip":"%s"}`, time.Now().Format(time.RFC3339), eventType, severity, username, ip)
	}
}
