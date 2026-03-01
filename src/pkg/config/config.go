// Package config 提供配置加载和管理功能
// 遵循《全平台通用开发任务设计规范文档》第6.1节配置管理规范
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用全局配置结构
type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Auth     AuthConfig     `yaml:"auth"`
	Storage  StorageConfig  `yaml:"storage"`
	Registry RegistryConfig `yaml:"registry"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
	Scanner  ScannerConfig  `yaml:"scanner"`
	Webhook  WebhookConfig  `yaml:"webhook"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name  string `yaml:"name"`
	Host  string `yaml:"host"`
	Port  int    `yaml:"port"`
	Env   string `yaml:"env"`
	Debug bool   `yaml:"debug"`
}

// DatabaseConfig PostgreSQL数据库配置
type DatabaseConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Name            string `yaml:"name"`
	SSLMode         string `yaml:"sslmode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"` // 秒
}

// DSN 获取数据库连接字符串
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.Username, d.Password, d.Name, d.SSLMode,
	)
}

// ConnMaxLifetimeDuration 获取连接最大存活时间
func (d DatabaseConfig) ConnMaxLifetimeDuration() time.Duration {
	return time.Duration(d.ConnMaxLifetime) * time.Second
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Password     string `yaml:"password"`
	DB           int    `yaml:"db"`
	PoolSize     int    `yaml:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns"`
	KeyPrefix    string `yaml:"key_prefix"`
}

// Addr 获取Redis地址
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWT        JWTConfig `yaml:"jwt"`
	PAT        PATConfig `yaml:"pat"`
	BcryptCost int       `yaml:"bcrypt_cost"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	AccessTokenExpire  int64  `yaml:"access_token_expire"`  // 秒
	RefreshTokenExpire int64  `yaml:"refresh_token_expire"` // 秒
	Secret             string `yaml:"secret"`
}

// PATConfig Personal Access Token配置
type PATConfig struct {
	Prefix string `yaml:"prefix"`
	Expire int64  `yaml:"expire"` // 秒
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type  string       `yaml:"type"`
	Local LocalStorage `yaml:"local"`
	MinIO MinIOStorage `yaml:"minio"`
}

// LocalStorage 本地存储配置
type LocalStorage struct {
	RootPath string `yaml:"root_path"`
}

// MinIOStorage MinIO存储配置
type MinIOStorage struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
}

// RegistryConfig 镜像仓库配置
type RegistryConfig struct {
	MaxLayerSize   int64 `yaml:"max_layer_size"`
	AllowAnonymous bool  `yaml:"allow_anonymous"`
	TokenExpire    int   `yaml:"token_expire"` // 秒
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	BruteForce BruteForceConfig `yaml:"brute_force"`
	CORS       CORSConfig       `yaml:"cors"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second"`
	Burst             int  `yaml:"burst"`
}

// BruteForceConfig 暴力破解防护配置
type BruteForceConfig struct {
	MaxAttemptsPerMinute int `yaml:"max_attempts_per_minute"`
	LockoutDuration      int `yaml:"lockout_duration"` // 秒
	MaxAttemptsPerIP     int `yaml:"max_attempts_per_ip"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string   `yaml:"level"`
	Format string   `yaml:"format"`
	Output string   `yaml:"output"`
	File   FileLog  `yaml:"file"`
	Trace  TraceLog `yaml:"trace"`
}

// FileLog 文件日志配置
type FileLog struct {
	Path       string `yaml:"path"`
	MaxSize    int    `yaml:"max_size"` // MB
	MaxAge     int    `yaml:"max_age"`  // 天
	MaxBackups int    `yaml:"max_backups"`
}

// TraceLog 链路追踪配置
type TraceLog struct {
	Enabled    bool    `yaml:"enabled"`
	SampleRate float64 `yaml:"sample_rate"`
}

// ScannerConfig 扫描器配置
type ScannerConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Severity        []string `yaml:"severity"`
	BlockOnCritical bool     `yaml:"block_on_critical"`
	Async           bool     `yaml:"async"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	MaxRetries      int    `yaml:"max_retries"`
	Timeout         int    `yaml:"timeout"` // 秒
	SignatureSecret string `yaml:"signature_secret"`
}

