package middleware

import (
	"bytes"
	"io"
	"time"

	logger "github.com/PrototypeSirius/ruglogger/ruglog"
	"github.com/gin-gonic/gin"
)

func StructuredLogHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(io.LimitReader(c.Request.Body, 16384))
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		c.Next()

		latency := time.Since(start)

		fields := map[string]any{
			"protocol":   c.Request.Proto,
			"status":     c.Writer.Status(),
			"latency_ms": latency.Milliseconds(),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		if len(body) > 0 {
			fields["body"] = string(body)
		}

		if c.Writer.Status() >= 400 {
			logger.Error("Request failed", nil, 0, fields)
		} else {
			logger.Info("Request processed", fields)
		}
	}
}
