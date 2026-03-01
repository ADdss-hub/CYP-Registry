// Package service 提供用户认证相关业务逻辑
// 遵循《全平台通用用户认证设计规范》
package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"gorm.io/gorm"

	"github.com/cyp-registry/registry/src/modules/auth/jwt"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/models"
)

var (
	defaultAdminMu    sync.Mutex
	defaultAdminCreds *DefaultAdminCreds
)

// generateRandomPassword 生成 10-15 位的密码，仅包含英文、数字和常见符号（不包含中文），以避免在命令行或部分客户端中出现编码兼容问题。
func generateRandomPassword() string {
	const lettersDigits = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const symbols = "~!@#$%^&*()-_=+[]{};:,.?/<>"

	// 仅使用英文、数字和常见符号，避免中文字符带来的输入法和编码问题
	var pool []rune
	for _, ch := range lettersDigits + symbols {
		pool = append(pool, ch)
	}

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	length := 10 + r.Intn(6) // 10-15
	out := make([]rune, length)
	for i := range out {
		out[i] = pool[r.Intn(len(pool))]
	}
	return string(out)
}

// ensureDefaultAdmin 在用户表为空时自动创建一个默认管理员账号。
// 默认账号仅用于首次登录：
// - 用户名：6-10 位，仅由英文字母和数字组成
// - 密码：10-15 位，仅由英文字母、数字和常见符号组成（不包含中文），并设置 FirstLogin=true。
func (s *Service) ensureDefaultAdmin() error {
	if database.DB == nil {
		return errors.ErrDatabaseError
	}

	var count int64
	if err := database.DB.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		// 已有用户时不再自动创建
		return nil
	}

	// 默认管理员用户名：固定为项目名称（APP_NAME / config.app.name），并做必要清洗以满足用户名规则
	username := defaultAdminUsernameFromAppName()
	password := generateRandomPassword()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BcryptCost)
	if err != nil {
		return fmt.Errorf("生成默认管理员密码失败: %w", err)
	}

	admin := &models.User{
		Username:   username,
		Email:      fmt.Sprintf("%s@localhost", username),
		Password:   string(hash),
		Nickname:   "默认管理员",
		IsActive:   true,
		IsAdmin:    true,
		FirstLogin: true,
	}

	if err := database.DB.Create(admin).Error; err != nil {
		return err
	}

	// 将默认管理员凭据保存到进程内存中，供前端通过一次性接口获取并提示用户保存。
	defaultAdminMu.Lock()
	defaultAdminCreds = &DefaultAdminCreds{
		Username:  username,
		Password:  password,
		CreatedAt: time.Now(),
	}
	defaultAdminMu.Unlock()

	fmt.Printf("[INIT] 已自动创建默认管理员账号，请尽快登录并修改密码：用户名=%s（固定为项目名称）、密码=%s（10-15 位英文+数字+符号随机）\n", username, password)
	return nil
}

// ConsumeDefaultAdminCreds 获取默认管理员凭据（仅返回一次），用于前端在登录页首次提示并复制保存。
func (s *Service) ConsumeDefaultAdminCreds() *DefaultAdminCreds {
	defaultAdminMu.Lock()
	defer defaultAdminMu.Unlock()
	creds := defaultAdminCreds
	defaultAdminCreds = nil
	return creds
}