var (
	cfg *Config
)

// Init 初始化配置
// 遵循《全平台通用容器开发设计规范》2.1节环境变量配置
func Init(path string) error {
	// 尝试读取配置文件（如果存在）
	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			cfg = &Config{}
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return fmt.Errorf("解析配置文件失败: %w", err)
			}
		} else {
			// 配置文件不存在，使用默认配置
			cfg = &Config{
				App: AppConfig{
					Name:  "CYP-Registry",
					Host:  "0.0.0.0",
					Port:  8080,
					Env:   "production",
					Debug: false,
				},
				Database: DatabaseConfig{
					Host:            "localhost",
					Port:            5432,
					Username:        "registry",
					Password:        "",
					Name:            "registry_db",
					SSLMode:         "disable",
					MaxOpenConns:    20,
					MaxIdleConns:    10,
					ConnMaxLifetime: 3600,
				},
				Redis: RedisConfig{
					Host:         "localhost",
					Port:         6379,
					Password:     "",
					DB:           0,
					PoolSize:     10,
					MinIdleConns: 5,
					KeyPrefix:    "registry:",
				},
				Auth: AuthConfig{
					JWT: JWTConfig{
						AccessTokenExpire:  7200,
						RefreshTokenExpire: 604800,
						Secret:             "",
					},
					PAT: PATConfig{
						Prefix: "cyp_",
						Expire: 2592000,
					},
					BcryptCost: 10,
				},
				Storage: StorageConfig{
					Type: "local",
					Local: LocalStorage{
						RootPath: "/data/storage",
					},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
					Output: "stdout",
				},
				Scanner: ScannerConfig{
					Enabled:         true,
					Severity:        []string{"CRITICAL", "HIGH"},
					BlockOnCritical: false,
					Async:           true,
				},
			}
		}
	} else {
		// 无配置文件，使用默认配置
		cfg = &Config{}
	}

	// 环境变量覆盖（优先级最高）
	applyEnvOverrides(cfg)

	return nil
}

// Get 获取全局配置
func Get() *Config {
	return cfg
}

