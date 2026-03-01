// Package service 提供用户认证相关业务逻辑
// 遵循《全平台通用用户认证设计规范》
package service

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/modules/auth/jwt"
	"github.com/cyp-registry/registry/src/modules/auth/pat"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// RefreshToken 刷新Token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	tokens, err := s.jwtSvc.RefreshTokenPair(refreshToken)
	if err != nil {
		return nil, err
	}

	// 更新RefreshToken记录
	s.rotateRefreshToken(ctx, refreshToken, tokens.RefreshToken)

	return tokens, nil
}

// ==================== PAT相关方法 ====================

// CreatePAT 创建Personal Access Token
// expireInSec:
//   - >0: 使用该值作为过期秒数
//   - =0: 使用PAT配置中的默认过期时间
//   - <0: 视为"永不过期"
func (s *Service) CreatePAT(ctx context.Context, userID uuid.UUID, name string, scopes []string, expireInSec int64) (*pat.TokenCreate, error) {
	return s.patSvc.Generate(userID, name, scopes, expireInSec)
}

// ListPAT 列出用户的PAT
func (s *Service) ListPAT(ctx context.Context, userID uuid.UUID) ([]pat.TokenResponse, error) {
	return s.patSvc.ListByUser(userID)
}

// RevokePAT 撤销PAT
func (s *Service) RevokePAT(ctx context.Context, userID, tokenID uuid.UUID) error {
	return s.patSvc.Revoke(tokenID, userID)
}

// ValidatePAT 验证PAT
func (s *Service) ValidatePAT(ctx context.Context, token string) (*models.PersonalAccessToken, error) {
	return s.patSvc.Validate(token)
}

// ==================== JWT相关方法 ====================

// ValidateAccessToken 验证Access Token
func (s *Service) ValidateAccessToken(token string) (*jwt.TokenClaims, error) {
	return s.jwtSvc.ValidateAccessToken(token)
}

// ValidateRefreshToken 验证Refresh Token
func (s *Service) ValidateRefreshToken(token string) (*jwt.TokenClaims, error) {
	return s.jwtSvc.ValidateRefreshToken(token)
}

// GetJWTService 获取JWT服务（用于生成token）
func (s *Service) GetJWTService() *jwt.Service {
	return s.jwtSvc
}

// ==================== RefreshToken管理 ====================

// saveRefreshToken 保存RefreshToken
func (s *Service) saveRefreshToken(_ context.Context, user *models.User, token, ip, userAgent string) {
	// 生成RefreshToken的hash
	tokenHash := hashToken(token)

	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		UserAgent: userAgent,
		IP:        ip,
	}

	// 保存RefreshToken，失败时仅记录错误，不影响登录流程
	if err := database.DB.Create(refreshToken).Error; err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"save_refresh_token","user_id":"%s","error":"failed to save refresh token: %v"}`, time.Now().Format(time.RFC3339), user.ID.String(), err)
	}
}

// rotateRefreshToken 轮换RefreshToken
func (s *Service) rotateRefreshToken(_ context.Context, oldToken, newToken string) {
	oldHash := hashToken(oldToken)
	newHash := hashToken(newToken)

	// 更新RefreshToken，失败时仅记录错误
	if err := database.DB.Model(&models.RefreshToken{}).
		Where("token = ?", oldHash).
		Updates(map[string]interface{}{
			"token":      newHash,
			"updated_at": time.Now(),
		}).Error; err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"rotate_refresh_token","error":"failed to rotate refresh token: %v"}`, time.Now().Format(time.RFC3339), err)
	}
}
