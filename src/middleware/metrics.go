// Prometheus 指标端点中间件
// 遵循《全平台通用开发任务设计规范文档》v1.0

package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP请求总数
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cyp_registry_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP请求持续时间
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cyp_registry_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// 认证失败次数（导出供其他模块使用）
	AuthFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cyp_registry_auth_failures_total",
			Help: "Total number of authentication failures",
		},
		[]string{"reason"},
	)

	// 镜像推送统计（导出供其他模块使用）
	ImagePushTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cyp_registry_image_push_total",
			Help: "Total number of image pushes",
		},
		[]string{"project", "status"},
	)

	// 镜像拉取统计（导出供其他模块使用）
	ImagePullTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cyp_registry_image_pull_total",
			Help: "Total number of image pulls",
		},
		[]string{"project", "status"},
	)

	// 漏洞扫描统计（导出供其他模块使用）
	ScanTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cyp_registry_scan_total",
			Help: "Total number of vulnerability scans",
		},
		[]string{"status", "severity"},
	)

	// 数据库连接池指标（导出供其他模块使用）
	DBConnectionPool = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cyp_registry_db_connections",
			Help: "Database connection pool status",
		},
		[]string{"state"},
	)

	// Redis连接池指标（导出供其他模块使用）
	RedisPoolSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cyp_registry_redis_pool_size",
			Help: "Redis connection pool size",
		},
	)
)

// MetricsMiddleware 返回Prometheus指标中间件
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 处理请求并在结束后记录指标
		c.Next()

		// 获取开始时间
		startRaw, ok := c.Get("request_start_time")
		var start time.Time
		if ok {
			if t, ok2 := startRaw.(time.Time); ok2 {
				start = t
			} else {
				start = time.Now()
			}
		} else {
			start = time.Now()
		}

		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		method := c.Request.Method
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		httpRequestsTotal.WithLabelValues(method, endpoint, strconv.Itoa(status)).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	}
}

// InitMetrics 初始化请求开始时间
func InitMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("request_start_time", time.Now())
		c.Next()
	}
}
