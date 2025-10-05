package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestIDKey 是存储请求ID的上下文键
const RequestIDKey = "request_id"

// RequestLogger 创建请求日志中间件
// 为每个请求生成唯一ID并记录请求/响应信息
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成唯一请求ID
		requestID := uuid.New().String()
		c.Set(RequestIDKey, requestID)

		// 记录请求开始
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		slog.Info("request_started",
			"request_id", requestID,
			"method", method,
			"path", path,
			"client_ip", clientIP,
			"user_agent", c.Request.UserAgent(),
		)

		// 处理请求
		c.Next()

		// 记录请求完成
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		logLevel := slog.LevelInfo
		if statusCode >= 500 {
			logLevel = slog.LevelError
		} else if statusCode >= 400 {
			logLevel = slog.LevelWarn
		}

		slog.Log(c.Request.Context(), logLevel, "request_completed",
			"request_id", requestID,
			"method", method,
			"path", path,
			"status", statusCode,
			"duration_ms", duration.Milliseconds(),
			"client_ip", clientIP,
		)
	}
}

// GetRequestID 从gin.Context中获取请求ID
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
