// Package service 提供用户认证相关业务逻辑
// 遵循《全平台通用用户认证设计规范》
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cyp-registry/registry/src/modules/auth/jwt"
	"github.com/cyp-registry/registry/src/modules/auth/pat"
	"github.com/cyp-registry/registry/src/pkg/cache"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// 用户相关错误
var (
	ErrUserNotFound       = errors.ErrUserNotFound
	ErrUserAlreadyExists  = errors.ErrUserAlreadyExists
	ErrPasswordIncorrect  = errors.ErrPasswordIncorrect
	ErrAccountLocked      = errors.ErrAccountLocked
	ErrBruteForceDetected = errors.ErrBruteForceDetected
)

// Config 服务配置
type Config struct {
	BcryptCost int
}

// Service 用户服务
type Service struct {
	cfg    *Config
	jwtSvc *jwt.Service
	patSvc *pat.Service
}

// DefaultAdminCreds 默认管理员凭据（仅在进程内短暂保存，用于前端首屏提示一次）
type DefaultAdminCreds struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	defaultAdminMu    sync.Mutex
	defaultAdminCreds *DefaultAdminCreds
)

// NotificationSettings 用户通知设置（服务内部结构）
type NotificationSettings struct {
	EmailEnabled         bool   `json:"email_enabled"`
	ScanCompleted        bool   `json:"scan_completed"`
	SecurityAlerts       bool   `json:"security_alerts"`
	WebhookNotifications bool   `json:"webhook_notifications"`
	Digest               string `json:"digest"`
	NotificationEmail    string `json:"notification_email"`
}

