// Package pat 提供Personal Access Token管理
// 遵循《全平台通用用户认证设计规范》PAT规范
package pat

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// Token Personal Access Token结构
type Token struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

// TokenResponse Token创建响应（不包含完整token）
type TokenResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenCreate 创建新Token的响应（只返回一次）
type TokenCreate struct {
	TokenResponse
	Token     string `json:"token"` // 只在创建时返回
	TokenType string `json:"token_type"`
}

// Service PAT服务
type Service struct {
	prefix    string
	expireSec int64
}

// NewService 创建PAT服务
func NewService(cfg *config.PATConfig) *Service {
	return &Service{
		prefix:    cfg.Prefix,
		expireSec: cfg.Expire,
	}
}

// calcExpiresAt 根据传入的过期秒数和默认配置计算过期时间。
// 规则：
//   - expireInSec > 0：按传入秒数计算过期时间
//   - expireInSec == 0：使用配置中的默认过期时间（s.expireSec）
//   - 若最终有效过期秒数 <= 0：视为“永不过期”，使用一个远未来的时间表示
func (s *Service) calcExpiresAt(expireInSec int64) time.Time {
	effective := expireInSec
	if effective == 0 {
		effective = s.expireSec
	}

	// small helper: treat <=0 as never expire
	if effective <= 0 {
		// 使用远未来时间表示“永不过期”，前端应将该时间识别并展示为“永不过期”
		return time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	return time.Now().Add(time.Duration(effective) * time.Second)
}

// Generate 生成新PAT
// expireInSec:
//   - >0: 使用该值作为过期秒数
//   - =0: 使用配置中的默认过期时间
//   - <0: 视为“永不过期”
func (s *Service) Generate(userID uuid.UUID, name string, scopes []string, expireInSec int64) (*TokenCreate, error) {
	// 规范化名称并做基本校验
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("PAT名称不能为空")
	}

	// 同一用户下 PAT 名称需唯一（仅针对未撤销的Token）
	{
		var count int64
		if err := database.DB.Model(&models.PersonalAccessToken{}).
			Where("user_id = ? AND name = ? AND revoked_at IS NULL", userID, name).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("检查访问令牌名称唯一性失败: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("访问令牌名称已存在，请更换名称")
		}
	}

	// 生成32字节随机token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("生成随机token失败: %w", err)
	}
	tokenStr := s.prefix + hex.EncodeToString(tokenBytes)

	// 计算token哈希
	tokenHash := s.hashToken(tokenBytes)

	// 计算过期时间
	expiresAt := s.calcExpiresAt(expireInSec)

	// 序列化scopes为JSON
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return nil, fmt.Errorf("序列化scopes失败: %w", err)
	}

	// 保存到数据库
	pat := &models.PersonalAccessToken{
		UserID:     userID,
		TokenHash:  tokenHash,
		Name:       name,
		Scopes:     string(scopesJSON),
		ExpiresAt:  expiresAt,
		LastUsedAt: nil,
		RevokedAt:  nil,
	}

	// 使用事务确保数据一致性
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pat).Error; err != nil {
			// 检查是否是唯一约束冲突（包括 token_hash 或 (user_id, name) 联合唯一索引）
			if strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "UNIQUE constraint") ||
				strings.Contains(err.Error(), "violates unique constraint") {
				return fmt.Errorf("访问令牌名称已存在，请更换名称")
			}
			return fmt.Errorf("保存token失败: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &TokenCreate{
		TokenResponse: TokenResponse{
			ID:        pat.ID,
			Name:      pat.Name,
			Scopes:    scopes,
			ExpiresAt: pat.ExpiresAt,
			CreatedAt: pat.CreatedAt,
		},
		Token:     tokenStr,
		TokenType: "pat",
	}, nil
}

// Validate 验证PAT
func (s *Service) Validate(token string) (*models.PersonalAccessToken, error) {
	// 检查前缀
	if len(token) < len(s.prefix) || token[:len(s.prefix)] != s.prefix {
		return nil, fmt.Errorf("无效的token格式")
	}

	// 提取token值
	tokenValue := token[len(s.prefix):]
	tokenBytes, err := hex.DecodeString(tokenValue)
	if err != nil {
		return nil, fmt.Errorf("无效的token编码")
	}
	tokenHash := s.hashToken(tokenBytes)

	// 从数据库查找
	var pat models.PersonalAccessToken
	result := database.DB.Where("token_hash = ?", tokenHash).
		Where("revoked_at IS NULL").
		Where("expires_at > ?", time.Now()).
		First(&pat)

	if result.Error != nil {
		return nil, fmt.Errorf("token无效或已过期")
	}

	// 更新最后使用时间
	database.DB.Model(&pat).Update("last_used_at", time.Now())

	return &pat, nil
}

// Revoke 撤销Token（对用户而言等同于“删除”）
//
// 为了保持数据库结构的简单性且兼容已有部署，这里采用“软删除”方式：
//   - 立即将 revoked_at 标记为当前时间
//   - 所有读取接口（列表/校验）都只返回 revoked_at IS NULL 的记录
//   - 后台定时任务 cleanup_expired_pat 会负责将已撤销/过期的记录归档并物理删除
//
// 因此前端看到的效果是“删除后立刻从列表消失，不能再使用”，满足“删除”语义。
func (s *Service) Revoke(tokenID uuid.UUID, userID uuid.UUID) error {
	result := database.DB.Model(&models.PersonalAccessToken{}).
		Where("id = ? AND user_id = ?", tokenID, userID).
		Update("revoked_at", time.Now())

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("token不存在")
	}
	return nil
}

// ListByUser 获取用户的所有Token
func (s *Service) ListByUser(userID uuid.UUID) ([]TokenResponse, error) {
	var tokens []models.PersonalAccessToken
	result := database.DB.Where("user_id = ?", userID).
		Where("revoked_at IS NULL").
		Order("created_at DESC").
		Find(&tokens)

	if result.Error != nil {
		return nil, result.Error
	}

	responses := make([]TokenResponse, len(tokens))
	for i, t := range tokens {
		responses[i] = TokenResponse{
			ID:        t.ID,
			Name:      t.Name,
			Scopes:    parseScopes(t.Scopes),
			ExpiresAt: t.ExpiresAt,
			CreatedAt: t.CreatedAt,
		}
	}

	return responses, nil
}

// CleanupExpired 清理过期Token（定时任务调用）
func (s *Service) CleanupExpired() error {
	result := database.DB.Where("expires_at < ?", time.Now()).
		Delete(&models.PersonalAccessToken{})

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// hashToken 计算token哈希
func (s *Service) hashToken(tokenBytes []byte) string {
	hash := sha256.Sum256(tokenBytes)
	return hex.EncodeToString(hash[:])
}

// parseScopes 解析scopes字符串
func parseScopes(scopesStr string) []string {
	if scopesStr == "" || scopesStr == "[]" {
		return []string{}
	}

	// 尝试解析JSON数组
	var scopes []string
	if err := json.Unmarshal([]byte(scopesStr), &scopes); err != nil {
		// 如果解析失败，尝试作为单个字符串处理（兼容旧数据）
		return []string{scopesStr}
	}

	return scopes
}

// CompareToken 安全比较token（防止时序攻击）
func (s *Service) CompareToken(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
