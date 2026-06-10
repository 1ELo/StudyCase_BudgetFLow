package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// StructuredLogger is a Gin middleware that logs HTTP requests in JSON format using slog.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		if raw != "" {
			path = path + "?" + raw
		}

		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Set log level based on status code
		level := slog.LevelInfo
		if status >= 400 && status < 500 {
			level = slog.LevelWarn
		} else if status >= 500 {
			level = slog.LevelError
		}

		slog.Log(c.Request.Context(), level, "HTTP Request",
			slog.Int("status", status),
			slog.String("method", method),
			slog.String("path", path),
			slog.String("ip", clientIP),
			slog.String("latency", latency.String()),
			slog.String("user_agent", userAgent),
			slog.String("error", errorMessage),
		)
	}
}