// NewService 创建用户服务
func NewService(jwtCfg *config.JWTConfig, patCfg *config.PATConfig, bcryptCost int) *Service {
	svc := &Service{
		cfg: &Config{
			BcryptCost: bcryptCost,
		},
		jwtSvc: jwt.NewService(jwtCfg),
		patSvc: pat.NewService(patCfg),
	}
	// 尝试在服务初始化时自动创建默认管理员账号（仅在用户表为空时执行一次）
	go func() {
		_ = svc.ensureDefaultAdmin()
	}()
	return svc
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

// defaultAdminUsernameFromAppName 将项目名称（APP_NAME / config.app.name）转换为合法用户名：
// - 允许字母数字及 _ . -，且必须以字母/数字开头
// - 长度 3-64
// 若无法得到合法值，则回退到 "cyp-registry"。
func defaultAdminUsernameFromAppName() string {
	// 优先使用全局配置（已应用环境变量覆盖）
	appName := ""
	if c := config.Get(); c != nil {
		appName = strings.TrimSpace(c.App.Name)
	}
	if appName == "" {
		appName = strings.TrimSpace(os.Getenv("APP_NAME"))
	}
	if appName == "" {
		appName = "CYP-Registry"
	}

	// 与单镜像入口脚本保持一致的规则：
	// 1. 统一转为小写
	// 2. 非 [a-z0-9] 字符全部替换为 '-'
	// 3. 连续的 '-' 折叠为一个
	// 4. 去掉首尾 '-'
	lower := strings.ToLower(appName)
	var b strings.Builder
	b.Grow(len(lower))
	prevDash := false
	for _, r := range lower {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			r = '-'
		}

		if r == '-' {
			if prevDash {
				continue
			}
			prevDash = true
		} else {
			prevDash = false
		}
		b.WriteRune(r)
	}
	username := strings.Trim(b.String(), "-")
	// 必须以字母/数字开头：否则加一个前缀
	if username == "" || !regexp.MustCompile(`^[a-z0-9]`).MatchString(username) {
		username = "cyp-" + username
		username = strings.Trim(username, "-")
	}
	// 限制长度 3-64
	if len(username) > 64 {
		username = username[:64]
		username = strings.Trim(username, "-")
	}
	if len(username) < 3 {
		username = "cyp-registry"
	}

	// 最终校验（与注册规则保持一致）
	usernameRe := regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,63}$`)
	if !usernameRe.MatchString(username) {
		return "cyp-registry"
	}
	return username
}

// ConsumeDefaultAdminCreds 获取默认管理员凭据（仅返回一次），用于前端在登录页首次提示并复制保存。
func (s *Service) ConsumeDefaultAdminCreds() *DefaultAdminCreds {
	defaultAdminMu.Lock()
	defer defaultAdminMu.Unlock()
	creds := defaultAdminCreds
	defaultAdminCreds = nil
	return creds
}

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

// defaultNotificationSettings 默认的通知设置
func defaultNotificationSettings() *NotificationSettings {
	return &NotificationSettings{
		EmailEnabled:         true,
		ScanCompleted:        true,
		SecurityAlerts:       true,
		WebhookNotifications: true,
		Digest:               "realtime",
		NotificationEmail:    "",
	}
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
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return user, nil
}

// Login 用户登录
func (s *Service) Login(ctx context.Context, username, password, ip, userAgent string) (*jwt.TokenPair, *models.User, error) {
	if database.DB == nil {
		return nil, nil, errors.ErrDatabaseError
	}

	// 检查登录失败次数（暴力破解防护）
	if s.isBruteForceAttack(ctx, username, ip) {
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
		return nil, nil, ErrUserNotFound
	}

	// 检查账户状态
	if !user.IsActive {
		return nil, nil, ErrAccountLocked
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		// 记录失败尝试
		s.recordLoginFailure(ctx, username, ip)
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
		// TODO: 使用结构化日志记录
	}

	// 记录RefreshToken
	s.saveRefreshToken(ctx, &user, tokens.RefreshToken, ip, userAgent)

	// 清除登录失败记录
	s.clearLoginFailure(ctx, username, ip)

	return tokens, &user, nil
}

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

// GetUserByID 根据ID获取用户
func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if database.DB == nil {
		return nil, errors.ErrDatabaseError
	}

	var user models.User
	result := database.DB.Where("id = ?", userID).
		Where("deleted_at IS NULL").
		First(&user)

	if result.Error != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *Service) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if database.DB == nil {
		return nil, errors.ErrDatabaseError
	}

	var user models.User
	result := database.DB.Where("username = ?", username).
		Where("deleted_at IS NULL").
		First(&user)

	if result.Error != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// UpdateUser 更新用户信息
func (s *Service) UpdateUser(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	if database.DB == nil {
		return errors.ErrDatabaseError
	}

	// GORM 的 Updates 方法可以接受 map，键名可以是结构体字段名（如 "Avatar"）或数据库字段名（如 "avatar"）
	// 为了确保兼容性，我们将所有键名统一转换为结构体字段名（首字母大写）
	// GORM 会自动将结构体字段名转换为数据库字段名（snake_case）
	dbUpdates := make(map[string]interface{})
	for k, v := range updates {
		// 将小写字段名转换为首字母大写的结构体字段名
		// 例如：avatar -> Avatar, nickname -> Nickname
		structFieldName := strings.ToUpper(k[:1]) + k[1:]
		dbUpdates[structFieldName] = v
	}

	result := database.DB.Model(&models.User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(dbUpdates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

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
		return result.Error
	}

	// 撤销所有RefreshToken
	if err := database.DB.Model(&models.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("revoked_at", time.Now()).Error; err != nil {
		// 记录错误但不影响密码修改流程
		// TODO: 使用结构化日志记录
	}

	return nil
}

// DeleteUser 软删除用户
func (s *Service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	if database.DB == nil {
		return errors.ErrDatabaseError
	}

	result := database.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Update("deleted_at", time.Now())

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ==================== 通知设置相关方法 ====================

// GetNotificationSettings 获取用户通知设置（从缓存中读取，不存在时返回默认值）
func (s *Service) GetNotificationSettings(ctx context.Context, userID uuid.UUID) (*NotificationSettings, error) {
	// 如果缓存未初始化，直接返回默认值，避免影响主流程
	if cache.Cache == nil {
		return defaultNotificationSettings(), nil
	}

	key := fmt.Sprintf("user:notification:%s", userID.String())

	exists, err := cache.Exists(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return defaultNotificationSettings(), nil
	}

	raw, err := cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var settings NotificationSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		// 解析失败时返回默认值，避免前端卡死
		return defaultNotificationSettings(), nil
	}

	// 兜底处理 Digest 字段
	if settings.Digest == "" {
		settings.Digest = "realtime"
	}

	return &settings, nil
}

// UpdateNotificationSettings 更新用户通知设置（写入缓存）
func (s *Service) UpdateNotificationSettings(ctx context.Context, userID uuid.UUID, settings *NotificationSettings) error {
	// 缓存未初始化时直接返回成功，避免影响主流程
	if cache.Cache == nil {
		return nil
	}

	// 兜底处理 Digest
	if settings.Digest == "" {
		settings.Digest = "realtime"
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("user:notification:%s", userID.String())
	// 不设置过期时间（0 表示不过期），由业务显式管理
	return cache.Set(ctx, key, string(data), 0)
}

// ListUsers 列出用户（管理员）
func (s *Service) ListUsers(ctx context.Context, page, pageSize int, keyword string) ([]models.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var users []models.User
	var total int64

	q := database.DB.Model(&models.User{}).Where("deleted_at IS NULL")
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username ILIKE ? OR email ILIKE ? OR nickname ILIKE ?", like, like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// ==================== PAT相关方法 ====================

// CreatePAT 创建Personal Access Token
// expireInSec:
//   - >0: 使用该值作为过期秒数
//   - =0: 使用PAT配置中的默认过期时间
//   - <0: 视为“永不过期”
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

// ==================== 暴力破解防护方法 ====================

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

	cache.Incr(ctx, userKey)
	cache.Expire(ctx, userKey, time.Minute) // 1分钟后重置

	cache.Incr(ctx, ipKey)
	cache.Expire(ctx, ipKey, time.Hour) // 1小时后重置

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

	cache.Del(ctx, userKey, ipKey)
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
		// TODO: 使用结构化日志记录
	}
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
		// TODO: 使用结构化日志记录
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
		// TODO: 使用结构化日志记录
	}
}

// hashToken 计算token哈希
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