// applyEnvOverrides 应用环境变量覆盖
// 遵循《全平台通用容器开发设计规范》2.1节环境变量配置
func applyEnvOverrides(c *Config) {
	// 应用基础配置
	if name := os.Getenv("APP_NAME"); name != "" {
		c.App.Name = name
	}
	if host := os.Getenv("APP_HOST"); host != "" {
		c.App.Host = host
	}
	if port := os.Getenv("APP_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			c.App.Port = p
		}
	}
	if env := os.Getenv("APP_ENV"); env != "" {
		c.App.Env = env
		c.App.Debug = false // 默认关闭 debug 模式
	}

	// 数据库配置
	// 兼容规范命名（APP_DB_*）与历史命名（DB_*）
	if host := os.Getenv("APP_DB_HOST"); host != "" {
		c.Database.Host = host
	} else if host := os.Getenv("DB_HOST"); host != "" {
		c.Database.Host = host
	}
	if port := os.Getenv("APP_DB_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			c.Database.Port = p
		}
	} else if port := os.Getenv("DB_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			c.Database.Port = p
		}
	}
	if user := os.Getenv("APP_DB_USER"); user != "" {
		c.Database.Username = user
	} else if user := os.Getenv("DB_USER"); user != "" {
		c.Database.Username = user
	}
	if password := os.Getenv("APP_DB_PASSWORD"); password != "" {
		c.Database.Password = password
	} else if password := os.Getenv("DB_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if name := os.Getenv("APP_DB_NAME"); name != "" {
		c.Database.Name = name
	} else if name := os.Getenv("DB_NAME"); name != "" {
		c.Database.Name = name
	}
	if sslmode := os.Getenv("APP_DB_SSLMODE"); sslmode != "" {
		c.Database.SSLMode = sslmode
	} else if sslmode := os.Getenv("DB_SSLMODE"); sslmode != "" {
		c.Database.SSLMode = sslmode
	}

	// Redis配置
	// 兼容规范命名（APP_REDIS_*）与历史命名（REDIS_*）
	if host := os.Getenv("APP_REDIS_HOST"); host != "" {
		c.Redis.Host = host
	} else if host := os.Getenv("REDIS_HOST"); host != "" {
		c.Redis.Host = host
	}
	if port := os.Getenv("APP_REDIS_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			c.Redis.Port = p
		}
	} else if port := os.Getenv("REDIS_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			c.Redis.Port = p
		}
	}
	if password := os.Getenv("APP_REDIS_PASSWORD"); password != "" {
		c.Redis.Password = password
	} else if password := os.Getenv("REDIS_PASSWORD"); password != "" {
		c.Redis.Password = password
	}
	if db := os.Getenv("APP_REDIS_DB"); db != "" {
		var d int
		if _, err := fmt.Sscanf(db, "%d", &d); err == nil {
			c.Redis.DB = d
		}
	} else if db := os.Getenv("REDIS_DB"); db != "" {
		var d int
		if _, err := fmt.Sscanf(db, "%d", &d); err == nil {
			c.Redis.DB = d
		}
	}

	// JWT配置
	// 兼容规范命名（APP_AUTH_JWT_SECRET）与历史命名（JWT_SECRET）
	if secret := os.Getenv("APP_AUTH_JWT_SECRET"); secret != "" {
		c.Auth.JWT.Secret = secret
	} else if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.Auth.JWT.Secret = secret
	}
	if expire := os.Getenv("JWT_ACCESS_TOKEN_EXPIRE"); expire != "" {
		var exp int64
		if _, err := fmt.Sscanf(expire, "%d", &exp); err == nil && exp > 0 {
			c.Auth.JWT.AccessTokenExpire = exp
		}
	}
	if expire := os.Getenv("JWT_REFRESH_TOKEN_EXPIRE"); expire != "" {
		var exp int64
		if _, err := fmt.Sscanf(expire, "%d", &exp); err == nil && exp > 0 {
			c.Auth.JWT.RefreshTokenExpire = exp
		}
	}

	// 存储配置
	// 兼容规范命名（APP_STORAGE_*）与历史命名（STORAGE_*）
	if stype := os.Getenv("APP_STORAGE_TYPE"); stype != "" {
		c.Storage.Type = stype
	} else if stype := os.Getenv("STORAGE_TYPE"); stype != "" {
		c.Storage.Type = stype
	}
	if path := os.Getenv("APP_STORAGE_LOCAL_ROOT_PATH"); path != "" {
		c.Storage.Local.RootPath = path
	} else if path := os.Getenv("STORAGE_LOCAL_ROOT_PATH"); path != "" {
		c.Storage.Local.RootPath = path
	}
	if endpoint := os.Getenv("APP_STORAGE_MINIO_ENDPOINT"); endpoint != "" {
		c.Storage.MinIO.Endpoint = endpoint
	} else if endpoint := os.Getenv("STORAGE_MINIO_ENDPOINT"); endpoint != "" {
		c.Storage.MinIO.Endpoint = endpoint
	} else if endpoint := os.Getenv("MINIO_ENDPOINT"); endpoint != "" {
		// 兼容全局配置中心中常见命名（MINIO_*）
		c.Storage.MinIO.Endpoint = endpoint
	}
	if key := os.Getenv("APP_STORAGE_MINIO_ACCESS_KEY"); key != "" {
		c.Storage.MinIO.AccessKey = key
	} else if key := os.Getenv("STORAGE_MINIO_ACCESS_KEY"); key != "" {
		c.Storage.MinIO.AccessKey = key
	} else if key := os.Getenv("MINIO_ACCESS_KEY"); key != "" {
		c.Storage.MinIO.AccessKey = key
	}
	if key := os.Getenv("APP_STORAGE_MINIO_SECRET_KEY"); key != "" {
		c.Storage.MinIO.SecretKey = key
	} else if key := os.Getenv("STORAGE_MINIO_SECRET_KEY"); key != "" {
		c.Storage.MinIO.SecretKey = key
	} else if key := os.Getenv("MINIO_SECRET_KEY"); key != "" {
		c.Storage.MinIO.SecretKey = key
	}
	if bucket := os.Getenv("APP_STORAGE_MINIO_BUCKET"); bucket != "" {
		c.Storage.MinIO.Bucket = bucket
	} else if bucket := os.Getenv("STORAGE_MINIO_BUCKET"); bucket != "" {
		c.Storage.MinIO.Bucket = bucket
	} else if bucket := os.Getenv("MINIO_BUCKET"); bucket != "" {
		c.Storage.MinIO.Bucket = bucket
	}

	// CORS 配置（优先从环境变量覆盖，便于容器化部署与快速联调）
	// - 兼容：CORS_ALLOWED_ORIGINS=http://a,http://b
	// - 可选：CORS_ALLOWED_METHODS / CORS_ALLOWED_HEADERS（逗号分隔）
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		parts := strings.Split(origins, ",")
		clean := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				clean = append(clean, p)
			}
		}
		if len(clean) > 0 {
			c.Security.CORS.AllowedOrigins = clean
		}
	}
	if methods := os.Getenv("CORS_ALLOWED_METHODS"); methods != "" {
		parts := strings.Split(methods, ",")
		clean := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				clean = append(clean, p)
			}
		}
		if len(clean) > 0 {
			c.Security.CORS.AllowedMethods = clean
		}
	}
	if headers := os.Getenv("CORS_ALLOWED_HEADERS"); headers != "" {
		parts := strings.Split(headers, ",")
		clean := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				clean = append(clean, p)
			}
		}
		if len(clean) > 0 {
			c.Security.CORS.AllowedHeaders = clean
		}
	}

	// 日志配置
	if level := os.Getenv("LOGGING_LEVEL"); level != "" {
		c.Logging.Level = level
	}
	if format := os.Getenv("LOGGING_FORMAT"); format != "" {
		c.Logging.Format = format
	}

	// 扫描器配置
	if enabled := os.Getenv("SCANNER_ENABLED"); enabled != "" {
		c.Scanner.Enabled = (enabled == "true" || enabled == "1")
	}
	if block := os.Getenv("SCANNER_BLOCK_ON_CRITICAL"); block != "" {
		c.Scanner.BlockOnCritical = (block == "true" || block == "1")
	}
}

