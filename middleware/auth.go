package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/atopos31/llmio/common"
	"github.com/gin-gonic/gin"
)

func Auth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := GetRequestID(c)

		// 不设置token，则不进行验证
		if token == "" {
			slog.Debug("auth_skipped_no_token_configured",
				"request_id", requestID,
			)
			return
		}
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			slog.Warn("auth_failed_missing_header",
				"request_id", requestID,
				"client_ip", c.ClientIP(),
			)
			common.ErrorWithHttpStatus(c, http.StatusUnauthorized, http.StatusUnauthorized, "Authorization header is missing")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			slog.Warn("auth_failed_invalid_format",
				"request_id", requestID,
				"client_ip", c.ClientIP(),
			)
			common.ErrorWithHttpStatus(c, http.StatusUnauthorized, http.StatusUnauthorized, "Invalid authorization header")
			c.Abort()
			return
		}

		tokenString := parts[1]
		if tokenString != token {
			slog.Warn("auth_failed_invalid_token",
				"request_id", requestID,
				"client_ip", c.ClientIP(),
			)
			common.ErrorWithHttpStatus(c, http.StatusUnauthorized, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		slog.Debug("auth_success",
			"request_id", requestID,
		)
	}
}

func AuthAnthropic(koken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := GetRequestID(c)

		// 不设置token，则不进行验证
		if koken == "" {
			slog.Debug("anthropic_auth_skipped_no_token_configured",
				"request_id", requestID,
			)
			return
		}
		authHeader := c.GetHeader("x-api-key")
		if authHeader == "" {
			slog.Warn("anthropic_auth_failed_missing_header",
				"request_id", requestID,
				"client_ip", c.ClientIP(),
			)
			common.ErrorWithHttpStatus(c, http.StatusUnauthorized, http.StatusUnauthorized, "x-api-key header is missing")
			c.Abort()
			return
		}
		if authHeader != koken {
			slog.Warn("anthropic_auth_failed_invalid_token",
				"request_id", requestID,
				"client_ip", c.ClientIP(),
			)
			common.ErrorWithHttpStatus(c, http.StatusUnauthorized, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		slog.Debug("anthropic_auth_success",
			"request_id", requestID,
		)
	}
}