// Register 用户注册
func (s *Service) Register(ctx context.Context, username, email, password, nickname string) (*models.User, error) {
	if database.DB == nil {
		return nil, errors.ErrDatabaseError
	}

	username = strings.TrimSpace(username)
	email = strings.TrimSpace(strings.ToLower(email))
	nickname = strings.TrimSpace(nickname)

	// 用户名格式：允许字母数字 + _ . -，且必须以字母/数字开头
	// 典型示例：alice、alice_01、alice.dev、alice-dev
	if username == "" || len(username) < 3 || len(username) > 64 {
		return nil, errors.NewCodeError(10001, "用户名长度需为 3-64")
	}
	usernameRe := regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{2,63}$`)
	if !usernameRe.MatchString(username) {
		return nil, errors.NewCodeError(10001, "用户名仅允许字母数字及 _ . -，且需以字母或数字开头")
	}

	// 检查用户名是否已存在
	var count int64
	if err := database.DB.Model(&models.User{}).Where("username = ? OR email = ?", username, email).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查用户是否存在失败: %w", err)
	}
	if count > 0 {
		return nil, ErrUserAlreadyExists
	}

	// 密码加密
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户（使用事务确保数据一致性）
	user := &models.User{
		Username: username,
		Email:    email,
		Password: string(hash),
		Nickname: nickname,
		IsActive: true,
		IsAdmin:  false,
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			// 检查唯一约束冲突
			if strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "UNIQUE constraint") ||
				strings.Contains(err.Error(), "violates unique constraint") {
				return ErrUserAlreadyExists
			}
			return fmt.Errorf("创建用户失败: %w", err)
		}
		return nil
	})

	if err != nil {
		if err == ErrUserAlreadyExists {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"register","username":"%s","email":"%s","error":"user already exists"}`, time.Now().Format(time.RFC3339), username, maskEmail(email))
			return nil, ErrUserAlreadyExists
		}
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"register","username":"%s","email":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), username, maskEmail(email), err)
		return nil, err
	}

	log.Printf(`{"timestamp":"%s","level":"info","module":"user","operation":"register","user_id":"%s","username":"%s","email":"%s","nickname":"%s"}`, time.Now().Format(time.RFC3339), user.ID.String(), username, maskEmail(email), nickname)
	return user, nil
}

// Login 用户登录
func (s *Service) Login(ctx context.Context, username, password, ip, userAgent string) (*jwt.TokenPair, *models.User, error) {
	if database.DB == nil {
		return nil, nil, errors.ErrDatabaseError
	}

	// 检查登录失败次数（暴力破解防护）
	if s.isBruteForceAttack(ctx, username, ip) {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"login","username":"%s","ip":"%s","error":"brute force attack detected"}`, time.Now().Format(time.RFC3339), username, ip)
		return nil, nil, ErrBruteForceDetected
	}

	// 查找用户
	var user models.User
	result := database.DB.Where("username = ? OR email = ?", username, username).
		Where("deleted_at IS NULL").
		First(&user)

	if result.Error != nil {
		// 记录失败尝试
		s.recordLoginFailure(ctx, username, ip)
		log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"login","username":"%s","ip":"%s","error":"user not found"}`, time.Now().Format(time.RFC3339), username, ip)
		return nil, nil, ErrUserNotFound
	}

	// 检查账户状态
	if !user.IsActive {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"login","user_id":"%s","username":"%s","ip":"%s","error":"account locked"}`, time.Now().Format(time.RFC3339), user.ID.String(), username, ip)
		return nil, nil, ErrAccountLocked
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		// 记录失败尝试
		s.recordLoginFailure(ctx, username, ip)
		log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"login","user_id":"%s","username":"%s","ip":"%s","error":"password incorrect"}`, time.Now().Format(time.RFC3339), user.ID.String(), username, ip)
		return nil, nil, ErrPasswordIncorrect
	}

	// 生成Token对
	tokens, err := s.jwtSvc.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		return nil, nil, fmt.Errorf("生成token失败: %w", err)
	}

	// 更新登录信息
	now := time.Now()
	updates := map[string]interface{}{
		"last_login_at": now,
		"last_login_ip": ip,
		"login_count":   user.LoginCount + 1,
	}
	if user.FirstLogin {
		// 首次成功登录后，关闭 FirstLogin 标记，避免反复提示
		updates["first_login"] = false
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		// 记录错误但不影响登录流程
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"login","user_id":"%s","username":"%s","ip":"%s","error":"failed to update login info: %v"}`, time.Now().Format(time.RFC3339), user.ID.String(), username, ip, err)
	}

	// 记录RefreshToken
	s.saveRefreshToken(ctx, &user, tokens.RefreshToken, ip, userAgent)

	// 清除登录失败记录
	s.clearLoginFailure(ctx, username, ip)

	log.Printf(`{"timestamp":"%s","level":"info","module":"user","operation":"login","user_id":"%s","username":"%s","ip":"%s","user_agent":"%s","first_login":%t}`, time.Now().Format(time.RFC3339), user.ID.String(), username, ip, userAgent, user.FirstLogin)
	return tokens, &user, nil
}