// Load 加载配置（供测试使用）
func Load(path string) (*Config, error) {
	if err := Init(path); err != nil {
		return nil, err
	}
	return cfg, nil
}

// GetString 获取字符串配置
func (c *Config) GetString(key string) string {
	switch key {
	case "storage.local.path":
		return c.Storage.Local.RootPath
	case "storage.type":
		return c.Storage.Type
	case "app.host":
		return c.App.Host
	case "app.env":
		return c.App.Env
	case "auth.jwt.secret":
		return c.Auth.JWT.Secret
	default:
		return ""
	}
}

// GetBool 获取布尔配置
func (c *Config) GetBool(key string) bool {
	switch key {
	case "storage.minio.use_ssl":
		return c.Storage.MinIO.UseSSL
	case "app.debug":
		return c.App.Debug
	case "registry.allow_anonymous":
		return c.Registry.AllowAnonymous
	case "security.rate_limit.enabled":
		return c.Security.RateLimit.Enabled
	default:
		return false
	}
}

// GetInt64 获取整型配置
func (c *Config) GetInt64(key string) int64 {
	switch key {
	case "registry.max_layer_size":
		return c.Registry.MaxLayerSize
	case "app.port":
		return int64(c.App.Port)
	default:
		return 0
	}
}

// Set 设置配置值
func (c *Config) Set(key string, value interface{}) {
	// 可以在此添加动态配置设置逻辑
}

// MinIOStorage MinIO存储配置（用于配置获取）
type MinIOStorageConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
	Location  string `yaml:"location"`
	PartSize  int64  `yaml:"part_size"`
}
