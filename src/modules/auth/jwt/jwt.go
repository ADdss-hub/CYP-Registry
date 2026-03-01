// Package jwt 提供JWT Token生成和验证
// 遵循《全平台通用用户认证设计规范》JWT规范
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/pkg/config"
)

var (
	ErrTokenExpired  = errors.New("token已过期")
	ErrTokenInvalid  = errors.New("token无效")
	ErrTokenMissing  = errors.New("token缺失")
	ErrInvalidClaims = errors.New("无效的token声明")
)

// Config JWT配置
type Config struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpire  int64 // 秒
	RefreshExpire int64 // 秒
}

// TokenClaims Token声明结构
type TokenClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	TokenType string    `json:"token_type"` // "access" 或 "refresh"
	jwtv5.RegisteredClaims
}

// TokenPair Token对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// Service JWT服务
type Service struct {
	config Config
}

// NewService 创建JWT服务
func NewService(cfg *config.JWTConfig) *Service {
	return &Service{
		config: Config{
			AccessSecret:  cfg.Secret + "_access",
			RefreshSecret: cfg.Secret + "_refresh",
			AccessExpire:  cfg.AccessTokenExpire,
			RefreshExpire: cfg.RefreshTokenExpire,
		},
	}
}

// GenerateTokenPair 生成Token对
func (s *Service) GenerateTokenPair(userID uuid.UUID, username string) (*TokenPair, error) {
	now := time.Now()
	accessExpires := now.Add(time.Duration(s.config.AccessExpire) * time.Second)
	refreshExpires := now.Add(time.Duration(s.config.RefreshExpire) * time.Second)

	// 生成Access Token
	accessToken, err := s.generateAccessToken(userID, username, now, accessExpires)
	if err != nil {
		return nil, fmt.Errorf("生成access token失败: %w", err)
	}

	// 生成Refresh Token
	refreshToken, err := s.generateRefreshToken(userID, username, now, refreshExpires)
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpires,
		TokenType:    "Bearer",
	}, nil
}

// generateAccessToken 生成Access Token
func (s *Service) generateAccessToken(userID uuid.UUID, username string, now, expires time.Time) (string, error) {
	claims := TokenClaims{
		UserID:    userID,
		Username:  username,
		TokenType: "access",
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    "cyp-registry",
			Subject:   userID.String(),
			Audience:  jwtv5.ClaimStrings{username},
			ExpiresAt: jwtv5.NewNumericDate(expires),
			IssuedAt:  jwtv5.NewNumericDate(now),
			NotBefore: jwtv5.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.AccessSecret))
}

// generateRefreshToken 生成Refresh Token
func (s *Service) generateRefreshToken(userID uuid.UUID, username string, now, expires time.Time) (string, error) {
	claims := TokenClaims{
		UserID:    userID,
		Username:  username,
		TokenType: "refresh",
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    "cyp-registry",
			Subject:   userID.String(),
			ExpiresAt: jwtv5.NewNumericDate(expires),
			IssuedAt:  jwtv5.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.RefreshSecret))
}

// ValidateAccessToken 验证Access Token
func (s *Service) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwtv5.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.AccessSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwtv5.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	if claims.TokenType != "access" {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// ValidateRefreshToken 验证Refresh Token
func (s *Service) ValidateRefreshToken(tokenString string) (*TokenClaims, error) {
	token, err := jwtv5.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.RefreshSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwtv5.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	if claims.TokenType != "refresh" {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// RefreshTokenPair 使用Refresh Token刷新Token对
func (s *Service) RefreshTokenPair(refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return s.GenerateTokenPair(claims.UserID, claims.Username)
}
